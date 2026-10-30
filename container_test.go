package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// テストは、ビルドした ./container を実際に実行して確かめる。
// そのため、必ず "make test" で実行する (バイナリのビルドと、root での実行を行ってくれる)。

const rootfsDir = "rootfs"

func TestMain(m *testing.M) {
	if os.Geteuid() != 0 {
		println("the tests need root privileges: run them with `make test`")
		os.Exit(1)
	}
	if _, err := os.Stat("container"); err != nil {
		println("./container not found: run them with `make test`")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

type result struct {
	stdout, stderr string
	code           int
}

// container は "./container run args..." を実行して、その出力と終了コードを返す。
func container(t *testing.T, args ...string) result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "./container", append([]string{"run"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()

	res := result{stdout: stdout.String(), stderr: stderr.String()}
	if exitErr, ok := err.(*exec.ExitError); ok {
		res.code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run ./container: %v", err)
	}
	return res
}

// withRootfs は、args の先頭に -rootfs オプションを足したものを返す。
func withRootfs(t *testing.T, args ...string) []string {
	t.Helper()
	if _, err := os.Stat(rootfsDir); err != nil {
		t.Fatalf("%s not found: run `make rootfs` first", rootfsDir)
	}
	return append([]string{"-rootfs", rootfsDir}, args...)
}

func expectOutput(t *testing.T, res result, want string) {
	t.Helper()
	if res.code != 0 || strings.TrimSpace(res.stdout) != want {
		t.Fatalf("want exit code 0 and output %q, got exit code %d, stdout %q, stderr %q",
			want, res.code, res.stdout, res.stderr)
	}
}

// ステップ 0

func TestStep0_RunsCommand(t *testing.T) {
	expectOutput(t, container(t, "echo", "hello"), "hello")
}

func TestStep0_ReturnsExitCode(t *testing.T) {
	res := container(t, "sh", "-c", "exit 3")
	if res.code != 3 {
		t.Fatalf("want exit code 3, got %d (stderr %q)", res.code, res.stderr)
	}
}

// ステップ 1

func TestStep1_ProcessIsPID1(t *testing.T) {
	expectOutput(t, container(t, "sh", "-c", "echo $$"), "1")
}

func TestExtra1_Hostname(t *testing.T) {
	before, _ := os.Hostname()
	expectOutput(t, container(t, "-hostname", "lecture", "cat", "/proc/sys/kernel/hostname"), "lecture")
	if after, _ := os.Hostname(); after != before {
		t.Fatalf("the hostname of the host changed from %q to %q", before, after)
	}
}

// ステップ 2

func TestStep2_RootfsIsAlpine(t *testing.T) {
	res := container(t, withRootfs(t, "cat", "/etc/os-release")...)
	if res.code != 0 || !strings.Contains(res.stdout, "Alpine") {
		t.Fatalf("want an Alpine Linux root file system, got exit code %d, stdout %q, stderr %q",
			res.code, res.stdout, res.stderr)
	}
}

func TestExtra2_OnlyContainerProcessesVisible(t *testing.T) {
	res := container(t, withRootfs(t, "ps", "-o", "pid,comm")...)
	lines := strings.Split(strings.TrimSpace(res.stdout), "\n")
	if res.code != 0 || len(lines) > 3 {
		t.Fatalf("want only the processes of the container, got exit code %d, stdout %q, stderr %q",
			res.code, res.stdout, res.stderr)
	}
}

func TestExtra2_HostMountsAreHidden(t *testing.T) {
	res := container(t, withRootfs(t, "sh", "-c", "wc -l < /proc/self/mountinfo")...)
	n, err := strconv.Atoi(strings.TrimSpace(res.stdout))
	if res.code != 0 || err != nil || n > 12 {
		t.Fatalf("want at most 12 mounts, got %q (exit code %d, stderr %q)", res.stdout, res.code, res.stderr)
	}
}

// ステップ 3

// hog は、シェルの変数を倍々に増やして、メモリを使い続ける。
const hog = `x=aaaaaaaaaaaaaaaa; while true; do x="$x$x"; done`

func TestStep3_MemoryLimitKillsProcess(t *testing.T) {
	res := container(t, withRootfs(t, "-memory", "32", "sh", "-c", hog)...)
	if res.code != 137 {
		t.Fatalf("want the process to be killed (exit code 137), got exit code %d, stderr %q", res.code, res.stderr)
	}
}

func TestStep3_ProcessUnderLimitSucceeds(t *testing.T) {
	expectOutput(t, container(t, withRootfs(t, "-memory", "64", "echo", "ok")...), "ok")
}

func TestStep3_CgroupIsRemoved(t *testing.T) {
	container(t, withRootfs(t, "-memory", "64", "true")...)
	if left, _ := filepath.Glob("/sys/fs/cgroup/container-*"); len(left) != 0 {
		t.Fatalf("cgroups were not removed: %v", left)
	}
}

func TestExtra3_PidsLimit(t *testing.T) {
	// コンテナのプロセスの Go のランタイムが複数のスレッドを使うので、上限を小さくしすぎない。
	res := container(t, withRootfs(t, "-pids", "20", "sh", "-c", "for i in $(seq 30); do sleep 1 & done; wait")...)
	if res.code == 0 || !strings.Contains(res.stderr, "fork") {
		t.Fatalf("want fork to fail, got exit code %d, stderr %q", res.code, res.stderr)
	}
}

func TestExtra3_CpuLimit(t *testing.T) {
	cmd := exec.Command("./container", "run", "-cpus", "0.5", "sleep", "3")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()

	for i := 0; i < 50; i++ {
		files, _ := filepath.Glob("/sys/fs/cgroup/container-*/cpu.max")
		for _, f := range files {
			if b, _ := os.ReadFile(f); strings.TrimSpace(string(b)) == "50000 100000" {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal(`want cpu.max to be "50000 100000"`)
}

// ステップ 4

func TestStep4_RootIsMappedToUnprivilegedID(t *testing.T) {
	res := container(t, "-userns", "cat", "/proc/self/uid_map")
	if fields := strings.Fields(res.stdout); res.code != 0 || strings.Join(fields, " ") != "0 100000 65536" {
		t.Fatalf(`want uid_map "0 100000 65536", got %q (exit code %d, stderr %q)`, res.stdout, res.code, res.stderr)
	}
}

func TestStep4_FilesAreOwnedByMappedID(t *testing.T) {
	path := filepath.Join(rootfsDir, "tmp", "owner-test")
	os.Remove(path)
	defer os.Remove(path)

	res := container(t, withRootfs(t, "-userns", "touch", "/tmp/owner-test")...)
	if res.code != 0 {
		t.Fatalf("exit code %d, stderr %q", res.code, res.stderr)
	}
	out, err := exec.Command("stat", "-c", "%u", path).Output()
	if err != nil || strings.TrimSpace(string(out)) != "100000" {
		t.Fatalf("want the file to be owned by uid 100000 on the host, got %q (%v)", out, err)
	}
}

// ステップ 5

func TestStep5_OnlyOwnInterfacesVisible(t *testing.T) {
	res := container(t, withRootfs(t, "-net", "sh", "-c", "ip -o link | wc -l")...)
	expectOutput(t, res, "2")
}

func TestStep5_CanReachHost(t *testing.T) {
	res := container(t, withRootfs(t, "-net", "ping", "-c", "1", "-W", "3", "10.0.0.1")...)
	if res.code != 0 {
		t.Fatalf("ping failed: exit code %d, stdout %q, stderr %q", res.code, res.stdout, res.stderr)
	}
}

// ステップ 6

func TestStep6_DropsCapabilities(t *testing.T) {
	// マウントには CAP_SYS_ADMIN が必要。何も残らないように、マウントしたものはすぐに外す。
	probe := []string{"sh", "-c", "mount -t tmpfs tmpfs /mnt && umount /mnt"}
	if res := container(t, withRootfs(t, probe...)...); res.code != 0 {
		t.Fatalf("want mount to work without -drop-caps, got exit code %d, stderr %q", res.code, res.stderr)
	}
	res := container(t, withRootfs(t, append([]string{"-drop-caps"}, probe...)...)...)
	if res.code == 0 {
		t.Fatal("want mount to fail with -drop-caps")
	}
}

func TestStep6_SeccompBlocksSyscalls(t *testing.T) {
	// unshare(2) に必要なのは CAP_SYS_ADMIN だけなので、ここで失敗させられるのは seccomp だけ。
	if res := container(t, withRootfs(t, "unshare", "-u", "true")...); res.code != 0 {
		t.Fatalf("want unshare to work without -seccomp, got exit code %d, stderr %q", res.code, res.stderr)
	}
	res := container(t, withRootfs(t, "-seccomp", "unshare", "-u", "true")...)
	if res.code == 0 || !strings.Contains(res.stderr, "not permitted") {
		t.Fatalf("want unshare to fail with EPERM, got exit code %d, stderr %q", res.code, res.stderr)
	}
}
