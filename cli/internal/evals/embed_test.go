package evals

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// repoFile resolves a path under the repository root from this test file's
// location, so the test reads the real tree whatever the working directory.
func repoFile(t *testing.T, parts ...string) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	root := filepath.Join(filepath.Dir(here), "..", "..", "..")
	return filepath.Join(append([]string{root}, parts...)...)
}

// A drift between the embedded copy and harness/evals/ would make `dotf eval
// check` validate other repositories' cases against a schema nobody reviewed.
func TestEmbeddedCaseSchemaEqualsTheRepoSchema(t *testing.T) {
	want, err := os.ReadFile(repoFile(t, "harness", "evals", "case.schema.json"))
	if err != nil {
		// Fail, not skip: a test that skips when its source is gone checks nothing.
		t.Fatalf("harness/evals/case.schema.json is not here: %v", err)
	}
	if !bytes.Equal(embeddedCaseSchema, want) {
		t.Fatal("cli/internal/evals/case.schema.json differs from harness/evals/case.schema.json;" +
			" re-sync it: cp harness/evals/case.schema.json cli/internal/evals/case.schema.json")
	}
}
