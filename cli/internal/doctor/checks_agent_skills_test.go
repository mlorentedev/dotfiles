package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The check this file covers existed only as a COMMENT until 2026-08-31:
// harness.Decide's EnforceUnset branch claimed "surfaced by `dotf doctor`", and
// nothing in production called UnmigratedSkills. These cases pin that the claim
// is now true, and that it stays true for the migration's whole duration —
// during which the answer changes on every record.
func writeSkillsFixture(t *testing.T, dir, persona, skillsBlock string) {
	t.Helper()
	recDir := filepath.Join(dir, "harness", "agents", persona)
	if err := os.MkdirAll(recDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "---\nid: agent-" + persona + "\nname: " + persona +
		"\nkind: invocable\n" + skillsBlock + "owner: manu\n---\n\n# " + persona + "\n"
	if err := os.WriteFile(filepath.Join(recDir, "AGENT.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("write record: %v", err)
	}
	mf := filepath.Join(dir, "harness", "manifest.json")
	if err := os.WriteFile(mf, []byte(`{"agents":{"record_dir":"harness/agents"}}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestCheckAgentSkillsMigrated(t *testing.T) {
	for _, tc := range []struct {
		name        string
		skillsBlock string
		wantLevel   string
		wantSubstr  string
	}{
		{
			name:        "the legacy inline form is a WARN naming what is not enforced",
			skillsBlock: "skills: [audit, adversarial-review]\n",
			wantLevel:   "warn",
			wantSubstr:  "2 of 2 persona skills carry no `enforce:`",
		},
		{
			name:        "a fully migrated record passes",
			skillsBlock: "skills:\n  - id: audit\n    enforce: warn\n  - id: adversarial-review\n    enforce: block\n",
			wantLevel:   "pass",
			wantSubstr:  "every persona skill declares an enforcement severity (2)",
		},
		{
			name:        "a PARTIALLY migrated record still warns, counting only the gap",
			skillsBlock: "skills:\n  - id: audit\n    enforce: block\n  - adversarial-review\n",
			wantLevel:   "warn",
			wantSubstr:  "1 of 2 persona skills",
		},
		{
			name:        "a record declaring no skills is skipped, not passed",
			skillsBlock: "",
			wantLevel:   "skip",
			wantSubstr:  "no persona declares any skill",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeSkillsFixture(t, dir, "reviewer", tc.skillsBlock)

			var buf bytes.Buffer
			rep := NewReport(&buf, true)
			checkAgentSkillsMigrated(&Config{DotfilesDir: dir}, rep)

			text := buf.String()
			if !strings.Contains(text, tc.wantSubstr) {
				t.Fatalf("report does not contain %q:\n%s", tc.wantSubstr, text)
			}
			// The LEVEL is load-bearing, not decoration: an unmigrated skill is a
			// known, deliberate state, so failing the machine's health command over
			// it would train the reader to ignore the line.
			if tc.wantLevel == "warn" && rep.Failures() != 0 {
				t.Errorf("an unmigrated skill must WARN, never FAIL:\n%s", text)
			}
		})
	}
}
