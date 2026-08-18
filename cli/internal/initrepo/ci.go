package initrepo

import (
	"os"
	"path/filepath"
)

// ciTemplates maps a stack to its embedded CI workflow template. A stack with no
// entry (notably "none") gets no CI scaffold. The Spec-Driven Development
// convention is documented in the generated AGENTS.md, not enforced in CI here —
// an enforced, portable spec-gate is deferred follow-up work (see ADR-022).
var ciTemplates = map[string]string{
	"go":     "ci-go.yml",
	"python": "ci-python.yml",
	"node":   "ci-node.yml",
	"ts":     "ci-node.yml",
}

// WriteCI writes a stack-appropriate .github/workflows/ci.yml under root,
// skip-if-present. It returns the action taken: "created", "skipped" (a ci.yml
// already exists), or "none" (no CI template applies to this stack).
func WriteCI(root, stack string) (string, error) {
	return WriteCIOpts(root, stack, false)
}

// WriteCIOpts is the parameterised form of WriteCI, with dry-run support.
func WriteCIOpts(root, stack string, dryRun bool) (string, error) {
	tmpl, ok := ciTemplates[stack]
	if !ok {
		return "none", nil
	}
	dest := filepath.Join(root, ".github", "workflows", "ci.yml")
	if _, err := os.Stat(dest); err == nil {
		return "skipped", nil
	}
	if dryRun {
		return "created", nil
	}
	raw, err := ReadTemplate(tmpl)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		return "", err
	}
	return "created", nil
}
