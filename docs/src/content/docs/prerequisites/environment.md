---
title: 2. 開発環境の準備
description: 演習用の環境構築と動作確認
---

コンテナの挙動は、動かすマシンのカーネルの設定によって変わることがあります。当日これらで躓かないように、事前に環境を用意して、動作を確かめておきましょう。

演習環境には、Dev Containers を使います。VS Code の拡張機能 [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) と [Docker](https://docs.docker.com/get-started/get-docker/) をインストールしておいてください。

演習用のリポジトリを Git を用いて clone します。

```sh
git clone https://github.com/n4mlz/seccamp-mini-2026.git
```

VS Code で clone したディレクトリを開き、コマンドパレット（`Ctrl+Shift+P` / `Cmd+Shift+P`）から「Dev Containers: Reopen in Container」を実行します。初回はイメージの取得に数分かかります。

コンテナが開いたら、ターミナルで次の3つのコマンドを実行してください。

```sh
go version
docker run --rm hello-world
unshare --user --map-root-user --pid --fork --mount-proc ps
```

`go version` が Go のバージョンを表示し、`hello-world` が `Hello from Docker!` を出力し、最後の `ps` の PID が `1` になっていれば成功です。特に最後のコマンドが成功しない場合は、講師やチューターに相談してください。

うまくいかない場合は、エラー文を検索してみるか、いただければ対応できます。
