package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

// TestIDPatternProseMatchesCode is the reconciliation guard for the feature-id
// grammar. The rule was declared in six places and two had already drifted
// silently: AGENTS.md and the agents-spec-section template both omitted the
// `[a-z]?` sub-id letter, so they rejected SDD-012b-guard — an id the code
// accepts and TestValidateID asserts. Nothing reconciled them, so they lied
// until someone crossed both paths.
//
// The guard is a literal compare against idPattern.String(), not a parse of the
// prose, which is why every copy must carry the Go string VERBATIM (the older
// copies used \d where the code uses [0-9] — semantically equal, textually not).
// The code is canonical because it is the enforcement point.
//
// In-repo copies are always checked. The vault copies skip when the vault is
// absent (ADR-013), same rationale as TestEmbeddedTemplatesMatchVault above.
// The enrich-us skill deliberately carries a SUBSET (no dated-slug alternative,
// since a dated slug is not a backlog id) and is therefore not listed here.
func TestIDPatternProseMatchesCode(t *testing.T) {
	want := idPattern.String()

	// RepoRoot must be given an ABSOLUTE path: it walks up with filepath.Dir,
	// and filepath.Dir(".") is "." — so a relative start exits the loop on the
	// first iteration and reports "not in a git repo" from inside one.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root, err := RepoRoot(wd)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}

	check := func(label, path string) {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", label, err)
			return
		}
		if !strings.Contains(string(b), want) {
			t.Errorf("%s does not carry the canonical feature-id grammar verbatim.\n"+
				"want the exact string: %s\n"+
				"fix: replace that file's regex with the string above — do not reword it",
				label, want)
		}
	}

	// The initrepo template is listed even though TestEmbeddedTemplatesMatchVault
	// already pins it byte-for-byte to the vault: that guard SKIPS without the
	// vault, so in CI the embedded copy would otherwise go unchecked. It is a
	// repo file, so checking it offline closes that gap.
	for _, rel := range []string{
		"AGENTS.md",
		filepath.Join("harness", "skills", "spec", "SKILL.md"),
		filepath.Join("cli", "internal", "initrepo", "templates", "agents-spec-section.md"),
	} {
		check(rel, filepath.Join(root, rel))
	}

	vault := os.Getenv("VAULT_PATH")
	if vault == "" {
		if home, err := os.UserHomeDir(); err == nil {
			vault = filepath.Join(home, "Projects", "knowledge")
		}
	}
	metaDir := filepath.Join(vault, "00_meta")
	if _, err := os.Stat(metaDir); err != nil {
		t.Logf("vault absent (%s) — checked in-repo copies only (ADR-013)", metaDir)
		return
	}
	for _, rel := range []string{
		filepath.Join("skills", "spec", "SKILL.md"),
		filepath.Join("templates", "agents-spec-section.md"),
	} {
		check("vault 00_meta/"+filepath.ToSlash(rel), filepath.Join(metaDir, rel))
	}
}

// eqIgnoringEOL compares ignoring CR, so CRLF and LF are equal. The embedded copy
// is EOL-normalized to LF by .gitattributes; the vault is a separate repo this one
// does not govern, so a byte-exact compare is line-ending-fragile across machines
// (CRLF on a Windows checkout) and masks real content drift (#597).
func eqIgnoringEOL(a, b []byte) bool {
	return bytes.Equal(bytes.ReplaceAll(a, []byte("\r"), nil), bytes.ReplaceAll(b, []byte("\r"), nil))
}
