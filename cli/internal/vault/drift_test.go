package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedTemplatesMatchVault is the drift guard: every template embedded in
// the binary must stay byte-identical to the vault SSOT it was vendored from.
//
// The dotfiles CI has no access to the private vault (ADR-013), so this test
// SKIPS (not fails) when the vault is absent — it does real work only on a
// machine where the vault is present, which is exactly where template edits
// happen. The skip is written so a skipped run never looks like a failure. This
// mirrors cli/internal/spec/drift_test.go and cli/internal/initrepo/drift_test.go.
func TestEmbeddedTemplatesMatchVault(t *testing.T) {
	vault := os.Getenv("VAULT_PATH")
	if vault == "" {
		if home, err := os.UserHomeDir(); err == nil {
			vault = filepath.Join(home, "Projects", "knowledge")
		}
	}
	tmplDir := filepath.Join(vault, "00_meta", "templates")
	if _, err := os.Stat(tmplDir); err != nil {
		t.Skipf("vault templates absent (%s) — drift guard runs only where the vault is present (ADR-013)", tmplDir)
	}

	// embedded name (under templates/) -> vault SSOT filename (identical here).
	names := []string{
		"work-sdk-context.md",
		"work-sdk-memory.md",
		"work-sdk-family.md",
	}
	for _, name := range names {
		want, err := os.ReadFile(filepath.Join(tmplDir, name))
		if err != nil {
			t.Fatalf("read vault %s: %v", name, err)
		}
		got, err := templatesFS.ReadFile("templates/" + name)
		if err != nil {
			t.Fatalf("read embedded %s: %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("embedded templates/%s drifted from vault %s.\n"+
				"Re-vendor: cp %s cli/internal/vault/templates/%s",
				name, name, filepath.Join(tmplDir, name), name)
		}
	}
}
