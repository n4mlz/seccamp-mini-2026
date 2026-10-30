---
title: "4. ステップ 0: プロセスの起動"
description: Go から子プロセスを起動する
---

ここから、コンテナランタイムを作り始めます。最初のステップは、ただのプロセスを起動するだけです。

## ゴール

`./container run` に渡したコマンドを子プロセスとして起動して、その終了を待ち、終了コードを返せるようにします。

```text
$ ./container run echo hello
hello
```

## 説明

シェルは、コマンドを実行するときに `fork` と `exec` を使って子プロセスを作っていました。Go では、`os/exec` パッケージの `exec.Command` を使えば、同じことができます。

これから作っていくのは、シェルを起動するためのプログラムです。ステップが進むごとに、起動したシェルを少しずつ閉じ込めていきます。といっても今の時点では、ただ子プロセスを起動しているだけなので、何も閉じ込められていません。どのくらい無防備なのかは、このステップの最後に試してみます。

## 手順

`container.go` の `run` 関数を実装します。コマンドライン引数の解析 (`options.go`) と、プロセスの終了状態を終了コードに変換する `exitCode` 関数 (`main.go`) は、あらかじめ用意してあります。

`opts.cmd` から `exec.Command` でコマンドを作って、標準入出力を自分のものにつなぎ、実行して終了を待ちます。最後に、その結果を `exitCode` に渡して返します。

<details>
<summary>ヒント 1</summary>

`opts.cmd` は、コマンドと引数を並べた文字列のスライスで、1つ目がコマンド名、残りが引数です。

`exec.Command(name, args...)` で作ったコマンドは、`Stdin`、`Stdout`、`Stderr` に `os.Stdin` などを代入すると、自分の入出力につながります。`Run()` で、実行して終了を待てます。

</details>

<details>
<summary>ヒント 2</summary>

```diff title="container.go"
 import (
 	"errors"
+	"os"
+	"os/exec"
 )
 
 func run(opts options) (int, error) {
-	// TODO (ステップ 0): opts.cmd を子プロセスとして起動して、終了を待ち、その終了コードを返す。
-	return 1, errors.New("not implemented")
+	cmd := exec.Command( /* コマンド名 */ , /* 引数 */ ...)
+	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
+	return exitCode( /* 実行して、終了を待つ */ )
 }
```

</details>

<details>
<summary>答え</summary>

```go title="container.go"
package main

import (
	"errors"
	"os"
	"os/exec"
)

// run は "container run" で実行される。
// コンテナを起動して、その終了を待ち、コンテナの終了コードを返す。
func run(opts options) (int, error) {
	cmd := exec.Command(opts.cmd[0], opts.cmd[1:]...)
	// 標準入出力は、そのまま引き継ぐ。
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	return exitCode(cmd.Run())
}

// child は "container child" で実行される。新しい namespace の中で動く。
// コンテナの準備を終えて、最後に、自分を指定されたコマンドに置き換える。
func child(opts options) error {
	// TODO (ステップ 1): コンテナの準備を終えて、opts.cmd を実行する。
	return errors.New("not implemented")
}
```

</details>

## テスト

```text
$ make test STEP=0
go build -o container .
go test -c -o container.test .
sudo ./container.test -test.v -test.count=1 -test.run '^TestStep[0-0]_'
=== RUN   TestStep0_RunsCommand
--- PASS: TestStep0_RunsCommand (0.01s)
=== RUN   TestStep0_ReturnsExitCode
--- PASS: TestStep0_ReturnsExitCode (0.01s)
PASS
```

2つとも `PASS` になれば、このステップは完了です。

## 動かして確認

ビルドして、実際にコマンドを実行してみます。これから先のステップでは root 権限が必要になるので、今から `sudo` を付けておきます。

```text
$ go build -o container .
$ sudo ./container run echo hello
hello
```

起動した子プロセスが、ホストから見えることも確認してみます。ターミナルを2つ開いて、片方で `sleep` を起動します。

```text
$ sudo ./container run sleep 1000
```

もう片方で `ps` を実行すると、`container` と、その子の `sleep` が見つかります。

```text
$ ps -o pid,ppid,comm | grep -E 'container|sleep'
     12       1 container
     19      12 sleep
```

`sleep` の親が `container` になっているので、`container` が作った子プロセスだということが分かります (数字は環境によって変わります)。この `sleep` の PID を指定して `kill` すれば、終了させることもできます。

## まだ隔離できていないもの

このコンテナは、ただの子プロセスなので、ホストとは何も隔離されていません。悪用することを考えると、例えば、ホストで動いている別のプロセスを、コンテナの中から終了させることはできるでしょうか。ホストで `sleep` を動かしておいて、その PID を指定して、コンテナの中から `kill` してみます。

```text
$ sleep 1000 &
[1] 16
$ sudo ./container run kill 16
$
[1]+  Terminated              sleep 1000
```

できてしまいました。PID さえ分かれば、コンテナの中から、ホストのプロセスを好きなように終了できてしまいます。他のプロセスは PID で指定するわけですから、**PID の振り方をコンテナごとに分けてしまえば、この問題は防げそうです。** 次のステップで試してみましょう。
