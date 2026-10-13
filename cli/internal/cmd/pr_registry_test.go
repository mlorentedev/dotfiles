package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/prland"
	"github.com/spf13/cobra"
)

// checkoutWithRegistry makes a git checkout whose registry declares no triage
// marker, so reading it fails with a message only the found file produces,
// and runs the test from a subdirectory of it.
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

// Both commands read the registry through the resolver: run from a
// subdirectory, each reaches the checkout's file.
func TestPrTriageQueue_FindsTheRegistryFromASubdirectory(t *testing.T) {
	checkoutWithRegistry(t)
	_, stderr, err := execute(t, "pr", "triage-queue", "--repo", "o/r")
	if err == nil || !strings.Contains(stderr, "declares no triage.marker") {
		t.Errorf("want the found registry's own error, got %v:\n%s", err, stderr)
	}
}

func TestPrLand_FindsTheRegistryFromASubdirectory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	checkoutWithRegistry(t)
	saved := prLandOptions
	var got string
	prLandOptions = func(repo, registry string) prland.Options {
		got = registry
		return prland.Options{Repo: repo, Run: func(context.Context, ...string) ([]byte, error) {
			return nil, errors.New("no gh in this test")
		}}
	}
	t.Cleanup(func() { prLandOptions = saved })

	_, _, _ = execute(t, "pr", "land", "30", "--repo", "o/r")
	if _, err := os.Stat(got); err != nil {
		t.Errorf("pr land was handed registry %q, which does not resolve from here: %v", got, err)
	}
}

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
