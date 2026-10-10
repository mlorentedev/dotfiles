//go:build linux || darwin

package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The real walk, not the seam: a shell sits in the worktree and runs this test
// binary as its child, which moves to / before looking. Only the ancestor walk
// can then find the shell; the cwd check cannot.
func TestCallerWalkFindsAShellSittingInTheWorktree(t *testing.T) {
	wt := t.TempDir()
	out := runWalkHelper(t, wt, wt, false)
	if !strings.HasPrefix(out, "inside ") || !strings.Contains(out, " sh ") {
		t.Fatalf("want the shell found inside %s, got %q", wt, out)
	}
}

// The process inside need not be the direct parent: here the shell in the
// worktree starts a second shell in /, which runs the binary.
func TestCallerWalkFindsAGrandparentInTheWorktree(t *testing.T) {
	wt := t.TempDir()
	out := runWalkHelper(t, wt, wt, true)
	if !strings.HasPrefix(out, "inside ") || !strings.Contains(out, " sh ") {
		t.Fatalf("want the outer shell found inside %s, got %q", wt, out)
	}
}

func TestCallerWalkIgnoresAShellOutsideTheWorktree(t *testing.T) {
	wt := t.TempDir()
	out := runWalkHelper(t, t.TempDir(), wt, true)
	if out != "outside" {
		t.Fatalf("a shell elsewhere must not block, got %q", out)
	}
}

// runWalkHelper starts `sh -c '<test binary> ...; true'` in shellDir. The
// trailing `; true` keeps sh from exec'ing the binary in its own place, so sh
// stays a live parent with its cwd in shellDir. nested puts a second sh, in /,
// between the two.
func runWalkHelper(t *testing.T, shellDir, target string, nested bool) string {
	t.Helper()
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	script := `"$0" -test.run='^TestCallerWalkHelper$'; true`
	if nested {
		script = `sh -c 'cd / && "$0" -test.run=^TestCallerWalkHelper$; true' "$0"; true`
	}
	cmd := exec.Command("sh", "-c", script, bin)
	cmd.Dir = shellDir
	cmd.Env = append(os.Environ(), "WALK_HELPER_TARGET="+resolved)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper: %v\n%s", err, b)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "WALK: ") {
			return strings.TrimPrefix(line, "WALK: ")
		}
	}
	t.Fatalf("helper printed no result:\n%s", b)
	return ""
}

// TestCallerWalkHelper is the child side of runWalkHelper, inert in a normal run.
func TestCallerWalkHelper(t *testing.T) {
	target := os.Getenv("WALK_HELPER_TARGET")
	if target == "" {
		t.Skip("runs only as runWalkHelper's child")
	}
	if err := os.Chdir("/"); err != nil {
		t.Fatal(err)
	}
	if a, inside := isCallerInside(target); inside {
		_, _ = os.Stdout.WriteString("WALK: inside " + a.Comm + " " + a.Cwd + "\n")
		return
	}
	_, _ = os.Stdout.WriteString("WALK: outside\n")
}
