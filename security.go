package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// keepCaps は、コンテナの中に残す capability。これ以外は、全て取り上げる。
var keepCaps = []int{
	unix.CAP_CHOWN, unix.CAP_DAC_OVERRIDE, unix.CAP_FOWNER, unix.CAP_FSETID,
	unix.CAP_KILL, unix.CAP_SETGID, unix.CAP_SETUID, unix.CAP_SETPCAP,
	unix.CAP_NET_BIND_SERVICE, unix.CAP_NET_RAW, unix.CAP_SYS_CHROOT,
	unix.CAP_MKNOD, unix.CAP_AUDIT_WRITE, unix.CAP_SETFCAP,
}

// dropCapabilities は、呼び出したスレッドから、keepCaps 以外の capability を全て取り除く。
func dropCapabilities() error {
	var keep [2]uint32 // capability のビットマスク (32 ビットを2つ)
	for _, c := range keepCaps {
		keep[c/32] |= 1 << (c % 32)
	}

	// bounding set は、exec したコマンドが得られる capability の上限。ここから外す。
	for c := 0; c <= unix.CAP_LAST_CAP; c++ {
		if keep[c/32]&(1<<(c%32)) != 0 {
			continue
		}
		if err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(c), 0, 0, 0); err != nil && err != unix.EINVAL {
			return fmt.Errorf("drop capability %d: %w", c, err)
		}
	}

	// このスレッド自身の capability も絞る。inheritable が残っていると、
	// root としてコマンドを exec したときに、外したはずの capability が戻ってくる。
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	data := [2]unix.CapUserData{
		{Effective: keep[0], Permitted: keep[0], Inheritable: keep[0]},
		{Effective: keep[1], Permitted: keep[1], Inheritable: keep[1]},
	}
	if err := unix.Capset(&hdr, &data[0]); err != nil {
		return fmt.Errorf("capset: %w", err)
	}
	if err := unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0); err != nil {
		return fmt.Errorf("clear ambient capabilities: %w", err)
	}
	return nil
}

const (
	seccompRetKillProcess = 0x80000000
	seccompRetErrno       = 0x00050000
	seccompRetAllow       = 0x7fff0000
)

// blockedSyscalls は、EPERM で失敗させるシステムコール。
// 普通のアプリケーションはほとんど使わないが、カーネルの攻撃対象を広げてしまうもの。
var blockedSyscalls = []uint32{
	unix.SYS_UNSHARE, unix.SYS_SETNS, unix.SYS_PTRACE, unix.SYS_REBOOT,
	unix.SYS_KEXEC_LOAD, unix.SYS_INIT_MODULE, unix.SYS_FINIT_MODULE, unix.SYS_DELETE_MODULE,
	unix.SYS_BPF, unix.SYS_PERF_EVENT_OPEN, unix.SYS_KEYCTL, unix.SYS_ADD_KEY, unix.SYS_REQUEST_KEY,
}

// auditArch は、このバイナリについて seccomp が報告する、アーキテクチャの番号を返す。
func auditArch() (uint32, error) {
	switch runtime.GOARCH {
	case "amd64":
		return 0xc000003e, nil // AUDIT_ARCH_X86_64
	case "arm64":
		return 0xc00000b7, nil // AUDIT_ARCH_AARCH64
	}
	return 0, fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
}

// applySeccomp は、blockedSyscalls を EPERM で失敗させるフィルタを設定する。
// フィルタは呼び出したスレッドに適用されて、execve した後も残る。
func applySeccomp() error {
	arch, err := auditArch()
	if err != nil {
		return err
	}
	// これがないと、フィルタの設定に CAP_SYS_ADMIN が必要になる。
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("set no_new_privs: %w", err)
	}

	const (
		load = unix.BPF_LD | unix.BPF_W | unix.BPF_ABS
		jeq  = unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K
		ret  = unix.BPF_RET | unix.BPF_K
	)
	n := len(blockedSyscalls)
	filter := []unix.SockFilter{
		{Code: load, K: 4},                    // seccomp_data.arch (アーキテクチャ)
		{Code: jeq, K: arch, Jt: 1},           // 想定したアーキテクチャなら、次の行を飛ばす
		{Code: ret, K: seccompRetKillProcess}, // 想定外のアーキテクチャなら、プロセスを終了させる
		{Code: load, K: 0},                    // seccomp_data.nr (システムコールの番号)
	}
	for i, nr := range blockedSyscalls {
		// 一致したら、残りの比較と「許可」の行を飛ばして、「拒否」の行に進む。
		filter = append(filter, unix.SockFilter{Code: jeq, K: nr, Jt: uint8(n - i)})
	}
	filter = append(filter,
		unix.SockFilter{Code: ret, K: seccompRetAllow},
		unix.SockFilter{Code: ret, K: seccompRetErrno | uint32(unix.EPERM)},
	)

	prog := unix.SockFprog{Len: uint16(len(filter)), Filter: &filter[0]}
	if err := unix.Prctl(unix.PR_SET_SECCOMP, unix.SECCOMP_MODE_FILTER, uintptr(unsafe.Pointer(&prog)), 0, 0); err != nil {
		return fmt.Errorf("install seccomp filter: %w", err)
	}
	return nil
}
