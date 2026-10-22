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

// probeFor writes a patch changing a.txt from "one" to body, outside the
// repository, and leaves the tree clean.
func probeFor(t *testing.T, dir, body string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(t.TempDir(), "probe.diff")
	if err := os.WriteFile(patch, []byte(run(t, dir, "diff")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "checkout", "--", "a.txt")
	return patch
}

// The probe is applied for the run and reverted after it, whatever the
// check says: the tree is left byte for byte as it was found.
func TestRunWithAProbe(t *testing.T) {
	for _, tc := range []struct {
		name  string
		check []string
		want  State
	}{
		{"the check fails under the probe", []string{"git", "grep", "-q", "one", "--", "a.txt"}, Reproduced},
		{"the check passes under the probe", []string{"git", "grep", "-q", "two", "--", "a.txt"}, DidNotReproduce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := newRepo(t)
			probe := probeFor(t, dir, "two\n")
			got, err := Run(context.Background(), Options{Dir: dir, Command: tc.check, Probe: probe})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got.State != tc.want {
				t.Errorf("State = %q, want %q (exit %d, output %q)", got.State, tc.want, got.Exit, got.Output)
			}
			if st := run(t, dir, "status", "--porcelain", "--untracked-files=all"); st != "" {
				t.Errorf("the probe was not reverted:\n%s", st)
			}
		})
	}
}

// A probe that does not change the tree must never read as a clean
// result: the check is not run at all, and the state says why.
func TestRunWithAProbeThatDoesNotLand(t *testing.T) {
	for name, patch := range map[string]func(t *testing.T, dir string) string{
		"context does not match": func(t *testing.T, dir string) string {
			p := probeFor(t, dir, "two\n")
			if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("other\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			run(t, dir, "commit", "-q", "-am", "move on")
			return p
		},
		"empty patch": func(t *testing.T, _ string) string {
			p := filepath.Join(t.TempDir(), "empty.diff")
			if err := os.WriteFile(p, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			return p
		},
		"mode-only patch": func(t *testing.T, _ string) string {
			p := filepath.Join(t.TempDir(), "mode.diff")
			body := "diff --git a/a.txt b/a.txt\nold mode 100644\nnew mode 100755\n"
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			return p
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := newRepo(t)
			probe := patch(t, dir)
			got, err := Run(context.Background(), Options{Dir: dir, Command: []string{"git", "status"}, Probe: probe})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got.State != DidNotLand {
				t.Errorf("State = %q, want %q", got.State, DidNotLand)
			}
			if st := run(t, dir, "status", "--porcelain", "--untracked-files=all"); st != "" {
				t.Errorf("the tree was left changed:\n%s", st)
			}
		})
	}
}

// A check that rewrites a probed file does not leave it rewritten: the
// targets are put back as the commit has them, not by a reverse patch that
// would no longer apply.
func TestRunRestoresAFileTheCheckRewrote(t *testing.T) {
	dir := newRepo(t)
	probe := probeFor(t, dir, "two\n")
	// The check puts a.txt back itself, so a reverse patch would not apply.
	check := []string{"git", "checkout", "--", "a.txt"}
	if _, err := Run(context.Background(), Options{Dir: dir, Command: check, Probe: probe}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := run(t, dir, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the tree was left changed:\n%s", st)
	}
}

// A probe that creates a file leaves nothing behind.
func TestRunRemovesAFileTheProbeCreated(t *testing.T) {
	dir := newRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", "-N", "new.txt")
	probe := filepath.Join(t.TempDir(), "create.diff")
	if err := os.WriteFile(probe, []byte(run(t, dir, "diff")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "rm", "-q", "--cached", "new.txt")
	if err := os.Remove(filepath.Join(dir, "new.txt")); err != nil {
		t.Fatal(err)
	}
	got, err := Run(context.Background(), Options{Dir: dir, Command: []string{"git", "status"}, Probe: probe})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.State != DidNotReproduce {
		t.Errorf("State = %q, want %q (output %q)", got.State, DidNotReproduce, got.Output)
	}
	if st := run(t, dir, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the probe's file was left behind:\n%s", st)
	}
}

// A patch names files from the repository root, and the check runs where
// it was asked to: a probe applied from a subdirectory lands, and the
// check sees it from there.
func TestRunWithAProbeFromASubdirectory(t *testing.T) {
	dir := newRepo(t)
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", "sub/b.txt")
	run(t, dir, "commit", "-q", "-m", "sub")
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(t.TempDir(), "probe.diff")
	if err := os.WriteFile(probe, []byte(run(t, dir, "diff")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "checkout", "--", "sub/b.txt")

	got, err := Run(context.Background(), Options{Dir: sub, Command: []string{"git", "grep", "-q", "one", "--", "b.txt"}, Probe: probe})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.State != Reproduced {
		t.Errorf("State = %q, want %q (output %q)", got.State, Reproduced, got.Output)
	}
	if st := run(t, dir, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the probe was not reverted:\n%s", st)
	}
}
