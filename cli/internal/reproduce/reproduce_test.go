package reproduce

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newRepo is a clean git repository with one commit.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "t"},
		{"config", "commit.gpgsign", "false"},
	} {
		run(t, dir, args...)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", "a.txt")
	run(t, dir, "commit", "-q", "-m", "init")
	return dir
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// The exit code decides the state, except where the command never ran as
// designed: a start failure, 126 or 127 from a shell, a signal or a timeout
// is could-not-run, never reproduced, although each is non-zero.
func TestClassify(t *testing.T) {
	for _, tc := range []struct {
		name    string
		exit    int
		started bool
		timeout bool
		want    State
	}{
		{"did not start", -1, false, false, CouldNotRun},
		{"not executable", 126, true, false, CouldNotRun},
		{"not found by a shell", 127, true, false, CouldNotRun},
		{"killed by a signal", -1, true, false, CouldNotRun},
		{"timed out", 1, true, true, CouldNotRun},
		{"passed", 0, true, false, DidNotReproduce},
		{"failed", 1, true, false, Reproduced},
		{"failed with 128", 128, true, false, Reproduced},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := classify(tc.exit, tc.started, tc.timeout); got != tc.want {
				t.Errorf("classify(%d, started=%v, timeout=%v) = %q, want %q", tc.exit, tc.started, tc.timeout, got, tc.want)
			}
		})
	}
}

// Option A (#1062): a reproduction runs against committed code. An untracked
// test or an edited file would let uncommitted code stand in for it.
func TestRunRefusesADirtyTree(t *testing.T) {
	for name, dirty := range map[string]func(dir string) error{
		"untracked": func(dir string) error { return os.WriteFile(filepath.Join(dir, "new_test.txt"), nil, 0o600) },
		"modified":  func(dir string) error { return os.WriteFile(filepath.Join(dir, "a.txt"), []byte("two\n"), 0o600) },
	} {
		t.Run(name, func(t *testing.T) {
			dir := newRepo(t)
			if err := dirty(dir); err != nil {
				t.Fatal(err)
			}
			_, err := Run(context.Background(), Options{Dir: dir, Command: []string{"git", "status"}})
			if !errors.Is(err, ErrDirty) {
				t.Fatalf("err = %v, want ErrDirty", err)
			}
		})
	}
}

func TestRunClassifiesACommittedReproduction(t *testing.T) {
	dir := newRepo(t)
	head := run(t, dir, "rev-parse", "HEAD")
	for _, tc := range []struct {
		name    string
		command []string
		want    State
	}{
		{"the check fails", []string{"git", "rev-parse", "--verify", "-q", "refs/heads/no-such-branch"}, Reproduced},
		{"the check passes", []string{"git", "status"}, DidNotReproduce},
		{"the check cannot start", []string{"tool017-no-such-binary"}, CouldNotRun},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Run(context.Background(), Options{Dir: dir, Command: tc.command})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got.State != tc.want {
				t.Errorf("State = %q, want %q (exit %d)", got.State, tc.want, got.Exit)
			}
			if got.Head != head {
				t.Errorf("Head = %q, want %q: the record must name the commit it ran on", got.Head, head)
			}
		})
	}
}

// A hung check is killed at the timeout and reads as could-not-run: the
// kill's non-zero exit must never be recorded as a reproduction.
func TestRunTimesOutAsCouldNotRun(t *testing.T) {
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep binary")
	}
	got, err := Run(context.Background(), Options{Dir: newRepo(t), Command: []string{sleep, "5"}, Timeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.State != CouldNotRun {
		t.Errorf("State = %q, want %q (exit %d)", got.State, CouldNotRun, got.Exit)
	}
}
