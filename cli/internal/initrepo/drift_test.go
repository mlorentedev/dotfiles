package initrepo

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
// mirrors cli/internal/spec/drift_test.go.
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

	// embedded name (under templates/) -> vault SSOT filename.
	pairs := map[string]string{
		"agents-spec-section.md": "agents-spec-section.md",
	}
	for embedded, vaultName := range pairs {
		want, err := os.ReadFile(filepath.Join(tmplDir, vaultName))
		if err != nil {
			t.Fatalf("read vault %s: %v", vaultName, err)
		}
		got, err := templatesFS.ReadFile("templates/" + embedded)
		if err != nil {
			t.Fatalf("read embedded %s: %v", embedded, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("embedded templates/%s drifted from vault %s.\n"+
				"Re-vendor: cp %s cli/internal/initrepo/templates/%s",
				embedded, vaultName, filepath.Join(tmplDir, vaultName), embedded)
		}
	}
}
