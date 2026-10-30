---
title: "10. ステップ 6: capability と seccomp (発展)"
description: root が使える権限と、呼び出せるシステムコールを絞る
---

ステップ 4 と同じく、講義の時間にはやらないステップで、これが最後です。コンテナの中の `root` が持っている権限と、コンテナの中から呼び出せるシステムコールを、絞っていきます。

## ゴール

`-drop-caps` で `root` の権限を必要最小限に絞って、`-seccomp` で危険なシステムコールを呼び出せないようにします。

```text
$ sudo ./container run -rootfs rootfs -drop-caps mount -t tmpfs tmpfs /mnt
mount: permission denied (are you root?)
$ sudo ./container run -rootfs rootfs -seccomp unshare -u true
unshare: unshare(0x4000000): Operation not permitted
```

## 説明

### capability

Linux では、`root` が持つ「何でもできる」権限が、細かい単位に分けられています。この1つ1つを capability と呼びます。例えば、マウントする権限 (`CAP_SYS_ADMIN`)、ファイルの持ち主を変える権限 (`CAP_CHOWN`)、ネットワークの設定を変える権限 (`CAP_NET_ADMIN`) などです。

ここまでのコンテナの中の `root` は、全ての capability を持っていました。ステップ 4 の user namespace のおかげで、その権限がホストには及ばなくなっていますが、コンテナの中では強力なままです。普通のアプリケーションには要らない capability は、取り上げてしまいましょう。

プロセスの capability には、いくつか種類があります。今回注目するのは、現在持っている capability と、`exec` で別のプログラムに置き換わるときに得られる capability の上限 (bounding set) です。新しいプログラムが持てる capability は、bounding set に入っているものに限られるので、bounding set から不要な capability を外しておけば、`exec` した後のコマンドの権限が下がります。

### seccomp

seccomp は、プロセスが呼び出せるシステムコールを、フィルタで制限する仕組みです。フィルタは BPF (Berkeley Packet Filter) という小さなプログラムで書いて、システムコールの番号を見て、許可するか、`EPERM` (許可されていない) で失敗させるかを返します。

capability が「何をする権限があるか」の制限なら、seccomp は「どのシステムコールを呼べるか」の制限です。観点が違うので、両方を組み合わせて守りを厚くします。

## 手順

`security.go` を新しく作って、`dropCapabilities()` と `applySeccomp()` を書きます。前者は、残す capability 以外を bounding set から外して、自分自身の capability も同じ集合に絞ります。後者は、呼び出しを禁止するシステムコールを `EPERM` で失敗させるフィルタを設定します。`container.go` の `child` では、`exec` の直前に、`-drop-caps` と `-seccomp` に応じてこれらを呼びます。

capability と seccomp の設定は、スレッド単位のものです。Go のプログラムは複数のスレッドで動いているので、`child` の最初で `runtime.LockOSThread()` を呼んで、設定と `exec` が同じスレッドで行われるようにします。

<details>
<summary>ヒント 1</summary>

capability と seccomp の設定には、`golang.org/x/sys/unix` パッケージの `Prctl` や `Capset` を使います。`go.mod` には、すでに依存関係を書いてあります。

bounding set から外すには、外したい capability ごとに `unix.Prctl(unix.PR_CAPBSET_DROP, capability の番号, 0, 0, 0)` を呼びます。seccomp のフィルタは `unix.SockFilter` を並べて書いて、`unix.Prctl(unix.PR_SET_SECCOMP, unix.SECCOMP_MODE_FILTER, ...)` で設定します。その前に `PR_SET_NO_NEW_PRIVS` を設定しておく必要があります。

</details>

<details>
<summary>ヒント 2</summary>

```diff title="container.go"
 func child(opts options) error {
+	runtime.LockOSThread()
+
 	...
+	if opts.dropCaps {
+		if err := dropCapabilities(); err != nil {
+			return err
+		}
+	}
+	if opts.seccomp {
+		if err := applySeccomp(); err != nil {
+			return err
+		}
+	}
+
 	path, err := exec.LookPath(opts.cmd[0])
```

seccomp のフィルタは、次の流れで判定します。まずシステムコールの番号を読み込んで、禁止リストの1つ1つと比べます。一致したら「失敗させる」にジャンプして、どれにも一致しなければ「許可」します。

</details>

<details>
<summary>答え</summary>

