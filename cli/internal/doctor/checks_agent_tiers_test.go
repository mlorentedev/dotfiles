package doctor

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
			// The render reads only the `targets:` line, which names nobody in
			// the block form, so the record deploys nowhere and cannot fail a
			// render. A FAIL here would be the false positive on correct data.
			name:         "a block-style targets list is judged as the render reads it",
			deployAgents: []string{"claude"},
			frontmatter:  "---\nname: curator\ndescription: x\nkind: invocable\nmodel: ultra\ntargets:\n  - claude\n---",
			wantNot:      []string{"checked"},
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

	t.Run("an unparseable manifest", func(t *testing.T) {
		dir := t.TempDir()
		mustMkdir(t, filepath.Join(dir, "harness"))
		mustWrite(t, filepath.Join(dir, "harness", "manifest.json"), `{"agents": `)
		var buf bytes.Buffer
		rep := NewReport(&buf, false)
		rep.Section("test")
		checkAgentTiersResolve(&Config{DotfilesDir: dir}, tierMap, rep)
		if rep.Failures() != 0 {
			t.Errorf("a manifest that does not parse is checkCompileHarnessDrift's failure, not this check's")
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

// TestRecordTargetsDefaultsToEveryHarness pins the direction that, inverted,
// turns correct data into noise: an ABSENT targets list means ALL harnesses.
// A PRESENT one is read as the render reads it (see recordTargets).
func TestRecordTargetsDefaultsToEveryHarness(t *testing.T) {
	tests := []struct {
		raw      string
		declared bool
		agent    string
		want     bool
	}{
		{"", false, "claude", true},
		{"", false, "opencode", true},
		{"[claude]", true, "claude", true},
		{"[claude]", true, "opencode", false},
		{"[claude, opencode]", true, "opencode", true},
		{"[opencode]", true, "claude", false},
		{`["opencode"]`, true, "opencode", true},
		{"", true, "claude", false}, // block style: the key line names nobody
	}
	for _, tt := range tests {
		if got := recordTargets(tt.raw, tt.declared, tt.agent); got != tt.want {
			t.Errorf("recordTargets(%q, %v, %q) = %v, want %v", tt.raw, tt.declared, tt.agent, got, tt.want)
		}
	}
}

// TestRecordTargetsAgreesWithTheRender runs the render's own predicate,
// `skill_targets_agent` from scripts/compile-harness.sh, against the same
// records the check reads. The check exists to predict the render, so a rule
// that is merely more reasonable than the render's is a check that is wrong.
func TestRecordTargetsAgreesWithTheRender(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil || runtime.GOOS == "windows" {
		t.Skip("needs bash to run the render's predicate")
	}
	script, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "compile-harness.sh"))
	if err != nil {
		t.Fatal(err)
	}
	fn := regexp.MustCompile(`(?ms)^skill_targets_agent\(\) \{\n.*?^\}\n`).Find(script)
	if fn == nil {
		t.Fatal("skill_targets_agent not found in compile-harness.sh; this test must follow it")
	}

	records := []string{
		"model: top",
		"model: top\ntargets: [claude]",
		"model: top\ntargets: [claude, opencode]",
		"model: top\ntargets: [\"opencode\"]",
		"model: top\ntargets:\n  - claude",
		"model: top\ntargets: []",
		"model: top\ntargets: [copilot]",
	}
	agents := []string{"claude", "opencode", "copilot", "codex", "pi", "agy"}
	for _, body := range records {
		path := filepath.Join(t.TempDir(), "AGENT.md")
		mustWrite(t, path, "---\nname: x\n"+body+"\n---\n\nBody.\n")
		fm, err := readAgentFrontmatter(path)
		if err != nil {
			t.Fatal(err)
		}
		targets, declared := fm["targets"]
		for _, agent := range agents {
			// #nosec G204 -- the script text is this repository's own file
			cmd := exec.Command(bash, "-c", string(fn)+`skill_targets_agent "$1" "$2"`, "_", path, agent)
			render := cmd.Run() == nil
			if got := recordTargets(targets, declared, agent); got != render {
				t.Errorf("record %q, harness %s: check says targeted=%v, render says %v", body, agent, got, render)
			}
		}
	}
}
