package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// The default registry is the checkout's, found from any directory in it, not
// a path relative to wherever the command happens to run.
func TestReviewerRegistryPath_TheDefaultResolvesFromTheCheckoutRoot(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	sub := filepath.Join(repo, "cli", "internal")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	c := &cobra.Command{}
	var registry string
	addReviewerRegistryFlag(c, &registry)
	got := reviewerRegistryPath(c, registry)

	want := filepath.Join(repo, "harness", "review-attestation.json")
	gotDir, err := filepath.EvalSymlinks(filepath.Dir(filepath.Dir(got)))
	if err != nil {
		t.Fatal(err)
	}
	wantDir, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != wantDir || filepath.Base(got) != filepath.Base(want) {
		t.Errorf("registryPath = %q, want %q", got, want)
	}
}

// An explicit --registry is used as given, relative paths included.
func TestReviewerRegistryPath_AnExplicitFlagIsUsedAsGiven(t *testing.T) {
	c := &cobra.Command{}
	var registry string
	addReviewerRegistryFlag(c, &registry)
	if err := c.Flags().Set("registry", "elsewhere.json"); err != nil {
		t.Fatal(err)
	}
	if got := reviewerRegistryPath(c, registry); got != "elsewhere.json" {
		t.Errorf("registryPath = %q, want the flag as given", got)
	}
}
