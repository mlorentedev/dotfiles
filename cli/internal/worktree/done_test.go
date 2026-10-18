package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestDoneTeardown(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")

	// 1. Refuse teardown if path is main repository
	_, err := Done(DoneOptions{
		RepoRoot:     mainRepo,
		WorktreePath: mainRepo,
	})
	if err == nil {
		t.Errorf("expected Done on main repo to fail")
	}

	// 2. Error if worktreePath is empty
	_, err = Done(DoneOptions{
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
	stubMergedPRHeads(t, nil, nil)

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
	_, err := Done(DoneOptions{
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
	_, err = Done(DoneOptions{
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
	_, err := Done(DoneOptions{
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

// samePath reports whether two paths name the same location, after resolving
// the two ways one location can spell itself differently here: on Windows
// t.TempDir() hands back the 8.3 short form (C:\Users\RUNNER~1\...) while git
// reports the long form (C:\Users\runneradmin\...), and on macOS /var is a
// symlink to /private/var. Comparing the raw strings passes on Linux and fails
// on the other two for reasons that have nothing to do with the resolvers.
func samePath(a, b string) bool {
	return normalizePath(a) == normalizePath(b)
}

func normalizePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(resolved)
}

// assertFilepathConvention pins that a resolver answers in filepath's separator
// convention, not git's. It has to be checked separately from samePath, which
// runs both sides through filepath.Clean and would therefore paper over exactly
// this: git emits forward slashes on Windows, so a resolver returning its raw
// stdout compares equal under samePath while handing every caller a path in the
// wrong convention. On Linux the two conventions coincide and this can only
// pass -- the assertion earns its keep on the Windows CI leg.
func assertFilepathConvention(t *testing.T, label, path string) {
	t.Helper()
	if path != filepath.Clean(path) {
		t.Errorf("%s returned %q, which is not in filepath convention (want %q)", label, path, filepath.Clean(path))
	}
}

func testResolveFromRoot(t *testing.T, wtDir, absRepoDir, absWTDir string) {
	mainRoot, err := ResolveMainRepoRoot(wtDir)
	if err != nil {
		t.Fatalf("unexpected error resolving main repo root: %v", err)
	}
	if !samePath(mainRoot, absRepoDir) {
		t.Errorf("expected mainRoot to be %s, got %s", absRepoDir, mainRoot)
	}

	wtRoot, err := ResolveWorktreeRoot(wtDir)
	if err != nil {
		t.Fatalf("unexpected error resolving worktree root: %v", err)
	}
	if !samePath(wtRoot, absWTDir) {
		t.Errorf("expected wtRoot to be %s, got %s", absWTDir, wtRoot)
	}

	assertFilepathConvention(t, "ResolveMainRepoRoot", mainRoot)
	assertFilepathConvention(t, "ResolveWorktreeRoot", wtRoot)
}

func testResolveFromSubdir(t *testing.T, wtDir, absRepoDir, absWTDir string) (string, string) {
	subDir := filepath.Join(wtDir, "sub", "nested")
	_ = os.MkdirAll(subDir, 0o755)

	subMainRoot, err := ResolveMainRepoRoot(subDir)
	if err != nil {
		t.Fatalf("unexpected error resolving main root from subDir: %v", err)
	}
	if !samePath(subMainRoot, absRepoDir) {
		t.Errorf("expected subMainRoot to be %s, got %s", absRepoDir, subMainRoot)
	}

	subWTRoot, err := ResolveWorktreeRoot(subDir)
	if err != nil {
		t.Fatalf("unexpected error resolving wt root from subDir: %v", err)
	}
	if !samePath(subWTRoot, absWTDir) {
		t.Errorf("expected subWTRoot to be %s, got %s", absWTDir, subWTRoot)
	}

	assertFilepathConvention(t, "ResolveMainRepoRoot (from subdir)", subMainRoot)
	assertFilepathConvention(t, "ResolveWorktreeRoot (from subdir)", subWTRoot)
	// Returned unnormalised on purpose: the caller feeds these straight into
	// Done, which is what the in-worktree CLI flow does with them.
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
	// Its own lock: the default is machine-wide, shared with every other
	// package's tests and any real `dotf worktree` running at the time.
	_, err := Done(DoneOptions{
		RepoRoot:     subMainRoot,
		WorktreePath: subWTRoot,
		Force:        false,
		LockPath:     filepath.Join(t.TempDir(), "worktree.lock"),
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
	_, err := Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        false,
	})
	if err == nil {
		t.Fatalf("expected Done without force to fail when non-disposable gitignored file exists")
	}
	// #1607: the refusal names the path and does not call it uncommitted work.
	if msg := err.Error(); !strings.Contains(msg, ".env") || !strings.Contains(msg, "not known build output") || strings.Contains(msg, "uncommitted") {
		t.Errorf("refusal must name the ignored path and not call it uncommitted, got: %v", err)
	}

	content, err := os.ReadFile(envFile)
	if err != nil || string(content) != "SECRET=precious_credentials" {
		t.Fatalf("expected .env to be preserved, err=%v content=%s", err, string(content))
	}

	// 2. Done with force removes worktree
	_, err = Done(DoneOptions{
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

// #2750: a merged worktree whose only leftovers are what a repo's test or
// bootstrap target regenerates (a lockfile, coverage reports, test caches) is
// removed without --force. Refusing it taught --force as the default and so
// disarmed the refusal for the file that IS work, which still stops it.
func TestDoneRemovesAWorktreeHoldingOnlyRegenerableOutput(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	if err := os.WriteFile(filepath.Join(repoDir, ".gitignore"), []byte(".env\npoetry.lock\ncoverage.xml\n.coverage\n.coverage.*\nhtmlcov/\n.pytest_cache/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"-C", repoDir, "add", ".gitignore"}, {"-C", repoDir, "commit", "-m", "add gitignore"}, {"-C", wtDir, "merge", "main"}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	for _, rel := range []string{"poetry.lock", "coverage.xml", ".coverage", ".coverage.host.42", "htmlcov/index.html", ".pytest_cache/v/cache/nodeids"} {
		p := filepath.Join(wtDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("regenerated\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := filepath.Join(wtDir, ".env")
	if err := os.WriteFile(env, []byte("SECRET=x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath}

	_, err := Done(opts)
	if err == nil || !strings.Contains(err.Error(), ".env") {
		t.Fatalf("a .env beside the regenerable files must still stop done, got: %v", err)
	}
	for _, name := range []string{"poetry.lock", "coverage.xml", "htmlcov", ".pytest_cache"} {
		if strings.Contains(err.Error(), name) {
			t.Errorf("the refusal must name only the .env, got: %v", err)
		}
	}

	if err := os.Remove(env); err != nil {
		t.Fatal(err)
	}
	if _, err := Done(opts); err != nil {
		t.Fatalf("regenerable output alone must not need --force, got: %v", err)
	}
	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("expected the worktree to be removed")
	}
}

// #1653 AC2: done refuses a worktree that holds the caller's working directory,
// and --force does not lift it. --force is about uncommitted changes, and it is
// routine; the removal it would allow here strands the caller, and a Claude Code
// session launched from the worktree cannot be resumed once it is gone.
func TestDoneRefusesTheWorktreeHoldingTheCallersCwdEvenWithForce(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	sub := filepath.Join(wtDir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, cwd := range []string{wtDir, sub} {
		_, err := Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath, Cwd: cwd, Force: true})
		if err == nil || !strings.Contains(err.Error(), "current directory") {
			t.Fatalf("cwd %s: want a refusal naming the current directory, got %v", cwd, err)
		}
		if _, statErr := os.Stat(wtDir); statErr != nil {
			t.Fatalf("cwd %s: the worktree must survive the refusal: %v", cwd, statErr)
		}
	}
}

// A sibling whose name extends the target's (repo-wt-feat-2 beside repo-wt-feat)
// is not inside it.
func TestDoneAllowsACwdInASiblingSharingANamePrefix(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	sibling := wtDir + "-2"
	if err := os.Mkdir(sibling, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath, Cwd: sibling}); err != nil {
		t.Fatalf("a sibling cwd must not block done: %v", err)
	}
}

// A cwd reached through a symlink is compared by where it resolves.
func TestDoneRefusesACwdReachedThroughASymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(wtDir, link); err != nil {
		t.Fatal(err)
	}
	_, err := Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath, Cwd: link, Force: true})
	if err == nil || !strings.Contains(err.Error(), "current directory") {
		t.Fatalf("want a refusal for a symlinked cwd, got %v", err)
	}
}

// A process the caller runs under (a shell, a Claude Code session) sitting in
// the worktree blocks done the same way, also under --force. The seam drives
// the refusing branch on every platform; done_proc_linux_test.go drives the
// real walk.
func TestDoneRefusesWhenAnAncestorProcessIsInsideEvenWithForce(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	orig := callerInside
	t.Cleanup(func() { callerInside = orig })
	callerInside = func(target string) (ancestor, bool) {
		return ancestor{PID: 4242, Comm: "claude", Cwd: target}, true
	}
	_, err := Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath, Force: true})
	if err == nil || !strings.Contains(err.Error(), "4242") || !strings.Contains(err.Error(), "claude") {
		t.Fatalf("want a refusal naming the process, got %v", err)
	}
	if _, statErr := os.Stat(wtDir); statErr != nil {
		t.Fatalf("the worktree must survive the refusal: %v", statErr)
	}
}

// #1607: a tracked change and ignored content are two facts. Dirt keeps them
// apart, and reads the ignored paths only when nothing tracked is dirty.
func TestRealGitRunnerDirtSeparatesTrackedFromIgnored(t *testing.T) {
	_, wtDir, _ := setupTestGitRepoAndWorktree(t)
	_ = os.WriteFile(filepath.Join(wtDir, ".gitignore"), []byte(".env\nnode_modules/\nnotes*\n"), 0o644)
	for _, args := range [][]string{{"add", ".gitignore"}, {"commit", "-m", "ignore"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = wtDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runner := &RealGitRunner{}

	_ = os.MkdirAll(filepath.Join(wtDir, "node_modules"), 0o755)
	_ = os.WriteFile(filepath.Join(wtDir, "node_modules", "x.js"), []byte("x"), 0o644)
	dirt, err := runner.Dirt(wtDir)
	if err != nil || dirt.Any() {
		t.Fatalf("disposable ignored content only: Dirt = %+v, err = %v; want clean", dirt, err)
	}

	_ = os.WriteFile(filepath.Join(wtDir, ".env"), []byte("K=V"), 0o644)
	dirt, err = runner.Dirt(wtDir)
	if err != nil || dirt.Tracked || !reflect.DeepEqual(dirt.Ignored, []string{".env"}) {
		t.Fatalf("ignored .env: Dirt = %+v, err = %v; want Ignored [.env] and nothing tracked", dirt, err)
	}

	// A name with a space is reported as itself, not in git's quotes.
	_ = os.WriteFile(filepath.Join(wtDir, "notes draft.txt"), []byte("n"), 0o644)
	dirt, err = runner.Dirt(wtDir)
	if err != nil || !reflect.DeepEqual(dirt.Ignored, []string{".env", "notes draft.txt"}) {
		t.Fatalf("ignored name with a space: Dirt = %+v, err = %v; want [.env notes draft.txt]", dirt, err)
	}

	_ = os.WriteFile(filepath.Join(wtDir, "initial.txt"), []byte("changed"), 0o644)
	dirt, err = runner.Dirt(wtDir)
	if err != nil || !dirt.Tracked {
		t.Fatalf("tracked change: Dirt = %+v, err = %v; want Tracked", dirt, err)
	}
}