```go title="container.go"
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// run は "container run" で実行される。
// コンテナを起動して、その終了を待ち、コンテナの終了コードを返す。
func run(opts options) (int, error) {
	// 資源の制限が指定されていれば、cgroup を作る。
	cg, err := newCgroup(opts)
	if err != nil {
		return 1, err
	}
	if cg != nil {
		defer cg.Close()
	}

	// 自分自身を "child" サブコマンドで再実行して、新しい namespace の中で動かす。
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	// 標準入出力は、そのまま引き継ぐ。
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	// clone のフラグ。作りたい namespace を、ここに足していく。
	// まずは PID namespace。
	flags := uintptr(syscall.CLONE_NEWPID)
	// UTS namespace: ホスト名を分ける。
	flags |= syscall.CLONE_NEWUTS
	// mount namespace: マウントを分ける。
	flags |= syscall.CLONE_NEWNS
	if opts.net {
		// network namespace: ネットワークを分ける。
		flags |= syscall.CLONE_NEWNET
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: flags}
	if cg != nil {
		// 子プロセスを、最初から cgroup の中で起動する。
		cmd.SysProcAttr.UseCgroupFD = true
		cmd.SysProcAttr.CgroupFD = int(cg.dir.Fd())
	}
	if opts.userns {
		// コンテナの中の root を、ホストでは権限のない ID の範囲に対応させる。
		cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWUSER
		idMap := []syscall.SysProcIDMap{{ContainerID: 0, HostID: 100000, Size: 65536}}
		cmd.SysProcAttr.UidMappings = idMap
		cmd.SysProcAttr.GidMappings = idMap
		// user namespace の中で root になる。これがないと、子プロセスは権限を持てない。
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0, NoSetGroups: true}
	}

	// 親がネットワークの準備を終えるまで、子はこのパイプで待つ。
	syncR, syncW, err := os.Pipe()
	if err != nil {
		return 1, err
	}
	defer syncW.Close()
	cmd.ExtraFiles = []*os.File{syncR}

	// 子を起動する。
	if err := cmd.Start(); err != nil {
		return 1, err
	}
	// 親は、パイプの読む側を使わないので閉じる。
	syncR.Close()

	if opts.net {
		// 親の側のネットワークを準備する。
		cleanup, err := setupHostNetwork(cmd.Process.Pid)
		if err != nil {
			cmd.Process.Kill()
			cmd.Wait()
			return 1, err
		}
		defer cleanup()
	}
	// パイプを閉じて、子に準備が終わったことを知らせる。
	syncW.Close()

	return exitCode(cmd.Wait())
}

// child は "container child" で実行される。新しい namespace の中で動く。
// コンテナの準備を終えて、最後に、自分を指定されたコマンドに置き換える。
func child(opts options) error {
	// capability と seccomp は、スレッド単位の設定。
	// コマンドを exec するまで、同じ OS スレッドで動き続けるようにする。
	runtime.LockOSThread()

	// 親のネットワークの準備が終わる (パイプが閉じられる) まで待つ。
	if _, err := io.Copy(io.Discard, os.NewFile(3, "sync")); err != nil {
		return err
	}
	if opts.net {
		if err := setupContainerNetwork(); err != nil {
			return err
		}
	}

	// コンテナだけのホスト名を設定する。
	if opts.hostname != "" {
		if err := syscall.Sethostname([]byte(opts.hostname)); err != nil {
			return fmt.Errorf("set hostname: %w", err)
		}
	}

	// / を rootfs に差し替える。
	if opts.rootfs != "" {
		if err := setupRootfs(opts.rootfs); err != nil {
			return err
		}
	}

	// 権限と、呼び出せるシステムコールを絞る。
	if opts.dropCaps {
		if err := dropCapabilities(); err != nil {
			return err
		}
	}
	if opts.seccomp {
		if err := applySeccomp(); err != nil {
			return err
		}
	}

	// コマンドの場所を探して、自分をそのコマンドに置き換える (成功すると戻ってこない)。
	path, err := exec.LookPath(opts.cmd[0])
	if err != nil {
		return err
	}
	return syscall.Exec(path, opts.cmd, os.Environ())
}
```

```go title="security.go"
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
```

</details>

答えのコードには、2つ補足があります。

- `keepCaps` に挙げた 14 個が、残す capability です。Docker が標準で残すものと同じです。
- `blockedSyscalls` は、`unshare` や `ptrace`、`reboot`、カーネルモジュールの読み込みなど、普通のアプリケーションには要らないシステムコールです。Docker の標準のフィルタも、似たものを禁止しています。

