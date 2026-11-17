package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// HARNESS-142 AC4: the reviewer works in the review worktree, on both runners.
// agy is told the directory through --add-dir, tmux starts there, the prompt
// names it, and deadline is told to copy the verdict back and remove it.
func TestReviewLaunchUsesReviewWorktree(t *testing.T) {
	root := makeRepo(t)
	tree := stubReviewTree(t)
	pool := `{"pool":[
 {"id":"nan/deepseek-v4-flash","runner":"pi","provider":"nan","model":"deepseek-v4-flash"},
 {"id":"agy/gemini-3.8-flash-high","runner":"agy","model":"gemini-3.8-flash-high","auth":"login"}]}`
	if err := os.MkdirAll(filepath.Join(root, "harness"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "harness", "reviewer-pool.json"), []byte(pool), 0o644); err != nil {
		t.Fatal(err)
	}
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, true)

	for _, reviewer := range []string{"agy/gemini-3.8-flash-high", "nan/deepseek-v4-flash"} {
		out, errOut, err := execute(t, "spec", "review", "AI-001-x", "--dry-run", "--reviewer", reviewer)
		if err != nil {
			t.Fatalf("%s: %v\n%s", reviewer, err, out+errOut)
		}
		// The tmux form quotes the whole pipeline once more, so compare with
		// the quoting stripped. Only the quote escape goes: a Windows path keeps
		// its backslashes.
		plain := strings.NewReplacer(`'\''`, "", "'", "").Replace(out)
		wants := []string{
			"-c " + tree + " ",
			"Repo: " + tree,
			"fresh worktree checked out at headheadhead",
			"--worktree " + tree + " ",
			"--copy-back " + filepath.Join(root, "specs", "AI-001-x") + " ",
			"--review " + filepath.Join(tree, "specs", "AI-001-x", "review.md") + " ",
		}
		if strings.HasPrefix(reviewer, "agy/") {
			wants = append(wants, "--add-dir "+tree+" ")
		}
		for _, want := range wants {
			if !strings.Contains(plain, want) {
				t.Errorf("%s: the launch does not contain %q:\n%s", reviewer, want, out)
			}
		}
		if strings.Contains(plain, "-c "+root+" ") {
			t.Errorf("%s: tmux still starts in the caller's tree:\n%s", reviewer, out)
		}
	}
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Errorf("a dry run created the review worktree: %v", err)
	}
}

// HARNESS-142 AC6: a second first-signer round runs in a worktree at HEAD, so
// the first round's uncommitted verdict reaches it only by the copy-in.
func TestReviewWorktreeCarriesEarlierRounds(t *testing.T) {
	root := makeRepo(t)
	tree := stubReviewTree(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, true)

	if out, errOut, err := execute(t, "spec", "review", "AI-001-x"); err != nil {
		t.Fatalf("launch failed: %v\n%s", err, out+errOut)
	}
	treeSpec := filepath.Join(tree, "specs", "AI-001-x")
	for _, f := range []string{"review-round1.md", "review-request-round1.json", "review-request.json", "proposal.md"} {
		if _, err := os.Stat(filepath.Join(treeSpec, f)); err != nil {
			t.Errorf("the review worktree lacks %s: %v", f, err)
		}
	}
	// seedSpec gave the caller a review.md, so this holds only if the copy-in
	// runs after the rotation. A verdict committed at HEAD is the other route
	// into the slot: TestReviewWorktreeDropsAVerdictTheCallerRotated.
	if _, err := os.Stat(filepath.Join(treeSpec, "review.md")); !os.IsNotExist(err) {
		t.Errorf("the rotated verdict is still in the review slot of the worktree: %v", err)
	}
}

// A verdict committed at HEAD is checked out into the worktree; once the caller
// rotated it aside, the worktree must not keep it, or a run that ends without a
// verdict copies the previous round's back as if it were new.
func TestReviewWorktreeDropsAVerdictTheCallerRotated(t *testing.T) {
	root := makeRepo(t)
	tree := stubReviewTree(t)
	createReviewTree = func(rt reviewTree, _ string, _ time.Duration) error {
		committed := filepath.Join(rt.path, "specs", "AI-001-x")
		if err := os.MkdirAll(committed, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(committed, "review.md"), []byte("---\nverdict: PASS\n---\n"), 0o644)
	}
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, true)

	if out, errOut, err := execute(t, "spec", "review", "AI-001-x"); err != nil {
		t.Fatalf("launch failed: %v\n%s", err, out+errOut)
	}
	if _, err := os.Stat(filepath.Join(tree, "specs", "AI-001-x", "review.md")); !os.IsNotExist(err) {
		t.Errorf("the worktree kept the committed verdict the caller rotated aside: %v", err)
	}
}

