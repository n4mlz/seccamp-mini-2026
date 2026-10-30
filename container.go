package main

import (
	"errors"
)

// run は "container run" で実行される。
// コンテナを起動して、その終了を待ち、コンテナの終了コードを返す。
func run(opts options) (int, error) {
	// TODO (ステップ 0): opts.cmd を子プロセスとして起動して、終了を待ち、その終了コードを返す。
	return 1, errors.New("not implemented")
}

// child は "container child" で実行される。新しい namespace の中で動く。
// コンテナの準備を終えて、最後に、自分を指定されたコマンドに置き換える。
func child(opts options) error {
	// TODO (ステップ 1): コンテナの準備を終えて、opts.cmd を実行する。
	return errors.New("not implemented")
}
