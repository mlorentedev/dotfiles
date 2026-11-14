package hooks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

func findLaneGuardScript(t *testing.T) string {
	t.Helper()
	// Walk up from cli/internal/hooks to repository root git-hooks/lib/lane-guard.sh.
	p, err := filepath.Abs(filepath.Join("..", "..", "..", "git-hooks", filepath.FromSlash(lane.GuardRelPath)))
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

// laneMarkerFixtureBody is the marker exactly as `dotf lane init` writes it
// (indented JSON, one array element per line) so the guard's parser is exercised
// against the real production shape rather than a convenient compact one.
func laneMarkerFixtureBody(t *testing.T) []byte {
	t.Helper()
	body, err := json.MarshalIndent(struct {
		Version         int      `json:"version"`
		CanonicalLanes  []string `json:"canonical_lanes"`
		ThirdPartyRoots []string `json:"third_party_roots"`
	}{1, lane.CanonicalLanes, lane.ThirdPartyWorktreeRoots}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(body, '\n')
}

// writeLanesMarkerFixture commits-to-disk the marker a repository declares its
// lanes with. It arms nothing: only the activation record does (#3437).
func writeLanesMarkerFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".dotf", "lanes.json"), laneMarkerFixtureBody(t), 0o644); err != nil {
		t.Fatal(err)
	}
}

// activateLanesFixture writes the activation record `dotf converge` leaves in the
// git common dir, carrying body (the marker's JSON, as lane.Activate writes it).
// This is the only thing that arms the guard.
func activateLanesFixture(t *testing.T, repo string, body []byte) {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		t.Fatalf("git-common-dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(strings.TrimSpace(string(out)), lane.ActivationFile), body, 0o644); err != nil {
		t.Fatal(err)
	}
}

