package harness

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// repoPersonas loads the real roster. Fixtures are deliberately NOT hand-written
// literals: AC1 requires the join to consume LoadPersona, so the tests must break
// if the parse breaks. A literal map would keep passing while the parse rotted —
// which is the exact failure mode this spec exists to prevent.
func repoPersonas(t *testing.T) []*Persona {
	t.Helper()
	personas, err := LoadPersonas(filepath.Join("..", "..", "..", "harness", "agents"))
	if err != nil {
		t.Fatalf("LoadPersonas: %v", err)
	}
	if len(personas) == 0 {
		t.Fatal("no personas loaded")
	}
	return personas
}

func repoTriggers(t *testing.T) *TriggerConfig {
	t.Helper()
	cfg, err := LoadTriggers(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("LoadTriggers: %v", err)
	}
	return cfg
}

func TestResolveRoles(t *testing.T) {
	personas := repoPersonas(t)

	got := ResolveRoles(Suggestion{Skills: []string{"test-driven-development"}}, personas)
	if len(got) != 1 || got[0] != "builder" {
		t.Errorf("test-driven-development should resolve to [builder], got %v", got)
	}

	// Sorted output is the determinism contract: same input, same bytes.
	multi := ResolveRoles(Suggestion{Skills: []string{"terraform", "audit", "read-all-adrs"}}, personas)
	for i := 1; i < len(multi); i++ {
		if multi[i-1] > multi[i] {
			t.Errorf("roles are not sorted: %v", multi)
		}
	}
}

func TestResolveRolesAmbiguity(t *testing.T) {
	personas := repoPersonas(t)

	// Ambiguity is a first-class output, never ranked and never tie-broken.
	// Both known cases are asserted so a future "improvement" that narrows to a
	// single role fails two tests, not one.
	//
	// The cases are keyed by RULE and their skills are read from triggers.json,
	// because that is where ambiguity actually lives. `spec-driven-development`
	// is ambiguous only as a rule: its `spec` reaches planner and its
	// `adversarial-review` reaches reviewer, and neither skill is ambiguous
	// alone. Asserting a single skill tested a claim the design never made.
	cases := map[string][]string{
		"code-complexity-and-refactor": {"builder", "reviewer"},
		"spec-driven-development":      {"planner", "reviewer"},
	}
	byID := map[string][]string{}
	for _, rule := range repoTriggers(t).Triggers {
		byID[rule.ID] = rule.Skills
	}
	for ruleID, want := range cases {
		skills, ok := byID[ruleID]
		if !ok {
			t.Fatalf("rule %q is gone from triggers.json — the fixture is stale", ruleID)
		}
		got := ResolveRoles(Suggestion{Skills: skills}, personas)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s (skills %v): want %v, got %v", ruleID, skills, want, got)
		}
	}

	// A rule with no skills resolves to nobody, and empty is not an error.
	if got := ResolveRoles(Suggestion{Skills: nil}, personas); len(got) != 0 {
		t.Errorf("no skills should resolve to no roles, got %v", got)
	}
	if got := ResolveRoles(Suggestion{Skills: []string{"no-such-skill"}}, personas); len(got) != 0 {
		t.Errorf("an unknown skill should resolve to no roles, got %v", got)
	}
}

// TestRoleJoinDrift is AC3: the guard spans triggers.json and the persona roster,
// which nothing else does. It goes red on the two ways the join can collapse.
func TestRoleJoinDrift(t *testing.T) {
	personas := repoPersonas(t)
	cfg := repoTriggers(t)

	// (1) Every persona must contribute at least one skill. A reader that returns
	// an empty set — a form it does not understand — is indistinguishable from a
	// persona that legitimately declares nothing, and both silently shrink the
	// join. This is what made three ad-hoc parsers disagree on unchanged input.
	for _, p := range personas {
		if len(p.Skills) == 0 {
			t.Errorf("persona %q contributes no skills — the join would silently shrink", p.Name)
		}
	}

	// (2) The resolving-rule count is the measurement the whole design rests on.
	resolving := 0
	for _, rule := range cfg.Triggers {
		if len(ResolveRoles(Suggestion{Skills: rule.Skills}, personas)) > 0 {
			resolving++
		}
	}
	// A FLOOR, not an equality. An exact assertion goes red when a persona gains
	// a skill that lifts a 17th rule — good news — and the only available fix is
	// to bump the constant. A guard that gets edited every time it fires is a
	// guard the next session learns to disable. Coverage going UP is never the
	// failure this watches for.
	const minResolving = 16
	if resolving < minResolving {
		t.Errorf("resolving rules = %d, want at least %d of %d. A renamed skill, a persona "+
			"losing its roster, or a parse that stopped working will land here first",
			resolving, minResolving, len(cfg.Triggers))
	}
}

func TestFormatSuggestion(t *testing.T) {
	// The single-role case names the role, the rule and the skills, and states
	// what to do — the owner's chosen shape (see proposal.md, Decisions).
	out := FormatSuggestion([]string{"builder"}, "testing-standards",
		[]string{"test", "test-driven-development"})
	for _, want := range []string{"builder", "testing-standards", "test-driven-development"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should name %q, got:\n%s", want, out)
		}
	}

	// Ambiguity must read as two paths, not as indecision, and must never
	// present one role as the answer.
	amb := FormatSuggestion([]string{"builder", "reviewer"}, "code-complexity-and-refactor",
		[]string{"cyclomatic-complexity"})
	if !strings.Contains(amb, "builder") || !strings.Contains(amb, "reviewer") {
		t.Errorf("both roles must appear, got:\n%s", amb)
	}

	// Zero roles prints nothing: two of the 18 rules are pattern-only and have no
	// owner, and a suggestion naming nobody is pure noise on every prompt.
	if got := FormatSuggestion(nil, "shell-standards", nil); got != "" {
		t.Errorf("no roles should print nothing, got %q", got)
	}
}

// TestRoleJoinLatencyBudget is AC8. The join is charged to EVERY prompt, so a
// regression here is invisible without a stated budget.
func TestRoleJoinLatencyBudget(t *testing.T) {
	personas := repoPersonas(t)
	cfg := repoTriggers(t)

	start := time.Now()
	for i := 0; i < 100; i++ {
		s := Suggest(cfg.Triggers, "refactor this deeply nested docker helper and add tests", nil)
		ResolveRoles(s, personas)
	}
	if per := time.Since(start) / 100; per > 20*time.Millisecond {
		t.Errorf("match+join took %v per prompt, budget is 20ms", per)
	}
}
