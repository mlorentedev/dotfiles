// Package vault implements the `dotf vault` domain: scaffolding knowledge-vault
// entries from templates embedded in the binary (//go:embed), drift-tested
// against the vault SSOT exactly like cli/internal/spec and cli/internal/initrepo
// (drift_test.go does real work only where the vault is present — ADR-013).
//
// dotf vault is the single home for vault-entry scaffolding, with the entry TYPE
// as the dimension (ADR-021 step 3): `vault work <family> <component>` scaffolds a
// work-SDK entry under 50_work/45-development/ (this package, CLI-015 #388);
// `vault project <repo>` (the 10_projects/ entry, extracted from dotf init) lands
// in the follow-up #395. Unlike `dotf init`, a missing vault is an error here, not
// a skip: `dotf vault` is a vault-only command.
package vault

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// templatesFS holds the vault-entry template set. Every file under templates/ is
// vendored into the binary and kept byte-identical to its vault SSOT counterpart
// under 00_meta/templates/ (guarded by drift_test.go).
//
//go:embed templates
var templatesFS embed.FS

// readTemplate returns the raw bytes of an embedded template by name (relative to
// templates/). Callers token-replace at render time.
func readTemplate(name string) ([]byte, error) {
	b, err := templatesFS.ReadFile("templates/" + name)
	if err != nil {
		return nil, fmt.Errorf("reading embedded template %q: %w", name, err)
	}
	return b, nil
}

// ResolveVault returns the vault root if it exists, honoring $VAULT_PATH and
// falling back to ~/Projects/knowledge; "" when no vault is present. This is the
// canonical home going forward — initrepo.ResolveVault folds in here when dotf
// init is rewired to call vault (#395).
func ResolveVault() string {
	v := os.Getenv("VAULT_PATH")
	if v == "" {
		if home, err := os.UserHomeDir(); err == nil {
			v = filepath.Join(home, "Projects", "knowledge")
		}
	}
	if v == "" {
		return ""
	}
	if info, err := os.Stat(v); err == nil && info.IsDir() {
		return v
	}
	return ""
}

// ResolveVaultStrict is ResolveVault for vault-only commands: a missing vault is
// an actionable error (naming $VAULT_PATH) rather than the skip signal dotf init
// uses. It is the resolution every `dotf vault` subcommand calls.
func ResolveVaultStrict() (string, error) {
	if v := ResolveVault(); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("no knowledge vault found: set $VAULT_PATH or create ~/Projects/knowledge")
}

// slugPattern accepts a path segment that is safe to join under the vault: it must
// start alphanumeric and contain only [A-Za-z0-9._-]. Combined with the explicit
// ".." check in validateSlug, it rejects path traversal and absolute paths — the
// family/component args become directory names, so they are untrusted input.
var slugPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// validateSlug guards a user-supplied path segment against traversal/injection.
func validateSlug(kind, s string) error {
	if s == "" {
		return fmt.Errorf("%s must not be empty", kind)
	}
	if strings.Contains(s, "..") || !slugPattern.MatchString(s) {
		return fmt.Errorf("invalid %s %q: use letters, digits, '.', '-', '_' (no path separators or '..')", kind, s)
	}
	return nil
}

// fileExists reports whether path exists (file or dir).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
