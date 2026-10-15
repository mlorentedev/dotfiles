package worktree

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func addTestOpts(t *testing.T) AddOptions {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "myrepo")
	_ = os.MkdirAll(filepath.Join(repo, ".git", "info"), 0o755)
	return AddOptions{RepoRoot: repo, WorktreeRoot: filepath.Join(t.TempDir(), "worktrees"), Slug: "x"}
}

func TestAddCreatesFromTheRemoteDefaultBranch(t *testing.T) {
	var gotBase string
	runner := &MockAddRunner{
		OnDefaultBase: func(string) (string, error) { return "origin/master", nil },
		OnWorktreeAdd: func(_, target, _, base string) error { gotBase = base; return os.MkdirAll(target, 0o755) },
	}
	if _, err := AddWithRunner(addTestOpts(t), runner, time.Now()); err != nil {
		t.Fatal(err)
	}
	if gotBase != "origin/master" {
		t.Errorf("base = %q, want the resolved remote default origin/master", gotBase)
	}
}

func TestAddKeepsAnExplicitBase(t *testing.T) {
	opts := addTestOpts(t)
	opts.BaseRef = "feat/parent"
	var gotBase string
	runner := &MockAddRunner{OnWorktreeAdd: func(_, target, _, base string) error { gotBase = base; return os.MkdirAll(target, 0o755) }}
	if _, err := AddWithRunner(opts, runner, time.Now()); err != nil {
		t.Fatal(err)
	}
	if gotBase != "feat/parent" || runner.DefaultBaseRan {
		t.Errorf("base = %q, default resolution ran = %v; want feat/parent and no resolution", gotBase, runner.DefaultBaseRan)
	}
}

func TestAddRefusesWhenNoRemoteDefaultResolves(t *testing.T) {
	called := false
	runner := &MockAddRunner{
		OnDefaultBase: func(string) (string, error) { return "", errors.New("no remote default branch") },
		OnWorktreeAdd: func(_, _, _, _ string) error { called = true; return nil },
	}
	if _, err := AddWithRunner(addTestOpts(t), runner, time.Now()); err == nil {
		t.Fatal("add created a worktree with no base to create it from")
	}
	if called {
		t.Error("git worktree add ran without a base, which would branch from the caller's HEAD")
	}
}

// The real resolution, against a local origin: a repository whose remote has
// only master (the vault) gets origin/master, and one with no origin refuses
// naming --base.
func TestRealDefaultBase(t *testing.T) {
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	tmp := t.TempDir()
	origin := filepath.Join(tmp, "origin")
	git(tmp, "init", "-q", "-b", "master", origin)
	git(origin, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	clone := filepath.Join(tmp, "clone")
	git(tmp, "clone", "-q", origin, clone)
	if got, err := (&RealAddRunner{}).DefaultBase(clone); err != nil || got != "origin/HEAD" {
		t.Errorf("with origin/HEAD: DefaultBase = %q, %v; want origin/HEAD", got, err)
	}

	// Without origin/HEAD, and with fetch told not to restore it (git 2.48+
	// does by default), the order falls through to master.
	git(clone, "config", "remote.origin.followRemoteHEAD", "never")
	git(clone, "remote", "set-head", "origin", "--delete")
	if got, err := (&RealAddRunner{}).DefaultBase(clone); err != nil || got != "origin/master" {
		t.Errorf("without origin/HEAD: DefaultBase = %q, %v; want origin/master", got, err)
	}

	lonely := filepath.Join(tmp, "lonely")
	git(tmp, "init", "-q", lonely)
	if _, err := (&RealAddRunner{}).DefaultBase(lonely); err == nil || !strings.Contains(err.Error(), "--base") {
		t.Errorf("no origin: got %v, want a refusal naming --base", err)
	}
}
