# STEP でテストを選ぶ。例えば "make test STEP=2" は、ステップ 0 から 2 までのテストを実行する。
STEP ?= 6

.PHONY: build rootfs test test-extra

build:
	go build -o container .

rootfs:
	./scripts/setup-rootfs.sh

# テストには root 権限が必要。ビルドまでは自分のユーザーで行って、テストのバイナリだけを sudo で実行する。
test: build
	go test -c -o container.test .
	sudo ./container.test -test.v -test.count=1 -test.run '^TestStep[0-$(STEP)]_'

# "make test-extra STEP=2" は、ステップ 2 の (発展) のテストを実行する。
test-extra: build
	go test -c -o container.test .
	sudo ./container.test -test.v -test.count=1 -test.run '^TestExtra$(STEP)_'
