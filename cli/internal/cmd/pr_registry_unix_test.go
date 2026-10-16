//go:build !windows

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGhServingRegistry puts a gh on PATH that serves registry as the named
// repository's harness/review-attestation.json and no open pull request. An
// empty registry makes the contents call answer 404, as GitHub does for a
// repository without the file. Every call is appended to the returned log.
func fakeGhServingRegistry(t *testing.T, registry string) string {
	t.Helper()
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls")
	contents := "printf '%s' '" + registry + "'"
	if registry == "" {
		contents = "echo 'gh: Not Found (HTTP 404)' >&2; exit 1"
	}
	script := "#!/bin/sh\necho \"$*\" >> '" + calls + "'\n" +
		"case \"$*\" in\n" +
		"*repos/o/r/contents/harness/review-attestation.json*) " + contents + " ;;\n" +
		"*repos/o/r/pulls*) printf '[]' ;;\n" +
		"*) echo \"unexpected gh call: $*\" >&2; exit 1 ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil { //nolint:gosec // an executable stub
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	return calls
}

// Without --registry, the registry is the named repository's, read from its
// default branch: the local checkout's (here one with no marker, which would
// fail) plays no part, and neither does the directory the command runs in.
func TestPrTriageQueue_ReadsTheNamedRepositorysRegistry(t *testing.T) {
	checkoutWithRegistry(t)
	calls := fakeGhServingRegistry(t, `{"triage":{"marker":"## Served triage"}}`)

	stdout, stderr, err := execute(t, "pr", "triage-queue", "--repo", "o/r")
	if err != nil || !strings.Contains(stdout, "[OK]") {
		t.Fatalf("want the served registry used and an empty queue, got %v:\n%s%s", err, stdout, stderr)
	}
	log, _ := os.ReadFile(calls)
	if !strings.Contains(string(log), "application/vnd.github.raw+json") {
		t.Errorf("want the file read raw, got calls:\n%s", log)
	}
}

// A repository without the file is an error naming it, never an empty queue.
func TestPrTriageQueue_ARepositoryWithoutARegistryIsAnError(t *testing.T) {
	fakeGhServingRegistry(t, "")

	stdout, stderr, err := execute(t, "pr", "triage-queue", "--repo", "o/r")
	if err == nil || strings.Contains(stdout, "[OK]") {
		t.Fatalf("want a refusal, got %v:\n%s", err, stdout)
	}
	if !strings.Contains(stderr, "o/r has no harness/review-attestation.json") {
		t.Errorf("want the repository and the file named, got:\n%s", stderr)
	}
}

// The served registry goes through the same checks as a file: one with no
// marker cannot answer, whichever way it arrived.
func TestPrTriageQueue_AServedRegistryWithoutAMarkerIsRefused(t *testing.T) {
	fakeGhServingRegistry(t, `{}`)

	_, stderr, err := execute(t, "pr", "triage-queue", "--repo", "o/r")
	if err == nil || !strings.Contains(stderr, "o/r:harness/review-attestation.json declares no triage.marker") {
		t.Errorf("want the marker refusal naming the served file, got %v:\n%s", err, stderr)
	}
}

// An explicit --registry still wins over the served one.
func TestPrTriageQueue_AnExplicitRegistryWins(t *testing.T) {
	repo := checkoutWithRegistry(t)
	fakeGhServingRegistry(t, `{"triage":{"marker":"## Served triage"}}`)

	_, stderr, err := execute(t, "pr", "triage-queue", "--repo", "o/r",
		"--registry", filepath.Join(repo, "harness", "review-attestation.json"))
	if err == nil || !strings.Contains(stderr, "declares no triage.marker") {
		t.Errorf("want the given file's own error, got %v:\n%s", err, stderr)
	}
}

// pr land reads the registry through the same resolver.
func TestPrLand_ReadsTheNamedRepositorysRegistry(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	fakeGhServingRegistry(t, "")

	load := reviewerRegistry("o/r", "")
	if _, err := load(t.Context()); err == nil || !strings.Contains(err.Error(), "o/r has no harness/review-attestation.json") {
		t.Errorf("want pr land's loader to read the named repository, got %v", err)
	}
}
