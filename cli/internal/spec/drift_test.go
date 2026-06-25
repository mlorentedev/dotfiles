package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedTemplatesMatchVault is the drift guard: the templates embedded in
// the binary must stay byte-identical to the vault SSOT they were vendored from.
//
// The dotfiles CI has no access to the private vault (ADR-013), so this test
// SKIPS (not fails) when the vault is absent — it does real work only on a
// machine where the vault is present, which is exactly where template edits
// happen. The skip is written so a skipped run never looks like a failure.
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

	pairs := map[string]string{
		"proposal.md":     "spec-proposal.md",
		"tasks.md":        "spec-tasks.md",
		"verification.md": "spec-verification.md",
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
		if !eqIgnoringEOL(got, want) {
			t.Errorf("embedded templates/%s drifted from vault %s.\n"+
				"Re-vendor: cp %s cli/internal/spec/templates/%s",
				embedded, vaultName, filepath.Join(tmplDir, vaultName), embedded)
		}
	}
}

// eqIgnoringEOL compares ignoring CR, so CRLF and LF are equal. The embedded copy
// is EOL-normalized to LF by .gitattributes; the vault is a separate repo this one
// does not govern, so a byte-exact compare is line-ending-fragile across machines
// (CRLF on a Windows checkout) and masks real content drift (#597).
func eqIgnoringEOL(a, b []byte) bool {
	return bytes.Equal(bytes.ReplaceAll(a, []byte("\r"), nil), bytes.ReplaceAll(b, []byte("\r"), nil))
}
