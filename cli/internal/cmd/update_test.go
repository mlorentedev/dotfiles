package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
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

// AC8: after a clean fast-forward, `dotf update` converges the machine from
// the checkout, and the setup script runs as converge's last step on Linux and
// Windows; with nothing to pull it converges nothing.
func TestUpdate_ConvergesAfterAFastForward(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	files, home := convergeFixture(t)
	root := t.TempDir()
	upstream, writer, checkout := filepath.Join(root, "up.git"), filepath.Join(root, "w"), filepath.Join(root, "dotfiles")
	gitIn(t, root, "init", "-q", "--bare", upstream)
	gitIn(t, root, "clone", "-q", upstream, writer)
	if err := os.CopyFS(writer, os.DirFS(files)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(writer, env.CheckoutMarker), []byte("DOTF_VERSION=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, writer, "add", "-A")
	gitIn(t, writer, "commit", "-q", "-m", "one")
	gitIn(t, writer, "push", "-q", "origin", "HEAD")
	gitIn(t, root, "clone", "-q", upstream, checkout)
	t.Setenv("DOTFILES_REPO_DIR", checkout)
	state := filepath.Join(home, ".local", "state")
	t.Setenv("XDG_STATE_HOME", state)

	setups := 0
	fixture := convergeOptions
	convergeOptions = func() converge.Options {
		o := fixture()
		o.RunSetup = func(converge.Env) error { setups++; return nil }
		return o
	}

	stdout, _, err := execute(t, "update")
	if err != nil || setups != 0 || strings.Contains(stdout, "converge on") {
		t.Fatalf("nothing to pull must converge nothing: err=%v setups=%d\n%s", err, setups, stdout)
	}
	assertUpdateRecord(t, filepath.Join(state, "dotfiles"), "current")

	writeCommit(t, writer, "README.md", "new\n")
	gitIn(t, writer, "push", "-q", "origin", "HEAD")

	stdout, _, err = execute(t, "update")
	if err != nil {
		t.Fatalf("update after a push: %v\n%s", err, stdout)
	}
	if !strings.Contains(stdout, "converge on") || !strings.Contains(stdout, "self-update complete") {
		t.Errorf("update must print the converge report and complete:\n%s", stdout)
	}
	assertUpdateRecord(t, filepath.Join(state, "dotfiles"), "updated")
	wantSetups := 1
	if runtime.GOOS == "darwin" {
		wantSetups = 0 // no setup script on macOS (ADR-045)
	}
	if setups != wantSetups {
		t.Errorf("setup ran %d time(s), want %d", setups, wantSetups)
	}
}

// AC7: every run leaves its outcome where doctor reads it.
func assertUpdateRecord(t *testing.T, stateDir, status string) {
	t.Helper()
	r, err := update.ReadRecord(stateDir)
	if err != nil {
		t.Fatalf("no update record after the run: %v", err)
	}
	if r.Status != status || r.Failed() {
		t.Errorf("update record = %+v, want status %q and no error", r, status)
	}
}

// A failed run is recorded with its error, and a record that cannot be
// written is said without changing the run's exit.
func TestRecordUpdate_KeepsTheFailureAndNeverFailsTheRun(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	var w strings.Builder
	recordUpdate(&w, update.Outcome{Status: "converge-failed", Message: "converge failed"}, errors.New("converge: exit 1"))
	r, err := update.ReadRecord(filepath.Join(state, "dotfiles"))
	if err != nil || r.Status != "converge-failed" || r.Error != "converge: exit 1" {
		t.Fatalf("record = %+v, %v", r, err)
	}
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", blocker)
	w.Reset()
	recordUpdate(&w, update.Outcome{Status: "current"}, nil)
	if !strings.Contains(w.String(), "warning: recording this run for doctor") {
		t.Errorf("an unwritable record went unsaid: %q", w.String())
	}
}
