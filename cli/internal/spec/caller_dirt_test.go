package spec

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// HARNESS-142 AC5: the reviewer reads a fresh worktree at the reviewed commit,
// so the caller's uncommitted work outside the spec folder is invisible to it.
// Dirt inside specs/<id>/ is carried into the worktree and does not count.
func TestCallerDirtNamesWhatTheReviewerWouldNotSee(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	if err := os.WriteFile(filepath.Join(repo, "keep.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	specDir := filepath.Join(repo, "specs", "AI-001-x")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "init")

	if got, err := CallerDirt(specDir); err != nil || len(got) != 0 {
		t.Fatalf("clean tree: %q, %v", got, err)
	}

	if err := os.WriteFile(filepath.Join(specDir, "tasks.md"), []byte("- [ ] x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := CallerDirt(specDir); err != nil || len(got) != 0 {
		t.Fatalf("dirt inside the spec folder counted: %q, %v", got, err)
	}

	if err := os.WriteFile(filepath.Join(repo, "keep.go"), []byte("package y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "tests", "probe_test.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := CallerDirt(specDir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"keep.go", "tests/probe_test.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("caller dirt = %q, want %q", got, want)
	}
}
