// Package consoletest compares console output with golden files (CLI-114).
package consoletest

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

// Golden compares got with testdata/<name>.golden in the calling package's
// directory, or rewrites the file under -update. A golden file is the
// console's contract: a change to it is a change to what the owner reads,
// reviewed as a diff.
func Golden(t testing.TB, name string, got []byte) {
	t.Helper()
	p := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil { //nolint:gosec // test fixtures
			t.Fatal(err)
		}
		if err := os.WriteFile(p, got, 0o644); err != nil { //nolint:gosec // test fixtures
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p) //nolint:gosec // the package's own testdata
	if err != nil {
		t.Fatalf("%v (run the test with -update to create it)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output differs from %s\n--- got\n%q\n--- want\n%q", p, got, want)
	}
}
