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

func TestResolveDependencies(t *testing.T) {
	// Synthetic ids: the resolver is blind to names, and a real skill's name here
	// reads as a live dependency, which is how retired names outlived SKILL-001.
	deps := map[string][]string{
		"root":    {"dep-a", "dep-b"},
		"chain-1": {"chain-2"},
		"chain-2": {"leaf-x", "leaf-y"},
		"cycle-a": {"cycle-b"},
		"cycle-b": {"cycle-a"},
	}

	tests := []struct {
		name     string
		initial  []string
		expected []string
	}{
		{
			name:     "single skill with direct dependencies",
			initial:  []string{"root"},
			expected: []string{"dep-a", "dep-b", "root"},
		},
		{
			name:     "multi-level transitive dependencies",
			initial:  []string{"chain-1"},
			expected: []string{"chain-1", "chain-2", "leaf-x", "leaf-y"},
		},
		{
			name:     "cycle protection",
			initial:  []string{"cycle-a"},
			expected: []string{"cycle-a", "cycle-b"},
		},
		{
			name:     "no dependencies",
			initial:  []string{"docker"},
			expected: []string{"docker"},
		},
		{
			name:     "empty input",
			initial:  []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveDependencies(tt.initial, deps)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ResolveDependencies(%v) = %v; want %v", tt.initial, got, tt.expected)
			}
		})
	}
}

func TestLoadSkillDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	skill1Dir := filepath.Join(tmpDir, "skill1")
	skill2Dir := filepath.Join(tmpDir, "skill2")
	_ = os.MkdirAll(skill1Dir, 0755)
	_ = os.MkdirAll(skill2Dir, 0755)

	skill1Content := `---
name: skill1
requires: [skill2, skill3]
---
# Skill 1`
	skill2Content := `---
name: skill2
---
# Skill 2`

	_ = os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(skill1Content), 0644)
	_ = os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(skill2Content), 0644)

	deps, err := LoadSkillDependencies(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillDependencies error: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 skill with dependencies, got %d: %+v", len(deps), deps)
	}
	expected := []string{"skill2", "skill3"}
	if !reflect.DeepEqual(deps["skill1"], expected) {
		t.Fatalf("deps[skill1] = %v; want %v", deps["skill1"], expected)
	}
}

// TestSuggestSharedPatternDoesNotCrossLinkSkills pins that skills follow the rule
// that matched, not the pattern it names.
//
// Two rules can name one pattern (the Go rule and the complexity rule both point
// at the language standards). Linking by pattern made a Python refactor prompt
// suggest golang-pro, because the sibling rule's pattern had been "triggered".
// Both entry points are covered: a keyword match and a path match reach the
// skills by different routes.
func TestSuggestSharedPatternDoesNotCrossLinkSkills(t *testing.T) {
	rules := []TriggerRule{
		{ID: "complexity", Pattern: "pattern-language-standards", Globs: []string{"*.py"},
			Keywords: []string{"refactor"}, Skills: []string{"cyclomatic-complexity"}},
		{ID: "golang", Pattern: "pattern-language-standards", Globs: []string{"*.go"},
			Keywords: []string{"goroutine"}, Skills: []string{"golang-pro"}},
	}

	tests := []struct {
		name   string
		prompt string
		paths  []string
		want   []string
	}{
		{"a prompt matching one rule", "refactor this", nil, []string{"cyclomatic-complexity"}},
		{"a prompt matching the other rule", "a goroutine leak", nil, []string{"golang-pro"}},
		{"a path matching one rule", "", []string{"main.py"}, []string{"cyclomatic-complexity"}},
		{"a path matching the other rule", "", []string{"main.go"}, []string{"golang-pro"}},
		{"paths matching both rules", "", []string{"a.py", "b.go"}, []string{"cyclomatic-complexity", "golang-pro"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestWithDeps(rules, tt.prompt, tt.paths, map[string][]string{})
			if !reflect.DeepEqual(got.Skills, tt.want) {
				t.Errorf("Skills = %v; want %v", got.Skills, tt.want)
			}
			if !reflect.DeepEqual(got.Patterns, []string{"pattern-language-standards"}) {
				t.Errorf("Patterns = %v; want the shared pattern reported once", got.Patterns)
			}
		})
	}
}

