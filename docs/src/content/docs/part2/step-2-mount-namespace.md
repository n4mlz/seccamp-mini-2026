---
title: "6. ステップ 2: mount namespace と rootfs"
description: コンテナの / を、用意したファイルシステムに差し替える
---

ステップ 1 では PID を隔離できましたが、ホストのファイルが丸見えのままでした。このステップでは、コンテナの `/` を、別のファイルシステムに差し替えます。

## ゴール

`-rootfs` で指定したディレクトリが、コンテナの中から見ると `/` になるようにします。ホストのファイルは、コンテナの中から見えなくなります。

```text
$ sudo ./container run -rootfs rootfs cat /etc/os-release
NAME="Alpine Linux"
ID=alpine
VERSION_ID=3.20.10
...(中略)...
```

## 説明

コンテナの中で動いているプログラムが見ているファイルシステムは、ホストのものではなく、コンテナイメージに入っていたファイルの集まりです。この集まりを rootfs と呼びます。Alpine Linux や Ubuntu といったディストリビューションのファイル一式 (`/bin`、`/etc`、`/usr` など) から、Linux カーネルを除いたものだと考えてください。[環境構築](../setup/)で用意した `rootfs` ディレクトリが、まさにこれです。

プロセスから見える `/` を差し替えるのが、`chroot` システムコールです。`chroot("rootfs")` を呼ぶと、そのプロセスにとっての `/` が `rootfs` ディレクトリになります。

もう1つ、mount namespace も使います。これは、ファイルシステムのマウント (どのディスクやファイルシステムが、どのディレクトリに見えているか) をプロセスごとに分ける仕組みで、`CLONE_NEWNS` というフラグで作ります。このステップではまだマウントをしないのですが、(発展) で必要になるので、一緒に用意しておきます。

## 手順

`rootfs.go` を新しく作って、`setupRootfs(rootfs string) error` を実装します。`syscall.Chroot` で `/` を差し替えて、作業ディレクトリも `/` に移します。そして `container.go` で、`Cloneflags` に `CLONE_NEWNS` を足して、`child` の中で、`-rootfs` が指定されていれば `setupRootfs` を呼びます。

`chroot` した後も、カレントディレクトリは元の場所のままなので、`os.Chdir("/")` を忘れないようにしてください。

<details>
<summary>ヒント 1</summary>

`setupRootfs` では、`syscall.Chroot(rootfs)` を呼んだ直後に、`os.Chdir("/")` を呼びます。

`child` で `setupRootfs` を呼ぶのは、コマンドを `exec` する前です。`exec` した後は、自分のコードは動きません。また、`exec.LookPath` でコマンドを探すのは、`chroot` の後にします。コンテナの中のファイルから探すためです。

</details>

<details>
<summary>ヒント 2</summary>

`rootfs.go` は、新しく作るファイルです。

```go title="rootfs.go"
package main

import (
	"fmt"
	"os"
	"syscall"
)

func setupRootfs(rootfs string) error {
	if err := syscall.Chroot( /* 新しい / */ ); err != nil {
		return fmt.Errorf("chroot: %w", err)
	}
	return os.Chdir( /* 新しい / の中の作業ディレクトリ */ )
}
```

`container.go` は、次のように変えます。

```diff title="container.go"
 	flags := uintptr(syscall.CLONE_NEWPID)
+	flags |= /* mount namespace を作るフラグ */
 	cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: flags}
 
 func child(opts options) error {
+	if opts.rootfs != "" {
+		if err := setupRootfs(opts.rootfs); err != nil {
+			return err
+		}
+	}
+
 	path, err := exec.LookPath(opts.cmd[0])
```

