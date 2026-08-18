package harness

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		glob     string
		path     string
		expected bool
	}{
		// specs/**
		{"specs/**", "specs/FEATURE-1/proposal.md", true},
		{"specs/**", "specs/tasks.md", true},
		{"specs/**", "specs", true},
		{"specs/**", "other/specs/tasks.md", false},
		{"**/specs/**", "sub/specs/doc.md", true},

		// Dockerfile*
		{"Dockerfile*", "Dockerfile", true},
		{"Dockerfile*", "Dockerfile.dev", true},
		{"Dockerfile*", "docker/Dockerfile", true},
		{"Dockerfile*", "Containerfile", false},
		{"**/Dockerfile*", "backend/docker/Dockerfile.prod", true},

		// *test*
		{"*test*", "main_test.go", true},
		{"*test*", "internal/cmd/spec_test.go", true},
		{"*test*", "tests/test_harness.py", true},
		{"*test*", "tests/unit/app.go", true},
		{"*test*", "pkg/server.go", false},

		// *.sh and *.ps1
		{"*.sh", "compile-harness.sh", true},
		{"*.sh", "scripts/test.sh", true},
		{"*.sh", "script.bash", false},
		{"*.ps1", "setup-windows.ps1", true},
		{"*.ps1", "scripts/vault.ps1", true},
		{"*.ps1", "scripts/vault.sh", false},

		// Edge cases
		{"", "specs/tasks.md", false},
		{"specs/**", "", false},
		{"exact.txt", "exact.txt", true},
		{"exact.txt", "dir/exact.txt", true},
		{"dir/exact.txt", "dir/exact.txt", true},
		{"dir/exact.txt", "other/exact.txt", false},
	}

	for _, tt := range tests {
		got := MatchGlob(tt.glob, tt.path)
		if got != tt.expected {
			t.Errorf("MatchGlob(%q, %q) = %v; want %v", tt.glob, tt.path, got, tt.expected)
		}
	}
}

func TestExtractPathsFromDiff(t *testing.T) {
	diff := `diff --git a/specs/TEST-001/proposal.md b/specs/TEST-001/proposal.md
index 1234567..89abcdef 100644
--- a/specs/TEST-001/proposal.md
+++ b/specs/TEST-001/proposal.md
@@ -1,3 +1,4 @@
+# Added line
diff --git a/Dockerfile b/Dockerfile
new file mode 100644
--- /dev/null
+++ b/Dockerfile
@@ -0,0 +1,5 @@
+FROM alpine:latest
diff --git a/deleted.sh b/deleted.sh
deleted file mode 100644
--- a/deleted.sh
+++ /dev/null
`
	paths := ExtractPathsFromDiff(diff)
	expected := []string{
		"specs/TEST-001/proposal.md",
		"Dockerfile",
		"deleted.sh",
	}

	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("ExtractPathsFromDiff() = %v; want %v", paths, expected)
	}
}

func TestMatchPaths(t *testing.T) {
	rules := []TriggerRule{
		{
			ID:      "spec",
			Pattern: "pattern-spec-driven-development",
			Globs:   []string{"specs/**", "**/specs/**"},
		},
		{
			ID:      "container",
			Pattern: "pattern-container-workflow",
			Globs:   []string{"Dockerfile*", "**/Dockerfile*"},
		},
		{
			ID:      "test",
			Pattern: "pattern-testing-standards",
			Globs:   []string{"*test*", "tests/**"},
		},
	}

	tests := []struct {
		name     string
		paths    []string
		expected []string
	}{
		{
			name:     "single match",
			paths:    []string{"specs/AI-001/proposal.md"},
			expected: []string{"pattern-spec-driven-development"},
		},
		{
			name:     "multiple matches and deduplication",
			paths:    []string{"specs/AI-001/proposal.md", "specs/AI-001/tasks.md", "Dockerfile", "pkg/app_test.go"},
			expected: []string{"pattern-container-workflow", "pattern-spec-driven-development", "pattern-testing-standards"},
		},
		{
			name:     "no matches",
			paths:    []string{"README.md", "LICENSE"},
			expected: []string{},
		},
		{
			name:     "empty paths",
			paths:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchPaths(rules, tt.paths)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("MatchPaths() = %v; want %v", got, tt.expected)
			}
		})
	}
}

