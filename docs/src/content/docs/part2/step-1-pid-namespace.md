---
title: "5. ステップ 1: PID namespace"
description: PID の振り方を、コンテナごとに分ける
---

ステップ 0 では、コンテナの中から、ホストのプロセスを `kill` できてしまいました。このステップでは、PID の振り方をコンテナごとに分けて、これを防ぎます。

## ゴール

コンテナの中のプロセスの PID が `1` になって、ホストの他のプロセスは見えなくなるようにします。

```text
$ sudo ./container run sh -c 'echo $$'
1
```

## 説明

プロセスが他のプロセスに影響を与えるには、PID を指定することが多いです。例えばプロセスを kill する時には、`kill <pid>` とコマンドを実行します。もし子プロセスのシェルから他のプロセスへ影響を与えられないようにしたければ、この PID の「採番を独立」させれば良いのです。

コンテナの隔離を支えている仕組みの1つが、namespace です。プロセスから見える世界を分ける仕組みで、PID、ファイルシステム、ネットワークなど、分ける対象ごとに種類があります。このステップで使うのは、PID を分ける PID namespace です。


PID namespace の中では、PID の振り方が外とは独立します。新しく作った namespace の最初のプロセスが PID `1` になって、その中からは、外のプロセスは見えません。例えばホストの PID 16 のプロセスは、コンテナの中には存在しないことになるので、`kill 16` はできなくなります。

このステップで足すのは、図の「追加」の部分です。

```mermaid
flowchart LR
    R["run<br/>(ホスト)"] -->|"CLONE_NEWPID で clone (追加)"| C["child<br/>新しい PID namespace の PID 1"] -->|"exec (追加)"| X["指定したコマンド<br/>PID 1 のまま動く"]
```

作り方は簡単で、`clone` システムコールにフラグ `CLONE_NEWPID` を渡すだけです。Go では、`exec.Cmd` の `SysProcAttr` の `Cloneflags` に指定します。

```go
cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWPID}
```

ただし、これだけでは少し足りません。新しい namespace に入った子プロセスは、そこで隔離のための準備をしてから (準備はステップが進むと増えていきます)、本当に実行したいコマンドに置き換わる (=`exec` する) 必要があります。そこで、親 (`run`) は自分自身 (`/proc/self/exe`) を `child` というサブコマンドで起動して、起動された子 (`child`) が、準備を終えてから `exec` で指定されたコマンドになる、という形にします。

```mermaid
sequenceDiagram
    participant R as run (親)
    participant C as child (子)
    R->>C: 自分自身を child として起動 (CLONE_NEWPID)
    Note over C: 新しい PID namespace の PID 1
    C->>C: 準備をする (ステップが進むと増える)
    C->>C: exec で、指定されたコマンドに置き換わる
```

<details>
<summary>なぜ自分自身を再実行するのか</summary>

Go のランタイムは、複数のスレッド (1つのプロセスの中で、並行して動く処理の単位) を使って動いています。ところが、`fork` で子プロセスにコピーされるのは、呼び出したスレッドだけです。この状態で、Go のコードを続けて実行するのは安全ではありません。C 言語なら、`fork` した後の子プロセスでそのまま準備を続けられるのですが、Go ではそれができないので、`clone` と同時に新しいプログラムを `exec` して、namespace の中で Go のプログラムを最初から起動し直す、という形を取ります。

</details>

## 手順

`container.go` を書き換えます。`run` では、コマンドを直接起動する代わりに、`/proc/self/exe` を `child` サブコマンドで起動して、`Cloneflags` に `CLONE_NEWPID` を指定します。`child` では、`exec.LookPath` でコマンドの場所を探して、`syscall.Exec` で自分をそのコマンドに置き換えます。

次の雛形の、コメントで示した場所に実装を書きます。`-` の行は消してください。

```diff title="container.go"
 func run(opts options) (int, error) {
-	cmd := exec.Command(opts.cmd[0], opts.cmd[1:]...)
+	// 直接起動する代わりに、自分自身 (/proc/self/exe) を "child" サブコマンドで起動する
+	// (child には、run に渡された引数 os.Args[2:] をそのまま渡す)
 	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
+	// Cloneflags に、PID namespace を作るフラグを指定する
 
 	return exitCode(cmd.Run())
 }
 
 func child(opts options) error {
-	// TODO (ステップ 1): コンテナの準備を終えて、opts.cmd を実行する。
-	return errors.New("not implemented")
+	// exec.LookPath で、コマンドの場所を探す
+	// syscall.Exec で、自分をそのコマンドに置き換える
 }
```

