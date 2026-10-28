---
title: 1. Go の基本
description: Go のチュートリアルと CLI プログラムの作成
---

当日の演習は Go で進めます。できるだけ易しいコードにしますが、変数、関数、条件分岐、繰り返し、エラー処理を読める程度には慣れておくと安心です。

## Go の環境構築

ここではせっかくなので、Docker を利用して Go の環境構築を一瞬で済ませてみましょう。

まず、Docker をインストールします。[公式のインストール手順](https://docs.docker.com/get-started/get-docker/) に従って、自分の使っているOS 向けのものをインストールしてください。インストールできたら、次のコマンドで動作を確認します。

```sh
docker run --rm hello-world
```

`Hello from Docker!` と表示されれば成功です。

次に、Go の公式イメージからコンテナを起動し、その中に入ります。

```sh
docker run -d --name go-env golang:1 sleep infinity
docker exec -it go-env bash
```

コンテナの中で、Go が使えることを確認します。

```sh
go version
```

Go のバージョンが表示されれば、環境構築は完了です。コンテナから出るときは `exit` を実行します。

## A Tour of Go

Go でのプログラミングに不安がある場合は、公式の [A Tour of Go](https://go.dev/tour/) から、「Using the tour」と「Basics」まで進めてください。ブラウザだけで進められます。

## 簡単な CLI を作る

Go のチュートリアルが終わったら、練習がてらに小さな CLI[^1] を作ってみましょう。以下は、コマンドを受け取って実行するプログラムです。講義で実装するコンテナランタイムも CLI として実装していくことになります。

コンテナの中で、次のコマンドを実行します。

```sh
cat > main.go
```

入力待ちになるので、下のコードを貼り付けて、改行したあとに `Ctrl+D` を押します。`main.go` が作成されます。

```go title="main.go"
package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: my-cli <command> [args...]")
		os.Exit(1)
	}

	// go run main.go echo hello の場合、os.Args[1] は "echo"、os.Args[2:] は ["hello"]
	// exec.Command("実行するプログラム", "引数1", "引数2", "引数3") のようにするとコマンドを実行できる
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

そのまま、次のように動かします。

```sh
go run main.go echo hello
go run main.go ls -l /
```

`hello` と `/` の中身が表示されれば成功です。

余裕があれば、各行が何をしているかをコメントに書き出したり、存在しないコマンドを渡してエラーを観察したりしてみてください。

当日の演習は、[開発環境](../environment/) で用意する Dev Container の上で行います。確認が終わったら、`docker rm -f go-env` でコンテナを削除して構いません。

[^1]: Command Line Interface の略。ターミナルから呼び出して実行するプログラム。
