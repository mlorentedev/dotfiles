package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
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

// pr land holds a PR to the same registry as triage-queue: the loader it is
// handed must run when land asks what is untriaged, and a loader error must
// stop it before any queue is computed.
func TestPrLandOptions_UntriagedReadsTheRegistryItIsGiven(t *testing.T) {
	var asked int
	loader := func(context.Context) (prtriage.Registry, error) {
		asked++
		return prtriage.Registry{}, errors.New("registry unreadable")
	}

	_, err := prLandOptions("o/r", loader).Untriaged(t.Context())
	if err == nil || !strings.Contains(err.Error(), "registry unreadable") {
		t.Fatalf("want the loader's error, got %v", err)
	}
	if asked != 1 {
		t.Errorf("want the loader asked once, got %d", asked)
	}
}