</details>

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
	// mount namespace: マウントを分ける。
	flags |= syscall.CLONE_NEWNS
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

	// / を rootfs に差し替える。
	if opts.rootfs != "" {
		if err := setupRootfs(opts.rootfs); err != nil {
			return err
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

```go title="rootfs.go"
package main

import (
	"fmt"
	"os"
	"syscall"
)

// setupRootfs は、rootfs をコンテナの / にする。
func setupRootfs(rootfs string) error {
	// / を rootfs に差し替える。
	if err := syscall.Chroot(rootfs); err != nil {
		return fmt.Errorf("chroot: %w", err)
	}
	// chroot しても、カレントディレクトリは元の場所のままなので、新しい / に移す。
	return os.Chdir("/")
}
```

</details>

## テスト

```text
$ make test STEP=2
go build -o container .
go test -c -o container.test .
sudo ./container.test -test.v -test.count=1 -test.run '^TestStep[0-2]_'
=== RUN   TestStep0_RunsCommand
--- PASS: TestStep0_RunsCommand (0.01s)
=== RUN   TestStep0_ReturnsExitCode
--- PASS: TestStep0_ReturnsExitCode (0.01s)
=== RUN   TestStep1_ProcessIsPID1
--- PASS: TestStep1_ProcessIsPID1 (0.01s)
=== RUN   TestStep2_RootfsIsAlpine
--- PASS: TestStep2_RootfsIsAlpine (0.01s)
PASS
```

## 動かして確認

`-rootfs rootfs` を付けて、シェルを起動してみます。ホストの `/home` などは見えなくて、Alpine Linux のファイルだけが見えます。

```text
$ sudo ./container run -rootfs rootfs sh
sh: can't access tty; job control turned off
/ # ls /
bin    etc    lib    mnt    proc   run    srv    tmp    var
dev    home   media  opt    root   sbin   sys    usr
/ # cat /etc/os-release | head -2
NAME="Alpine Linux"
ID=alpine
/ # ls /home
/ # exit
```

最初の `can't access tty` というメッセージは、気にしなくて大丈夫です。`/ #` というプロンプトは Alpine Linux の `sh` のもので、ホストのシェルとは別物であることが分かります。

## まだ隔離できていないもの

ホストのファイルは見えなくなったので、別の悪用を考えてみます。ホストのメモリを使い切ってしまうのはどうでしょうか。次のコマンドは、シェルの変数を倍々に増やしていって、約 256 MB のメモリを使います。

```text
$ sudo ./container run -rootfs rootfs sh -c 'x=aaaaaaaaaaaaaaaa; i=0; while [ $i -lt 24 ]; do x="$x$x"; i=$((i+1)); done; echo done'
done
```

別のターミナルで `free -m` や `top` を見ていると、その間だけメモリの使用量が増えます。ここでは安全のために 24 回で止めていますが、ループを止めなければ、ホストのメモリを好きなだけ使えてしまいます。今のコンテナには、使えるメモリの上限がありません。使える資源に上限をかける方法があれば、これも止められそうです。次のステップで見ていきます。

## (発展) /proc と /dev のマウント、pivot_root

やらなくても、次のステップに進めます。

実は、ここまでのコンテナには、おかしな点があります。PID を隔離したはずなのに、コンテナの中で `ps` を実行すると、ホストのプロセスが全部見えてしまうのです (`-rootfs` を付けずに実行しています)。

```text
$ sudo ./container run ps -o pid,ppid,comm
    PID    PPID COMMAND
      1       0 bash
     23       1 container
     31      23 ps
```

`ps` は `/proc` というディレクトリを読んでプロセスの一覧を出していて、その `/proc` がホストのものだからです。`-rootfs rootfs` を付けて `/` を差し替えると、今度は逆に、何も表示されなくなります。

```text
$ sudo ./container run -rootfs rootfs ps
PID   USER     TIME  COMMAND
```

`rootfs` の中の `/proc` は空のディレクトリで、何もマウントされていないからです。

`/proc` には、`proc` というファイルシステムをマウントします。PID namespace の中でマウントした `proc` には、その namespace のプロセスだけが見えます。`/dev/null` や `/dev/zero` といったデバイスファイルも、同じようにホストのものを bind マウントして使えるようにします。

さらに、`chroot` を `pivot_root` に置き換えます。`chroot` は、プロセスから見える `/` を変えるだけで、元のファイルシステムはマウントされたままなので、権限を持つプロセスなら抜け出せることが知られています。`pivot_root` はルートファイルシステムそのものを入れ替えるので、その後で古いルートをアンマウントすれば、ホストのファイルシステムは完全に見えなくなります。

つまずきやすい点が1つあって、mount namespace を作っただけだと、マウントの変更がホストにも伝わってしまうことがあります (マウントの伝播)。これを防ぐために、最初に `/` を `private` にしておきます。

<details>
<summary>答え</summary>

```go title="rootfs.go"
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// devices は、ホストから bind マウントするデバイスファイル。
// user namespace の中ではデバイスファイルを作れないので、ホストのものを使う。
var devices = []string{"null", "zero", "full", "random", "urandom", "tty"}

// setupRootfs は、rootfs をコンテナの / にする。
func setupRootfs(rootfs string) error {
	rootfs, err := filepath.Abs(rootfs)
	if err != nil {
		return err
	}

	// 以降のマウントが、ホストに伝わらないようにする。
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("make mounts private: %w", err)
	}
	// pivot_root は、新しい / がマウントポイントであることを要求するので、自分自身に bind マウントしておく。
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount rootfs: %w", err)
	}

	// /proc には、この PID namespace のプロセスだけが見える proc をマウントする。
	proc := filepath.Join(rootfs, "proc")
	if err := syscall.Mount("proc", proc, "proc", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, ""); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}
	// デバイスファイルを、ホストから持ってくる。
	for _, name := range devices {
		if err := bindDevice(rootfs, name); err != nil {
			return err
		}
	}

	// pivot_root(".", ".") は、古い / を新しい / の上に重ねる。
	// 古い / の置き場所のディレクトリが要らず、その後に切り離せば、ホストのファイルが見えなくなる。
	if err := os.Chdir(rootfs); err != nil {
		return err
	}
	if err := syscall.PivotRoot(".", "."); err != nil {
		return fmt.Errorf("pivot_root: %w", err)
	}
	if err := syscall.Unmount(".", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root: %w", err)
	}
	return os.Chdir("/")
}

// bindDevice は、ホストの /dev/name を、rootfs の中で見えるようにする。
func bindDevice(rootfs, name string) error {
	target := filepath.Join(rootfs, "dev", name)
	f, err := os.OpenFile(target, os.O_CREATE|os.O_RDONLY, 0o644)
	if err != nil {
		return err
	}
	f.Close()
	if err := syscall.Mount("/dev/"+name, target, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("bind mount /dev/%s: %w", name, err)
	}
	return nil
}
```

</details>

```text
$ make test-extra STEP=2
go build -o container .
go test -c -o container.test .
sudo ./container.test -test.v -test.count=1 -test.run '^TestExtra2_'
=== RUN   TestExtra2_OnlyContainerProcessesVisible
--- PASS: TestExtra2_OnlyContainerProcessesVisible (0.01s)
=== RUN   TestExtra2_HostMountsAreHidden
--- PASS: TestExtra2_HostMountsAreHidden (0.01s)
PASS
```

```text
$ sudo ./container run -rootfs rootfs sh
/ # ps
PID   USER     TIME  COMMAND
    1 root      0:00 sh
    6 root      0:00 ps
/ # ls /proc | head -3
1
7
8
/ # exit
```

コンテナの中のプロセスだけが見えて、`sh` が PID `1` になっています。
