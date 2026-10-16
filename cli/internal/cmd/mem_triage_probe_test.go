package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// The session-start probe finds the reviewer registry from any directory of
// the checkout. Read relative to cwd, a session started in cli/ saw no
// registry and printed nothing, which reads as a clear queue (#1281).
func TestMemTriageQueue_FindsTheRegistryFromASubdirectory(t *testing.T) {
	repo := t.TempDir()
	for _, dir := range []string{".git", "harness", filepath.Join("cli", "internal")} {
		if err := os.MkdirAll(filepath.Join(repo, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "harness", "review-attestation.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, cwd := range []string{repo, filepath.Join(repo, "cli"), filepath.Join(repo, "cli", "internal")} {
		if memTriageQueue(cwd) == nil {
			t.Errorf("from %s: no probe, want one for the checkout's registry", cwd)
		}
	}
}

// A checkout without a registry, and a directory outside any checkout, stay
// silent: not applicable, never "could not be computed".
func TestMemTriageQueue_NoRegistryIsNotApplicable(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if memTriageQueue(repo) != nil {
		t.Error("a checkout with no registry must not get a probe")
	}
	if memTriageQueue(t.TempDir()) != nil {
		t.Error("a directory outside any checkout must not get a probe")
	}
}
