package spec

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// #1649, fourth instance: a reviewer restored deleted files from the parent
// commit and left them STAGED. A staged change to a tracked file, and a staged
// rename, are leftovers as much as an untracked probe is.
func TestStrayWritesSeeStagedChangesAndRenames(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	for _, f := range []string{"keep.go", "old.go"} {
		if err := os.WriteFile(filepath.Join(repo, f), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "init")
	review := filepath.Join(repo, "specs", "AI-001-x", "review.md")
	if err := os.MkdirAll(filepath.Dir(review), 0o755); err != nil {
		t.Fatal(err)
	}

	before, err := SnapshotTree(review)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "keep.go"), []byte("package y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "mv", "old.go", "new.go")
	git(t, repo, "add", "keep.go")
	if err := os.WriteFile(review, []byte("verdict\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := SnapshotTree(review)
	if err != nil {
		t.Fatal(err)
	}

	got := before.StrayWrites(after)
	if want := []string{"keep.go", "new.go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("stray writes = %q, want %q (the rename's source must not be read as a path of its own)", got, want)
	}
	if before.Folder() != "specs/AI-001-x/" {
		t.Errorf("folder = %q, want the spec folder relative to the repository root", before.Folder())
	}
}
