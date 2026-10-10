//go:build darwin

package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Gate f against the real ps and lsof, with real child processes, as the Linux
// suite runs against the real /proc: a fixture would be built to match the
// assumption under test.

// startChildIn launches a process whose cwd is dir and returns once lsof sees
// it there.
func startChildIn(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatalf("could not start a child process in %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		out, _ := exec.Command("lsof", "-a", "-p", strconv.Itoa(cmd.Process.Pid), "-d", "cwd", "-F", "n").Output()
		if strings.Contains(string(out), "\nn"+resolved+"\n") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("lsof never reported a cwd of %s for the child", resolved)
}

func TestGateFSeesAProcessLivingInTheWorktree(t *testing.T) {
	dir := t.TempDir()
	startChildIn(t, dir)
	if !isHostProcessInside(dir).Inside {
		t.Error("a live process works in this directory and Gate f did not see it; " +
			"the caller would delete the directory out from under it")
	}
}

func TestGateFDoesNotInventAProcessInAnEmptyWorktree(t *testing.T) {
	if isHostProcessInside(t.TempDir()).Inside {
		t.Error("Gate f reported a process inside a directory nothing is using; a gate " +
			"that always says yes passes the test above without discriminating")
	}
}

// t.TempDir is under /var/folders, which every process reports as
// /private/var/folders, so the plain test above already crosses a symlink. This
// one adds a link of its own, the shape of a worktree reached through one.
func TestGateFSeesAProcessInAWorktreeReachedThroughASymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	startChildIn(t, dir)
	if !isHostProcessInside(link).Inside {
		t.Error("the worktree was named through a symlink and Gate f compared the unresolved path")
	}
}

func TestParsePs_KeepsTheBaseNameOfACommandWithSpaces(t *testing.T) {
	out := []byte("  1     0 /sbin/launchd\n 501   1 /Applications/Visual Studio Code.app/Contents/MacOS/Electron\nbad line\n")
	procs := parsePs(out)
	if len(procs) != 2 {
		t.Fatalf("parsed %d processes, want 2: %v", len(procs), procs)
	}
	if p := procs[501]; p.PPID != 1 || p.Comm != "Electron" {
		t.Errorf("pid 501 = %+v", p)
	}
}

func TestParseLsofCwds_ReadsEachProcessCwd(t *testing.T) {
	out := []byte("p1\nfcwd\nn/\np42\nfcwd\nn/private/var/folders/x/T/a b\npbad\nn/ignored\np7\nfcwd\nn (deleted)\n")
	cwds := parseLsofCwds(out)
	if len(cwds) != 2 || cwds[1] != "/" || cwds[42] != "/private/var/folders/x/T/a b" {
		t.Errorf("cwds = %v", cwds)
	}
}

// With no lsof the scan cannot run, and a scan that cannot run must not read as
// "nobody is inside": the caller deletes on that answer. ps alone is on PATH.
func TestGateFFailsClosedWithoutLsof(t *testing.T) {
	bin := t.TempDir()
	if err := os.Symlink("/bin/ps", filepath.Join(bin, "ps")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if r := isHostProcessInside(t.TempDir()); !r.Inside {
		t.Errorf("Gate f answered %+v with no lsof; want Inside", r)
	}
	if _, inside := isCallerInside(t.TempDir()); inside {
		t.Error("done's walk refused with no process table; it must stay permissive")
	}
}
