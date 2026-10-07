package initrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// ciBranchToken is the trigger branch as the templates spell it; WriteCIOpts
// replaces it with the repository's default branch.
const ciBranchToken = "branches: [main]"

// WriteCI writes a stack-appropriate ci.yml under root's workflow directory,
// skip-if-present. It returns the action taken: "created", "skipped" (a ci.yml
// already exists), or "none" (no CI template applies to this stack).
func WriteCI(root, stack string) (string, error) {
	return WriteCIOpts(root, stack, false)
}

// WriteCIOpts is the parameterised form of WriteCI, with dry-run support. The
// directory follows the forge (WorkflowDir) and the push trigger follows the
// default branch (DefaultBranch), so a Gitea repository on `master` gets CI
// that runs (#1962).
func WriteCIOpts(root, stack string, dryRun bool) (string, error) {
	tmpl, ok := ciTemplates[stack]
	if !ok {
		return "none", nil
	}
	// Every forge's directory, not only the one WorkflowDir picks: a checkout
	// migrated between Gitea and Forgejo can hold a ci.yml under the other one.
	for _, d := range []string{".forgejo", ".gitea", ".github"} {
		dir := filepath.Join(d, "workflows")
		if _, err := os.Stat(filepath.Join(root, dir, "ci.yml")); err == nil {
			return "skipped", nil
		}
	}
	if dryRun {
		return "created", nil
	}
	raw, err := ReadTemplate(tmpl)
	if err != nil {
		return "", err
	}
	if strings.Count(string(raw), ciBranchToken) != 1 {
		return "", fmt.Errorf("template %s must hold %q exactly once", tmpl, ciBranchToken)
	}
	body := strings.Replace(string(raw), ciBranchToken, "branches: ["+yamlBranch(DefaultBranch(root))+"]", 1)
	dest := filepath.Join(root, WorkflowDir(root), "ci.yml")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, []byte(body), 0o644); err != nil {
		return "", err
	}
	return "created", nil
}
