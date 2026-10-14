package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// checkoutWithRegistry makes a git checkout whose registry declares no triage
// marker, so reading it fails with a message only that file produces, and runs
// the test from a subdirectory of it.
func checkoutWithRegistry(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.MkdirAll(filepath.Join(repo, "harness"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "harness", "review-attestation.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "cli", "internal")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	return repo
}
