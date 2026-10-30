package main

import (
	"errors"
	"flag"
)

// options は、"container run" のコマンドラインオプション。
// 各オプションは、ハンズオンのどれかのステップで使う。
type options struct {
	rootfs   string  // ステップ 2: ルートファイルシステムにするディレクトリ
	hostname string  // ステップ 1 (発展): コンテナの中のホスト名
	memoryMB int     // ステップ 3: メモリの上限 (MiB)
	pids     int     // ステップ 3 (発展): プロセス数の上限
	cpus     float64 // ステップ 3 (発展): CPU の上限 (例: 0.5 なら半コア)
	userns   bool    // ステップ 4: user namespace を使う
	net      bool    // ステップ 5: network namespace を使う
	dropCaps bool    // ステップ 6: capability を絞る
	seccomp  bool    // ステップ 6: seccomp のフィルタを設定する
	cmd      []string
}

func parseOptions(args []string) (options, error) {
	var o options
	fs := flag.NewFlagSet("container", flag.ContinueOnError)
	fs.StringVar(&o.rootfs, "rootfs", "", "directory to use as the root file system")
	fs.StringVar(&o.hostname, "hostname", "", "hostname inside the container")
	fs.IntVar(&o.memoryMB, "memory", 0, "memory limit in MiB")
	fs.IntVar(&o.pids, "pids", 0, "maximum number of processes")
	fs.Float64Var(&o.cpus, "cpus", 0, "CPU limit (e.g. 0.5)")
	fs.BoolVar(&o.userns, "userns", false, "use a user namespace")
	fs.BoolVar(&o.net, "net", false, "use a network namespace")
	fs.BoolVar(&o.dropCaps, "drop-caps", false, "drop capabilities")
	fs.BoolVar(&o.seccomp, "seccomp", false, "apply a seccomp filter")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	o.cmd = fs.Args()
	if len(o.cmd) == 0 {
		return o, errors.New("no command given")
	}
	return o, nil
}