## テスト

```text
$ make test STEP=6
go build -o container .
go test -c -o container.test .
sudo ./container.test -test.v -test.count=1 -test.run '^TestStep[0-6]_'
=== RUN   TestStep0_RunsCommand
--- PASS: TestStep0_RunsCommand (0.01s)
=== RUN   TestStep0_ReturnsExitCode
--- PASS: TestStep0_ReturnsExitCode (0.01s)
=== RUN   TestStep1_ProcessIsPID1
--- PASS: TestStep1_ProcessIsPID1 (0.01s)
=== RUN   TestStep2_RootfsIsAlpine
--- PASS: TestStep2_RootfsIsAlpine (0.01s)
=== RUN   TestStep3_MemoryLimitKillsProcess
--- PASS: TestStep3_MemoryLimitKillsProcess (0.11s)
=== RUN   TestStep3_ProcessUnderLimitSucceeds
--- PASS: TestStep3_ProcessUnderLimitSucceeds (0.01s)
=== RUN   TestStep3_CgroupIsRemoved
--- PASS: TestStep3_CgroupIsRemoved (0.01s)
=== RUN   TestStep4_RootIsMappedToUnprivilegedID
--- PASS: TestStep4_RootIsMappedToUnprivilegedID (0.01s)
=== RUN   TestStep4_FilesAreOwnedByMappedID
--- PASS: TestStep4_FilesAreOwnedByMappedID (0.01s)
=== RUN   TestStep5_OnlyOwnInterfacesVisible
--- PASS: TestStep5_OnlyOwnInterfacesVisible (0.03s)
=== RUN   TestStep5_CanReachHost
--- PASS: TestStep5_CanReachHost (0.04s)
=== RUN   TestStep6_DropsCapabilities
--- PASS: TestStep6_DropsCapabilities (0.01s)
=== RUN   TestStep6_SeccompBlocksSyscalls
--- PASS: TestStep6_SeccompBlocksSyscalls (0.01s)
PASS
```

## 動かして確認

まずは capability です。`-drop-caps` を付けなければ、コンテナの中で `mount` が成功します。付けると、権限がなくなって失敗します。

```text
$ sudo ./container run -rootfs rootfs sh -c 'mount -t tmpfs tmpfs /mnt && umount /mnt'
$ sudo ./container run -rootfs rootfs -drop-caps sh -c 'mount -t tmpfs tmpfs /mnt && umount /mnt'
mount: permission denied (are you root?)
```

seccomp も同じように確認します。`unshare` は新しい namespace を作るシステムコールで、`-seccomp` を付けると禁止されるので、`EPERM` で失敗します。

```text
$ sudo ./container run -rootfs rootfs unshare -u true
$ sudo ./container run -rootfs rootfs -seccomp unshare -u true
unshare: unshare(0x4000000): Operation not permitted
```

ステップ 2 の (発展) で `/proc` をマウントしていれば、プロセスの状態から設定の中身も確認できます。`CapEff` は有効な capability を16進数で表したもので、`Seccomp:` が `2` ならフィルタが有効です。

```text
$ sudo ./container run -rootfs rootfs grep CapEff /proc/self/status
CapEff:	000001ffffffffff
$ sudo ./container run -rootfs rootfs -drop-caps grep CapEff /proc/self/status
CapEff:	00000000a80425fb
$ sudo ./container run -rootfs rootfs -seccomp grep Seccomp: /proc/self/status
Seccomp:	2
```

## ここまでのまとめ

ただのプロセスを起動するところから始めて、ここまでで次のものを足してきました。

| ステップ | 足したもの | 隔離したもの |
| --- | --- | --- |
| 1 | PID namespace | プロセスの見え方 |
| 2 | mount namespace、`chroot` | ファイルシステム |
| 3 | cgroup | 使える資源の量 |
| 4 | user namespace | ユーザー、権限の範囲 |
| 5 | network namespace | ネットワーク |
| 6 | capability、seccomp | 使える権限、システムコール |

どれも、Linux カーネルの機能を組み合わせたものです。ということは、同じカーネルを共有している以上、カーネルの実装に誤りがあれば、この隔離が破られてしまう可能性があります。これについては、第3章で改めて扱います。

この先には、AppArmor や SELinux によるファイルへのアクセスの制御や、ブリッジと NAT によるインターネットへの接続 (ステップ 5 の続き)、root 権限なしでコンテナを起動する rootless コンテナなどがあります。