func TestMatchDiff(t *testing.T) {
	rules := []TriggerRule{
		{
			ID:      "spec",
			Pattern: "pattern-spec-driven-development",
			Globs:   []string{"specs/**"},
		},
		{
			ID:      "container",
			Pattern: "pattern-container-workflow",
			Globs:   []string{"Dockerfile*"},
		},
	}

	diff := `diff --git a/specs/TEST-001/proposal.md b/specs/TEST-001/proposal.md
--- a/specs/TEST-001/proposal.md
+++ b/specs/TEST-001/proposal.md
`
	got := MatchDiff(rules, diff)
	expected := []string{"pattern-spec-driven-development"}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("MatchDiff() = %v; want %v", got, expected)
	}
}

func TestLoadTriggers(t *testing.T) {
	// Test loading embedded defaults
	cfg, err := LoadTriggers("")
	if err != nil {
		t.Fatalf("LoadTriggers(\"\") error: %v", err)
	}
	if cfg == nil || len(cfg.Triggers) == 0 {
		t.Fatal("expected non-empty embedded triggers")
	}

	// Test loading from custom repo root
	tmpDir := t.TempDir()
	harnessDir := filepath.Join(tmpDir, "harness")
	if err := os.MkdirAll(harnessDir, 0755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}

	customJSON := `{
		"version": 1,
		"triggers": [
			{"id": "custom", "pattern": "pattern-custom", "globs": ["custom/**"]}
		]
	}`
	if err := os.WriteFile(filepath.Join(harnessDir, "triggers.json"), []byte(customJSON), 0644); err != nil {
		t.Fatalf("write custom triggers error: %v", err)
	}

	customCfg, err := LoadTriggers(tmpDir)
	if err != nil {
		t.Fatalf("LoadTriggers(tmpDir) error: %v", err)
	}
	if len(customCfg.Triggers) != 1 || customCfg.Triggers[0].Pattern != "pattern-custom" {
		t.Fatalf("unexpected custom config: %+v", customCfg)
	}
}

func TestMatchPrompt(t *testing.T) {
	rules := []TriggerRule{
		{
			ID:       "spec",
			Pattern:  "pattern-spec-driven-development",
			Skills:   []string{"spec", "adversarial-review"},
			Keywords: []string{"spec", "sdd", "proposal", "criterio de aceptacion"},
		},
		{
			ID:       "docker",
			Pattern:  "pattern-container-workflow",
			Skills:   []string{"docker"},
			Keywords: []string{"docker", "container", "compose", "contenedor"},
		},
	}

	tests := []struct {
		prompt   string
		wantPats []string
		wantSk   []string
	}{
		{
			prompt:   "ayudame a redactar una proposal para el nuevo endpoint",
			wantPats: []string{"pattern-spec-driven-development"},
			wantSk:   []string{"adversarial-review", "spec"},
		},
		{
			prompt:   "crea un contenedor docker con go y redis",
			wantPats: []string{"pattern-container-workflow"},
			wantSk:   []string{"docker"},
		},
		{
			prompt:   "una consulta general sobre la base de datos",
			wantPats: []string{},
			wantSk:   []string{},
		},
	}

	for _, tt := range tests {
		gotPats, gotSk := MatchPrompt(rules, tt.prompt)
		if !reflect.DeepEqual(gotPats, tt.wantPats) {
			t.Errorf("MatchPrompt(%q) patterns = %v; want %v", tt.prompt, gotPats, tt.wantPats)
		}
		if !reflect.DeepEqual(gotSk, tt.wantSk) {
			t.Errorf("MatchPrompt(%q) skills = %v; want %v", tt.prompt, gotSk, tt.wantSk)
		}
	}
}

