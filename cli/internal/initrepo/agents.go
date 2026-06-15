package initrepo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Markers and headings used to locate the SDD section. The snippet markers match
// the vault SSOT (agents-spec-section.md); sddHeading is the section anchor used
// for idempotency and in-place replacement — the Go twin of init-repo-agents.sh's
// grep '^## Spec-Driven Development$' and its awk section surgery.
const (
	snippetBegin = "## --- BEGIN SNIPPET ---"
	snippetEnd   = "## --- END SNIPPET ---"
	sddHeading   = "## Spec-Driven Development"
)

// agentsHeader is the minimal AGENTS.md scaffold written when the target repo has
// no AGENTS.md yet, before the SDD section is appended.
const agentsHeader = "# AGENTS.md\n\n" +
	"> Instructions for AI coding agents (Claude, Copilot, Cursor, Codex) operating in this repo.\n"

// AgentsSnippet returns the self-contained `## Spec-Driven Development` section
// extracted from the embedded template (the content between the BEGIN/END SNIPPET
// markers, exclusive), trimmed of surrounding blank lines.
//
// It fails closed (#248): if the extracted snippet still carries an unexpanded
// $VAULT_PATH — i.e. vault drift reintroduced one — it returns an error rather
// than emit a section that leaks on a vault-less machine. This is the render-time
// guard backing the choice to vendor the snippet self-contained at the SSOT.
func AgentsSnippet() (string, error) {
	raw, err := ReadTemplate("agents-spec-section.md")
	if err != nil {
		return "", err
	}
	var body []string
	inSnippet := false
	for _, ln := range strings.Split(string(raw), "\n") {
		switch ln {
		case snippetBegin:
			inSnippet = true
		case snippetEnd:
			inSnippet = false
		default:
			if inSnippet {
				body = append(body, ln)
			}
		}
	}
	snippet := strings.Trim(strings.Join(body, "\n"), "\n")
	if snippet == "" {
		return "", fmt.Errorf("BEGIN/END SNIPPET markers not found (or empty) in embedded agents-spec-section.md")
	}
	if strings.Contains(snippet, "$VAULT_PATH") {
		return "", fmt.Errorf("embedded SDD snippet leaks $VAULT_PATH — re-vendor from a self-contained vault SSOT (#248)")
	}
	return snippet, nil
}

// AgentsResult reports what BootstrapAgents did to the target AGENTS.md.
type AgentsResult struct {
	Path   string // absolute path to AGENTS.md
	Action string // "created" | "appended" | "replaced" | "unchanged"
}

// BootstrapAgents writes (or refreshes) the SDD section in repoRoot/AGENTS.md,
// mirroring init-repo-agents.sh:
//
//   - no AGENTS.md            -> create it with a minimal header + the section
//   - AGENTS.md, no section   -> append the section, preserving prior content
//   - AGENTS.md, has section  -> no-op, unless force replaces it in place
//
// It is idempotent: a re-run without force is a safe no-op.
func BootstrapAgents(repoRoot string, force bool) (AgentsResult, error) {
	snippet, err := AgentsSnippet()
	if err != nil {
		return AgentsResult{}, err
	}
	path := filepath.Join(repoRoot, "AGENTS.md")
	res := AgentsResult{Path: path}

	existing, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.WriteFile(path, []byte(agentsHeader+"\n"+snippet+"\n"), 0o644); err != nil {
			return res, err
		}
		res.Action = "created"
		return res, nil
	case err != nil:
		return res, fmt.Errorf("reading %s: %w", path, err)
	}

	content := string(existing)
	switch {
	case hasSDDSection(content) && !force:
		res.Action = "unchanged"
		return res, nil
	case hasSDDSection(content):
		if err := os.WriteFile(path, []byte(replaceSDDSection(content, snippet)), 0o644); err != nil {
			return res, err
		}
		res.Action = "replaced"
		return res, nil
	default:
		appended := strings.TrimRight(content, "\n") + "\n\n" + snippet + "\n"
		if err := os.WriteFile(path, []byte(appended), 0o644); err != nil {
			return res, err
		}
		res.Action = "appended"
		return res, nil
	}
}

// hasSDDSection reports whether content already carries a `## Spec-Driven
// Development` H2 (matched as a full line, like the script's anchored grep).
func hasSDDSection(content string) bool {
	for _, ln := range strings.Split(content, "\n") {
		if ln == sddHeading {
			return true
		}
	}
	return false
}

// replaceSDDSection swaps the existing SDD section for snippet, preserving
// everything before it and from the next H2 (or EOF) onward — the Go twin of
// init-repo-agents.sh's awk: at the SDD heading print the fresh snippet and skip
// until the next `## ` heading.
func replaceSDDSection(content, snippet string) string {
	var out []string
	skipping := false
	for _, ln := range strings.Split(content, "\n") {
		switch {
		case ln == sddHeading:
			out = append(out, snippet)
			skipping = true
		case skipping && strings.HasPrefix(ln, "## "):
			skipping = false
			out = append(out, ln)
		case skipping:
			// inside the old section body — drop it
		default:
			out = append(out, ln)
		}
	}
	return strings.Join(out, "\n")
}
