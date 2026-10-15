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
// per-test too, and WORKTREE_ROOT so add never writes under the real
// ~/Projects/worktrees.
func worktreeTestRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("WORKTREE_ROOT", filepath.Join(t.TempDir(), "worktrees"))
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	// A local bare origin: add creates from origin's default branch.
	origin := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", origin).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "Test"},
		{"config", "user.email", "test@example.com"},
		{"commit", "-q", "--allow-empty", "-m", "initial"},
		{"remote", "add", "origin", origin},
		{"push", "-q", "origin", "main"},
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

// rootWorktree is where add puts the worktree for slug under the test's
// WORKTREE_ROOT: the default branch feat/<slug>, slugified.
func rootWorktree(slug string) string {
	return filepath.Join(os.Getenv("WORKTREE_ROOT"), "repo", "feat-"+slug)
}

// legacySibling is the <repo>-wt-<slug> layout that predates the one root.
func legacySibling(repo, slug string) string {
	return filepath.Join(filepath.Dir(repo), "repo-wt-"+slug)
}

func TestWorktreeDoneTakesTheSlugAddWasGiven(t *testing.T) {
	repo := worktreeTestRepo(t)
	if out, err := executeWorktree(t, "add", "demo", "--repo", repo); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	wt := rootWorktree("demo")
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

// With -b, the directory follows the branch, not the slug (#2527 review): add
// says which name done takes, and done accepts the branch and its slug.
func TestWorktreeDoneTakesTheBranchGivenToAdd(t *testing.T) {
	for _, name := range []string{"fix/login", "fix-login"} {
		t.Run(name, func(t *testing.T) {
			repo := worktreeTestRepo(t)
			out, err := executeWorktree(t, "add", "demo", "-b", "fix/login", "--repo", repo)
			if err != nil {
				t.Fatalf("add: %v\n%s", err, out)
			}
			if !strings.Contains(out, "dotf worktree done fix-login") {
				t.Errorf("add -b does not say which name done takes:\n%s", out)
			}
			wt := filepath.Join(os.Getenv("WORKTREE_ROOT"), "repo", "fix-login")
			if out, err := executeWorktree(t, "done", name, "--repo", repo); err != nil {
				t.Fatalf("done %s: %v\n%s", name, err, out)
			}
			if _, err := os.Stat(wt); !os.IsNotExist(err) {
				t.Errorf("%s still exists after done %s (stat err %v)", wt, name, err)
			}
		})
	}
}

// A worktree in the legacy <repo>-wt-<slug> layout is still found by its slug,
// so done keeps working on trees made before the one root, until migrate.
func TestWorktreeDoneTakesTheSlugOfALegacySibling(t *testing.T) {
	repo := worktreeTestRepo(t)
	wt := legacySibling(repo, "old")
	if out, err := executeWorktree(t, "add", "old", "--repo", repo, "--path", wt); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	if out, err := executeWorktree(t, "done", "old", "--repo", repo); err != nil {
		t.Fatalf("done old: %v\n%s", err, out)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("%s still exists after done old (stat err %v)", wt, err)
	}
}

// A slug that names a worktree in both layouts is refused, naming both, rather
// than resolved to whichever layout is checked first.
func TestWorktreeDoneRefusesASlugInBothLayouts(t *testing.T) {
	repo := worktreeTestRepo(t)
	if out, err := executeWorktree(t, "add", "twin", "--repo", repo); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	legacy := legacySibling(repo, "twin")
	if out, err := executeWorktree(t, "add", "twin", "--repo", repo, "--path", legacy, "-b", "fix/twin"); err != nil {
		t.Fatalf("add legacy: %v\n%s", err, out)
	}
	t.Chdir(t.TempDir())

	_, err := executeWorktree(t, "done", "twin", "--repo", repo)
	if err == nil || !strings.Contains(err.Error(), legacy) || !strings.Contains(err.Error(), rootWorktree("twin")) {
		t.Fatalf("got %v, want a refusal naming %s and %s", err, legacy, rootWorktree("twin"))
	}
}

// From inside a worktree, a slug still names a worktree of the main repository,
// never one relative to the worktree the caller stands in.
func TestWorktreeDoneResolvesASlugBesideTheMainRepository(t *testing.T) {
	repo := worktreeTestRepo(t)
	for _, slug := range []string{"one", "two"} {
		if out, err := executeWorktree(t, "add", slug, "--repo", repo); err != nil {
			t.Fatalf("add %s: %v\n%s", slug, err, out)
		}
	}
	t.Chdir(rootWorktree("one"))

	if out, err := executeWorktree(t, "done", "two"); err != nil {
		t.Fatalf("done two from inside feat-one: %v\n%s", err, out)
	}
	if _, err := os.Stat(rootWorktree("two")); !os.IsNotExist(err) {
		t.Errorf("feat-two still exists (stat err %v)", err)
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
		legacySibling(repo, "nosuch"),
		rootWorktree("nosuch"),
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
	wt := rootWorktree("demo")
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

func TestWorktreeDoneSaysWhenTheRootWasNotSearched(t *testing.T) {
	tried := []string{"/a/x", "/a/repo-wt-x"}
	if err := doneNotFound("x", tried, ""); !strings.Contains(err.Error(), "WORKTREE_ROOT could not be resolved") {
		t.Errorf("unresolved root: got %v, want it named", err)
	}
	if err := doneNotFound("x", tried, "/w"); strings.Contains(err.Error(), "WORKTREE_ROOT") {
		t.Errorf("resolved root: got %v, want no root note", err)
	}
}
