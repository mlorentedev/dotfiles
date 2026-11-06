package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		// the quoting stripped.
		plain := strings.NewReplacer("'", "", "\\", "").Replace(out)
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
	if _, err := os.Stat(filepath.Join(treeSpec, "review.md")); !os.IsNotExist(err) {
		t.Errorf("the rotated verdict is still in the review slot of the worktree: %v", err)
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