// armLaneGuard marks repo as running lanes with the standard marker content.
func armLaneGuard(t *testing.T, repo string) {
	t.Helper()
	activateLanesFixture(t, repo, laneMarkerFixtureBody(t))
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
	armLaneGuard(t, vaultRepo)
	if out, err := runLaneGuardInDir(t, script, vaultRepo); err != nil {
		t.Fatalf("expected Obsidian vault repo to pass on main, got err=%v out=%s", err, out)
	}

	// 3. Repo WITH .dotf/lanes.json on main at main checkout -> blocked (exit 1).
	laneRepo := filepath.Join(t.TempDir(), "lane-repo")
	initGitRepoForLaneGuard(t, laneRepo, "main")
	writeLanesMarkerFixture(t, laneRepo)
	armLaneGuard(t, laneRepo)
	if out, err := runLaneGuardInDir(t, script, laneRepo); err == nil || !strings.Contains(out, "dotf lane switch") {
		t.Fatalf("expected commit on main in .dotf/lanes.json repo to be blocked with 'dotf lane switch', got err=%v out=%s", err, out)
	}

	// 3a. The block says what it found. At the main checkout it names the main
	// checkout; in a linked worktree that happens to hold main it names that
	// worktree and never "the main checkout", with the recovery for each.
	if out, err := runLaneGuardInDir(t, script, laneRepo); err == nil || !strings.Contains(out, "'main' at the main checkout") {
		t.Fatalf("expected the main-checkout block to say so, got err=%v out=%s", err, out)
	}
	// From a subdirectory git answers --git-dir absolute and --git-common-dir
	// relative to that subdirectory; the guard must still know it is the main
	// checkout rather than a linked worktree.
	subdir := filepath.Join(laneRepo, "sub", "dir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, subdir); err == nil || !strings.Contains(out, "'main' at the main checkout") {
		t.Fatalf("expected the block to name the main checkout from a subdirectory too, got err=%v out=%s", err, out)
	}
	mainRepo := filepath.Join(t.TempDir(), "main-repo")
	initGitRepoForLaneGuard(t, mainRepo, "main")
	writeLanesMarkerFixture(t, mainRepo)
	armLaneGuard(t, mainRepo)
	if out, err := exec.Command("git", "-C", mainRepo, "switch", "-q", "-c", "parked-elsewhere").CombinedOutput(); err != nil {
		t.Fatalf("switch: %v\n%s", err, string(out))
	}
	mainWT := filepath.Join(t.TempDir(), "main-wt")
	if out, err := exec.Command("git", "-C", mainRepo, "worktree", "add", "-q", mainWT, "main").CombinedOutput(); err != nil {
		t.Fatalf("worktree add main: %v\n%s", err, string(out))
	}
	out, err := runLaneGuardInDir(t, script, mainWT)
	if err == nil || !strings.Contains(out, "'main' in the linked worktree") || !strings.Contains(out, "main-wt") ||
		strings.Contains(out, "main checkout") || !strings.Contains(out, "git switch -c <type>/<issue>-<slug>") {
		t.Fatalf("expected a linked worktree on main to be blocked as such, with its own recovery, got err=%v out=%s", err, out)
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
	out, err = runLaneGuardInDir(t, script, scratchWT)
	if err == nil || !strings.Contains(out, "without a lease at") || !strings.Contains(out, "scratch-wt") {
		t.Fatalf("expected a detached linked worktree without a lease to be blocked with its own path and state, got err=%v out=%s", err, out)
	}
	if strings.Contains(out, "main checkout") {
		t.Fatalf("a linked worktree is not the main checkout; the message must not say so:\n%s", out)
	}
	// The block names the way out (#3436): adopt keeps this worktree and its
	// commits, a branch ends the detached state. It must not send the user to
	// `lane switch`, which claims a different worktree and leaves this one as is.
	for _, want := range []string{"dotf worktree adopt", "git switch -c <type>/<issue>-<slug>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected the leaseless detached block to name %q as recovery, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dotf lane switch") {
		t.Fatalf("a leaseless worktree is not a lane: lane switch is not its recovery:\n%s", out)
	}
	// The parenthesised repository is the main root, not the scratch path and
	// not empty: git answers with the physical path, so compare base names.
	root := regexp.MustCompile(`repository \((.*)\)\.`).FindStringSubmatch(out)
	if root == nil || root[1] == "" || filepath.Base(root[1]) != "lane-repo" || strings.Contains(root[1], "scratch-wt") {
		t.Fatalf("expected the message to name the main repository root in parentheses, got:\n%s", out)
	}

	// 6c. The admin-dir lease (ADR-062): the file `git clean` never reaches is
	// the one both rules read, with no in-tree copy at all, and it outranks a
	// stale in-tree copy when both exist, as worktree.LoadMetadata reads them.
	adminWT := filepath.Join(t.TempDir(), "admin", "lane-repo", "feat-lane-secondary")
	if err := os.MkdirAll(filepath.Dir(adminWT), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", laneRepo, "worktree", "add", "--detach", adminWT, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("worktree add admin: %v\n%s", err, string(out))
	}
	adminGitDirOut, err := exec.Command("git", "-C", adminWT, "rev-parse", "--absolute-git-dir").Output()
	if err != nil {
		t.Fatal(err)
	}
	adminLease := filepath.Join(strings.TrimSpace(string(adminGitDirOut)), "dotf-worktree.json")
	if err := os.WriteFile(adminLease, []byte(`{"lane":"feat-lane-secondary","state":"PARKED","reap_ok":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, adminWT); err == nil || !strings.Contains(out, "PARKED") {
		t.Fatalf("expected a PARKED admin-dir lease with no in-tree copy to block the commit, got err=%v out=%s", err, out)
	}
	if err := os.WriteFile(adminLease, []byte(`{"lane":"feat-lane-secondary","state":"ACTIVE","reap_ok":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, adminWT); err != nil {
		t.Fatalf("expected a detached worktree with an ACTIVE admin-dir lease and no in-tree copy to pass rule 6, got err=%v out=%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(adminWT, ".dotf-worktree.json"), []byte(`{"lane":"feat-lane-secondary","state":"PARKED","reap_ok":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runLaneGuardInDir(t, script, adminWT); err != nil {
		t.Fatalf("expected the ACTIVE admin-dir lease to outrank a stale PARKED in-tree copy, got err=%v out=%s", err, out)
	}
	// 6d. The main checkout on a detached HEAD keeps its own wording.
	detachedMain := filepath.Join(t.TempDir(), "detached-main")
	initGitRepoForLaneGuard(t, detachedMain, "main")
	writeLanesMarkerFixture(t, detachedMain)
	armLaneGuard(t, detachedMain)
	if out, err := exec.Command("git", "-C", detachedMain, "checkout", "--detach").CombinedOutput(); err != nil {
		t.Fatalf("checkout --detach: %v\n%s", err, string(out))
	}
	if out, err := runLaneGuardInDir(t, script, detachedMain); err == nil || !strings.Contains(out, "at the main checkout") {
		t.Fatalf("expected a detached HEAD at the main checkout to be blocked with the main-checkout message, got err=%v out=%s", err, out)
	}
	if out, err := runLaneGuardInDir(t, script, detachedMain); err == nil || !strings.Contains(out, "dotf lane switch <type>/<issue>-<slug>") || strings.Contains(out, "worktree adopt") {
		t.Fatalf("the main checkout is no worktree to adopt: its recovery is lane switch, got err=%v out=%s", err, out)
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
	customMarker := `{"version":1,"canonical_lanes":["feat-lane-primary"],"third_party_roots":["acme-sessions"]}`
	activateLanesFixture(t, customRepo, []byte(customMarker))
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
	activateLanesFixture(t, norootsRepo, []byte(`{"version":1,"canonical_lanes":["feat-lane-primary"]}`))
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
	activateLanesFixture(t, altRepo, []byte(`{"version":1,"canonical_lanes":["spike-lane"]}`))
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

func gitInLaneGuardTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

// The guard arms from the activation record and from nothing else (#3437): a
// marker in a working tree, committed or not, arms no checkout.
func TestPreCommitLaneGuard_ArmsOnlyFromTheActivationRecord(t *testing.T) {
	script := findLaneGuardScript(t)

	t.Run("a marker with no record blocks nothing, at the main checkout or in a worktree", func(t *testing.T) {
		repo := filepath.Join(t.TempDir(), "marker-only")
		initGitRepoForLaneGuard(t, repo, "main")
		writeLanesMarkerFixture(t, repo)
		gitInLaneGuardTest(t, repo, "add", ".dotf/lanes.json")
		gitInLaneGuardTest(t, repo, "commit", "-q", "-m", "declare lanes", "--no-verify")
		if out, err := runLaneGuardInDir(t, script, repo); err != nil {
			t.Fatalf("a committed marker alone must not arm the main checkout, got err=%v out=%s", err, out)
		}
		linked := filepath.Join(t.TempDir(), "linked-main")
		gitInLaneGuardTest(t, repo, "switch", "-q", "-c", "elsewhere")
		gitInLaneGuardTest(t, repo, "worktree", "add", "-q", linked, "main")
		if out, err := runLaneGuardInDir(t, script, linked); err != nil {
			t.Fatalf("a committed marker alone must not arm a worktree holding main, got err=%v out=%s", err, out)
		}
	})

	t.Run("a record arms main in the main checkout and passes a branch worktree", func(t *testing.T) {
		repo := filepath.Join(t.TempDir(), "armed")
		initGitRepoForLaneGuard(t, repo, "main")
		armLaneGuard(t, repo)
		if out, err := runLaneGuardInDir(t, script, repo); err == nil || !strings.Contains(out, "'main' at the main checkout") {
			t.Fatalf("expected main at the main checkout to be blocked, got err=%v out=%s", err, out)
		}
		branchWT := filepath.Join(t.TempDir(), "branch-wt")
		gitInLaneGuardTest(t, repo, "worktree", "add", "-q", "-b", "fix/1-thing", branchWT, "HEAD")
		if out, err := runLaneGuardInDir(t, script, branchWT); err != nil {
			t.Fatalf("expected a branch worktree to pass, got err=%v out=%s", err, out)
		}
	})

	t.Run("the deploy clone is never armed, even with the committed marker", func(t *testing.T) {
		// One origin, two clones: the machine's dev clone is activated (the
		// record lives in ITS git common dir); the deploy clone carries the very
		// same committed marker and has no record.
		origin := filepath.Join(t.TempDir(), "origin")
		initGitRepoForLaneGuard(t, origin, "main")
		writeLanesMarkerFixture(t, origin)
		gitInLaneGuardTest(t, origin, "add", ".dotf/lanes.json")
		gitInLaneGuardTest(t, origin, "commit", "-q", "-m", "declare lanes", "--no-verify")
		root := t.TempDir()
		dev, deploy := filepath.Join(root, "dev"), filepath.Join(root, "deploy")
		for _, dir := range []string{dev, deploy} {
			if out, err := exec.Command("git", "clone", "-q", origin, dir).CombinedOutput(); err != nil {
				t.Fatalf("clone: %v\n%s", err, string(out))
			}
		}
		armLaneGuard(t, dev)
		if out, err := runLaneGuardInDir(t, script, dev); err == nil || !strings.Contains(out, "'main' at the main checkout") {
			t.Fatalf("expected the activated clone to block main, got err=%v out=%s", err, out)
		}
		if _, err := os.Stat(filepath.Join(deploy, ".dotf", "lanes.json")); err != nil {
			t.Fatalf("the deploy clone must carry the committed marker for this case to mean anything: %v", err)
		}
		if out, err := runLaneGuardInDir(t, script, deploy); err != nil {
			t.Fatalf("expected the deploy clone to pass on main despite the marker, got err=%v out=%s", err, out)
		}
	})
}

// An expired lease does not block a commit, it warns (#3438): the work may be
// done, but the person committing is not the one to be stopped for it. The
// instant is compared across UTC offsets, because dotf writes local time.
func TestPreCommitLaneGuard_ExpiredLeaseWarnsAndDoesNotBlock(t *testing.T) {
	script := findLaneGuardScript(t)
	repo := filepath.Join(t.TempDir(), "armed")
	initGitRepoForLaneGuard(t, repo, "main")
	armLaneGuard(t, repo)
	wt := filepath.Join(t.TempDir(), "review-wt")
	gitInLaneGuardTest(t, repo, "worktree", "add", "-q", "--detach", wt, "HEAD")
	lease := filepath.Join(gitInLaneGuardTest(t, wt, "rev-parse", "--absolute-git-dir"), "dotf-worktree.json")

	// at renders an instant, offset from now, in a zone far from UTC.
	at := func(d time.Duration, zone *time.Location) string {
		return time.Now().Add(d).In(zone).Format(time.RFC3339Nano)
	}
	west, east := time.FixedZone("west", -12*3600), time.FixedZone("east", 14*3600)
	cases := []struct {
		name    string
		state   string
		expires string
		warns   bool
	}{
		{"expired, written in UTC", "ACTIVE", at(-time.Hour, time.UTC), true},
		{"expired, written east of UTC", "ACTIVE", at(-30*time.Minute, east), true},
		{"still valid, written west of UTC", "ACTIVE", at(30*time.Minute, west), false},
		{"still valid, written in UTC", "ACTIVE", at(time.Hour, time.UTC), false},
		{"no expiry (the zero time)", "ACTIVE", "0001-01-01T00:00:00Z", false},
		{"no expiry field", "ACTIVE", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"lane":"","state":"` + tc.state + `"`
			if tc.expires != "" {
				body += `,"lease_expires_at":"` + tc.expires + `"`
			}
			if err := os.WriteFile(lease, []byte(body+"}"), 0o644); err != nil {
				t.Fatal(err)
			}
			out, err := runLaneGuardInDir(t, script, wt)
			if err != nil {
				t.Fatalf("a lease must never block a commit here, got err=%v out=%s", err, out)
			}
			if got := strings.Contains(out, "WARN:") && strings.Contains(out, "lease expired"); got != tc.warns {
				t.Fatalf("warns = %v, want %v; out=%q", got, tc.warns, out)
			}
		})
	}
}
