#!/bin/sh
# Docker を使わずに、Alpine Linux の minirootfs (約 3 MB) をダウンロードして、./rootfs を作る。
# "make rootfs" が使えないときの代わりで、できあがる rootfs は同じ内容になる。
set -eu

dir="$(dirname "$0")/../rootfs"
if [ -e "$dir" ]; then
	echo "$dir already exists" >&2
	exit 0
fi

# CPU のアーキテクチャに合うものを選ぶ。
case "$(uname -m)" in
x86_64) arch=x86_64 ;;
aarch64 | arm64) arch=aarch64 ;;
*)
	echo "unsupported architecture: $(uname -m)" >&2
	exit 1
	;;
esac

# docker の alpine:3.20 と同じ、3.20 系の最新のものを使う。
base="https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/$arch"

fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -q "$1" -O "$2"
	else
		echo "curl or wget is required" >&2
		exit 1
	fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# 最新のファイル名とチェックサムは、latest-releases.yaml に書いてある。
fetch "$base/latest-releases.yaml" "$tmp/latest-releases.yaml"
file=$(awk '/^- *$/ { m = 0; f = ""; s = "" } /flavor: alpine-minirootfs/ { m = 1 } m && /^ *file:/ { f = $2 } m && /^ *sha256:/ { s = $2 } m && f != "" && s != "" { print f, s; exit }' "$tmp/latest-releases.yaml")
name=${file% *}
sum=${file#* }
if [ -z "$name" ] || [ -z "$sum" ]; then
	echo "could not find alpine-minirootfs in latest-releases.yaml" >&2
	exit 1
fi

echo "downloading $name"
fetch "$base/$name" "$tmp/$name"
echo "$sum  $tmp/$name" | sha256sum -c - >/dev/null

# 途中で失敗したら、中途半端な rootfs を残さない。
trap 'rm -rf "$tmp"; sudo rm -rf "$dir"' EXIT
sudo mkdir -p "$dir"
sudo tar -xzf "$tmp/$name" -C "$dir"
# ホストのデバイスファイルを bind マウントするための、マウントポイント (ステップ 2 の (発展) で使う)。
for n in null zero full random urandom tty; do
	sudo touch "$dir/dev/$n"
done
trap 'rm -rf "$tmp"' EXIT
echo "created $dir"
