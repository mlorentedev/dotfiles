package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

// caseFoldedTwin returns dir spelled with its last element lowercased, and
// skips when the filesystem is case-sensitive: there the two spellings are two
// directories, and the case under test does not exist.
func caseFoldedTwin(t *testing.T) (dir, twin string) {
	t.Helper()
	dir = filepath.Join(t.TempDir(), "Checkout")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	twin = filepath.Join(filepath.Dir(dir), "checkout")
	if _, err := os.Stat(twin); err != nil {
		t.Skip("case-sensitive filesystem: the two spellings name two directories")
	}
	return dir, twin
}

// macOS's default APFS volume is case-insensitive, like NTFS, and
// filepath.EvalSymlinks keeps the spelling it was given, so folding case only
// on Windows reported one checkout as two (#2094).
func TestSameCheckoutRoot_OneDirectoryInAnotherCaseIsTheSame(t *testing.T) {
	dir, twin := caseFoldedTwin(t)
	if !sameCheckoutRoot(twin, dir) {
		t.Errorf("%s and %s are one directory", twin, dir)
	}
}

func TestSamePath_OneDirectoryInAnotherCaseIsTheSame(t *testing.T) {
	dir, twin := caseFoldedTwin(t)
	if !samePath(twin+"/", dir) {
		t.Errorf("%s and %s are one directory", twin, dir)
	}
}

func TestSameCheckoutRootAndSamePath_TellTwoDirectoriesApart(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	if sameCheckoutRoot(a, b) || samePath(a, b) {
		t.Errorf("%s and %s are two directories", a, b)
	}
}

// A hooksPath that does not exist yet is still compared, by its spelling.
func TestSamePath_ComparesMissingPathsBySpelling(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "hooks")
	if !samePath(filepath.ToSlash(missing)+"/", missing) {
		t.Errorf("%s must match its own slash-and-trailing-separator spelling", missing)
	}
}
