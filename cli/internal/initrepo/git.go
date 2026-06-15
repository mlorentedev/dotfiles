package initrepo

import (
	"os"
	"os/exec"
	"path/filepath"
)

// GitInit runs `git init` in root when it is not already a git repo. It returns
// "initialized" when it created the repo or "exists" when one was already there
// (idempotent). git is assumed present — it is the one hard dependency of a
// scaffolder, and the orchestrator surfaces the error if it is missing.
func GitInit(root string) (string, error) {
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		return "exists", nil
	}
	if err := exec.Command("git", "-C", root, "init", "-q").Run(); err != nil {
		return "", err
	}
	return "initialized", nil
}

// PreCommitInstall runs `pre-commit install` in root when pre-commit is on PATH
// and a .pre-commit-config.yaml is present. A missing tool or config is a
// graceful "skipped" (the hooks just aren't active until the user installs
// pre-commit and re-runs) — never an error.
func PreCommitInstall(root string) (string, error) {
	if _, err := exec.LookPath("pre-commit"); err != nil {
		return "skipped", nil
	}
	if _, err := os.Stat(filepath.Join(root, ".pre-commit-config.yaml")); err != nil {
		return "skipped", nil
	}
	cmd := exec.Command("pre-commit", "install")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		// An install failure (e.g. not a git repo yet) degrades to skipped: the
		// scaffold continues and the user can re-run `pre-commit install`.
		return "skipped", nil
	}
	return "installed", nil
}
