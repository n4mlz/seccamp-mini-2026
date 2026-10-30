---
title: 2. コンテナを使ってみる
description: Docker でコンテナを起動し、中を触ってから終了するまで
---

コンテナを触った事がない場合は、まずはコンテナを実際に動かしてみましょう。コンテナを使ったことがある場合は、復習だと思ってください。

## コンテナの起動

以下のコマンドにより、Ubuntu のコンテナを起動して、その中に入ってみる様子を見てみます。

```text
$ docker run -it --rm ubuntu:24.04 bash
```

`ubuntu:24.04` は使うイメージで、`イメージ名:タグ` の形式で指定します。タグはバージョンの指定などに使われます。最後の `bash` は、コンテナの中で実行するコマンドです。手元にイメージが無い場合は、初回のみダウンロードに時間がかかります。

起動に成功すると、プロンプトが次のように変わります。

```text
root@3f2a1b9c8d7e:/#
```

ここがコンテナの中です。`@` の後ろの文字列はコンテナごとに変わるので、皆さんの環境では別のものが表示されるはずです。`cat /etc/os-release` を実行すると、Ubuntu 24.04 の環境になっていることが確認できます。

```text
root@3f2a1b9c8d7e:/# cat /etc/os-release
PRETTY_NAME="Ubuntu 24.04.5 LTS"
NAME="Ubuntu"
VERSION_ID="24.04"
VERSION="24.04.5 LTS (Noble Numbat)"
...(中略)...
UBUNTU_CODENAME=noble
LOGO=ubuntu-logo
root@3f2a1b9c8d7e:/#
```

## コンテナ内の操作と終了

最初に適当なファイルを作り、内容を表示します。また、せっかくなので普通は絶対やらないような操作をしてみましょう。`ls` コマンドの実行ファイルを削除します。

```text
root@3f2a1b9c8d7e:/# echo hello > /tmp/memo.txt
root@3f2a1b9c8d7e:/# cat /tmp/memo.txt
hello
root@3f2a1b9c8d7e:/# rm /bin/ls
root@3f2a1b9c8d7e:/# ls
```

最後の `ls` は、実行ファイルが無いため、次のようなエラーになります。

```text
bash: ls: command not found
root@3f2a1b9c8d7e:/#
```

普通のマシンでこんなことをすれば、`ls` という常に使うようなコマンドが使えなくなるので大惨事です。しかし、これはコンテナの中の環境であるので、全然問題ありません。`exit` でコンテナから出て、同じコマンドでもう一度コンテナを起動してみます。

```text
root@3f2a1b9c8d7e:/# exit
exit
$ docker run -it --rm ubuntu:24.04 bash
root@a81c04e9d2b5:/#
```

```text
root@a81c04e9d2b5:/# ls /tmp
root@a81c04e9d2b5:/# ls /
bin  boot  dev  etc  home  lib  lib64  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var
root@a81c04e9d2b5:/# exit
exit
$
```

最初に作っていた `/tmp/memo.txt` は存在せず、`ls` も元どおり使えています。消したはずのものが元に戻っているのは、壊れていない状態の綺麗なイメージから毎回新しくコンテナを起動しているからです。前のコンテナの中で行った変更は引き継がれません。終了したコンテナは削除されており、コンテナを動かしているマシン (ホストと呼びます) の側にも、この操作の痕跡は残っていません。

## 別の環境を起動してみる

イメージを変えれば、別の Linux ディストリビューションもすぐに使えます。次のコマンドで、以下では、Alpine Linux のコンテナが起動します。

```text
$ docker run -it --rm alpine sh
/ #
```

`cat /etc/os-release` を実行すると、Ubuntu とは別のディストリビューションであることが分かります。

```text
/ # cat /etc/os-release
NAME="Alpine Linux"
ID=alpine
VERSION_ID=3.24.3
PRETTY_NAME="Alpine Linux v3.24"
HOME_URL="https://alpinelinux.org/"
BUG_REPORT_URL="https://gitlab.alpinelinux.org/alpine/aports/-/issues"
/ # exit
$
```
