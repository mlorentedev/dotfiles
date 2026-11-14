package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvalCheckPassesARepositoryWithoutCases(t *testing.T) {
	stdout, _, err := execute(t, "eval", "check", "--repo-root", t.TempDir())
	if err != nil || !strings.Contains(stdout, "no eval cases") {
		t.Fatalf("a repository without cases: err %v\n%s", err, stdout)
	}
}

func TestEvalCheckFailsAndNamesABrokenCase(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".dotf", "evals", "cases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"id": "bad", "kind": "x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := execute(t, "eval", "check", "--repo-root", root)
	if err == nil {
		t.Fatalf("a broken case passed:\n%s", stdout)
	}
	if !strings.Contains(stdout, ".dotf/evals/cases/bad.json") || !strings.Contains(stdout, "schema") {
		t.Fatalf("the report must name the case file and its problem:\n%s", stdout)
	}
}
