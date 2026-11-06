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
	writeLintFile(t, repo, "harness/doc-paths.json", `{
		"version":1,
		"governed_basenames":["AGENTS.md"],
		"known_extensions":[".sh"],
		"required_members":["AGENTS.md"]
	}`)
	writeLintFile(t, repo, "AGENTS.md", "Run `scripts/missing.sh`.\n")
	writeLintFile(t, repo, "scripts/exists.sh", "ok\n")
	initLintRepo(t, repo)

	var stdout, stderr strings.Builder
	err := runLintDocPaths(&stdout, &stderr, repo, "", nil)
	if err == nil {
		t.Fatal("runLintDocPaths() succeeded with a missing path")
	}
	if !strings.Contains(stderr.String(), "AGENTS.md: referenced path does not exist: scripts/missing.sh") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	writeLintFile(t, repo, "AGENTS.md", "Run `scripts/exists.sh`.\n")
	stdout.Reset()
	stderr.Reset()
	if err := runLintDocPaths(&stdout, &stderr, repo, "", []string{"AGENTS.md"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "doc-paths: OK AGENTS.md") || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestLintDocPathsResolvesExplicitFilesFromWorkingDirectory(t *testing.T) {
	repo := t.TempDir()
	writeLintFile(t, repo, "harness/doc-paths.json", `{
		"version":1,
		"governed_basenames":["README.md"],
		"known_extensions":[".sh"],
		"required_members":[]
	}`)
	writeLintFile(t, repo, "README.md", "Root file is clean.\n")
	writeLintFile(t, repo, "ai/nan/README.md", "Missing: `scripts/missing.sh`.\n")
	writeLintFile(t, repo, "scripts/exists.sh", "ok\n")
	initLintRepo(t, repo)
	chdirForTest(t, filepath.Join(repo, "ai", "nan"))

	var stdout, stderr strings.Builder
	err := runLintDocPaths(&stdout, &stderr, "", "", []string{"README.md"})
	if err == nil {
		t.Fatal("runLintDocPaths() scanned the repository-root README instead of the working-directory README")
	}
	if !strings.Contains(stderr.String(), "ai/nan/README.md: referenced path does not exist: scripts/missing.sh") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLintDocPathsSanitizesGitConfigWhenDiscoveringRepo(t *testing.T) {
	repo := t.TempDir()
	writeLintFile(t, repo, "harness/doc-paths.json", `{
		"version":1,
		"governed_basenames":["AGENTS.md"],
		"known_extensions":[".sh"],
		"required_members":[]
	}`)
	writeLintFile(t, repo, "AGENTS.md", "No path claims.\n")
	initLintRepo(t, repo)
	chdirForTest(t, repo)
	setIncompleteGitConfig(t)

	var stdout, stderr strings.Builder
	if err := runLintDocPaths(&stdout, &stderr, "", "", []string{"AGENTS.md"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "doc-paths: OK AGENTS.md") || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestLintDocPathsIncludesGitDiscoveryFailure(t *testing.T) {
	outside := t.TempDir()
	cmd := exec.Command("git", "-C", outside, "rev-parse", "--show-toplevel")
	raw, directErr := cmd.CombinedOutput()
	if directErr == nil {
		t.Fatal("fixture unexpectedly resolved as a Git checkout")
	}
	detail := strings.TrimSpace(string(raw))
	chdirForTest(t, outside)

	var stdout, stderr strings.Builder
	err := runLintDocPaths(&stdout, &stderr, "", "", []string{"AGENTS.md"})
	if err == nil {
		t.Fatal("runLintDocPaths() succeeded outside a Git checkout")
	}
	if !strings.Contains(err.Error(), "git rev-parse --show-toplevel") || !strings.Contains(err.Error(), detail) {
		t.Fatalf("error = %q, want Git command and %q", err, detail)
	}
}

func writeLintFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func initLintRepo(t *testing.T, repo string) {
	t.Helper()
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func chdirForTest(t *testing.T, path string) {
	t.Helper()
	before, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(before); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func setIncompleteGitConfig(t *testing.T) {
	t.Helper()
	for _, name := range []string{"GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0"} {
		value, ok := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if ok {
				_ = os.Setenv(name, value)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
}
