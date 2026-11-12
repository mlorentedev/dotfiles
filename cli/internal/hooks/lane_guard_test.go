package hooks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
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

// writeLanesMarkerFixture writes the marker exactly as `dotf lane init` writes it
// (indented JSON, one array element per line) so the guard's parser is exercised
// against the real production shape rather than a convenient compact one.
func writeLanesMarkerFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := json.MarshalIndent(struct {
		Version         int      `json:"version"`
		CanonicalLanes  []string `json:"canonical_lanes"`
		ThirdPartyRoots []string `json:"third_party_roots"`
	}{1, lane.CanonicalLanes, lane.ThirdPartyWorktreeRoots}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".dotf", "lanes.json"), append(body, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPreCommitLaneGuard(t *testing.T) {
	script := findLaneGuardScript(t)
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("lane-guard.sh missing at %s: %v", script, err)
	}

	// 1. Repo WITHOUT .dotf/lanes.json on main -> allowed (exit 0).
	plainRepo := filepath.Join(t.TempDir(), "plain-repo")
	initGitRepoForLaneGuard(t, plainRepo, "main")
	if out, err := runLaneGuardInDir(t, script, plainRepo); err != nil {
		t.Fatalf("expected repo without .dotf/lanes.json to pass on main, got err=%v out=%s", err, out)
	}

	// 2. Repo WITH .obsidian/ even if .dotf/lanes.json is present -> allowed (exit 0).
	vaultRepo := filepath.Join(t.TempDir(), "vault-repo")
	initGitRepoForLaneGuard(t, vaultRepo, "main")
	if err := os.MkdirAll(filepath.Join(vaultRepo, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLanesMarkerFixture(t, vaultRepo)
	if out, err := runLaneGuardInDir(t, script, vaultRepo); err != nil {
		t.Fatalf("expected Obsidian vault repo to pass on main, got err=%v out=%s", err, out)
	}

	// 3. Repo WITH .dotf/lanes.json on main at main checkout -> blocked (exit 1).
	laneRepo := filepath.Join(t.TempDir(), "lane-repo")
	initGitRepoForLaneGuard(t, laneRepo, "main")
	writeLanesMarkerFixture(t, laneRepo)
	if out, err := runLaneGuardInDir(t, script, laneRepo); err == nil || !strings.Contains(out, "dotf lane switch") {
		t.Fatalf("expected commit on main in .dotf/lanes.json repo to be blocked with 'dotf lane switch', got err=%v out=%s", err, out)
	}

	// 3b. Repo with ONLY .dotf/lane.json (the issue-queue config from #2532) ->
	// allowed. The guard must not be switched on by another feature's file.
	queueRepo := filepath.Join(t.TempDir(), "queue-repo")
	initGitRepoForLaneGuard(t, queueRepo, "main")
	if err := os.MkdirAll(filepath.Join(queueRepo, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queueRepo, ".dotf", "lane.json"), []byte(`{"queue":[2501],"branch_prefix":"fix","base":"origin/main"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, queueRepo); err != nil {
		t.Fatalf("expected .dotf/lane.json (issue-queue config) NOT to activate the guard, got err=%v out=%s", err, out)
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

	// 6b. A detached linked worktree without a lease is blocked, and the message
	// names that state and that path, never "the main checkout" (#3324).
	scratchWT := filepath.Join(t.TempDir(), "scratch", "lane-repo", "scratch-wt")
	if err := os.MkdirAll(filepath.Dir(scratchWT), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", laneRepo, "worktree", "add", "--detach", scratchWT, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("worktree add scratch: %v\n%s", err, string(out))
	}
	out, err := runLaneGuardInDir(t, script, scratchWT)
	if err == nil || !strings.Contains(out, "without a lease at") || !strings.Contains(out, "scratch-wt") {
		t.Fatalf("expected a detached linked worktree without a lease to be blocked with its own path and state, got err=%v out=%s", err, out)
	}
	if strings.Contains(out, "main checkout") {
		t.Fatalf("a linked worktree is not the main checkout; the message must not say so:\n%s", out)
	}
	// 6c. The main checkout on a detached HEAD keeps its own wording.
	detachedMain := filepath.Join(t.TempDir(), "detached-main")
	initGitRepoForLaneGuard(t, detachedMain, "main")
	writeLanesMarkerFixture(t, detachedMain)
	if out, err := exec.Command("git", "-C", detachedMain, "checkout", "--detach").CombinedOutput(); err != nil {
		t.Fatalf("checkout --detach: %v\n%s", err, string(out))
	}
	if out, err := runLaneGuardInDir(t, script, detachedMain); err == nil || !strings.Contains(out, "at the main checkout") {
		t.Fatalf("expected a detached HEAD at the main checkout to be blocked with the main-checkout message, got err=%v out=%s", err, out)
	}

	// 7. Commit inside copilot-worktrees/ for a .dotf/lanes.json repo -> blocked (m4).
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

	// 8. A third-party root is data, not code: declaring a different host name in
	// the marker blocks that root too, with no change to the guard.
	customRepo := filepath.Join(t.TempDir(), "custom-repo")
	initGitRepoForLaneGuard(t, customRepo, "main")
	if err := os.MkdirAll(filepath.Join(customRepo, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	customMarker := `{"version":1,"canonical_lanes":["feat-lane-primary"],"third_party_roots":["acme-sessions"]}`
	if err := os.WriteFile(filepath.Join(customRepo, ".dotf", "lanes.json"), []byte(customMarker), 0o644); err != nil {
		t.Fatal(err)
	}
	customWT := filepath.Join(t.TempDir(), "acme-sessions", "custom-repo", "ephemeral-wt")
	if err := os.MkdirAll(filepath.Dir(customWT), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", customRepo, "worktree", "add", "-b", "feat/998-acme", customWT, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("worktree add acme: %v\n%s", err, string(out))
	}
	if out, err := runLaneGuardInDir(t, script, customWT); err == nil || !strings.Contains(out, "acme-sessions") {
		t.Fatalf("expected declared third-party root 'acme-sessions' to be blocked, got err=%v out=%s", err, out)
	}

	// 9. No host is privileged in code: a marker that declares no third_party_roots
	// leaves copilot-worktrees/ unblocked. The guard must not fall back to a
	// hardcoded host name.
	norootsRepo := filepath.Join(t.TempDir(), "noroots-repo")
	initGitRepoForLaneGuard(t, norootsRepo, "main")
	if err := os.MkdirAll(filepath.Join(norootsRepo, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(norootsRepo, ".dotf", "lanes.json"), []byte(`{"version":1,"canonical_lanes":["feat-lane-primary"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	norootsWT := filepath.Join(t.TempDir(), "copilot-worktrees", "noroots-repo", "ephemeral-wt")
	if err := os.MkdirAll(filepath.Dir(norootsWT), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", norootsRepo, "worktree", "add", "-b", "feat/997-noroots", norootsWT, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("worktree add noroots: %v\n%s", err, string(out))
	}
	if out, err := runLaneGuardInDir(t, script, norootsWT); err != nil {
		t.Fatalf("expected undeclared root copilot-worktrees/ to pass when the marker declares no third_party_roots, got err=%v out=%s", err, out)
	}

	// 10. A lane's identity is the record `dotf lane init` leaves in the
	// worktree's git admin dir, not its leaf name: (a) an unleased worktree that
	// merely shares a declared lane's name passes; (b) with the record present
	// the commit is refused, by the guard running as a real pre-commit hook;
	// (c) the record follows the worktree through `git worktree move`, so the
	// block survives a rename to a name the marker never declared.
	altRepo := filepath.Join(t.TempDir(), "alt-repo")
	initGitRepoForLaneGuard(t, altRepo, "main")
	if err := os.MkdirAll(filepath.Join(altRepo, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(altRepo, ".dotf", "lanes.json"), []byte(`{"version":1,"canonical_lanes":["spike-lane"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	altWT := filepath.Join(t.TempDir(), "spike-lane")
	if out, err := exec.Command("git", "-C", altRepo, "worktree", "add", "-b", "feat/996-spike", altWT, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("worktree add alt: %v\n%s", err, string(out))
	}
	if out, err := runLaneGuardInDir(t, script, altWT); err != nil {
		t.Fatalf("expected an unregistered worktree merely named 'spike-lane' to pass, got err=%v out=%s", err, out)
	}

	gitDirOut, gitDirErr := exec.Command("git", "-C", altWT, "rev-parse", "--absolute-git-dir").Output()
	if gitDirErr != nil {
		t.Fatalf("rev-parse --absolute-git-dir: %v", gitDirErr)
	}
	if err := os.WriteFile(filepath.Join(strings.TrimSpace(string(gitDirOut)), lane.LaneIdentityFile), []byte("spike-lane\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooksDir := t.TempDir()
	hook := "#!/bin/sh\nexec \"" + filepath.ToSlash(findGitPOSIXShell(t)) + "\" \"" + filepath.ToSlash(script) + "\"\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", altRepo, "config", "core.hooksPath", filepath.ToSlash(hooksDir)).CombinedOutput(); err != nil {
		t.Fatalf("config core.hooksPath: %v\n%s", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(altWT, "work.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", altWT, "add", "work.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, string(out))
	}
	commitOut, commitErr := exec.Command("git", "-C", altWT, "commit", "-m", "spike").CombinedOutput()
	if commitErr == nil || !strings.Contains(string(commitOut), "lease metadata") || !strings.Contains(string(commitOut), "spike-lane") {
		t.Fatalf("expected the registered lane 'spike-lane' without a lease to be refused by the pre-commit hook, got err=%v out=%s", commitErr, string(commitOut))
	}

	// An empty record is no record: Go's lane.Identity and the guard agree.
	if err := os.WriteFile(filepath.Join(strings.TrimSpace(string(gitDirOut)), lane.LaneIdentityFile), []byte(" \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, altWT); err != nil {
		t.Fatalf("expected an empty identity record to be read as no record, got err=%v out=%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(strings.TrimSpace(string(gitDirOut)), lane.LaneIdentityFile), []byte("spike-lane\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A record that exists but cannot be read fails closed, not open.
	if runtime.GOOS != "windows" && os.Getuid() != 0 {
		record := filepath.Join(strings.TrimSpace(string(gitDirOut)), lane.LaneIdentityFile)
		if err := os.Chmod(record, 0o000); err != nil {
			t.Fatal(err)
		}
		out, err := runLaneGuardInDir(t, script, altWT)
		if cerr := os.Chmod(record, 0o644); cerr != nil {
			t.Fatal(cerr)
		}
		if err == nil || !strings.Contains(out, "not readable") {
			t.Fatalf("expected an unreadable identity record to block, got err=%v out=%s", err, out)
		}
	}

	movedWT := filepath.Join(filepath.Dir(altWT), "not-a-lane-name")
	if out, err := exec.Command("git", "-C", altRepo, "worktree", "move", altWT, movedWT).CombinedOutput(); err != nil {
		t.Fatalf("worktree move: %v\n%s", err, string(out))
	}
	if out, err := runLaneGuardInDir(t, script, movedWT); err == nil || !strings.Contains(out, "lease metadata") || !strings.Contains(out, "spike-lane") {
		t.Fatalf("expected the moved lane to stay blocked by its record, got err=%v out=%s", err, out)
	}
}
