package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoneTeardown(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")

	// 1. Refuse teardown if path is main repository
	err := Done(DoneOptions{
		RepoRoot:     mainRepo,
		WorktreePath: mainRepo,
	})
	if err == nil {
		t.Errorf("expected Done on main repo to fail")
	}

	// 2. Error if worktreePath is empty
	err = Done(DoneOptions{
		RepoRoot:     mainRepo,
		WorktreePath: "",
	})
	if err == nil {
		t.Errorf("expected empty worktreePath to fail")
	}
}

func setupTestGitRepoAndWorktree(t *testing.T) (repoDir, wtDir, lockPath string) {
	tmpDir := t.TempDir()
	repoDir = filepath.Join(tmpDir, "repo")
	wtDir = filepath.Join(tmpDir, "repo-wt-feat")
	lockPath = filepath.Join(tmpDir, "test.lock")

	runGit := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(out))
		}
	}

	_ = os.MkdirAll(repoDir, 0o755)
	runGit(repoDir, "init", "-b", "main")
	runGit(repoDir, "config", "user.name", "Test User")
	runGit(repoDir, "config", "user.email", "test@example.com")
	_ = os.WriteFile(filepath.Join(repoDir, "initial.txt"), []byte("hello"), 0o644)
	runGit(repoDir, "add", "initial.txt")
	runGit(repoDir, "commit", "-m", "initial commit")

	runGit(repoDir, "worktree", "add", "-b", "feat/test", wtDir)
	return repoDir, wtDir, lockPath
}

func TestDoneRefusesUnpushedCommitsWithNoUpstream(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)

	// Add an unpushed commit on feat/test (which has no upstream)
	filePath := filepath.Join(wtDir, "wip.txt")
	_ = os.WriteFile(filePath, []byte("unpushed commit content"), 0o644)
	cmd := exec.Command("git", "add", "wip.txt")
	cmd.Dir = wtDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "local commit without upstream")
	cmd.Dir = wtDir
	_ = cmd.Run()

	// 1. Done without force must refuse to remove
	err := Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        false,
	})
	if err == nil {
		t.Fatalf("expected Done to fail on branch with unpushed commits and no upstream, but got nil")
	}
	if !strings.Contains(err.Error(), "unpushed commit") {
		t.Errorf("expected error to mention unpushed commit, got: %v", err)
	}

	// Verify worktree still exists
	if _, statErr := os.Stat(wtDir); os.IsNotExist(statErr) {
		t.Errorf("worktree directory was deleted despite unpushed commits!")
	}

	// 2. Done with force must succeed
	err = Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        true,
	})
	if err != nil {
		t.Fatalf("expected Done with Force=true to succeed, got: %v", err)
	}

	// Verify worktree is removed
	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("expected worktree directory to be removed after force Done")
	}
}

func TestDoneSucceedsOnCleanBranchWithoutCommits(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)

	// Fresh worktree has 0 commits ahead of main and no upstream
	err := Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        false,
	})
	if err != nil {
		t.Fatalf("expected Done on fresh clean worktree to succeed, got: %v", err)
	}

	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("expected worktree directory to be removed")
	}
}

func testResolveFromRoot(t *testing.T, wtDir, absRepoDir, absWTDir string) {
	mainRoot, err := ResolveMainRepoRoot(wtDir)
	if err != nil {
		t.Fatalf("unexpected error resolving main repo root: %v", err)
	}
	if mainRoot != absRepoDir {
		t.Errorf("expected mainRoot to be %s, got %s", absRepoDir, mainRoot)
	}

	wtRoot, err := ResolveWorktreeRoot(wtDir)
	if err != nil {
		t.Fatalf("unexpected error resolving worktree root: %v", err)
	}
	if wtRoot != absWTDir {
		t.Errorf("expected wtRoot to be %s, got %s", absWTDir, wtRoot)
	}
}

func testResolveFromSubdir(t *testing.T, wtDir, absRepoDir, absWTDir string) (string, string) {
	subDir := filepath.Join(wtDir, "sub", "nested")
	_ = os.MkdirAll(subDir, 0o755)

	subMainRoot, err := ResolveMainRepoRoot(subDir)
	if err != nil {
		t.Fatalf("unexpected error resolving main root from subDir: %v", err)
	}
	if subMainRoot != absRepoDir {
		t.Errorf("expected subMainRoot to be %s, got %s", absRepoDir, subMainRoot)
	}

	subWTRoot, err := ResolveWorktreeRoot(subDir)
	if err != nil {
		t.Fatalf("unexpected error resolving wt root from subDir: %v", err)
	}
	if subWTRoot != absWTDir {
		t.Errorf("expected subWTRoot to be %s, got %s", absWTDir, subWTRoot)
	}
	return subMainRoot, subWTRoot
}

func TestResolveWorktreeAndMainRepoRoot(t *testing.T) {
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	absRepoDir, _ := filepath.Abs(repoDir)
	absWTDir, _ := filepath.Abs(wtDir)

	// 1. From root of linked worktree
	testResolveFromRoot(t, wtDir, absRepoDir, absWTDir)

	// 2. From subdirectory within linked worktree
	subMainRoot, subWTRoot := testResolveFromSubdir(t, wtDir, absRepoDir, absWTDir)

	// 3. Teardown using the resolved roots (the in-worktree CLI flow)
	err := Done(DoneOptions{
		RepoRoot:     subMainRoot,
		WorktreePath: subWTRoot,
		Force:        false,
	})
	if err != nil {
		t.Fatalf("expected Done with resolved roots to succeed, got: %v", err)
	}

	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("expected worktree directory to be removed after teardown")
	}
}

func TestDonePreservesGitignoredLocalFiles(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)

	// Add .gitignore with .env and node_modules
	gitIgnorePath := filepath.Join(repoDir, ".gitignore")
	_ = os.WriteFile(gitIgnorePath, []byte(".env\n*.tmp\nnode_modules/\n"), 0o644)
	cmd := exec.Command("git", "add", ".gitignore")
	cmd.Dir = repoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add gitignore")
	cmd.Dir = repoDir
	_ = cmd.Run()

	// Update feat/test to include gitignore and remain clean
	cmd = exec.Command("git", "merge", "main")
	cmd.Dir = wtDir
	_ = cmd.Run()

	// Plant a gitignored .env file
	envFile := filepath.Join(wtDir, ".env")
	if err := os.WriteFile(envFile, []byte("SECRET=precious_credentials"), 0o644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	// 1. Done without force must refuse to remove and protect .env
	err := Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        false,
	})
	if err == nil {
		t.Fatalf("expected Done without force to fail when non-disposable gitignored file exists")
	}

	content, err := os.ReadFile(envFile)
	if err != nil || string(content) != "SECRET=precious_credentials" {
		t.Fatalf("expected .env to be preserved, err=%v content=%s", err, string(content))
	}

	// 2. Done with force removes worktree
	err = Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        true,
	})
	if err != nil {
		t.Fatalf("expected Done with force to succeed, got: %v", err)
	}

	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("expected worktree directory to be removed after forced teardown")
	}
}
