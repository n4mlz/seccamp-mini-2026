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
