package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeOpts builds a WorkEntryOptions rooted at a fresh temp vault.
func writeOpts(t *testing.T) WorkEntryOptions {
	t.Helper()
	return WorkEntryOptions{
		VaultPath: t.TempDir(),
		Family:    "acme-sensors",
		Component: "edge-fw",
		Date:      "2026-06-16",
	}
}

func TestWriteWorkEntryCreatesFilesWithTokensSubstituted(t *testing.T) {
	opts := writeOpts(t)
	res, err := WriteWorkEntry(opts)
	if err != nil {
		t.Fatalf("WriteWorkEntry: %v", err)
	}

	wantDir := filepath.Join(opts.VaultPath, "50_work", "45-development", "acme-sensors", "edge-fw")
	if res.EntryDir != wantDir {
		t.Errorf("EntryDir = %q, want %q", res.EntryDir, wantDir)
	}
	if res.Family != "created" {
		t.Errorf("Family = %q, want created", res.Family)
	}

	ctx := readFile(t, filepath.Join(wantDir, "context.md"))
	for _, want := range []string{
		`id: "acme-sensors-edge-fw"`,
		"# edge-fw: Work SDK Context",
		"acme-sensors family context",
		`created: "2026-06-16"`,
	} {
		if !strings.Contains(ctx, want) {
			t.Errorf("context.md missing %q", want)
		}
	}
	// ${PROJECTS_PATH} is a literal env-var the user fills later, NOT a token.
	if !strings.Contains(ctx, "${PROJECTS_PATH}") {
		t.Errorf("context.md should keep literal ${PROJECTS_PATH}")
	}
	// No token delimiter should survive rendering.
	if strings.Contains(ctx, "{{") {
		t.Errorf("context.md has unrendered token:\n%s", ctx)
	}

	mem := readFile(t, filepath.Join(wantDir, "memory", "MEMORY.md"))
	if !strings.Contains(mem, "# edge-fw — Work SDK Session Memory") {
		t.Errorf("MEMORY.md missing component header:\n%s", mem)
	}

	fam := readFile(t, filepath.Join(opts.VaultPath, "50_work", "45-development", "acme-sensors", "context.md"))
	for _, want := range []string{"# acme-sensors: Product Family Context", "| edge-fw |"} {
		if !strings.Contains(fam, want) {
			t.Errorf("family context.md missing %q", want)
		}
	}
}

func TestWriteWorkEntrySkipsExistingThenForceRegenerates(t *testing.T) {
	opts := writeOpts(t)
	if _, err := WriteWorkEntry(opts); err != nil {
		t.Fatalf("first write: %v", err)
	}
	ctxPath := filepath.Join(opts.VaultPath, "50_work", "45-development", "acme-sensors", "edge-fw", "context.md")
	if err := os.WriteFile(ctxPath, []byte("HAND-EDITED"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-run without --force: the hand-edited file is preserved.
	res, err := WriteWorkEntry(opts)
	if err != nil {
		t.Fatalf("re-run: %v", err)
	}
	if !contains(res.Skipped, "context.md") {
		t.Errorf("expected context.md skipped, got Skipped=%v Created=%v", res.Skipped, res.Created)
	}
	if got := readFile(t, ctxPath); got != "HAND-EDITED" {
		t.Errorf("skip-if-present clobbered the file: %q", got)
	}

	// Re-run with --force: regenerated from the template.
	opts.Force = true
	res, err = WriteWorkEntry(opts)
	if err != nil {
		t.Fatalf("force re-run: %v", err)
	}
	if !contains(res.Created, "context.md") {
		t.Errorf("expected context.md regenerated under --force, got Created=%v", res.Created)
	}
	if got := readFile(t, ctxPath); got == "HAND-EDITED" {
		t.Errorf("--force did not regenerate the file")
	}
}

func TestWriteWorkEntryFamilyContextNotClobberedBySecondComponent(t *testing.T) {
	vault := t.TempDir()
	first := WorkEntryOptions{VaultPath: vault, Family: "acme-sensors", Component: "edge-fw", Date: "2026-06-16"}
	if _, err := WriteWorkEntry(first); err != nil {
		t.Fatalf("first component: %v", err)
	}
	famPath := filepath.Join(vault, "50_work", "45-development", "acme-sensors", "context.md")
	// Simulate the family file having accumulated a hand-maintained repo table.
	if err := os.WriteFile(famPath, []byte("ACCUMULATED FAMILY TABLE"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A second component in the same family must NOT overwrite the family file,
	// even with --force.
	second := WorkEntryOptions{VaultPath: vault, Family: "acme-sensors", Component: "gateway", Date: "2026-06-16", Force: true}
	res, err := WriteWorkEntry(second)
	if err != nil {
		t.Fatalf("second component: %v", err)
	}
	if res.Family != "exists" {
		t.Errorf("Family = %q, want exists (not regenerated)", res.Family)
	}
	if got := readFile(t, famPath); got != "ACCUMULATED FAMILY TABLE" {
		t.Errorf("family context clobbered by second component: %q", got)
	}
}

func TestWriteWorkEntryRejectsPathTraversal(t *testing.T) {
	cases := []struct{ name, family, component string }{
		{"family traversal", "../../etc", "edge-fw"},
		{"component traversal", "acme", "../../../tmp/pwned"},
		{"component dotdot", "acme", ".."},
		{"family with slash", "a/b", "edge-fw"},
		{"empty component", "acme", ""},
		{"leading dot", ".hidden", "edge-fw"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vault := t.TempDir()
			_, err := WriteWorkEntry(WorkEntryOptions{VaultPath: vault, Family: tc.family, Component: tc.component, Date: "2026-06-16"})
			if err == nil {
				t.Fatalf("expected error for family=%q component=%q", tc.family, tc.component)
			}
			// Nothing should have been written outside a valid slug dir.
			if entries, _ := os.ReadDir(filepath.Join(vault, "50_work", "45-development")); len(entries) > 0 {
				t.Errorf("traversal wrote %d dirs under 45-development", len(entries))
			}
		})
	}
}

func TestResolveVaultStrictErrorsWhenAbsent(t *testing.T) {
	t.Setenv("VAULT_PATH", filepath.Join(t.TempDir(), "does-not-exist"))
	if _, err := ResolveVaultStrict(); err == nil {
		t.Fatal("expected error when vault is absent")
	}
}

func TestResolveVaultStrictHonorsVaultPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VAULT_PATH", dir)
	got, err := ResolveVaultStrict()
	if err != nil {
		t.Fatalf("ResolveVaultStrict: %v", err)
	}
	if got != dir {
		t.Errorf("ResolveVaultStrict = %q, want %q", got, dir)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
