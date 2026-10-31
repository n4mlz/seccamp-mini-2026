---
title: 11. トラブルシューティング
description: ハンズオンで起きやすいエラーと、その対処
---

ハンズオンで詰まりやすい点をまとめます。ここに載っていないエラーは、講師やチューターに聞いてください。

## テストが動かない

`the tests need root privileges` と出る場合は、`go test` を直接実行していないでしょうか。テストには root 権限が必要なので、`make test` を使ってください。`make test` は、ビルドまでを自分のユーザーで行って、テストのバイナリだけを `sudo` で実行します。

`rootfs not found: run make rootfs first` と出る場合は、`make rootfs` をまだ実行していません。ステップ 2 以降で必要です。途中で失敗して、`rootfs` が中途半端な状態で残ってしまったときは、`sudo rm -rf rootfs` で消してから、やり直してください。`make rootfs` が Docker のエラーで失敗する場合は、`make rootfs-download` を試してください。

ステップ 6 で `golang.org/x/sys` に関するエラーが出る場合は、ライブラリの取得に失敗しています。ネットワークにつながっているかを確認して、`go mod download` を試してみてください。

## コンテナが起動しない

`operation not permitted` と出る場合は、まず `sudo` を付けているかを確認してください。付けているのに出るなら、Dev Container の外で実行していないでしょうか。namespace やマウントの操作は、普通のコンテナの中では許可されていないことがあります。

`create cgroup` に関するエラーが出る場合は、cgroup (v2) が使えない状態かもしれません。次のコマンドで確認します。

```sh
cat /sys/fs/cgroup/cgroup.controllers
mount | grep cgroup
```

`cgroup2` が表示されて、`memory` が含まれていれば使えています。表示されない場合は、Docker Desktop や WSL2 の設定が原因のことがあるので、講師に相談してください。

`open .../memory.max: no such file or directory` と出る場合は、`memory` の controller が有効になっていません。Dev Container の中では、普通は有効になっています。

```sh
cat /sys/fs/cgroup/cgroup.subtree_control
```

ここに `memory` が含まれているか確認してください。

ステップ 5 で `ip link add` が `Operation not supported` になる場合は、`veth` を作る機能が使えない状態です。Dev Container を作り直してみてください。

## 後片付け

プロセスが途中で終了したときなどに、次のものが残ってしまうことがあります。

cgroup は、`/sys/fs/cgroup/container-*` というディレクトリが残ります。中のプロセスがいないことを確認して、`sudo rmdir /sys/fs/cgroup/container-*` で消せます。veth は、`ip link` に `veth...` が残っていたら、`sudo ip link del veth...` で消せます。プロセスが残っている場合は、`ps aux | grep container` で探して、`sudo kill` で終了してください。

namespace の中で行ったマウントは、プロセスが終わると自動で消えるので、ホスト側に残ることはありません。

## 環境ごとの注意

Windows (WSL2) や macOS (Docker Desktop) では、Dev Container は Linux の仮想マシンの中で動きます。そのため、ここで使っている Linux の機能は、その仮想マシンのカーネルで動くことになります。Apple Silicon (arm64) でも動くようにコードを書いていますが、`make rootfs` ではイメージを取得するので、ネットワークが遅いと時間がかかります。