実装できたら、`go build -o container .` でビルドが通ることを確認してください。

動きを確かめたいときは、`child` の最初に `fmt.Fprintln(os.Stderr, "child:", os.Getpid())` を足してみてください。PID namespace の中で動いているので、`1` と表示されるはずです (`fmt` を `import` する必要があります)。

<details>
<summary>ヒント 1</summary>

`run` が起動するのは `/proc/self/exe` で、引数は、`os.Args[2:]` の先頭に `"child"` を足したものです。`os.Args[0]` は実行ファイル名、`os.Args[1]` は `run` です。

`child` で `syscall.Exec(path, argv, envv)` を呼ぶと、今のプロセスが、指定したプログラムに置き換わります。`path` はコマンドの場所 (`exec.LookPath` で探せます)、`argv` は `opts.cmd`、`envv` は `os.Environ()` を渡します。

</details>

<details>
<summary>ヒント 2</summary>

実際の差分です。`+` の行を足して、`-` の行を消します。

```diff title="container.go"
--- a/container.go
+++ b/container.go
@@ -1,17 +1,23 @@
 package main
 
 import (
-	"errors"
 	"os"
 	"os/exec"
+	"syscall"
 )
 
 // run は "container run" で実行される。
 // コンテナを起動して、その終了を待ち、コンテナの終了コードを返す。
 func run(opts options) (int, error) {
-	cmd := exec.Command(opts.cmd[0], opts.cmd[1:]...)
+	// 自分自身を "child" サブコマンドで再実行して、新しい namespace の中で動かす。
+	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
 	// 標準入出力は、そのまま引き継ぐ。
 	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
+
+	// clone のフラグ。作りたい namespace を、ここに足していく。
+	// まずは PID namespace。
+	flags := uintptr(syscall.CLONE_NEWPID)
+	cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: flags}
 
 	return exitCode(cmd.Run())
 }
@@ -19,6 +25,10 @@
 // child は "container child" で実行される。新しい namespace の中で動く。
 // コンテナの準備を終えて、最後に、自分を指定されたコマンドに置き換える。
 func child(opts options) error {
-	// TODO (ステップ 1): コンテナの準備を終えて、opts.cmd を実行する。
-	return errors.New("not implemented")
+	// コマンドの場所を探して、自分をそのコマンドに置き換える (成功すると戻ってこない)。
+	path, err := exec.LookPath(opts.cmd[0])
+	if err != nil {
+		return err
+	}
+	return syscall.Exec(path, opts.cmd, os.Environ())
 }
```

</details>

<details>
<summary>答え</summary>

```go title="container.go"
package main

import (
	"os"
	"os/exec"
	"syscall"
)

// run は "container run" で実行される。
// コンテナを起動して、その終了を待ち、コンテナの終了コードを返す。
func run(opts options) (int, error) {
	// 自分自身を "child" サブコマンドで再実行して、新しい namespace の中で動かす。
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	// 標準入出力は、そのまま引き継ぐ。
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	// clone のフラグ。作りたい namespace を、ここに足していく。
	// まずは PID namespace。
	flags := uintptr(syscall.CLONE_NEWPID)
	cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: flags}

	return exitCode(cmd.Run())
}

// child は "container child" で実行される。新しい namespace の中で動く。
// コンテナの準備を終えて、最後に、自分を指定されたコマンドに置き換える。
func child(opts options) error {
	// コマンドの場所を探して、自分をそのコマンドに置き換える (成功すると戻ってこない)。
	path, err := exec.LookPath(opts.cmd[0])
	if err != nil {
		return err
	}
	return syscall.Exec(path, opts.cmd, os.Environ())
}
```

</details>

## テスト

