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
	}{
		{
			name:         "a tier the deploy target can answer",
			deployAgents: []string{"claude"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: top\n---",
		},
		{
			// The drift this check exists for. `top` names only claude, so a
			// second deploy target makes the record unrenderable for it.
			name:         "a tier one deploy target cannot answer",
			deployAgents: []string{"claude", "opencode"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: top\n---",
			wantFail:     true,
			wantSubs:     []string{"top", "opencode", "AGENT.md"},
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
			rep := NewReport(&buf, false)
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
		})
	}
}

// TestAgentTiersMissingInputsAreNotFailures pins that this check stays silent
// about things it does not own. A missing manifest or record dir is diagnosed by
// checkCompileHarnessDrift, and duplicating that here would print two failures
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

// TestRecordTargetsDefaultsToEveryHarness pins the direction that, inverted,
// turns correct data into noise: an ABSENT targets list means ALL harnesses.
func TestRecordTargetsDefaultsToEveryHarness(t *testing.T) {
	tests := []struct {
		raw, agent string
		want       bool
	}{
		{"", "claude", true},
		{"   ", "opencode", true},
		{"[claude]", "claude", true},
		{"[claude]", "opencode", false},
		{"[claude, opencode]", "opencode", true},
		{"[opencode]", "claude", false},
	}
	for _, tt := range tests {
		if got := recordTargets(tt.raw, tt.agent); got != tt.want {
			t.Errorf("recordTargets(%q, %q) = %v, want %v", tt.raw, tt.agent, got, tt.want)
		}
	}
}
