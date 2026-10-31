---
title: 3. 環境構築
description: ハンズオンの準備と、動作の確認
---

ハンズオンは、[事前学習](../../prerequisites/environment/)で用意した Dev Container の中で行います。まだの方は、先にそちらを済ませてください。

## ブランチを切り替える

このリポジトリには、`main` と `skeleton` という2つのブランチがあります。`main` は全てのステップが完成した状態で、`skeleton` はコンテナの本体がまだ何も書かれていない出発点です。ハンズオンは `skeleton` で進めるので、Dev Container のターミナルで切り替えておきます。詰まったときは、`main` のコードが参考になります。

```text
$ git switch skeleton
```

## 動作の確認

事前学習でも試した、次の3つのコマンドを実行します。

```sh
go version
docker run --rm hello-world
unshare --user --map-root-user --pid --fork --mount-proc ps
```

最後の `ps` の PID が `1` になっていれば成功です。ステップ 3 で cgroup も使うので、こちらも確認しておきます。

```sh
cat /sys/fs/cgroup/cgroup.controllers
```

表示された中に `memory`、`pids`、`cpu` が含まれていれば大丈夫です。

## rootfs を用意する

ステップ 2 からは、コンテナの中の `/` になる、Linux のファイル一式を使います。次のコマンドで、`rootfs` ディレクトリに用意しておきます。

```sh
make rootfs
```

やっていることは、Docker で Alpine Linux のイメージを取得して、その中身のファイルを `rootfs` に取り出しているだけです。コンテナイメージの正体が、ただのファイルの集まりだということが分かると思います。少し時間がかかります。

Docker がうまく動かないときは、代わりに `make rootfs-download` でも用意できます。Alpine Linux の配布元から、3 MB ほどの `rootfs` 用のファイルを直接ダウンロードして展開するので、Docker は使いません。できあがるものは、`make rootfs` とほぼ同じです。

## テストを実行してみる

テストは `make test` で実行します。まだ何も書いていないので、失敗するのが正しい状態です。

```text
$ make test STEP=0
go build -o container .
go test -c -o container.test .
sudo ./container.test -test.v -test.count=1 -test.run '^TestStep[0-0]_'
=== RUN   TestStep0_RunsCommand
    container_test.go:77: want exit code 0 and output "hello", got exit code 1, stdout "", stderr "error: not implemented\n"
--- FAIL: TestStep0_RunsCommand (0.00s)
=== RUN   TestStep0_ReturnsExitCode
    container_test.go:83: want exit code 3, got 1 (stderr "error: not implemented\n")
--- FAIL: TestStep0_ReturnsExitCode (0.00s)
FAIL
```

`STEP=` には、いま取り組んでいるステップの番号を指定します。指定したステップまでのテストが、まとめて実行されます。(発展) のテストは、`make test-extra STEP=1` のように別で実行します。

`error: not implemented` と表示されれば、準備は完了です。

## 困ったとき

うまくいかないときは、講師やチューターに声をかけてください。よくあるエラーは、[トラブルシューティング](../troubleshooting/)にまとめてあります。
