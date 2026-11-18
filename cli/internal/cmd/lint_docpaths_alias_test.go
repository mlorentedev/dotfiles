package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// An absolute file argument that reaches the checkout through another name
// for its directory (macOS's /var for /private/var, a symlinked workspace) is
// inside the repository, and is reported by its path there (CLI-141).
func TestLintDocPathsAcceptsAnAbsoluteFileThroughAnAlias(t *testing.T) {
	repo := t.TempDir()
	writeLintFile(t, repo, "harness/doc-paths.json", `{
		"version":1,
		"governed_basenames":["README.md"],
		"known_extensions":[".sh"],
		"required_members":[]
	}`)
	writeLintFile(t, repo, "ai/nan/README.md", "Missing: `scripts/missing.sh`.\n")
	writeLintFile(t, repo, "scripts/exists.sh", "ok\n")
	initLintRepo(t, repo)
	alias := filepath.Join(t.TempDir(), "alias")
	aliasDir(t, repo, alias)

	var stdout, stderr strings.Builder
	err := runLintDocPaths(&stdout, &stderr, repo, "", []string{filepath.Join(alias, "ai", "nan", "README.md")})
	if err == nil || strings.Contains(err.Error(), "outside repository root") {
		t.Fatalf("err = %v, want the file's one finding", err)
	}
	if !strings.Contains(stderr.String(), "ai/nan/README.md: referenced path does not exist: scripts/missing.sh") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

// aliasDir makes link another name for the directory target: a symlink, or on
// Windows, where an account without the privilege cannot make one, a junction.
func aliasDir(t *testing.T, target, link string) {
	t.Helper()
	err := os.Symlink(target, link)
	if err != nil && runtime.GOOS == "windows" {
		err = exec.Command("cmd", "/c", "mklink", "/J", link, target).Run()
	}
	if err != nil {
		t.Skipf("no way to alias a directory here: %v", err)
	}
}

// A file outside the checkout stays outside, whatever name reaches it, and a
// governed file that is itself a link keeps its own name.
func TestLintDocPathsKeepsTheRepositoryBoundary(t *testing.T) {
	repo := t.TempDir()
	writeLintFile(t, repo, "harness/doc-paths.json", `{
		"version":1,
		"governed_basenames":["README.md", "CLAUDE.md"],
		"known_extensions":[".sh"],
		"required_members":[]
	}`)
	writeLintFile(t, repo, "README.md", "Clean.\n")
	initLintRepo(t, repo)
	outside := t.TempDir()
	writeLintFile(t, outside, "README.md", "Elsewhere.\n")

	var stdout, stderr strings.Builder
	err := runLintDocPaths(&stdout, &stderr, repo, "", []string{filepath.Join(outside, "README.md")})
	if err == nil || !strings.Contains(err.Error(), "outside repository root") {
		t.Fatalf("err = %v, want the file refused as outside the repository", err)
	}
}

// A governed file that is a link to another is read under its own name.
func TestLintDocPathsReportsALinkedFileUnderItsOwnName(t *testing.T) {
	repo := t.TempDir()
	writeLintFile(t, repo, "harness/doc-paths.json", `{
		"version":1,
		"governed_basenames":["README.md", "CLAUDE.md"],
		"known_extensions":[".sh"],
		"required_members":[]
	}`)
	writeLintFile(t, repo, "README.md", "Clean.\n")
	initLintRepo(t, repo)
	if err := os.Symlink(filepath.Join(repo, "README.md"), filepath.Join(repo, "CLAUDE.md")); err != nil {
		t.Skipf("file symlinks unavailable: %v", err)
	}
	var stdout, stderr strings.Builder
	if err := runLintDocPaths(&stdout, &stderr, repo, "", []string{filepath.Join(repo, "CLAUDE.md")}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "doc-paths: OK CLAUDE.md") {
		t.Fatalf("stdout = %q, want the link reported under its own name", stdout.String())
	}
}
