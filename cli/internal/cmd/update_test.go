package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// TestRepoForUpdatePrefersExistingCascadeDir: a cascade value that names a real
// directory is used as-is.
func TestRepoForUpdatePrefersExistingCascadeDir(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("DOTFILES_REPO_DIR", repo)
	if got := repoForUpdate(); got != repo {
		t.Errorf("repoForUpdate() = %q, want existing cascade dir %q", got, repo)
	}
}

// TestRepoForUpdateFallsBackToWalkUpWhenCascadeMissing: when the cascade resolves
// to a non-existent path (the #696 phantom-default class), repoForUpdate must
// fall through to the .git walk-up instead of returning the dead path.
func TestRepoForUpdateFallsBackToWalkUpWhenCascadeMissing(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, env.CheckoutMarker), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "cli")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", filepath.Join(repo, "does-not-exist"))
	t.Chdir(sub)

	got, err := filepath.EvalSymlinks(repoForUpdate()) // tmp dirs may be symlinked (macOS)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("repoForUpdate() = %q, want walk-up root %q", got, want)
	}
}

// A `dotf update` run from inside another project must not fast-forward that
// project: the walk-up only counts when it lands on a dotfiles checkout.
func TestRepoForUpdateIgnoresAnotherProjectsRepository(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("DOTFILES_REPO_DIR", filepath.Join(home, "does-not-exist"))
	other := t.TempDir()
	if err := os.Mkdir(filepath.Join(other, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(other)

	if got, want := repoForUpdate(), env.DefaultCheckoutDir(home); got != want {
		t.Errorf("repoForUpdate() = %q, want the default checkout %q", got, want)
	}
}
