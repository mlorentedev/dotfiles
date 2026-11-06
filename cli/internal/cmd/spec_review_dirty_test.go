package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// HARNESS-142 AC5, against a real repository: the reviewer runs in a fresh
// worktree, so a change outside specs/<id>/ would be reviewed as if absent.
func TestReviewRefusesDirtyTreeWithoutAllowDirty(t *testing.T) {
	root := makeRepo(t)
	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "init", "-q")
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	gitIn(t, root, "add", ".")
	gitIn(t, root, "commit", "-q", "-m", "init")
	prev := callerDirt
	callerDirt = spec.CallerDirt
	t.Cleanup(func() { callerDirt = prev })

	// Dirt inside the spec folder is carried to the reviewer: no refusal.
	if err := os.WriteFile(filepath.Join(root, "specs", "AI-001-x", "tasks.md"), []byte("- [ ] x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, errOut, err := execute(t, "spec", "review", "AI-001-x", "--dry-run"); err != nil {
		t.Fatalf("dirt inside specs/<id>/ refused the launch: %v\n%s", err, out+errOut)
	}

	if err := os.WriteFile(filepath.Join(root, "probe.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err == nil {
		t.Fatalf("an untracked file outside specs/<id>/ did not refuse:\n%s", out+errOut)
	}
	for _, want := range []string{"probe.go", "--allow-dirty"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not name %q: %v", want, err)
		}
	}
	if strings.Contains(out, "DRY RUN") {
		t.Errorf("printed a launch it refused:\n%s", out)
	}

	out, errOut, err = execute(t, "spec", "review", "AI-001-x", "--dry-run", "--allow-dirty")
	if err != nil {
		t.Fatalf("--allow-dirty still refused: %v\n%s", err, out+errOut)
	}
	if !strings.Contains(errOut, "probe.go") || !strings.Contains(errOut, "will not see") {
		t.Errorf("--allow-dirty launched without saying what the reviewer will not see:\n%s", errOut)
	}
}
