package doctor

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// agentTierFixture builds a deploy dir holding a model map, a manifest and one
// agent record, so a test can put the two committed files out of step on purpose.
func agentTierFixture(t *testing.T, deployAgents []string, recordFrontmatter string) *Config {
	t.Helper()
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "harness", "agents", "curator"))

	manifest := map[string]any{
		"version": 1,
		"agents": map[string]any{
			"record_dir": "harness/agents",
			"deploy": func() []any {
				out := make([]any, 0, len(deployAgents))
				for _, a := range deployAgents {
					out = append(out, map[string]any{"agent": a, "render": "agent-md", "dir": ".x/agents"})
				}
				return out
			}(),
		},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "harness", "manifest.json"), string(raw))
	mustWrite(t, filepath.Join(dir, "harness", "agents", "curator", "AGENT.md"),
		recordFrontmatter+"\n\n# Curator\n\nBody.\n")
	return &Config{DotfilesDir: dir}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// tierMap is the `tiers` block the checks below resolve against: claude has all
// three, opencode only mid — the shipped map's real asymmetry.
var tierMap = map[string]any{
	"tiers": map[string]any{
		"top": map[string]any{"claude": "opus"},
		"mid": map[string]any{"claude": "sonnet", "opencode": "qwen3.6-plus"},
		"low": map[string]any{"claude": "haiku"},
	},
}

