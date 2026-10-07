package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessInstructions_DeploysThenReportsCurrent(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	for rel, body := range map[string]string{
		"harness/manifest.json": `{"agents":{"presence":[{"agent":"claude","file":".claude/CLAUDE.md","source":"ai/claude/CLAUDE.md"}]}}`,
		"ai/claude/CLAUDE.md":   "# CLAUDE\n",
	} {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	args := []string{"harness", "instructions", "--repo-root", repo, "--home", home}

	stdout, _, err := execute(t, args...)
	if err != nil || !strings.Contains(stdout, "[deploy] instructions -> ") {
		t.Fatalf("first run: %v\n%s", err, stdout)
	}
	stdout, _, err = execute(t, args...)
	if err != nil || !strings.Contains(stdout, "[deploy] instructions current: ") {
		t.Fatalf("second run should find the file current: %v\n%s", err, stdout)
	}
}
