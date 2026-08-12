package doctor

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Real-git fixtures for #806. Every other test in this package models the
// vault's layout with plain files (vaultTree, gitRepo) — sufficient for the
// hooksPath/dispatcher logic, but not for THIS bug, which is specifically
// that a linked worktree's ".git" is a gitdir: pointer FILE rather than a
// directory. A synthetic fixture that writes ".git/hooks/..." can only ever
// produce a directory, so it cannot reproduce the layout the bug depends on.
// These tests run real `git` instead.

// realGit runs a git command for real against dir, failing the test on error.
// Fixture scaffolding only — the check under test still goes through the
// System seam (see realVaultSystem).
func realGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=doctor-test", "GIT_AUTHOR_EMAIL=doctor-test@example.com",
		"GIT_COMMITTER_NAME=doctor-test", "GIT_COMMITTER_EMAIL=doctor-test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s (in %s) failed: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return string(out)
}

// isolateGitConfig points git's GLOBAL and SYSTEM config at /dev/null for the
// remainder of the test, via t.Setenv (auto-restored, and incompatible with
// t.Parallel — none of these tests use it).
//
// This machine (like every machine this repo's setup provisions) has
// core.hooksPath set globally to the GUARD-001 dispatcher — without this,
// `hooksDirFor` resolves every fixture repo's hooks to the REAL dispatcher on
// $PATH, which happily reports the gate active for a repo that never had
// anything installed. Isolating global/system config is what makes these
// tests deterministic on a provisioned dev machine and in a bare CI runner
// alike.
func isolateGitConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
}

// realLinkedWorktree builds an actual checkout plus an actual LINKED WORKTREE
// via `git worktree add`, and asserts the fixture really has the shape the
// bug depends on before trusting anything built on top of it.
func realLinkedWorktree(t *testing.T) (main, worktree string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	isolateGitConfig(t)
	root := t.TempDir()
	main = filepath.Join(root, "main")
	mkdirAll(t, main)
	realGit(t, main, "init", "-q", "-b", "main")
	writeFile(t, filepath.Join(main, ".pre-commit-config.yaml"), "repos: []\n")
	realGit(t, main, "add", ".")
	realGit(t, main, "commit", "-q", "-m", "seed")

	worktree = filepath.Join(root, "linked")
	realGit(t, main, "worktree", "add", "-q", worktree, "-b", "wt-branch")

	info, err := os.Lstat(filepath.Join(worktree, ".git"))
	if err != nil {
		t.Fatalf("worktree .git missing: %v", err)
	}
	if info.IsDir() {
		t.Fatalf("fixture bug: %s/.git is a directory, not the gitdir: pointer file "+
			"this test needs — `git worktree add` did not do what was expected", worktree)
	}
	return main, worktree
}

// commonHooksDir resolves the shared hooks directory for a checkout via real
// git, mirroring what hooksDirFor asks in production.
func commonHooksDir(t *testing.T, repo string) string {
	t.Helper()
	common := strings.TrimSpace(realGit(t, repo, "rev-parse", "--git-common-dir"))
	if !filepath.IsAbs(common) {
		common = filepath.Join(repo, common)
	}
	return filepath.Join(common, "hooks")
}

// realVaultSystem wires the real git/exec seams (realSystem) but overrides
// VAULT_PATH to point at the fixture, without touching the process's actual
// environment or other seams.
func realVaultSystem(vault string) *System {
	sys := realSystem()
	sys.Getenv = func(k string) string {
		if k == "VAULT_PATH" {
			return vault
		}
		return os.Getenv(k)
	}
	return sys
}

// #806 AC: a linked worktree with the gate missing must FAIL, not SKIP.
// Before the fix, isDir(worktree/.git) was false (it's a pointer file), so
// this reported "no vault checkout" — a false negative on a present,
// unprotected vault, which is worse than a FAIL because it reads as healthy.
func TestVaultHooks_LinkedWorktree_GateMissing_MustFail(t *testing.T) {
	_, worktree := realLinkedWorktree(t)

	var buf bytes.Buffer
	rep := capture(&buf)
	checkVaultHooks(realVaultSystem(worktree), rep, false)

	if rep.Failures() != 1 {
		t.Fatalf("a linked worktree with no gate installed must FAIL, got %d failures:\n%s",
			rep.Failures(), buf.String())
	}
	if strings.Contains(buf.String(), "no vault checkout") {
		t.Errorf("must not misread the worktree as absent, got: %s", buf.String())
	}
}

// #806 AC: a linked worktree with the gate installed must PASS. Hooks live in
// the COMMON git dir, shared across every worktree — installed once, and a
// linked worktree must see it exactly like the main checkout does.
func TestVaultHooks_LinkedWorktree_GateInstalled_MustPass(t *testing.T) {
	main, worktree := realLinkedWorktree(t)

	hooks := commonHooksDir(t, main)
	for _, stage := range []string{"pre-commit", "pre-push"} {
		writeExecFile(t, filepath.Join(hooks, stage),
			"#!/usr/bin/env bash\n# File generated by pre-commit: https://pre-commit.com\n")
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkVaultHooks(realVaultSystem(worktree), rep, false)

	if rep.Failures() != 0 {
		t.Fatalf("a linked worktree with the gate installed must PASS, got %d failures:\n%s",
			rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "gitleaks gate active") {
		t.Errorf("want the installed PASS, got: %s", buf.String())
	}
}

// #806 AC: ordinary (non-worktree) checkout behaviour is unchanged by the fix.
func TestVaultHooks_RegularCheckout_StillWorks(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	isolateGitConfig(t)
	root := t.TempDir()
	main := filepath.Join(root, "main")
	mkdirAll(t, main)
	realGit(t, main, "init", "-q", "-b", "main")
	writeFile(t, filepath.Join(main, ".pre-commit-config.yaml"), "repos: []\n")
	realGit(t, main, "add", ".")
	realGit(t, main, "commit", "-q", "-m", "seed")

	var buf bytes.Buffer
	rep := capture(&buf)
	checkVaultHooks(realVaultSystem(main), rep, false)

	if rep.Failures() != 1 {
		t.Fatalf("an ordinary checkout with no gate installed must still FAIL, got %d:\n%s",
			rep.Failures(), buf.String())
	}
}

// #806 AC: --fix from a linked worktree installs into the shared hooks dir
// (so every worktree sees it, not just the one that ran --fix) and is
// idempotent on a second run.
func TestVaultHooks_LinkedWorktree_Fix_InstallsIntoCommonDirAndIsIdempotent(t *testing.T) {
	if _, err := exec.LookPath("pre-commit"); err != nil {
		t.Skip("pre-commit not on PATH")
	}
	main, worktree := realLinkedWorktree(t)

	var buf bytes.Buffer
	rep := capture(&buf)
	checkVaultHooks(realVaultSystem(worktree), rep, true)

	if rep.Failures() != 0 {
		t.Fatalf("--fix with pre-commit available must not FAIL, got %d:\n%s", rep.Failures(), buf.String())
	}

	hooks := commonHooksDir(t, main)
	if _, err := os.Stat(filepath.Join(hooks, "pre-commit")); err != nil {
		t.Fatalf("--fix must install into the COMMON git dir (%s), not the worktree: %v", hooks, err)
	}

	var buf2 bytes.Buffer
	rep2 := capture(&buf2)
	checkVaultHooks(realVaultSystem(worktree), rep2, true)
	if rep2.Failures() != 0 {
		t.Fatalf("a second --fix run must be idempotent, got %d failures:\n%s", rep2.Failures(), buf2.String())
	}
}