```text
$ make test STEP=1
go build -o container .
go test -c -o container.test .
sudo ./container.test -test.v -test.count=1 -test.run '^TestStep[0-1]_'
=== RUN   TestStep0_RunsCommand
--- PASS: TestStep0_RunsCommand (0.01s)
=== RUN   TestStep0_ReturnsExitCode
--- PASS: TestStep0_ReturnsExitCode (0.01s)
=== RUN   TestStep1_ProcessIsPID1
--- PASS: TestStep1_ProcessIsPID1 (0.01s)
PASS
```

ステップ 0 のテストも一緒に実行されます。全て `PASS` になれば完了です。

## 動かして確認

コンテナの中のシェルの PID が、`1` になっています。

```text
$ sudo ./container run sh -c 'echo $$'
1
```

ステップ 0 で試した、ホストのプロセスへの `kill` もできなくなりました。ホストの PID 16 は、コンテナの中には存在しないからです。

```text
$ sleep 1000 &
[1] 16
$ sudo ./container run kill 16
kill: (16): No such process
```

## うまくいかないとき

`sudo` を付けないと、namespace を作れずに失敗します。

```text
$ ./container run echo hi
error: fork/exec /proc/self/exe: operation not permitted
```

`sudo ./container run ...` の形で実行してください。

`echo $$` が `1` にならずに、大きな数字が表示される場合は、`Cloneflags` が反映されていません。`cmd.SysProcAttr` を設定してから、`cmd.Run()` を呼べているかを確認してください。

## まだ隔離できていないもの

PID は隔離できましたが、ファイルシステムはホストのままです。悪用しようと思えばホストのファイルを書き換えたり、消したりすることができてしまいます。コンテナの中から、ホストの `/tmp` にファイルを作ってみます。

```text
$ sudo ./container run sh -c 'echo hacked > /tmp/hacked'
$ cat /tmp/hacked
hacked
```

コンテナ (子プロセスのシェル) の中で作ったファイルが、ホストにできてしまいました。同じように、`rm` でホストのファイルを消すこともできます。

原因は、コンテナの中から見えるファイルシステムが、ホストのままだということです。コンテナの `/` を、ホストとは別のものに差し替えられれば、防げそうです。次のステップで試します。

## (発展) ホスト名の隔離

やらなくても、次のステップに進めます。

もう1つ、コンテナの中からできてしまう悪さがあります。ホストの名前を勝手に変えることです。

```text
$ hostname
5bd9ee5a4b1a
$ sudo ./container run hostname ultra-vulnerable-machine-2026
$ hostname
ultra-vulnerable-machine-2026
```

ホスト名は、UTS namespace で分けられます。`CLONE_NEWUTS` を `CLONE_NEWPID` に足して、`child` の中で `syscall.Sethostname` を呼べば、コンテナだけのホスト名を持てます。`-hostname` オプションで名前を指定できるようにします (オプションの解析は用意してあります)。

<details>
<summary>答え</summary>

```go title="container.go"
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// run は "container run" で実行される。
// コンテナを起動して、その終了を待ち、コンテナの終了コードを返す。
func run(opts options) (int, error) {
	// 自分自身を "child" サブコマンドで再実行して、新しい namespace の中で動かす。
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	// 標準入出力は、そのまま引き継ぐ。
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	// clone のフラグ。作りたい namespace を、ここに足していく。
	// まずは PID namespace。
	flags := uintptr(syscall.CLONE_NEWPID)
	// UTS namespace: ホスト名を分ける。
	flags |= syscall.CLONE_NEWUTS
	cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: flags}

	return exitCode(cmd.Run())
}

// child は "container child" で実行される。新しい namespace の中で動く。
// コンテナの準備を終えて、最後に、自分を指定されたコマンドに置き換える。
func child(opts options) error {
	// コンテナだけのホスト名を設定する。
	if opts.hostname != "" {
		if err := syscall.Sethostname([]byte(opts.hostname)); err != nil {
			return fmt.Errorf("set hostname: %w", err)
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

</details>

```text
$ make test-extra STEP=1
go build -o container .
go test -c -o container.test .
sudo ./container.test -test.v -test.count=1 -test.run '^TestExtra1_'
=== RUN   TestExtra1_Hostname
--- PASS: TestExtra1_Hostname (0.01s)
PASS
```

```text
$ hostname
5bd9ee5a4b1a
$ sudo ./container run -hostname lecture hostname
lecture
$ hostname
5bd9ee5a4b1a
```
