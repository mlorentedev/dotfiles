package hooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func findLaneGuardScript(t *testing.T) string {
	t.Helper()
	// Walk up from cli/internal/hooks to repository root git-hooks/lib/lane-guard.sh.
	p, err := filepath.Abs(filepath.Join("..", "..", "..", "git-hooks", "lib", "lane-guard.sh"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func runLaneGuardInDir(t *testing.T, script, dir string) (string, error) {
	t.Helper()
	shPath := findGitPOSIXShell(t)
	cmd := exec.Command(shPath, filepath.ToSlash(script))
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func findGitPOSIXShell(t *testing.T) string {
	t.Helper()
	// Prefer Git-for-Windows' bundled sh/bash over C:\Windows\System32\bash.exe (WSL).
	for _, candidate := range []string{
		`C:\Program Files\Git\bin\bash.exe`,
		`C:\Program Files\Git\usr\bin\bash.exe`,
		`C:\Program Files\Git\bin\sh.exe`,
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX sh not on PATH")
	}
	return sh
}

func initGitRepoForLaneGuard(t *testing.T, dir, branch string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", branch},
		{"config", "user.name", "Test"},
		{"config", "user.email", "test@example.com"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "README.md"},
		{"commit", "-m", "initial", "--no-verify"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
}

func TestPreCommitLaneGuard(t *testing.T) {
	script := findLaneGuardScript(t)
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("lane-guard.sh missing at %s: %v", script, err)
	}

	// 1. Repo WITHOUT .dotf/lane.json on main -> allowed (exit 0).
	plainRepo := filepath.Join(t.TempDir(), "plain-repo")
	initGitRepoForLaneGuard(t, plainRepo, "main")
	if out, err := runLaneGuardInDir(t, script, plainRepo); err != nil {
		t.Fatalf("expected repo without .dotf/lane.json to pass on main, got err=%v out=%s", err, out)
	}

	// 2. Repo WITH .obsidian/ even if .dotf/lane.json is present -> allowed (exit 0).
	vaultRepo := filepath.Join(t.TempDir(), "vault-repo")
	initGitRepoForLaneGuard(t, vaultRepo, "main")
	if err := os.MkdirAll(filepath.Join(vaultRepo, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(vaultRepo, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vaultRepo, ".dotf", "lane.json"), []byte(`{"base":"origin/main"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, vaultRepo); err != nil {
		t.Fatalf("expected Obsidian vault repo to pass on main, got err=%v out=%s", err, out)
	}

	// 3. Repo WITH .dotf/lane.json on main at main checkout -> blocked (exit 1).
	laneRepo := filepath.Join(t.TempDir(), "lane-repo")
	initGitRepoForLaneGuard(t, laneRepo, "main")
	if err := os.MkdirAll(filepath.Join(laneRepo, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(laneRepo, ".dotf", "lane.json"), []byte(`{"base":"origin/main"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, laneRepo); err == nil || !strings.Contains(out, "dotf lane switch") {
		t.Fatalf("expected commit on main in .dotf/lane.json repo to be blocked with 'dotf lane switch', got err=%v out=%s", err, out)
	}

	// 4. PARKED worktree -> blocked when NOT mid-rebase/cherry-pick.
	parkedWT := filepath.Join(t.TempDir(), "worktrees", "lane-repo", "feat-lane-primary")
	if err := os.MkdirAll(filepath.Dir(parkedWT), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", laneRepo, "worktree", "add", "--detach", parkedWT, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("worktree add: %v\n%s", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(parkedWT, ".dotf-worktree.json"), []byte(`{"lane":"feat-lane-primary","state":"PARKED","reap_ok":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, parkedWT); err == nil || !strings.Contains(out, "PARKED") {
		t.Fatalf("expected commit in PARKED lane to be blocked, got err=%v out=%s", err, out)
	}

	// 5. Detached HEAD during active rebase/cherry-pick -> ALLOWED (B6).
	gitDirOut, err := exec.Command("git", "-C", parkedWT, "rev-parse", "--git-dir").Output()
	if err != nil {
		t.Fatal(err)
	}
	wtGitDir := strings.TrimSpace(string(gitDirOut))
	if !filepath.IsAbs(wtGitDir) {
		wtGitDir = filepath.Join(parkedWT, wtGitDir)
	}
	rebaseMergeDir := filepath.Join(wtGitDir, "rebase-merge")
	if err := os.MkdirAll(rebaseMergeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, parkedWT); err != nil {
		t.Fatalf("expected commit during active git rebase (rebase-merge present) to be allowed (B6), got err=%v out=%s", err, out)
	}
	_ = os.RemoveAll(rebaseMergeDir)

	// 6. Commit in ACTIVE named lane on a feature branch -> ALLOWED.
	if out, err := exec.Command("git", "-C", parkedWT, "checkout", "-b", "feat/3255-test").CombinedOutput(); err != nil {
		t.Fatalf("checkout feat branch: %v\n%s", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(parkedWT, ".dotf-worktree.json"), []byte(`{"lane":"feat-lane-primary","state":"ACTIVE","reap_ok":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, parkedWT); err != nil {
		t.Fatalf("expected commit in ACTIVE named lane on feature branch to pass, got err=%v out=%s", err, out)
	}

	// 7. Commit inside copilot-worktrees/ for a .dotf/lane.json repo -> blocked (m4).
	copilotWT := filepath.Join(t.TempDir(), "copilot-worktrees", "lane-repo", "ephemeral-wt")
	if err := os.MkdirAll(filepath.Dir(copilotWT), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", laneRepo, "worktree", "add", "-b", "feat/999-copilot", copilotWT, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("worktree add copilot: %v\n%s", err, string(out))
	}
	if out, err := runLaneGuardInDir(t, script, copilotWT); err == nil || !strings.Contains(out, "dotf lane switch") {
		t.Fatalf("expected commit in copilot-worktrees/ to be blocked (m4), got err=%v out=%s", err, out)
	}
}
