package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const cgroupRoot = "/sys/fs/cgroup"

// cgroup は、コンテナが使える資源を制限する cgroup v2 のディレクトリ。
type cgroup struct {
	path string
	dir  *os.File // 子プロセスを、直接この中で起動できるように、開いたままにしておく
}

// newCgroup は、opts に従って設定した cgroup を作る。
// 制限が何も指定されていなければ、nil を返す。
func newCgroup(opts options) (*cgroup, error) {
	if opts.memoryMB == 0 && opts.pids == 0 && opts.cpus == 0 {
		return nil, nil
	}

	// 上限を書くファイルが現れるように、親の cgroup で controller を有効にしておく。
	// Dev Container ではすでに有効なので、失敗しても構わない (念のための処理)。
	_ = os.WriteFile(filepath.Join(cgroupRoot, "cgroup.subtree_control"), []byte("+memory +pids +cpu"), 0o644)

	// cgroup は、ディレクトリを作るだけで作れる。
	path := filepath.Join(cgroupRoot, fmt.Sprintf("container-%d", os.Getpid()))
	if err := os.Mkdir(path, 0o755); err != nil {
		return nil, fmt.Errorf("create cgroup: %w", err)
	}
	cg := &cgroup{path: path}
	if err := cg.configure(opts); err != nil {
		cg.Close()
		return nil, err
	}
	// 子プロセスを直接 cgroup の中で起動できるように、ディレクトリを開いておく。
	dir, err := os.Open(path)
	if err != nil {
		cg.Close()
		return nil, err
	}
	cg.dir = dir
	return cg, nil
}

func (cg *cgroup) configure(opts options) error {
	if opts.memoryMB > 0 {
		// メモリの上限 (バイト単位)。
		if err := cg.write("memory.max", strconv.Itoa(opts.memoryMB<<20)); err != nil {
			return err
		}
		// スワップに逃げられないようにする。上限を超えると、すぐに OOM killer が動く。
		if err := cg.write("memory.swap.max", "0"); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	if opts.pids > 0 {
		// プロセス (スレッド) の数の上限。
		if err := cg.write("pids.max", strconv.Itoa(opts.pids)); err != nil {
			return err
		}
	}
	if opts.cpus > 0 {
		// cpu.max は "<上限> <周期>" の形式で、単位はマイクロ秒。
		const period = 100000
		quota := int(opts.cpus * period)
		if err := cg.write("cpu.max", fmt.Sprintf("%d %d", quota, period)); err != nil {
			return err
		}
	}
	return nil
}

// write は、cgroup の制御用ファイルに value を書き込む。
func (cg *cgroup) write(name, value string) error {
	f, err := os.OpenFile(filepath.Join(cg.path, name), os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(value); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// Close は、cgroup を削除する。中のプロセスが全て終了するまでは、削除できない。
func (cg *cgroup) Close() {
	if cg.dir != nil {
		cg.dir.Close()
	}
	for i := 0; i < 50; i++ {
		if err := os.Remove(cg.path); err == nil || errors.Is(err, fs.ErrNotExist) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}
