package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePersona(t *testing.T, dir, name, frontmatter string) string {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "AGENT.md")
	body := "---\n" + strings.TrimPrefix(frontmatter, "\n") + "\n---\n\n# " + name + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// The shape this spec introduces: severity declared per skill.
func TestPersonaParsesDeclaredSeverity(t *testing.T) {
	p, err := LoadPersona(writePersona(t, t.TempDir(), "reviewer", `
name: reviewer
kind: invocable
model: mid
skills:
  - id: adversarial-review
    enforce: block
  - id: cyclomatic-complexity
    enforce: warn`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := p.Blocking(); len(got) != 1 || got[0] != "adversarial-review" {
		t.Errorf("Blocking() = %v, want [adversarial-review]", got)
	}
	if got := p.UnmigratedSkills(); len(got) != 0 {
		t.Errorf("nothing should be unmigrated here, got %v", got)
	}
}

// The legacy flat form still parses, and every entry lands as EnforceUnset —
// NOT as a default. A default would either make the gate silently inert
// (`warn`) or turn every existing skill into a hard block overnight.
func TestPersonaLegacyFlatFormCarriesNoSeverity(t *testing.T) {
	p, err := LoadPersona(writePersona(t, t.TempDir(), "builder", `
name: builder
kind: invocable
model: mid
skills: [test, systematic-debugging]`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(p.Skills) != 2 {
		t.Fatalf("want 2 skills, got %d", len(p.Skills))
	}
	if got := p.Blocking(); len(got) != 0 {
		t.Errorf("an unmigrated skill must never block, got %v", got)
	}
	if got := p.UnmigratedSkills(); len(got) != 2 {
		t.Errorf("both skills should be reported unmigrated, got %v", got)
	}
}

// AC7 — the load-bearing one. A `skills:` key the parser cannot read must be a
// loud failure, never an empty list.
//
// This is not hypothetical: `specs/HARNESS-046/check-roster-consistency.py`
// matches `^skills:\s*\[(.*?)\]` and falls back to `[]`, so under the block form
// it returns "no skills" in silence and the roster drift guard stops guarding.
// This test pins the opposite behaviour here.
func TestPersonaRefusesToReadSkillsAsEmpty(t *testing.T) {
	for _, tc := range []struct{ name, fm, want string }{
		{"skills is a scalar", "name: a\nkind: invocable\nskills: audit", "want a list"},
		{"skills is a map", "name: a\nkind: invocable\nskills:\n  audit: block", "want a list"},
		{"mapping without enforce", "name: a\nkind: invocable\nskills:\n  - id: audit", "declares no `enforce`"},
		{"mapping without id", "name: a\nkind: invocable\nskills:\n  - enforce: block", "has no `id`"},
		{"unknown severity", "name: a\nkind: invocable\nskills:\n  - id: audit\n    enforce: maybe", "want block or warn"},
		{"empty string entry", "name: a\nkind: invocable\nskills: ['']", "empty string"},
		{"entry is a number", "name: a\nkind: invocable\nskills: [7]", "want a string or an"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadPersona(writePersona(t, t.TempDir(), "a", tc.fm))
			if err == nil {
				t.Fatalf("%s loaded clean — a skills key that cannot be read must never resolve to no skills", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

// An absent `skills:` is legitimately no skills; only an unreadable one is an
// error. Without this the previous test's rule would forbid a valid record.
func TestPersonaAbsentSkillsIsNotAnError(t *testing.T) {
	p, err := LoadPersona(writePersona(t, t.TempDir(), "steward", "name: steward\nkind: catalog"))
	if err != nil {
		t.Fatalf("a record with no skills must load: %v", err)
	}
	if len(p.Skills) != 0 {
		t.Errorf("want no skills, got %v", p.Skills)
	}
}

func TestPersonaMalformedFrontmatterIsLoud(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "broken")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, body, want string }{
		{"no fence", "# just a heading\n", "no frontmatter fence"},
		{"unclosed", "---\nname: a\n", "not closed"},
		{"invalid yaml", "---\nname: [a\n---\n", "not valid YAML"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(d, "AGENT.md")
			if err := os.WriteFile(p, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadPersona(p); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want an error mentioning %q, got %v", tc.want, err)
			}
		})
	}
}

// An absent targets list means EVERY harness. Getting this backwards scopes a
// persona to one harness and fails it against all the others — a false positive
// on correct data.
func TestPersonaTargetsDefaultToEveryHarness(t *testing.T) {
	all, err := LoadPersona(writePersona(t, t.TempDir(), "a", "name: a\nkind: invocable"))
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{"claude", "pi", "opencode", "anything"} {
		if !all.AppliesTo(h) {
			t.Errorf("no targets must mean every harness, but %s was excluded", h)
		}
	}
	scoped, err := LoadPersona(writePersona(t, t.TempDir(), "b", "name: b\nkind: invocable\ntargets: [pi]"))
	if err != nil {
		t.Fatal(err)
	}
	if !scoped.AppliesTo("pi") || scoped.AppliesTo("claude") {
		t.Error("an explicit targets list must scope")
	}
}

// Effect, not shape: the real records in this repository must load, and the
// legacy form they still carry must be reported rather than silently defaulted.
func TestRealPersonaRecordsLoadAndReportTheirMigrationState(t *testing.T) {
	dir := filepath.Join(repoRootForTest(t), "harness", "agents")
	personas, err := LoadPersonas(dir)
	if err != nil {
		t.Fatalf("the shipped persona records do not load: %v", err)
	}
	if len(personas) < 5 {
		t.Fatalf("expected the full roster, got %d", len(personas))
	}
	unmigrated := 0
	for _, p := range personas {
		if p.Name == "" {
			t.Errorf("%s declares no name", p.Path)
		}
		unmigrated += len(p.UnmigratedSkills())
	}
	// Recorded rather than asserted to a number: this count is the migration's
	// progress, and pinning it would make the test a chore on every persona edit.
	t.Logf("%d personas loaded, %d skills still in the legacy flat form", len(personas), unmigrated)
}

// The MIXED list — mapping entries beside bare strings — is what makes a partial
// migration expressible, and nothing covered it until builder used it.
//
// It matters because the alternative shapes are both wrong. Gating all of a
// persona's skills means naming situational ones (debug-hardware, mcp-builder)
// on every call, which teaches the reader to scroll past `[gate]`; gating none
// leaves the persona inert while every check reports it as wired. The mixed form
// is the only one that says "these two are obligations, those seven are a menu"
// — and it is a real parse path, not a formatting preference, so it gets a test.
func TestPersonaMixedFormGatesOnlyTheMappingEntries(t *testing.T) {
	p, err := LoadPersona(writePersona(t, t.TempDir(), "builder", `
name: builder
kind: invocable
model: mid
skills:
  - id: test-driven-development
    enforce: warn
  - id: test
    enforce: warn
  - golang-pro
  - debug-hardware`))
	if err != nil {
		t.Fatalf("a mixed skills list must load: %v", err)
	}

	if len(p.Skills) != 4 {
		t.Fatalf("all four entries must survive, got %d: %v", len(p.Skills), p.Skills)
	}

	gated := map[string]Enforcement{}
	for _, s := range p.Skills {
		if s.Enforce != EnforceUnset {
			gated[s.ID] = s.Enforce
		}
	}
	if len(gated) != 2 || gated["test"] != EnforceWarn || gated["test-driven-development"] != EnforceWarn {
		t.Errorf("gated = %v, want exactly test and test-driven-development at warn", gated)
	}

	// The bare strings are DECLARED and ungated, and both halves are the point:
	// UnmigratedSkills is what keeps them visible instead of silently absent.
	unmigrated := map[string]bool{}
	for _, s := range p.UnmigratedSkills() {
		unmigrated[s] = true
	}
	if len(unmigrated) != 2 || !unmigrated["golang-pro"] || !unmigrated["debug-hardware"] {
		t.Errorf("UnmigratedSkills() = %v, want [golang-pro debug-hardware]", p.UnmigratedSkills())
	}
	if got := p.Blocking(); len(got) != 0 {
		t.Errorf("nothing is block here, got %v", got)
	}
}

// The migrations that have actually happened, asserted by name.
//
// TestRealPersonaRecordsLoadAndReportTheirMigrationState deliberately only LOGS
// the unmigrated count, so a persona silently reverting to the flat form is
// invisible to it — and that is not hypothetical: `compile-harness.sh --refresh`
// regenerates these records from the vault, so a checkout whose vault half is
// stale rewrites a migrated record back to inline form and the gate goes quiet
// with nothing red. Measured on 2026-09-05, on a different record.
//
// A named list rather than a count: each migration is a decision, so each costs
// one line here, and adding an eighth persona never breaks this test.
func TestMigratedPersonasStillGateSomething(t *testing.T) {
	personas, err := LoadPersonas(filepath.Join(repoRootForTest(t), "harness", "agents"))
	if err != nil {
		t.Fatalf("the shipped persona records do not load: %v", err)
	}
	byName := map[string]*Persona{}
	for _, p := range personas {
		byName[p.Name] = p
	}

	for _, name := range []string{"reviewer", "builder"} {
		p, ok := byName[name]
		if !ok {
			t.Errorf("%s is missing from the roster", name)
			continue
		}
		gated := 0
		for _, s := range p.Skills {
			if s.Enforce != EnforceUnset {
				gated++
			}
		}
		if gated == 0 {
			t.Errorf("%s has been migrated but now gates nothing — reverted to the flat form?", name)
		}
	}
}
