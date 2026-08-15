package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessTriggersCmd(t *testing.T) {
	t.Run("matching file paths directly", func(t *testing.T) {
		stdout, stderr, err := execute(t, "harness", "triggers", "specs/AI-001/proposal.md", "Dockerfile")
		if err != nil {
			t.Fatalf("unexpected error: %v, stderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "pattern-spec-driven-development") {
			t.Errorf("expected pattern-spec-driven-development in stdout: %s", stdout)
		}
		if !strings.Contains(stdout, "pattern-container-workflow") {
			t.Errorf("expected pattern-container-workflow in stdout: %s", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		stdout, stderr, err := execute(t, "harness", "triggers", "--json", "specs/tasks.md")
		if err != nil {
			t.Fatalf("unexpected error: %v, stderr: %s", err, stderr)
		}

		var patterns []string
		if err := json.Unmarshal([]byte(stdout), &patterns); err != nil {
			t.Fatalf("failed to parse JSON output: %v, stdout: %s", err, stdout)
		}
		if len(patterns) != 1 || patterns[0] != "pattern-spec-driven-development" {
			t.Errorf("unexpected patterns: %v", patterns)
		}
	})

	t.Run("diff mode from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		diffPath := filepath.Join(tmpDir, "sample.diff")
		diffContent := `diff --git a/Dockerfile b/Dockerfile
index 123..456 100644
--- a/Dockerfile
+++ b/Dockerfile
@@ -1,2 +1,3 @@
+ENV FOO=bar
`
		if err := os.WriteFile(diffPath, []byte(diffContent), 0644); err != nil {
			t.Fatalf("write diff error: %v", err)
		}

		stdout, stderr, err := execute(t, "harness", "triggers", "--diff", diffPath)
		if err != nil {
			t.Fatalf("unexpected error: %v, stderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "pattern-container-workflow") {
			t.Errorf("expected pattern-container-workflow in stdout: %s", stdout)
		}
	})
}
