package initrepo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCreatesStructureAndFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "myproj")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Scaffold(root)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	wantDirs := []string{"src", "tests", "scripts", "specs", "docs/adr", "docs/runbooks", "docs/troubleshooting", ".claude"}
	for _, d := range wantDirs {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(d))); err != nil || !info.IsDir() {
			t.Errorf("expected directory %s: %v", d, err)
		}
	}

	wantFiles := []string{".gitignore", ".pre-commit-config.yaml", "CLAUDE.md", "env-contract.json", "docs/lessons.md"}
	for _, f := range wantFiles {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(f))); err != nil {
			t.Errorf("expected file %s: %v", f, err)
		}
	}
	if len(res.Created) != len(wantFiles) {
		t.Errorf("Created = %v, want %d files", res.Created, len(wantFiles))
	}

	// CLAUDE.md is a thin pointer to AGENTS.md.
	claude := readFile(t, filepath.Join(root, "CLAUDE.md"))
	if !strings.Contains(claude, "AGENTS.md") {
		t.Errorf("CLAUDE.md should point at AGENTS.md:\n%s", claude)
	}

	// {{repo}} is substituted with the repo basename in lessons.md.
	lessons := readFile(t, filepath.Join(root, "docs", "lessons.md"))
	if strings.Contains(lessons, "{{repo}}") {
		t.Errorf("lessons.md still contains the {{repo}} placeholder:\n%s", lessons)
	}
	if !strings.Contains(lessons, "myproj") {
		t.Errorf("lessons.md should mention the repo name 'myproj':\n%s", lessons)
	}
}

func TestScaffoldGitignoreHasMemorySinkBlock(t *testing.T) {
	// GUARD-001 AC4: a fresh repo is born convention-correct — its .gitignore
	// excludes the memory-sink artifacts so MEMORY.md / memory/ can never be
	// committed to a code repo (memory lives only in the vault).
	root := filepath.Join(t.TempDir(), "myproj")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Scaffold(root); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	gitignore := readFile(t, filepath.Join(root, ".gitignore"))
	for _, want := range []string{"MEMORY.md", "memory/"} {
		if !strings.Contains(gitignore, want) {
			t.Errorf(".gitignore must ignore %q (single-sink convention):\n%s", want, gitignore)
		}
	}
}

func TestScaffoldEnvContractIsValidJSON(t *testing.T) {
	root := t.TempDir()
	if _, err := Scaffold(root); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	raw := readFile(t, filepath.Join(root, "env-contract.json"))
	var c struct {
		EnvVars          []any `json:"env_vars"`
		RequiredBinaries []any `json:"required_binaries"`
	}
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatalf("scaffolded env-contract.json is not valid JSON: %v\n%s", err, raw)
	}
}

func TestScaffoldSkipsExistingFiles(t *testing.T) {
	root := t.TempDir()
	custom := "# my own gitignore\nsecret.txt\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Scaffold(root)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if got := readFile(t, filepath.Join(root, ".gitignore")); got != custom {
		t.Errorf("Scaffold clobbered an existing .gitignore:\n%s", got)
	}
	found := false
	for _, s := range res.Skipped {
		if s == ".gitignore" {
			found = true
		}
	}
	if !found {
		t.Errorf("existing .gitignore should be reported in Skipped, got %v", res.Skipped)
	}
}
