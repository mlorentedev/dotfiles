package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// worktreeTestRepo is a git repository with one commit, in a directory whose
// siblings are the test's own. TMPDIR is redirected so done's lock file is
// per-test too.
func worktreeTestRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("TMPDIR", t.TempDir())
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "Test"},
		{"config", "user.email", "test@example.com"},
		{"commit", "-q", "--allow-empty", "-m", "initial"},
	} {
		c := exec.Command("git", args...)
		c.Dir = repo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return repo
}

func executeWorktree(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := New("dev", "")
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"worktree"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func TestWorktreeDoneTakesTheSlugAddWasGiven(t *testing.T) {
	repo := worktreeTestRepo(t)
	if out, err := executeWorktree(t, "add", "demo", "--repo", repo); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	wt := filepath.Join(filepath.Dir(repo), "repo-wt-demo")
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("add did not create %s: %v", wt, err)
	}

	out, err := executeWorktree(t, "done", "demo", "--repo", repo)
	if err != nil {
		t.Fatalf("done demo: %v\n%s", err, out)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("%s still exists after done demo (stat err %v)", wt, err)
	}
}

// From inside a worktree, a slug still names a sibling of the main repository,
// never <worktree>-wt-<slug>.
func TestWorktreeDoneResolvesASlugBesideTheMainRepository(t *testing.T) {
	repo := worktreeTestRepo(t)
	for _, slug := range []string{"one", "two"} {
		if out, err := executeWorktree(t, "add", slug, "--repo", repo); err != nil {
			t.Fatalf("add %s: %v\n%s", slug, err, out)
		}
	}
	t.Chdir(filepath.Join(filepath.Dir(repo), "repo-wt-one"))

	if out, err := executeWorktree(t, "done", "two"); err != nil {
		t.Fatalf("done two from inside repo-wt-one: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(repo), "repo-wt-two")); !os.IsNotExist(err) {
		t.Errorf("repo-wt-two still exists (stat err %v)", err)
	}
}

func TestWorktreeDoneNamesBothPathsForAnUnknownName(t *testing.T) {
	repo := worktreeTestRepo(t)
	t.Chdir(t.TempDir())

	_, err := executeWorktree(t, "done", "nosuch", "--repo", repo)
	if err == nil {
		t.Fatal("done nosuch succeeded")
	}
	cwd, _ := os.Getwd()
	for _, want := range []string{
		filepath.Join(cwd, "nosuch"),
		filepath.Join(filepath.Dir(repo), "repo-wt-nosuch"),
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %s", err, want)
		}
	}
	if strings.Contains(err.Error(), "exit status") {
		t.Errorf("error %q is a bare git exit status", err)
	}
}

// A bare name that is both a directory in the cwd and a sibling worktree is
// refused, naming both, rather than resolved to whichever is checked first.
func TestWorktreeDoneRefusesANameThatIsBothAPathAndASlug(t *testing.T) {
	repo := worktreeTestRepo(t)
	if out, err := executeWorktree(t, "add", "demo", "--repo", repo); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	cwd := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwd, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	_, err := executeWorktree(t, "done", "demo", "--repo", repo)
	wt := filepath.Join(filepath.Dir(repo), "repo-wt-demo")
	if err == nil || !strings.Contains(err.Error(), wt) || !strings.Contains(err.Error(), filepath.Join(cwd, "demo")) {
		t.Fatalf("got %v, want a refusal naming both %s and the cwd's demo", err, wt)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Errorf("the sibling worktree was touched: %v", err)
	}
}

// A directory where a slug points that is not a git worktree is named as such,
// not handed to git status to fail with a bare exit status.
func TestWorktreeDoneNamesASiblingThatIsNotAWorktree(t *testing.T) {
	repo := worktreeTestRepo(t)
	stray := filepath.Join(filepath.Dir(repo), "repo-wt-stray")
	if err := os.Mkdir(stray, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	_, err := executeWorktree(t, "done", "stray", "--repo", repo)
	if err == nil || !strings.Contains(err.Error(), stray) || !strings.Contains(err.Error(), "not a git worktree") {
		t.Fatalf("got %v, want an error naming %s as not a git worktree", err, stray)
	}
}

// A cwd that cannot be read must not switch the caller check off: the shell's
// $PWD stands in for it.
func TestCallerCwdFallsBackToPWD(t *testing.T) {
	env := func(k string) string {
		if k == "PWD" {
			return "/repo-wt-x/sub"
		}
		return ""
	}
	if got := callerCwd(func() (string, error) { return "/elsewhere", nil }, env); got != "/elsewhere" {
		t.Fatalf("a readable cwd must win, got %q", got)
	}
	unreadable := func() (string, error) { return "", errors.New("getwd: permission denied") }
	if got := callerCwd(unreadable, env); got != "/repo-wt-x/sub" {
		t.Fatalf("an unreadable cwd must fall back to $PWD, got %q", got)
	}
}
