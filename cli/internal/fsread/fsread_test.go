package fsread

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// The paths Glob would match, in its order: a file that is an entry of root
// (ENOTDIR on the way down) and a child without rel are skipped, and a
// directory at rel counts, as with Glob.
func TestChildFilesMatchesGlob(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "b", "context.md"))
	write(t, filepath.Join(root, "a", "context.md"))
	write(t, filepath.Join(root, "c", "other.md"))
	write(t, filepath.Join(root, "file.md"))
	if err := os.MkdirAll(filepath.Join(root, "d", "context.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	found, unreadable, err := ChildFiles(root, "context.md")
	want, _ := filepath.Glob(filepath.Join(root, "*", "context.md"))
	if err != nil || len(unreadable) > 0 || !slices.Equal(found, want) {
		t.Fatalf("ChildFiles = %v, %v, %v; want %v", found, unreadable, err, want)
	}
}

func TestChildFilesMissingRootHoldsNothing(t *testing.T) {
	found, unreadable, err := ChildFiles(filepath.Join(t.TempDir(), "absent"), "context.md")
	if err != nil || found != nil || unreadable != nil {
		t.Fatalf("ChildFiles = %v, %v, %v; want nothing and no error", found, unreadable, err)
	}
}

// The two errors Glob drops: an unreadable root, and a child that cannot be
// searched. Neither may read as "no match".
func TestChildFilesReportsWhatItCannotRead(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a mode bit that denies the reader: not on Windows, not as root")
	}
	root := t.TempDir()
	write(t, filepath.Join(root, "a", "context.md"))
	write(t, filepath.Join(root, "locked", "context.md"))
	locked := filepath.Join(root, "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	found, unreadable, err := ChildFiles(root, "context.md")
	if err != nil || len(found) != 1 || len(unreadable) != 1 || unreadable[0].Path != filepath.Join(locked, "context.md") {
		t.Fatalf("ChildFiles = %v, %v, %v; want a/context.md found and locked/context.md unreadable", found, unreadable, err)
	}

	if err := os.Chmod(root, 0o300); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	if _, _, err := ChildFiles(root, "context.md"); err == nil {
		t.Fatal("ChildFiles on an unreadable root returned no error")
	}
}
