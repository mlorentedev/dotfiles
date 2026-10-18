package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dotfilesTree makes dir a dotfiles checkout as the resolvers see one: a .git
// (a worktree's is a file) and the checkout marker.
func dotfilesTree(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{".git": "gitdir: elsewhere\n", "versions.conf": "DOTF_VERSION=0.0.0\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// A worktree is usually a feature branch days behind main. Converging the
// machine from it wrote that branch's pins into machine-global state and
// pruned a tool main pins (#2711), so the declared checkout wins.
func TestConvergeCheckout_PrefersTheDeclaredCheckoutOverTheCwdWorktree(t *testing.T) {
	_, home := convergeFixture(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	declared := dotfilesTree(t, filepath.Join(t.TempDir(), "dotfiles"))
	t.Setenv("DOTFILES_REPO_DIR", declared)
	t.Chdir(dotfilesTree(t, filepath.Join(t.TempDir(), "worktree")))

	if got := convergeCheckout(home); got != declared {
		t.Errorf("convergeCheckout() = %q, want the declared checkout %q", got, declared)
	}
}

// A checkout cloned somewhere other than the declared path, on a machine where
// the declared path does not exist yet, is the tree ./install.sh runs from: it
// still converges from it rather than cloning a second one.
func TestConvergeCheckout_TheCwdCheckoutWinsWhileTheDeclaredOneIsAbsent(t *testing.T) {
	_, home := convergeFixture(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("DOTFILES_REPO_DIR", filepath.Join(t.TempDir(), "not-cloned-yet"))
	cwd := dotfilesTree(t, filepath.Join(t.TempDir(), "clone"))
	t.Chdir(cwd)

	got := convergeCheckout(home)
	if !sameFile(got, cwd) {
		t.Errorf("convergeCheckout() = %q, want the checkout the cwd is in %q", got, cwd)
	}
}

// A declared path that exists but is no dotfiles checkout (emptied, replaced)
// must not beat a real one in the cwd: `--only tools` skips the checkout step
// that would refuse it, and the tools step would read its pins from nothing.
func TestConvergeCheckout_ADeclaredPathThatIsNoCheckoutDoesNotWin(t *testing.T) {
	_, home := convergeFixture(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("DOTFILES_REPO_DIR", t.TempDir())
	cwd := dotfilesTree(t, filepath.Join(t.TempDir(), "clone"))
	t.Chdir(cwd)

	if got := convergeCheckout(home); !sameFile(got, cwd) {
		t.Errorf("convergeCheckout() = %q, want the checkout the cwd is in %q", got, cwd)
	}
}

// The choice is visible: run from another checkout, converge names the tree
// it converges and how to converge the other one.
func TestConverge_NamesTheTreeWhenTheCwdIsAnotherCheckout(t *testing.T) {
	repo, home := convergeFixture(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	dotfilesTree(t, repo)
	t.Setenv("DOTFILES_REPO_DIR", repo)
	wt := dotfilesTree(t, filepath.Join(t.TempDir(), "worktree"))
	t.Chdir(wt)

	stdout, _, err := execute(t, "converge", "--plan", "--only", "checkout")
	if err != nil {
		t.Fatalf("converge --plan: %v\n%s", err, stdout)
	}
	if !strings.Contains(stdout, "converging "+repo) || !strings.Contains(stdout, "--repo") {
		t.Errorf("want the converged tree named, with --repo for the other one:\n%s", stdout)
	}

	t.Chdir(repo)
	stdout, _, err = execute(t, "converge", "--plan", "--only", "checkout")
	if err != nil {
		t.Fatalf("converge --plan: %v\n%s", err, stdout)
	}
	if strings.Contains(stdout, "converging ") {
		t.Errorf("from the declared checkout itself there is nothing to say:\n%s", stdout)
	}

	other := t.TempDir()
	if err := os.Mkdir(filepath.Join(other, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(other)
	stdout, _, err = execute(t, "converge", "--plan", "--only", "checkout")
	if err != nil {
		t.Fatalf("converge --plan: %v\n%s", err, stdout)
	}
	if strings.Contains(stdout, "converging ") {
		t.Errorf("another project's repository is not a tree converge could have meant:\n%s", stdout)
	}
}
