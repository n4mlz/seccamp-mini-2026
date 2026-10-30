package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// devices は、ホストから bind マウントするデバイスファイル。
// user namespace の中ではデバイスファイルを作れないので、ホストのものを使う。
var devices = []string{"null", "zero", "full", "random", "urandom", "tty"}

// setupRootfs は、rootfs をコンテナの / にする。
func setupRootfs(rootfs string) error {
	rootfs, err := filepath.Abs(rootfs)
	if err != nil {
		return err
	}

	// 以降のマウントが、ホストに伝わらないようにする。
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("make mounts private: %w", err)
	}
	// pivot_root は、新しい / がマウントポイントであることを要求するので、自分自身に bind マウントしておく。
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount rootfs: %w", err)
	}

	// /proc には、この PID namespace のプロセスだけが見える proc をマウントする。
	proc := filepath.Join(rootfs, "proc")
	if err := syscall.Mount("proc", proc, "proc", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, ""); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}
	// デバイスファイルを、ホストから持ってくる。
	for _, name := range devices {
		if err := bindDevice(rootfs, name); err != nil {
			return err
		}
	}

	// pivot_root(".", ".") は、古い / を新しい / の上に重ねる。
	// 古い / の置き場所のディレクトリが要らず、その後に切り離せば、ホストのファイルが見えなくなる。
	if err := os.Chdir(rootfs); err != nil {
		return err
	}
	if err := syscall.PivotRoot(".", "."); err != nil {
		return fmt.Errorf("pivot_root: %w", err)
	}
	if err := syscall.Unmount(".", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root: %w", err)
	}
	return os.Chdir("/")
}

// bindDevice は、ホストの /dev/name を、rootfs の中で見えるようにする。
func bindDevice(rootfs, name string) error {
	target := filepath.Join(rootfs, "dev", name)
	f, err := os.OpenFile(target, os.O_CREATE|os.O_RDONLY, 0o644)
	if err != nil {
		return err
	}
	f.Close()
	if err := syscall.Mount("/dev/"+name, target, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("bind mount /dev/%s: %w", name, err)
	}
	return nil
}
