// Package spec implements the `dot spec` scaffold domain: feature-id
// validation, the GitHub work-gate, and rendering the per-feature spec folder
// from templates embedded in the binary. It mirrors scripts/init-spec.sh, but
// the templates are vendored via //go:embed so the binary is self-contained and
// works on machines without the vault checked out (the SELF-002 class, Go side).
package spec

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// repoSlug is the GitHub repo the work-gate issue belongs to. It is written
// into the proposal frontmatter `issue:` field as "<repoSlug>#<num>" — the fix
// for init-spec.sh, which injected the Why provenance comment but left the
// frontmatter field empty.
const repoSlug = "dotfiles"

// templateNames are the spec files rendered into specs/<id>/, in a stable
// order. proposal.md is special-cased (it carries the issue: frontmatter and
// the ## Why provenance comment); the others are pure placeholder substitution.
var templateNames = []string{"proposal.md", "tasks.md", "verification.md"}

//go:embed templates/proposal.md templates/tasks.md templates/verification.md
var templatesFS embed.FS

// idPattern mirrors init-spec.sh exactly: an AREA-NNN ticket with an optional
// sub-id letter (SDD-012b) and optional slug, or a YYYY-MM-DD dated slug.
var idPattern = regexp.MustCompile(`^([A-Z]+-[0-9]+[a-z]?(-[a-z0-9-]+)?|[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9-]+)$`)

// ValidateID returns an error naming the expected grammar if id is not a valid
// feature-id.
func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf(
			"invalid feature-id: %s\n        expected: TICKET-NNN[letter][-slug] "+
				"(e.g. AI-001-ollama-public, SDD-012b-guard) or YYYY-MM-DD-slug", id)
	}
	return nil
}

// RepoRoot walks up from start until it finds a directory containing .git
// (a directory in a normal clone, a file in a git worktree — both are accepted).
func RepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a git repo (no .git found from %s)", start)
		}
		dir = parent
	}
}

// Gate verifies that issue #num exists and is OPEN, returning its title. It
// shells out to `gh issue view` — a faithful twin of init-spec.sh, and the same
// path the bitácora workflow uses — rather than reimplementing the GitHub API.
// Callers skip Gate entirely under --force-no-gate.
func Gate(num int) (title string, err error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh CLI not found — cannot verify work-gate issue #%d "+
			"(install gh, or use --force-no-gate)", num)
	}
	out, err := exec.Command("gh", "issue", "view", strconv.Itoa(num),
		"--json", "state,title", "--jq", `.state + "\t" + .title`).Output()
	if err != nil {
		return "", fmt.Errorf("work-gate issue #%d not found (or gh failed): %s",
			num, strings.TrimSpace(stderrOf(err)))
	}
	state, title, ok := strings.Cut(strings.TrimRight(string(out), "\n"), "\t")
	if !ok {
		return "", fmt.Errorf("unexpected gh output verifying issue #%d: %q", num, string(out))
	}
	if state != "OPEN" {
		return "", fmt.Errorf("work-gate issue #%d is not open (state: %s) — "+
			"the work-gate is an OPEN issue; reopen it or pick the right one", num, state)
	}
	return title, nil
}

// stderrOf extracts captured stderr from an *exec.ExitError, falling back to the
// error string. It keeps Gate's message useful when gh fails.
func stderrOf(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(ee.Stderr) > 0 {
		return string(ee.Stderr)
	}
	return err.Error()
}

// Render returns each spec file's content keyed by filename, with placeholders
// substituted. When issueNum > 0 the proposal frontmatter `issue:` field is set
// to "<repo>#<num>"; when a non-empty issueTitle is also given, the ## Why
// provenance comment is injected. date is the value for created: (YYYY-MM-DD).
func Render(id, date string, issueNum int, issueTitle string) (map[string]string, error) {
	files := make(map[string]string, len(templateNames))
	for _, name := range templateNames {
		raw, err := templatesFS.ReadFile("templates/" + name)
		if err != nil {
			return nil, fmt.Errorf("reading embedded template %s: %w", name, err)
		}
		s := string(raw)
		s = strings.ReplaceAll(s, "<feature-id>", id)
		s = strings.ReplaceAll(s, "{TITLE}", id)
		s = strings.ReplaceAll(s, "{{date:YYYY-MM-DD}}", date)
		if name == "proposal.md" {
			if issueNum > 0 {
				s = strings.Replace(s, `issue: ""`,
					fmt.Sprintf(`issue: "%s#%d"`, repoSlug, issueNum), 1)
			}
			if issueNum > 0 && issueTitle != "" {
				comment := fmt.Sprintf("<!-- from issue #%d: %s -->", issueNum, issueTitle)
				s = strings.Replace(s, "## Why\n", "## Why\n\n"+comment+"\n", 1)
			}
		}
		files[name] = s
	}
	return files, nil
}

// Scaffold renders the spec for id and writes it under repoRoot/specs/<id>. It
// refuses to overwrite an existing specs/<id> (returns an error). If <id>
// already exists under specs/archive/, it returns a non-empty warning but still
// scaffolds. issueNum/issueTitle are passed through to Render.
func Scaffold(repoRoot, id, date string, issueNum int, issueTitle string) (warning string, err error) {
	specDir := filepath.Join(repoRoot, "specs", id)
	if _, err := os.Stat(specDir); err == nil {
		return "", fmt.Errorf("already exists: %s", specDir)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "specs", "archive", id)); err == nil {
		warning = fmt.Sprintf("%s exists in specs/archive/. Possibly reviving.", id)
	}
	files, err := Render(id, date, issueNum, issueTitle)
	if err != nil {
		return warning, err
	}
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return warning, err
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(specDir, name), []byte(content), 0o644); err != nil {
			return warning, err
		}
	}
	return warning, nil
}
