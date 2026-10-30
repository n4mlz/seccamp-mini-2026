package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

const (
	hostAddr      = "10.0.0.1/24"
	containerAddr = "10.0.0.2/24"
)

// ip は、ip コマンドを args つきで実行する。
func ip(args ...string) error {
	cmd := exec.Command("ip", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ip %v: %w", args, err)
	}
	return nil
}

// setupHostNetwork は、親が呼ぶ。veth の組を作って、片方 (名前は eth0) を
// プロセス pid の network namespace に入れて、もう片方を設定する。
// 返される関数は、ホスト側の veth を削除する。
func setupHostNetwork(pid int) (func(), error) {
	// veth の組を作る。片方は、最初から子の network namespace に入れてしまう。
	veth := fmt.Sprintf("veth%d", pid)
	if err := ip("link", "add", veth, "type", "veth", "peer", "name", "eth0", "netns", strconv.Itoa(pid)); err != nil {
		return nil, err
	}
	cleanup := func() { _ = exec.Command("ip", "link", "del", veth).Run() }
	// ホスト側の端に、IP アドレスを設定して有効にする。
	if err := ip("addr", "add", hostAddr, "dev", veth); err != nil {
		cleanup()
		return nil, err
	}
	if err := ip("link", "set", veth, "up"); err != nil {
		cleanup()
		return nil, err
	}
	return cleanup, nil
}

// setupContainerNetwork は、子が呼ぶ。新しい network namespace の中の lo と eth0 を設定する。
func setupContainerNetwork() error {
	// lo と eth0 を有効にして、eth0 に IP アドレスを設定する。
	for _, args := range [][]string{
		{"link", "set", "lo", "up"},
		{"addr", "add", containerAddr, "dev", "eth0"},
		{"link", "set", "eth0", "up"},
	} {
		if err := ip(args...); err != nil {
			return err
		}
	}
	return nil
}
