package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// writeRecord drops one agent record into a temp dir and returns its path.
func writeRecord(t *testing.T, frontmatter string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENT.md")
	body := "---\n" + frontmatter + "\n---\n\n# Persona\n\nbody\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write record: %v", err)
	}
	return path
}

// TestHarnessResolveSkills pins the stdout contract.
//
// captureRealStreams rather than execute(), for the same reason resolve-tier
// gives: the sole consumer is `compile-harness.sh` reading through a shell
// `$(...)`, where a value written to stderr arrives as an empty string. An empty
// string is precisely the failure this subcommand exists to prevent, so a test
// that could not tell stdout from stderr would be blind to it (BUG-070 #915).
func TestHarnessResolveSkills(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		wantStdout  string
		wantErrSubs []string
	}{
		{
			name:        "the legacy inline form",
			frontmatter: "name: x\nkind: invocable\nskills: [audit, cyclomatic-complexity]",
			wantStdout:  "[audit, cyclomatic-complexity]",
		},
		{
			// The whole point of the subcommand. skill_field returns EMPTY here,
			// which renders "MUST consume: none" -- enforcement removed silently.
			name:        "the mapping form carrying severity",
			frontmatter: "name: x\nkind: invocable\nskills:\n  - id: audit\n    enforce: block\n  - id: insights\n    enforce: warn",
			wantStdout:  "[audit, insights]",
		},
		{
			// Severity is the gate's input, never the presence text's. Emitting
			// it here would also push the doctrine payload past .gemini's hard
			// 12000-character cap, which the roster has already breached twice.
			name:        "severity never reaches stdout",
			frontmatter: "name: x\nkind: invocable\nskills:\n  - id: audit\n    enforce: block",
			wantStdout:  "[audit]",
		},
		{
			// Distinct from an unreadable one, and the caller renders "none".
			name:        "a record declaring no skills prints nothing and succeeds",
			frontmatter: "name: x\nkind: invocable",
			wantStdout:  "",
		},
		{
			name:        "the mapping form with no severity is refused, not defaulted",
			frontmatter: "name: x\nkind: invocable\nskills:\n  - id: audit",
			wantErrSubs: []string{"enforce", "audit"},
		},
		{
			name:        "an unknown severity is refused",
			frontmatter: "name: x\nkind: invocable\nskills:\n  - id: audit\n    enforce: nag",
			wantErrSubs: []string{"nag", "block"},
		},
		{
			name:        "a scalar skills value is refused rather than coerced",
			frontmatter: "name: x\nkind: invocable\nskills: audit",
			wantErrSubs: []string{"want a list"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := writeRecord(t, tc.frontmatter)
			stdout, _, err := captureRealStreams(t, "harness", "resolve-skills", rec)

			if len(tc.wantErrSubs) > 0 {
				if err == nil {
					t.Fatalf("want an error, got stdout %q", stdout)
				}
				// C15: a refusal must write NOTHING to stdout. A message there
				// would be captured by the shell's $(...) and rendered into the
				// presence block as if it were a skill list.
				if strings.TrimSpace(stdout) != "" {
					t.Errorf("a refusal wrote to stdout: %q", stdout)
				}
				for _, sub := range tc.wantErrSubs {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("error %q does not mention %q", err, sub)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := strings.TrimSpace(stdout); got != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", got, tc.wantStdout)
			}
		})
	}
}

// TestResolveSkillsMatchesShippedRecords is the equivalence proof, run against
// the records the repository actually ships rather than a fixture.
//
// `compile-harness.sh` previously read this field with an awk one-liner that
// takes the text after `skills:` verbatim. Go's strings.Join NORMALISES the
// separator to ", ", so a record written `[a,b]` would render differently
// through the two paths and silently change every deployed instructions file.
// Byte-equality across all shipped records is what makes the substitution
// behaviour-preserving in fact rather than by argument.
func TestResolveSkillsMatchesShippedRecords(t *testing.T) {
	root := repoRootForTest(t)
	dir := filepath.Join(root, "harness", "agents")

	personas, err := harness.LoadPersonas(dir)
	if err != nil {
		t.Fatalf("load personas: %v", err)
	}

	for _, p := range personas {
		t.Run(p.Name, func(t *testing.T) {
			raw, err := os.ReadFile(p.Path)
			if err != nil {
				t.Fatalf("read %s: %v", p.Path, err)
			}
			awkLike := legacySkillField(string(raw))
			if awkLike == "" {
				t.Skipf("%s declares no inline skills line", p.Name)
			}
			stdout, _, err := captureRealStreams(t, "harness", "resolve-skills", p.Path)
			if err != nil {
				t.Fatalf("resolve-skills: %v", err)
			}
			if got := strings.TrimSpace(stdout); got != awkLike {
				t.Errorf("delegation changes the rendered line for %s:\n  awk: %q\n  go : %q",
					p.Name, awkLike, got)
			}
		})
	}
}

// legacySkillField reproduces the awk skill_field read that compile-harness.sh
// used before the delegation: the verbatim text after `skills:` in frontmatter.
func legacySkillField(raw string) string {
	fences := 0
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "---" {
			fences++
			if fences >= 2 {
				return ""
			}
			continue
		}
		if fences != 1 {
			continue
		}
		if after, ok := strings.CutPrefix(line, "skills:"); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
