package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// presenceRepo mirrors tests/compile-harness.bats' seed_agents_fixture: one
// invocable persona (curator) targeting all four harnesses, plus whatever the
// test adds.
func presenceRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	write(t, filepath.Join(repo, "harness", "manifest.json"), `{ "version": 1,
  "agents": { "record_dir": "harness/agents",
    "presence": [
      { "agent": "claude",   "file": ".claude/CLAUDE.md" },
      { "agent": "opencode", "file": ".config/opencode/AGENTS.md" },
      { "agent": "pi",       "file": ".pi/agent/AGENTS.md" },
      { "agent": "copilot",  "file": ".copilot/copilot-instructions.md", "requires_command": "copilot" } ] } }`)
	writeRecord(t, repo, "curator", "---\nname: curator\ndescription: Crystallize-phase persona.\nkind: invocable\nmodel: top\nskills: [vault-doctor, crystallize, genre-picker]\ntargets: [claude, opencode, pi, copilot]\n---\n\n# Curator\n")
	return repo
}

func writeRecord(t *testing.T, repo, name, body string) {
	t.Helper()
	write(t, filepath.Join(repo, "harness", "agents", name, "AGENT.md"), body)
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const instructionsFixture = "user intro\n\n<!-- BEGIN HARNESS GENERATED -->\npatterns content\n<!-- END HARNESS GENERATED -->\n\nuser outro\n"

func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // test fixture
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The block is byte-identical to compile-harness.sh's build_agent_presence:
// header once, one bullet per invocable persona in name order, ids as the
// resolve-skills flow sequence, "none" for a persona without skills.
func TestBuildPresence_MatchesTheShellRendering(t *testing.T) {
	repo := presenceRepo(t)
	writeRecord(t, repo, "auto", "---\nname: auto\ndescription: never listed.\nkind: autonomous\nskills: [x]\n---\n")
	writeRecord(t, repo, "bare", "---\nname: bare\ndescription: no skills.\nkind: invocable\n---\n")
	personas, err := LoadPersonas(filepath.Join(repo, "harness", "agents"))
	if err != nil {
		t.Fatal(err)
	}
	got := BuildPresence(personas, "claude")
	want := "## Active agent personas — forced skills\n\nWhen acting as one, you MUST consume its skills.\n\n" +
		"- **bare** — MUST consume: none\n" +
		"- **curator** — MUST consume: [vault-doctor, crystallize, genre-picker]\n"
	if got != want {
		t.Errorf("block differs from the shell rendering:\n got: %q\nwant: %q", got, want)
	}
	if PresenceSHA(got) != PresenceSHA(strings.ReplaceAll(got, "\n", "\r\n")) {
		t.Error("the sha must not depend on line endings")
	}
}

// HARNESS-045 AC7, carried over from tests/compile-harness.bats: a record whose
// skills are in the mapping form (per-skill severity) renders its ids and only
// its ids — never "none", never the severity, which belongs to the gate.
func TestBuildPresence_RendersMappingFormSkillsWithoutSeverity(t *testing.T) {
	repo := t.TempDir()
	writeRecord(t, repo, "curator", "---\nname: curator\ndescription: x\nkind: invocable\nskills:\n  - id: crystallize\n    enforce: block\n  - id: insights\n    enforce: warn\n---\n")
	personas, err := LoadPersonas(filepath.Join(repo, "harness", "agents"))
	if err != nil {
		t.Fatal(err)
	}
	got := BuildPresence(personas, "claude")
	if !strings.Contains(got, "- **curator** — MUST consume: [crystallize, insights]\n") {
		t.Errorf("mapping-form skills must render their ids:\n%s", got)
	}
	if strings.Contains(got, "none") || strings.Contains(got, "enforce") || strings.Contains(got, "block") {
		t.Errorf("neither 'none' nor severity may appear:\n%s", got)
	}
}

func TestBuildPresence_RespectsTargetsAndSkipsAutonomous(t *testing.T) {
	repo := presenceRepo(t)
	writeRecord(t, repo, "scribe", "---\nname: scribe\ndescription: opencode only.\nkind: invocable\nskills: [docs-skill]\ntargets: [opencode]\n---\n")
	personas, err := LoadPersonas(filepath.Join(repo, "harness", "agents"))
	if err != nil {
		t.Fatal(err)
	}
	claude, opencode := BuildPresence(personas, "claude"), BuildPresence(personas, "opencode")
	if strings.Contains(claude, "scribe") || !strings.Contains(claude, "curator") {
		t.Errorf("claude must list curator only:\n%s", claude)
	}
	if !strings.Contains(opencode, "scribe") || !strings.Contains(opencode, "curator") {
		t.Errorf("opencode must list both:\n%s", opencode)
	}
	// targets: [pi] must not match "copilot" — the shell's substring test did.
	writeRecord(t, repo, "piper", "---\nname: piper\ndescription: pi only.\nkind: invocable\ntargets: [pi]\n---\n")
	personas, _ = LoadPersonas(filepath.Join(repo, "harness", "agents"))
	if strings.Contains(BuildPresence(personas, "copilot"), "piper") {
		t.Error("a persona targeting pi must not appear for copilot")
	}
}

func TestBuildPresence_EmptyWhenNoPersonaTargetsTheHarness(t *testing.T) {
	repo := t.TempDir()
	writeRecord(t, repo, "scribe", "---\nname: scribe\ndescription: opencode only.\nkind: invocable\ntargets: [opencode]\n---\n")
	personas, err := LoadPersonas(filepath.Join(repo, "harness", "agents"))
	if err != nil {
		t.Fatal(err)
	}
	if got := BuildPresence(personas, "claude"); got != "" {
		t.Errorf("no persona targets claude → empty block, got %q", got)
	}
}

// AC1/AC2 (HARNESS-092, #1326): inject into every target, leave user content
// and the GENERATED region untouched, and be idempotent — two runs, one region.
func TestDeployPresence_InjectsEveryTargetOnceAndKeepsTheRest(t *testing.T) {
	repo, home := presenceRepo(t), t.TempDir()
	files := []string{".claude/CLAUDE.md", ".config/opencode/AGENTS.md", ".pi/agent/AGENTS.md", ".copilot/copilot-instructions.md"}
	for _, f := range files {
		write(t, filepath.Join(home, filepath.FromSlash(f)), instructionsFixture)
	}

	first, err := DeployPresence(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range first {
		if o.Status != "injected" {
			t.Errorf("%s: want injected, got %s", o.Agent, o.Status)
		}
	}
	second, err := DeployPresence(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range second {
		if o.Status != "unchanged" {
			t.Errorf("%s: second run must be unchanged, got %s", o.Agent, o.Status)
		}
	}
	for _, f := range files {
		got := read(t, filepath.Join(home, filepath.FromSlash(f)))
		for _, needle := range []string{"user intro", "patterns content", "user outro", "curator", "vault-doctor", PresenceEndMarker} {
			if !strings.Contains(got, needle) {
				t.Errorf("%s lacks %q:\n%s", f, needle, got)
			}
		}
		if strings.Count(got, PresenceBeginPrefix) != 1 || strings.Count(got, "BEGIN HARNESS GENERATED") != 1 {
			t.Errorf("%s: exactly one presence region and one patterns region expected:\n%s", f, got)
		}
		if !strings.HasPrefix(got, "user intro") || !strings.HasSuffix(got, PresenceEndMarker+"\n") {
			t.Errorf("%s: appended region must follow the content and end with a newline:\n%q", f, got)
		}
	}
}

func TestInjectPresence_ReplacesAStaleRegionInPlace(t *testing.T) {
	home := t.TempDir()
	f := filepath.Join(home, "AGENTS.md")
	stale := "intro\n\n" + PresenceBeginPrefix + " (sha256:0000000000000000)" + presenceNote + "\nold roster\n" + PresenceEndMarker + "\n\noutro\n"
	write(t, f, stale)

	changed, err := InjectPresence(f, "new roster\n")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	got := read(t, f)
	if strings.Contains(got, "old roster") || !strings.Contains(got, "new roster") {
		t.Errorf("region not replaced:\n%s", got)
	}
	if !strings.HasPrefix(got, "intro\n\n") || !strings.HasSuffix(got, PresenceEndMarker+"\n\noutro\n") {
		t.Errorf("content around the region must be byte-identical:\n%q", got)
	}
	if state, _ := PresenceStatus(f, "new roster\n"); state != PresenceCurrent {
		t.Errorf("after injection the status is current, got %s", state)
	}
	if state, _ := PresenceStatus(f, "newer roster\n"); state != PresenceStale {
		t.Errorf("a changed roster reads as stale, got %s", state)
	}
}

// A CRLF instructions file (the Windows checkout copies them CRLF) stays CRLF,
// and carries the same sha its LF twin would.
func TestInjectPresence_HonoursCRLF(t *testing.T) {
	home := t.TempDir()
	f := filepath.Join(home, "CLAUDE.md")
	write(t, f, strings.ReplaceAll(instructionsFixture, "\n", "\r\n"))

	if _, err := InjectPresence(f, "roster\n"); err != nil {
		t.Fatal(err)
	}
	got := read(t, f)
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a CRLF file must not gain bare LF lines:\n%q", got)
	}
	if !strings.Contains(got, "(sha256:"+PresenceSHA("roster\n")+")") {
		t.Error("the sha must be the LF sha")
	}
	if state, _ := PresenceStatus(f, "roster\n"); state != PresenceCurrent {
		t.Errorf("CRLF region must read as current, got %s", state)
	}
}

func TestDeployPresence_AbsentTargetIsASkipAndEmptyRosterInjectsNothing(t *testing.T) {
	repo, home := presenceRepo(t), t.TempDir()
	writeRecord(t, repo, "curator", "---\nname: curator\ndescription: opencode only now.\nkind: invocable\nskills: [vault-doctor]\ntargets: [opencode]\n---\n")
	write(t, filepath.Join(home, ".claude", "CLAUDE.md"), instructionsFixture)
	// opencode's file is absent on purpose.

	out, err := DeployPresence(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"claude": "empty", "opencode": "absent", "pi": "empty", "copilot": "empty"}
	for _, o := range out {
		if o.Status != want[o.Agent] {
			t.Errorf("%s: want %s, got %s", o.Agent, want[o.Agent], o.Status)
		}
	}
	if strings.Contains(read(t, filepath.Join(home, ".claude", "CLAUDE.md")), PresenceBeginPrefix) {
		t.Error("an empty roster must inject no region")
	}
	if state, _ := PresenceStatus(filepath.Join(home, ".claude", "CLAUDE.md"), ""); state != PresenceMissing {
		t.Errorf("no region reads as missing, got %s", state)
	}
}

func TestLoadPresence_ManifestWithoutPresenceIsNoTargets(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "harness", "manifest.json"), `{"version":1,"agents":{"record_dir":"harness/agents"}}`)
	_, targets, err := LoadPresence(filepath.Join(repo, "harness", "manifest.json"))
	if err != nil || len(targets) != 0 {
		t.Errorf("want no targets and no error, got %v %v", targets, err)
	}
	write(t, filepath.Join(repo, "harness", "manifest.json"), `{"version":1,"agents":{"presence":[{"agent":"claude"}]}}`)
	if _, _, err := LoadPresence(filepath.Join(repo, "harness", "manifest.json")); err == nil {
		t.Error("an entry without a file must be rejected")
	}
}
