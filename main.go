package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: container run [options] command [args...]")
		os.Exit(2)
	}
	// run と child は、同じコマンドライン引数を解析する。
	opts, err := parseOptions(os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	// run は利用者が実行するサブコマンドで、child は run が自分自身を再実行して呼ぶ。
	switch os.Args[1] {
	case "run":
		code, err := run(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			if code == 0 {
				code = 1
			}
		}
		os.Exit(code)
	case "child":
		// child は、成功すると指定されたコマンドに置き換わるので、失敗したときしか戻ってこない。
		if err := child(opts); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown subcommand %q\n", os.Args[1])
		os.Exit(2)
	}
}

// exitCode は、cmd.Run や cmd.Wait の結果を終了コードに変換する。
// シグナルで終了した場合は、シェルと同じように 128 + シグナル番号 にする。
func exitCode(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal()), nil
		}
		return exitErr.ExitCode(), nil
	}
	return 1, err
}
