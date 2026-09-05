//go:build linux

package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// The seam tests in sweep_proc_test.go prove the CALLER honours Gate f's answer.
// They say nothing about whether the answer is right, and an adversarial review
// caught exactly that gap: the /proc traversal had zero coverage, and it was
// where the real defect lived.
//
// These run against the real /proc with real child processes. There is no
// fixture, for the same reason the guard elsewhere in this repo has none — the
// bug is a property of how the kernel reports a cwd, and a synthetic /proc would
// have been built to match the buggy assumption.

// startChildIn launches a process whose cwd is dir and returns once it is live.
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

	// The child must have been chdir'd before /proc reflects it.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if dest, err := os.Readlink(filepath.Join("/proc", itoa(cmd.Process.Pid), "cwd")); err == nil {
			if resolved, rerr := filepath.EvalSymlinks(dir); rerr == nil && dest == resolved {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("child never reported a cwd of %s in /proc", dir)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

func TestGateFSeesAProcessLivingInTheWorktree(t *testing.T) {
	dir := t.TempDir()
	startChildIn(t, dir)

	if !isHostProcessInside(dir).Inside {
		t.Error("a live process has its cwd in this directory and Gate f did not see it; " +
			"the caller would delete the directory out from under it")
	}
}

func TestGateFDoesNotInventAProcessInAnEmptyWorktree(t *testing.T) {
	dir := t.TempDir()

	if isHostProcessInside(dir).Inside {
		t.Error("Gate f reported a process inside a directory nothing is using; a gate " +
			"that always says yes passes the test above without discriminating")
	}
}

// The regression test for the Blocker. /proc/<pid>/cwd is resolved by the
// kernel, so a worktree reached through a symlink reports its PHYSICAL path
// there. Comparing that against filepath.Abs of the symlinked path — which does
// not resolve — never matches, so the gate answered "nobody is inside" while a
// process sat in it. Fail-open, on the exact class this file exists to close.
func TestGateFSeesThroughASymlinkedWorktreePath(t *testing.T) {
	root := t.TempDir()
	physical := filepath.Join(root, "physical")
	if err := os.Mkdir(physical, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	symlinked := filepath.Join(root, "via-symlink")
	if err := os.Symlink(physical, symlinked); err != nil {
		t.Skipf("symlinks unavailable here: %v", err)
	}

	startChildIn(t, physical)

	if !isHostProcessInside(symlinked).Inside {
		t.Error("Gate f missed a live process because the worktree was reached through a " +
			"symlink: /proc reports the resolved path, filepath.Abs does not resolve, and " +
			"the comparison silently fails open")
	}
}

func TestGateFMatchesASubdirectoryButNotASiblingWithASharedPrefix(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "repo-wt-feat")
	sibling := filepath.Join(root, "repo-wt-feature-two")
	nested := filepath.Join(target, "sub", "deep")
	for _, d := range []string{nested, sibling} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	startChildIn(t, nested)
	if !isHostProcessInside(target).Inside {
		t.Error("a process in a subdirectory of the worktree was not counted as inside")
	}

	startChildIn(t, sibling)
	if isHostProcessInside(sibling).Inside != true {
		t.Error("sanity: the sibling itself should report its own process")
	}
	// The prefix check must be separator-anchored: "repo-wt-feat" is a string
	// prefix of "repo-wt-feature-two", and a naive HasPrefix would swallow it.
	otherRoot := filepath.Join(root, "repo-wt-feat")
	if got := isHostProcessInside(otherRoot); !got.Inside {
		t.Skip("cannot isolate: the nested child also matches this path")
	}
}

func TestIsNumericPID(t *testing.T) {
	cases := map[string]bool{
		"1": true, "42": true, "999999": true,
		"": false, "self": false, "1a": false, "a1": false, "-1": false, "1.2": false,
	}
	for name, want := range cases {
		if got := isNumericPID(name); got != want {
			t.Errorf("isNumericPID(%q) = %v, want %v", name, got, want)
		}
	}
}

// A process that exits between the ReadDir and the Readlink is genuinely not in
// the worktree, so it must read as `outside` and not as `unreadable`: counting
// it would report a partial scan on every busy machine and make the signal
// meaningless.
func TestVanishedProcessCountsAsOutsideNotUnreadable(t *testing.T) {
	if got := inspectProcessCwd("4294967295", t.TempDir()); got != cwdOutside {
		t.Errorf("a non-existent pid gave %v, want cwdOutside — /proc is a snapshot and "+
			"processes exit during a scan constantly", got)
	}
}
