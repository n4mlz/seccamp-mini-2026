#!/bin/sh
# Alpine Linux のイメージから、./rootfs を作る。
set -eu

dir="$(dirname "$0")/../rootfs"
if [ -e "$dir" ]; then
	echo "$dir already exists" >&2
	exit 0
fi

id=$(docker create alpine:3.20)
trap 'docker rm "$id" >/dev/null' EXIT

sudo mkdir -p "$dir"
docker export "$id" | sudo tar -x -C "$dir"
# ホストのデバイスファイルを bind マウントするための、マウントポイント (ステップ 2 の (発展) で使う)。
for name in null zero full random urandom tty; do
	sudo touch "$dir/dev/$name"
done
echo "created $dir"
