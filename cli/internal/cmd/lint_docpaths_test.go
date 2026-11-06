package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintDocPathsReportsFindingsAndCleanFiles(t *testing.T) {
	repo := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("harness/doc-paths.json", `{
		"version":1,
		"governed_basenames":["AGENTS.md"],
		"known_extensions":[".sh"],
		"required_members":["AGENTS.md"]
	}`)
	write("AGENTS.md", "Run `scripts/missing.sh`.\n")
	write("scripts/exists.sh", "ok\n")
	cmd := exec.Command("git", "-C", repo, "init", "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", repo, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	var stdout, stderr strings.Builder
	err := runLintDocPaths(&stdout, &stderr, repo, "", nil)
	if err == nil {
		t.Fatal("runLintDocPaths() succeeded with a missing path")
	}
	if !strings.Contains(stderr.String(), "AGENTS.md: referenced path does not exist: scripts/missing.sh") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	write("AGENTS.md", "Run `scripts/exists.sh`.\n")
	stdout.Reset()
	stderr.Reset()
	if err := runLintDocPaths(&stdout, &stderr, repo, "", []string{"AGENTS.md"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "doc-paths: OK AGENTS.md") || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