// TestMatchPathsNeverReportsAnEmptyPattern: a rule with no pattern must not put
// "" into the result, where a caller would print it as a pattern name.
func TestMatchPathsNeverReportsAnEmptyPattern(t *testing.T) {
	rules := []TriggerRule{{ID: "bare", Globs: []string{"*.tf"}, Skills: []string{"terraform"}}}
	if got := MatchPaths(rules, []string{"main.tf"}); len(got) != 0 {
		t.Errorf("MatchPaths() = %q; want no patterns", got)
	}
}

// TestRealTriggersFileValid guards harness/triggers.json against silent rot or schema deviation (GUARD #1137).
func TestRealTriggersFileValid(t *testing.T) {
	cfg, err := ParseTriggers(defaultTriggersJSON)
	if err != nil {
		t.Fatalf("ParseTriggers(defaultTriggersJSON) failed: %v", err)
	}
	if cfg.Version < 1 {
		t.Errorf("expected Version >= 1, got %d", cfg.Version)
	}
	if len(cfg.Triggers) == 0 {
		t.Fatal("expected non-empty triggers array")
	}

	seenIDs := make(map[string]bool)
	for i, rule := range cfg.Triggers {
		if rule.ID == "" {
			t.Errorf("trigger[%d] has empty ID", i)
		}
		if seenIDs[rule.ID] {
			t.Errorf("duplicate trigger ID %q", rule.ID)
		}
		seenIDs[rule.ID] = true

		if rule.Pattern == "" {
			t.Errorf("trigger %q has empty Pattern", rule.ID)
		}
		if len(rule.Globs) == 0 && len(rule.Keywords) == 0 {
			t.Errorf("trigger %q has neither globs nor keywords", rule.ID)
		}
	}
}

// TestTriggersEmbeddedMatchesDiskSSOT guards against drift between harness/triggers.json and embedded copy (#1137).
func TestTriggersEmbeddedMatchesDiskSSOT(t *testing.T) {
	root := "../../.."
	diskPath := filepath.Join(root, "harness", "triggers.json")
	diskBytes, err := os.ReadFile(diskPath)
	if err != nil {
		t.Skipf("skipping drift test outside repository root: %v", err)
		return
	}
	if !reflect.DeepEqual(diskBytes, defaultTriggersJSON) {
		t.Errorf("embedded triggers.json drifted from %s; re-sync copies", diskPath)
	}
}

// TestEverySkillTheRouterNamesHasARecord is TestEveryDeclaredSkillHasARecord's
// sibling for the two other places a skill id is typed by hand: a trigger rule's
// `skills` and DefaultSkillDependencies. Retiring a skill drops its record under
// harness/skills/, and neither list was checked against the records, so a retired
// id would go on being suggested by `dotf harness suggest` and the prompt hook: a
// skill name that nothing can invoke. SKILL-001 retired eight skills that both
// lists still named.
func TestEverySkillTheRouterNamesHasARecord(t *testing.T) {
	root := repoRootForTest(t)
	cfg, err := ParseTriggers(defaultTriggersJSON)
	if err != nil {
		t.Fatalf("the embedded triggers do not parse: %v", err)
	}

	named := map[string]string{}
	for _, rule := range cfg.Triggers {
		for _, s := range rule.Skills {
			named[s] = "trigger " + rule.ID
		}
	}
	for skill, deps := range DefaultSkillDependencies {
		named[skill] = "DefaultSkillDependencies"
		for _, d := range deps {
			named[d] = "DefaultSkillDependencies[" + skill + "]"
		}
	}
	if len(named) == 0 {
		t.Fatal("neither the triggers nor the dependency map name a skill; nothing was checked")
	}

	for skill, where := range named {
		// IsRegular, as in the persona guard: os.Stat also succeeds on a directory.
		info, err := os.Stat(filepath.Join(root, "harness", "skills", skill, "SKILL.md"))
		if err != nil || !info.Mode().IsRegular() {
			t.Errorf("%s names skill %q, but harness/skills/%s/SKILL.md is not a readable file: "+
				"the router would suggest a skill nothing can invoke", where, skill, skill)
		}
	}
}
