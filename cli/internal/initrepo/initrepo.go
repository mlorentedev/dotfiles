// Package initrepo implements the `dotf init` repo-scaffolder domain: the
// practice-stack templates a fresh repo is built from, vendored into the binary
// via //go:embed so init works on a machine with no vault checked out (the
// SELF-001 class, #248) and never leaks an unexpanded $VAULT_PATH. It mirrors
// cli/internal/spec: the embedded templates are kept byte-identical to the vault
// SSOT they were vendored from, guarded by drift_test.go, which does real work
// only where the vault is present and skips elsewhere (ADR-013).
//
// This package is built up across the CLI-014 steps: this first cut establishes
// the embed + the drift guard for the AGENTS.md SDD snippet (the one template
// with a vault SSOT today); the `dotf init agents`/`github` subcommands and the
// orchestrator's static artefacts (.gitignore, pre-commit, CI, env-contract)
// land in the following steps.
package initrepo

import (
	"embed"
	"fmt"
)

// templatesFS holds the whole practice-stack template set. The directory is the
// embed contract: every file under templates/ is vendored into the binary.
// agents-spec-section.md has a vault SSOT and is drift-tested (drift_test.go);
// the rest are self-contained static scaffolding with no vault counterpart.
//
//go:embed templates
var templatesFS embed.FS

// ReadTemplate returns the raw bytes of the embedded template named by its path
// relative to templates/ (e.g. "agents-spec-section.md"). It is the load
// primitive every init subcommand builds on; callers transform the bytes
// (snippet extraction, placeholder substitution) at render time. It errors if
// the binary does not vendor a template by that name.
func ReadTemplate(name string) ([]byte, error) {
	b, err := templatesFS.ReadFile("templates/" + name)
	if err != nil {
		return nil, fmt.Errorf("reading embedded template %q: %w", name, err)
	}
	return b, nil
}