// TestAgentTiersResolve is #1164: two COMMITTED files can disagree — a record
// declares a tier, and model-map decides which harnesses that tier covers — and
// before this check nothing noticed until a deploy ran on someone's machine.
func TestAgentTiersResolve(t *testing.T) {
	tests := []struct {
		name         string
		deployAgents []string
		frontmatter  string
		wantFail     bool
		wantSubs     []string
		wantNot      []string
	}{
		{
			name:         "a tier the deploy target can answer",
			deployAgents: []string{"claude"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: top\n---",
			// The count tells "everything resolved" from "nothing was looked at".
			wantSubs: []string{"(1 checked)"},
		},
		{
			name:         "every pair counts toward the pass line",
			deployAgents: []string{"claude", "opencode"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: mid\n---",
			wantSubs:     []string{"(2 checked)"},
		},
		{
			// The drift this check exists for. `top` names only claude, so a
			// second deploy target makes the record unrenderable for it.
			name:         "a tier one deploy target cannot answer",
			deployAgents: []string{"claude", "opencode"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: top\n---",
			wantFail:     true,
			wantSubs:     []string{"top", "opencode", "AGENT.md"},
			// A FAIL beside an OK line for the same check is a contradiction.
			wantNot: []string{"checked)"},
		},
		{
			// Declaring no tier is not an error: the render emits no model line.
			name:         "a record declaring no tier is not drift",
			deployAgents: []string{"claude", "opencode"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\n---",
		},
		{
			// The false positive that would train an operator to ignore this
			// line: a record scoped to one harness must not be judged against
			// the others.
			name:         "a record scoped by targets is judged only against those",
			deployAgents: []string{"claude", "opencode"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: top\ntargets: [claude]\n---",
			wantSubs:     []string{"(1 checked)"},
		},
		{
			// A quoted entry still targets its harness in the render, so the
			// drift behind it must not go silent.
			name:         "a quoted targets entry is judged like a bare one",
			deployAgents: []string{"claude", "opencode"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: top\ntargets: [\"opencode\"]\n---",
			wantFail:     true,
			wantSubs:     []string{"top", "opencode"},
		},
		{
			// The Go render parses the list as YAML, so the block form targets
			// the harnesses it names (#1733); the shell's read it as nobody.
			name:         "a block-style targets list targets the harnesses it names",
			deployAgents: []string{"claude"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: ultra\ntargets:\n  - claude\n---",
			wantFail:     true,
			wantSubs:     []string{"ultra", "claude"},
		},
		{
			// Matched by name, not by substring: copilot does not target pi.
			name:         "a harness name inside another is not a target",
			deployAgents: []string{"pi"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: top\ntargets: [copilot]\n---",
			wantNot:      []string{"checked", "pi"},
		},
		{
			// The render normalises line endings, so a CRLF record resolves.
			name:         "a CRLF record resolves its tier",
			deployAgents: []string{"claude"},
			frontmatter:  "---\r\nname: curator\r\ndescription: x\r\nkind: invocable\r\nmodel: top\r\n---\r",
			wantSubs:     []string{"(1 checked)"},
		},
		{
			// YAML refuses a repeated key, and the render with it.
			name:         "a record the render cannot parse fails once",
			deployAgents: []string{"claude", "opencode"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: top\ntargets: [claude]\ntargets: [opencode]\n---",
			wantFail:     true,
			wantSubs:     []string{"cannot be read for its targets"},
		},
		{
			// An indented key belongs to a nested block, not to the record.
			name:         "an indented model key is not the record's tier",
			deployAgents: []string{"claude"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmeta:\n  model: ultra\n---",
			wantNot:      []string{"ultra", "checked"},
		},
		{
			name:         "a tier no tier block declares at all",
			deployAgents: []string{"claude"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: ultra\n---",
			wantFail:     true,
			wantSubs:     []string{"ultra", "claude"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := agentTierFixture(t, tt.deployAgents, tt.frontmatter)
			var buf bytes.Buffer
			// Verbose, so the pass line and its count are in the output.
			rep := NewReport(&buf, true)
			rep.Section("test")
			checkAgentTiersResolve(cfg, tierMap, rep)

			if tt.wantFail && rep.Failures() == 0 {
				t.Fatal("expected a FAIL; a record whose tier the map cannot answer will break the next deploy")
			}
			if !tt.wantFail && rep.Failures() != 0 {
				t.Fatalf("unexpected FAIL on valid data: %s", buf.String())
			}
			text := buf.String()
			for _, sub := range tt.wantSubs {
				if !strings.Contains(text, sub) {
					t.Errorf("report does not name %q, so the operator cannot tell which record or harness:\n%s", sub, text)
				}
			}
			for _, sub := range tt.wantNot {
				if strings.Contains(text, sub) {
					t.Errorf("report says %q, but the render deploys this record nowhere:\n%s", sub, text)
				}
			}
		})
	}
}

// TestAgentTiersMissingInputsAreNotFailures pins that this check stays silent
// about things it does not own. A missing manifest or record dir is diagnosed by
// checkRecordDrift, and duplicating that here would print two failures
// for one cause.
func TestAgentTiersMissingInputsAreNotFailures(t *testing.T) {
	t.Run("no manifest at all", func(t *testing.T) {
		var buf bytes.Buffer
		rep := NewReport(&buf, false)
		rep.Section("test")
		checkAgentTiersResolve(&Config{DotfilesDir: t.TempDir()}, tierMap, rep)
		if rep.Failures() != 0 {
			t.Errorf("an absent manifest is not this check's failure to report")
		}
	})

	t.Run("an unparseable manifest", func(t *testing.T) {
		dir := t.TempDir()
		mustMkdir(t, filepath.Join(dir, "harness"))
		mustWrite(t, filepath.Join(dir, "harness", "manifest.json"), `{"agents": `)
		var buf bytes.Buffer
		rep := NewReport(&buf, false)
		rep.Section("test")
		checkAgentTiersResolve(&Config{DotfilesDir: dir}, tierMap, rep)
		if rep.Failures() != 0 {
			t.Errorf("a manifest that does not parse is checkRecordDrift's failure, not this check's")
		}
		if !strings.Contains(buf.String(), "does not parse") {
			t.Errorf("the check must say it did not run, or silence reads as a pass:\n%s", buf.String())
		}
	})

	t.Run("a manifest with no deploy targets", func(t *testing.T) {
		dir := t.TempDir()
		mustMkdir(t, filepath.Join(dir, "harness"))
		mustWrite(t, filepath.Join(dir, "harness", "manifest.json"), `{"version":1}`)
		var buf bytes.Buffer
		rep := NewReport(&buf, false)
		rep.Section("test")
		checkAgentTiersResolve(&Config{DotfilesDir: dir}, tierMap, rep)
		if rep.Failures() != 0 {
			t.Errorf("nothing renders agent definitions, so there is no tier to disagree about")
		}
	})
}

// TestAgentTiersDefaultRecordDir pins the manifest default: no record_dir
// means harness/agents, the render's own default.
func TestAgentTiersDefaultRecordDir(t *testing.T) {
	cfg := agentTierFixture(t, []string{"claude"}, "---\nname: curator\nmodel: ultra\n---")
	mustWrite(t, filepath.Join(cfg.DotfilesDir, "harness", "manifest.json"),
		`{"version":1,"agents":{"deploy":[{"agent":"claude"}]}}`)
	var buf bytes.Buffer
	rep := NewReport(&buf, false)
	rep.Section("test")
	checkAgentTiersResolve(cfg, tierMap, rep)
	if rep.Failures() != 1 {
		t.Errorf("the default record dir must be read, so the undeclared tier fails:\n%s", buf.String())
	}
}
