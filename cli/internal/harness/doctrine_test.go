package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func enforcedRepo(t *testing.T, records map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for id, body := range records {
		writeFile(t, filepath.Join(repo, "harness", "enforced", id+".md"), body)
	}
	return repo
}

func TestCompactDoctrine_DropsOnlyWhatItCanIdentify(t *testing.T) {
	repo := enforcedRepo(t, map[string]string{
		"a": "> Injected verbatim into every agent's instructions (harness `enforced` id a).\n\nRule A.\n",
		"b": "Rule B.\n<!-- full-only:begin -->\nthe exception\n<!-- full-only:end -->\n\nRule B2.\n",
	})
	got, err := CompactDoctrine(repo, []string{"a", "b"})
	if err != nil || got != "Rule A.\nRule B.\nRule B2.\n" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestCompactDoctrine_Refuses(t *testing.T) {
	repo := enforcedRepo(t, map[string]string{"open": "Rule.\n<!-- full-only:begin -->\nlost\n"})
	if _, err := CompactDoctrine(repo, []string{"open"}); err == nil || !strings.Contains(err.Error(), "full-only") {
		t.Errorf("an unclosed full-only region was accepted: %v", err)
	}
	if _, err := CompactDoctrine(repo, []string{"missing"}); err == nil || !strings.Contains(err.Error(), "missing.md") {
		t.Errorf("a missing record was accepted: %v", err)
	}
}

func TestFoldToASCII(t *testing.T) {
	got := FoldToASCII("a \u2014 b \u2013 \u201cq\u201d \u2019s\u2026 \u2192 \u2190 \u2265 \u2264 \u00d7\u00a0\u00a74 bit\u00e1cora \u00f1 \u00dc")
	if got != `a -- b - "q" 's... -> <- >= <= x Section 4 bitacora n U` {
		t.Errorf("got %q", got)
	}
	if runs := nonASCIIRuns("ok \u2603 and \u2603 and \u00df"); strings.Join(runs, " ") != "c39f e29883" {
		t.Errorf("leftover runs: %v", runs)
	}
}

func TestInjectDoctrine(t *testing.T) {
	cases := []struct{ name, file, want string }{
		{"a region is replaced in place",
			"top\n<!-- BEGIN HARNESS GENERATED (sha256:0) -->\nold\n<!-- END HARNESS GENERATED -->\ntail\n",
			"top\nB\np1\np2\n<!-- END HARNESS GENERATED -->\ntail\n"},
		{"a file with no region gets one appended after a blank line",
			"top\n", "top\n\nB\np1\np2\n<!-- END HARNESS GENERATED -->\n"},
		{"a BEGIN without an END is no region", "<!-- BEGIN HARNESS GENERATED -->\n",
			"<!-- BEGIN HARNESS GENERATED -->\n\nB\np1\np2\n<!-- END HARNESS GENERATED -->\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := injectDoctrine(tc.file, "B", "p1\np2\n"); got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestMigratePreamble_ExactLineOnly(t *testing.T) {
	if got := migratePreamble("# x\n" + legacyDoctrinePreamble + "\n"); got != "# x\n"+DoctrinePreamble+"\n" {
		t.Errorf("the legacy line was not migrated: %q", got)
	}
	lookalike := "# x\n" + legacyDoctrinePreamble + " (mine)\n"
	if got := migratePreamble(lookalike); got != lookalike {
		t.Errorf("a look-alike was rewritten: %q", got)
	}
}

func TestCapWarnings_NameWhoseContentBreached(t *testing.T) {
	if w := capWarnings("f", "small", "small", 100); len(w) != 0 {
		t.Errorf("under the cap: %v", w)
	}
	if w := capWarnings("f", strings.Repeat("x", 120), strings.Repeat("x", 120), 100); len(w) != 1 || !strings.Contains(w[0], "GENERATED doctrine alone") {
		t.Errorf("ours: %v", w)
	}
	if w := capWarnings("f", strings.Repeat("x", 120), "x", 100); len(w) != 1 || !strings.Contains(w[0], "the file's own content") {
		t.Errorf("the user's: %v", w)
	}
	// Bytes, not only characters: 60 two-byte characters breach a 100 cap.
	if w := capWarnings("f", strings.Repeat("\u00df", 60), "x", 100); len(w) != 1 {
		t.Errorf("bytes were not measured: %v", w)
	}
}

func doctrineRepo(t *testing.T) (repo, home string) {
	t.Helper()
	repo = enforcedRepo(t, map[string]string{"rule": "Rule \u2014 one.\n"})
	home = t.TempDir()
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"vault_subpath":"00_meta/patterns",
		"agents":{"record_dir":"harness/agents","deploy":[]},
		"doctrine":{"inject":["rule"],"deploy":[
			{"agent":"agy","file":".gemini/GEMINI.md","char_cap":12000},
			{"agent":"codex","file":".codex/AGENTS.md","shadowed_by":".codex/AGENTS.override.md"}]}}`)
	writeFile(t, filepath.Join(repo, "harness", "agents", "demo", "AGENT.md"), "---\nname: demo\ndescription: d\nkind: invocable\n---\nbody\n")
	return repo, home
}

func TestDeployDoctrine_StartsFillsThenConverges(t *testing.T) {
	repo, home := doctrineRepo(t)
	writeFile(t, filepath.Join(home, ".codex", "AGENTS.override.md"), "mine\n")
	out, err := DeployDoctrine(repo, home, false)
	if err != nil || len(out) != 2 || !out[0].Created || out[0].Status != SkillDeployed {
		t.Fatalf("outcomes: %+v, %v", out, err)
	}
	gemini, _ := os.ReadFile(filepath.Join(home, ".gemini", "GEMINI.md"))
	if !strings.HasPrefix(string(gemini), "# Global rules\n\n"+DoctrinePreamble+"\n\n"+GeneratedBeginPrefix) ||
		!strings.Contains(string(gemini), "Rule -- one.") {
		t.Errorf("the capped file was not started and folded:\n%s", gemini)
	}
	codex, _ := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if !strings.Contains(string(codex), "Rule \u2014 one.") {
		t.Errorf("an uncapped payload was folded:\n%s", codex)
	}
	if len(out[1].Warnings) != 1 || !strings.Contains(out[1].Warnings[0], "never read") {
		t.Errorf("the shadow was not reported: %+v", out[1])
	}
	again, err := DeployDoctrine(repo, home, false)
	if err != nil || again[0].Status != SkillCurrent || again[1].Status != SkillCurrent || again[0].Created {
		t.Errorf("second run: %+v, %v", again, err)
	}
}

func TestDeployDoctrine_DryRunWritesNothing(t *testing.T) {
	repo, home := doctrineRepo(t)
	out, err := DeployDoctrine(repo, home, true)
	if err != nil || len(out) != 2 || out[0].Status != SkillDeployed {
		t.Fatalf("outcomes: %+v, %v", out, err)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("a dry run wrote %d entries", len(entries))
	}
}

func TestLoadDoctrine_NeedsDoctrineAndAgents(t *testing.T) {
	p := filepath.Join(t.TempDir(), "manifest.json")
	writeFile(t, p, `{"doctrine":{"deploy":[{"agent":"agy","file":"f"}]}}`)
	if m, err := LoadDoctrine(p); m != nil || err != nil {
		t.Errorf("doctrine without agents: %+v, %v", m, err)
	}
	writeFile(t, p, `{"agents":{},"doctrine":{"deploy":[{"agent":"agy"}]}}`)
	if _, err := LoadDoctrine(p); err == nil {
		t.Error("a target without a file was accepted")
	}
}
