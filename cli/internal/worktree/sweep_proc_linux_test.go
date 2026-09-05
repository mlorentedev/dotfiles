//go:build linux

package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
		if dest, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(cmd.Process.Pid), "cwd")); err == nil {
			if resolved, rerr := filepath.EvalSymlinks(dir); rerr == nil && dest == resolved {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("child never reported a cwd of %s in /proc", dir)
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

func TestGateFMatchesAProcessInASubdirectory(t *testing.T) {
	target := filepath.Join(t.TempDir(), "repo-wt-feat")
	nested := filepath.Join(target, "sub", "deep")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	startChildIn(t, nested)

	if !isHostProcessInside(target).Inside {
		t.Error("a process in a subdirectory of the worktree was not counted as inside; " +
			"a shell two levels down still loses its work when the worktree is removed")
	}
}

// The negative half, in its own test with its own root.
//
// Round 3 caught the previous version asserting nothing here: it spawned a child
// inside the target as well, then re-derived the target's own path and called
// that "the sibling", so the comparison was a tautology and its failure branch
// was a t.Skip. The naive-prefix mutation survived the whole suite.
//
// Nothing lives inside target here, which is what makes the assertion mean
// something: the only process anywhere near is in a directory whose name has
// target's name as a string prefix.
func TestGateFDoesNotMatchASiblingSharingANamePrefix(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "repo-wt-feat")
	sibling := filepath.Join(root, "repo-wt-feature-two")
	for _, d := range []string{target, sibling} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	startChildIn(t, sibling)

	if isHostProcessInside(target).Inside {
		t.Error("a process in repo-wt-feature-two was reported as inside repo-wt-feat: the " +
			"prefix test is not separator-anchored, so every worktree whose name prefixes " +
			"another one is now unreapable while nothing is in it")
	}
}

// The producer of the uninspectable count, which no test drove before round 3 —
// replacing `reading.Uninspectable++` with a no-op left the suite green, so the
// count could have pinned at 0 forever while the report claimed a reach the scan
// never had.
func TestUninspectableProcessesAreCountedByTheRealScan(t *testing.T) {
	// Anchor first: this test rests on there being at least one process whose
	// cwd this user may not read, and it must not silently become vacuous on a
	// machine where that is untrue. /proc/1 is the natural witness — it is
	// root's, and ptrace_may_access denies every non-root caller.
	//
	// The skip is narrow and it is LOUD, because a skip and a pass are the same
	// colour in CI: `go test` without -v prints neither, so a test that quietly
	// stopped running looks exactly like one that keeps passing. Whoever reads
	// this on a machine where it skipped gets told which condition did it.
	euid := os.Geteuid()
	if _, err := os.Readlink("/proc/1/cwd"); err == nil {
		t.Skipf("NOT EXECUTED: /proc/1/cwd is readable at euid=%d, so every process on this "+
			"machine is inspectable and the counter has nothing to count. This is expected "+
			"under root and nowhere else; the CI leg that pins this behaviour runs unprivileged.",
			euid)
	}

	// An empty directory, so the scan runs to completion instead of returning
	// early on the first process found inside.
	reading := isHostProcessInside(t.TempDir())

	if reading.Inside {
		t.Fatal("a freshly created temp directory reported a process inside it; the fixture " +
			"is wrong and the count below would not mean what it says")
	}
	if reading.Uninspectable == 0 {
		t.Error("the scan reported full reach over every process on a machine where " +
			"/proc/1/cwd is unreadable; an uncounted unreadable process is one the gate " +
			"silently treated as outside")
	}
}

// AC2's fail-closed branch for an unresolvable target. It needs no injection —
// only a path that does not exist — which is why round 3 rejected the claim that
// this branch could not be reached by a test.
func TestGateFRefusesWhenTheTargetCannotBeResolved(t *testing.T) {
	if got := isHostProcessInside(filepath.Join(t.TempDir(), "no-such-worktree")); !got.Inside {
		t.Error("Gate f answered \"nobody is inside\" for a path it could not resolve; the " +
			"caller deletes on that answer, so an unresolvable target must read as occupied")
	}
}

// The Linux half of AC4. The !linux half is pinned by
// TestUnsupportedPlatformDoesNotAdvertiseDiscovery on the Windows leg; this side
// had only a test that skipped when the constant was false and then asserted a
// package-level var was non-nil, which it cannot be. Flipping the constant to
// false left the suite green while sweep would have printed "no process-liveness
// check on this platform" next to a working one.
func TestProcessDiscoveryIsAdvertisedOnLinux(t *testing.T) {
	if !processDiscoverySupported {
		t.Error("Linux walks /proc and answers from real processes, but the constant says " +
			"this platform has no process discovery; sweep would announce a refusal it is " +
			"not making and explain away every empty result")
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
// the worktree, so it must read as `outside` and not as `unreadable`: /proc is a
// snapshot and processes leave during every scan.
func TestVanishedProcessCountsAsOutsideNotUnreadable(t *testing.T) {
	if got := inspectProcessCwd("4294967295", t.TempDir()); got != cwdOutside {
		t.Errorf("a non-existent pid gave %v, want cwdOutside — /proc is a snapshot and "+
			"processes exit during a scan constantly", got)
	}
}
