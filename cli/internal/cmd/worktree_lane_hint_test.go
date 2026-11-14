package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

const laneEntranceHint = "dotf lane switch <branch>"

// activate writes the record that makes repo run lanes, by hand: the hint reads
// it, and how Activate makes it is lane's own tests' business.
func activate(t *testing.T, repo string) {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(strings.TrimSpace(string(out)), lane.ActivationFile)
	if err := os.WriteFile(record, []byte("active\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// executeWorktreeSplit is executeWorktree with the two streams kept apart: the
// hint is for stderr, and a script reading add's stdout must not see it.
func executeWorktreeSplit(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := New("dev", "")
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(append([]string{"worktree"}, args...))
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}

// HARNESS-195 AC4: add still creates the worktree, exactly as before, and where
// the repository runs lanes it says so on stderr.
func TestWorktreeAddPointsAtTheLaneEntranceWhereLanesRun(t *testing.T) {
	cases := []struct {
		name     string
		active   bool
		wantHint bool
	}{
		{"a repository that runs lanes", true, true},
		{"a repository that runs none", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := worktreeTestRepo(t)
			if c.active {
				activate(t, repo)
			}
			stdout, stderr, err := executeWorktreeSplit(t, "add", "demo", "--repo", repo)
			if err != nil {
				t.Fatalf("add: %v\n%s%s", err, stdout, stderr)
			}
			if _, err := os.Stat(rootWorktree("demo")); err != nil {
				t.Errorf("add did not create the worktree: %v", err)
			}
			if !strings.Contains(stdout, "Created worktree at") {
				t.Errorf("add's report moved: %q", stdout)
			}
			if strings.Contains(stdout, laneEntranceHint) {
				t.Errorf("the hint is on stdout, where a script reading add's output would take it:\n%s", stdout)
			}
			if got := strings.Contains(stderr, laneEntranceHint); got != c.wantHint {
				t.Errorf("stderr names %q = %v, want %v:\n%s", laneEntranceHint, got, c.wantHint, stderr)
			}
			if c.wantHint && strings.Count(stderr, "\n") != 1 {
				t.Errorf("the hint is one line, got:\n%s", stderr)
			}
		})
	}
}

// A refused add made no worktree, so it points nowhere.
func TestWorktreeAddRefusedPrintsNoLaneHint(t *testing.T) {
	repo := worktreeTestRepo(t)
	activate(t, repo)
	if _, _, err := executeWorktreeSplit(t, "add", "demo", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := executeWorktreeSplit(t, "add", "demo", "--repo", repo)
	if err == nil {
		t.Fatal("a second add of the same slug succeeded")
	}
	if strings.Contains(stderr, laneEntranceHint) {
		t.Errorf("a refused add printed the hint:\n%s", stderr)
	}
}

// Every position in an active repository gets the entrance; an unlaned one
// gets nothing; an unreadable directory says why on the stream it was given.
func TestLaneHintByPosition(t *testing.T) {
	for _, state := range []worktree.LifecycleState{worktree.StateActive, worktree.StateParked} {
		f := newLaneFixture(t, true, state)
		for name, cwd := range map[string]string{"main checkout": f.main, "lane": f.lane} {
			var errOut bytes.Buffer
			if got := laneHint(cwd, &errOut); !strings.Contains(got, laneEntranceHint) {
				t.Errorf("%s (%s lease): hint %q, want it to name %q", name, state, got, laneEntranceHint)
			}
		}
	}
	idle := newLaneFixture(t, false, worktree.StateActive)
	if got := laneHint(idle.main, io.Discard); got != "" {
		t.Errorf("a repository without lanes got the hint %q", got)
	}
	// A lease Where cannot parse is a question it cannot answer: the lane's own
	// directory, not a missing one (git answers a missing directory with "no
	// repository", which is Inactive and silent).
	broken := newLaneFixture(t, true, worktree.StateActive)
	admin, err := worktree.AdminDir(broken.lane)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(admin, worktree.LeaseFileName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	if got := laneHint(broken.lane, &errOut); got != "" {
		t.Errorf("a lane whose lease cannot be read got the hint %q", got)
	}
	if !strings.Contains(errOut.String(), "lane hint skipped") {
		t.Errorf("a directory Where cannot read was not reported on errOut: %q", errOut.String())
	}
}

// AddDetached is what `spec review` calls. It prints nothing, in a repository
// that runs lanes too: the hint belongs to the command a person types.
func TestAddDetachedPrintsNoLaneHint(t *testing.T) {
	f := newLaneFixture(t, true, worktree.StateActive)
	root := filepath.Join(t.TempDir(), "worktrees")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prev := os.Stderr
	os.Stderr = w
	_, addErr := worktree.AddDetached(worktree.DetachedOptions{
		RepoRoot: f.main, WorktreeRoot: root, Name: "review-x", Commit: "HEAD", Kind: worktree.KindReview,
	})
	os.Stderr = prev
	_ = w.Close()
	captured, _ := io.ReadAll(r)
	if addErr != nil {
		t.Fatal(addErr)
	}
	if len(captured) != 0 {
		t.Errorf("AddDetached wrote to stderr:\n%s", captured)
	}
}
