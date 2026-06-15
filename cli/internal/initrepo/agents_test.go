package initrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAgentsSnippetIsSelfContained is the #248 guard at the unit level: the SDD
// snippet the binary emits must be non-empty, headed by the SDD H2, point at the
// dotf CLI, and carry zero unexpanded $VAULT_PATH literals.
func TestAgentsSnippetIsSelfContained(t *testing.T) {
	snippet, err := AgentsSnippet()
	if err != nil {
		t.Fatalf("AgentsSnippet: %v", err)
	}
	if !strings.HasPrefix(snippet, "## Spec-Driven Development") {
		t.Errorf("snippet should start with the SDD heading, got:\n%s", snippet)
	}
	if strings.Contains(snippet, "$VAULT_PATH") {
		t.Errorf("snippet leaks $VAULT_PATH (#248):\n%s", snippet)
	}
	if !strings.Contains(snippet, "dotf spec init") {
		t.Errorf("snippet should point at `dotf spec init`, got:\n%s", snippet)
	}
	// The markers themselves must never bleed into the emitted section.
	for _, marker := range []string{"BEGIN SNIPPET", "END SNIPPET"} {
		if strings.Contains(snippet, marker) {
			t.Errorf("snippet should not contain the %q marker", marker)
		}
	}
}

func TestBootstrapAgentsCreatesWhenMissing(t *testing.T) {
	root := t.TempDir()
	res, err := BootstrapAgents(root, false)
	if err != nil {
		t.Fatalf("BootstrapAgents: %v", err)
	}
	if res.Action != "created" {
		t.Errorf("Action = %q, want %q", res.Action, "created")
	}
	got := readFile(t, filepath.Join(root, "AGENTS.md"))
	if !strings.HasPrefix(got, "# AGENTS.md") {
		t.Errorf("created AGENTS.md should open with the header, got:\n%s", got)
	}
	if !sddSectionCount(got, 1) {
		t.Errorf("created AGENTS.md should have exactly one SDD section:\n%s", got)
	}
	if strings.Contains(got, "$VAULT_PATH") {
		t.Errorf("created AGENTS.md leaks $VAULT_PATH:\n%s", got)
	}
}

func TestBootstrapAgentsIdempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := BootstrapAgents(root, false); err != nil {
		t.Fatalf("first BootstrapAgents: %v", err)
	}
	first := readFile(t, filepath.Join(root, "AGENTS.md"))

	res, err := BootstrapAgents(root, false)
	if err != nil {
		t.Fatalf("second BootstrapAgents: %v", err)
	}
	if res.Action != "unchanged" {
		t.Errorf("re-run Action = %q, want %q", res.Action, "unchanged")
	}
	second := readFile(t, filepath.Join(root, "AGENTS.md"))
	if first != second {
		t.Errorf("re-run mutated AGENTS.md\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if !sddSectionCount(second, 1) {
		t.Errorf("re-run must not duplicate the SDD section:\n%s", second)
	}
}

func TestBootstrapAgentsAppendsToExisting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	prior := "# AGENTS.md\n\n## House Rules\n\nBe excellent.\n"
	if err := os.WriteFile(path, []byte(prior), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := BootstrapAgents(root, false)
	if err != nil {
		t.Fatalf("BootstrapAgents: %v", err)
	}
	if res.Action != "appended" {
		t.Errorf("Action = %q, want %q", res.Action, "appended")
	}
	got := readFile(t, path)
	if !strings.Contains(got, "## House Rules") {
		t.Errorf("append must preserve prior content:\n%s", got)
	}
	if !sddSectionCount(got, 1) {
		t.Errorf("append should add exactly one SDD section:\n%s", got)
	}
}

func TestBootstrapAgentsForceReplaces(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	stale := "# AGENTS.md\n\n## Spec-Driven Development\n\nOLD STALE BODY with $VAULT_PATH leak.\n\n## After\n\nkeep me.\n"
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	// Without --force it is a no-op even if the section is stale.
	res, err := BootstrapAgents(root, false)
	if err != nil {
		t.Fatalf("BootstrapAgents(no force): %v", err)
	}
	if res.Action != "unchanged" {
		t.Errorf("no-force Action = %q, want %q", res.Action, "unchanged")
	}

	// With --force the stale section is replaced in place.
	res, err = BootstrapAgents(root, true)
	if err != nil {
		t.Fatalf("BootstrapAgents(force): %v", err)
	}
	if res.Action != "replaced" {
		t.Errorf("force Action = %q, want %q", res.Action, "replaced")
	}
	got := readFile(t, path)
	if strings.Contains(got, "OLD STALE BODY") || strings.Contains(got, "$VAULT_PATH") {
		t.Errorf("force must replace the stale section (no old body, no leak):\n%s", got)
	}
	if !strings.Contains(got, "## After\n\nkeep me.") {
		t.Errorf("force must preserve the section after the SDD one:\n%s", got)
	}
	if !sddSectionCount(got, 1) {
		t.Errorf("force must leave exactly one SDD section:\n%s", got)
	}
}

// --- helpers ---

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// sddSectionCount reports whether the file has exactly n SDD H2 headings.
func sddSectionCount(content string, n int) bool {
	count := 0
	for _, ln := range strings.Split(content, "\n") {
		if ln == "## Spec-Driven Development" {
			count++
		}
	}
	return count == n
}