// A foreground run whose reviewer never started (here, the transcript could not
// be created) leaves no review worktree behind for the lease to outlive.
func TestReviewForegroundRemovesTheWorktreeWhenTheRunNeverStarts(t *testing.T) {
	root := makeRepo(t)
	tree := stubReviewTree(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	prev := runForeground
	runForeground = func(string, []string, string) error { return os.ErrPermission }
	t.Cleanup(func() { runForeground = prev })

	if _, _, err := execute(t, "spec", "review", "AI-001-x", "--foreground"); err == nil {
		t.Fatal("a foreground run that never started succeeded")
	}
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Errorf("the review worktree outlived a foreground run that never started: %v", err)
	}
}

// A launch that fails after the worktree was made removes it again.
func TestReviewLaunchRemovesTheWorktreeWhenTmuxFails(t *testing.T) {
	root := makeRepo(t)
	tree := stubReviewTree(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, true)
	prev := runCommand
	runCommand = func(string, []string) error { return os.ErrPermission }
	t.Cleanup(func() { runCommand = prev })

	if _, _, err := execute(t, "spec", "review", "AI-001-x"); err == nil {
		t.Fatal("a launch whose tmux session could not start succeeded")
	}
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Errorf("the review worktree outlived a failed launch: %v", err)
	}
}

// A reviewer that dies on startup never reaches deadline, which is what removes
// the worktree, so the launcher removes it (#3223): otherwise it waits out a
// lease of deadline plus the margin with nothing running in it.
func TestReviewLaunchRemovesTheWorktreeWhenTheReviewerDiesOnStartup(t *testing.T) {
	root := makeRepo(t)
	tree := stubReviewTree(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, false)

	if _, _, err := execute(t, "spec", "review", "AI-001-x"); err == nil {
		t.Fatal("a launch whose reviewer died on startup succeeded")
	}
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Errorf("the review worktree outlived a reviewer that died on startup: %v", err)
	}
}

// The same premise as TestReviewWorktreeDropsAVerdictTheCallerRotated, out of
// real git rather than a stubbed createReviewTree (#3223): `git worktree add
// --detach` does check HEAD's committed verdict out into the review worktree,
// and the copy-in of the caller's folder, which rotated it aside, drops it.
func TestReviewWorktreeDropsACommittedVerdictOutOfRealGit(t *testing.T) {
	repo := worktreeTestRepo(t)
	callerSpec := filepath.Join(repo, "specs", "AI-001-x")
	if err := os.MkdirAll(callerSpec, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"proposal.md": "# AI-001-x\n", "review.md": "---\nverdict: PASS\n---\n"} {
		if err := os.WriteFile(filepath.Join(callerSpec, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) string {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = repo
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("add", "specs")
	git("commit", "-q", "-m", "the previous round's verdict")
	head := git("rev-parse", "HEAD")
	// The caller rotates the verdict aside before the launch.
	if err := os.Remove(filepath.Join(callerSpec, "review.md")); err != nil {
		t.Fatal(err)
	}

	tree, err := planReviewTree(repo, "AI-001-x", spec.FirstSigner)
	if err != nil {
		t.Fatal(err)
	}
	if err := createReviewTree(tree, head, time.Hour); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = removeReviewTree(tree.path) })
	treeSpec := filepath.Join(tree.path, "specs", "AI-001-x")
	if _, err := os.Stat(filepath.Join(treeSpec, "review.md")); err != nil {
		t.Fatalf("the premise does not hold: git did not check the committed verdict out: %v", err)
	}
	if err := mirrorSpecFolder(callerSpec, treeSpec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(treeSpec, "review.md")); !os.IsNotExist(err) {
		t.Errorf("the worktree kept the committed verdict the caller rotated aside: %v", err)
	}
}
