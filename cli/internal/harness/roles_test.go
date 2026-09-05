package harness

import (
	"encoding/json"
	"os"
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

	// `pr-review-triage` is named by the `git-workflow` rule and, until shipper
	// adopted it, was declared by no persona at all — so the join read it, found
	// no owner and dropped it. The rule still resolved through a sibling skill,
	// which is exactly why nothing was red. Asserting the skill ALONE is what
	// distinguishes "the rule resolves" from "this skill has an owner"; only the
	// second is what changed. See #1499 for the guard that cannot tell them apart.
	if got := ResolveRoles(Suggestion{Skills: []string{"pr-review-triage"}}, personas); len(got) != 1 || got[0] != "shipper" {
		t.Errorf("pr-review-triage should resolve to [shipper], got %v", got)
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
	// The single-role case names the role, the pattern and the skills, and states
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

// TestManifestEmitsPromptHook is AC4: the suggester is bound FROM THE MANIFEST,
// so a deploy propagates it. Hand-writing it into settings.json would make it
// present on exactly one machine and absent everywhere a deploy runs.
//
// `emit_hooks` is not new — the gate and both mem hooks already ride it. This
// asserts the fourth entry exists, targets UserPromptSubmit, and invokes the
// stdin mode rather than a --prompt argument (AC5's injection surface).
func TestManifestEmitsPromptHook(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "harness", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var m struct {
		Agents struct {
			Bind []struct {
				Agent     string `json:"agent"`
				EmitHooks []struct {
					ID      string `json:"id"`
					Event   string `json:"event"`
					Command string `json:"command"`
					Timeout int    `json:"timeout"`
				} `json:"emit_hooks"`
			} `json:"bind"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	for _, target := range m.Agents.Bind {
		if target.Agent != "claude" {
			continue
		}
		for _, h := range target.EmitHooks {
			if h.ID != "suggest-role" {
				continue
			}
			if h.Event != "UserPromptSubmit" {
				t.Errorf("event = %q, want UserPromptSubmit", h.Event)
			}
			if !strings.Contains(h.Command, "--from-hook") {
				t.Errorf("command = %q, must use the stdin mode", h.Command)
			}
			if strings.Contains(h.Command, "--prompt") {
				t.Errorf("command = %q passes the prompt as an argument — that is the injection surface AC5 forbids", h.Command)
			}
			// A bounded timeout. This hook sits on the INTERACTIVE path: what a
			// hang delays is the user's own prompt, not a background tool call.
			// AC8 budgets 20ms, so any positive bound is enormous headroom and
			// the point is only that the damage is bounded at all.
			if h.Timeout <= 0 {
				t.Errorf("timeout = %d; an unbounded hook on UserPromptSubmit can stall the user's prompt", h.Timeout)
			}
			return
		}
		t.Fatal("claude bind target has no suggest-role hook — the suggester would never fire")
	}
	t.Fatal("no claude bind target in the manifest")
}

// TestResolveRolesExcludesNonInvocable closes the gap the independent review
// found by mutation: deleting the `kind: invocable` filter left every test
// green, because `hermes-nan`'s only skill is absent from triggers.json today.
// The filter was load-bearing for a future case and asserted by nothing, which
// makes it exactly the kind of code a later cleanup deletes as dead.
func TestResolveRolesExcludesNonInvocable(t *testing.T) {
	autonomous := &Persona{
		Name:   "hermes-nan",
		Kind:   "autonomous",
		Skills: []SkillBinding{{ID: "test-driven-development"}},
	}
	invocable := &Persona{
		Name:   "builder",
		Kind:   "invocable",
		Skills: []SkillBinding{{ID: "test-driven-development"}},
	}

	got := ResolveRoles(Suggestion{Skills: []string{"test-driven-development"}},
		[]*Persona{autonomous, invocable})
	if len(got) != 1 || got[0] != "builder" {
		t.Errorf("a kind:autonomous persona must never be suggested — a session cannot adopt it. got %v", got)
	}

	// And on its own it resolves to nobody rather than to itself.
	if only := ResolveRoles(Suggestion{Skills: []string{"test-driven-development"}},
		[]*Persona{autonomous}); len(only) != 0 {
		t.Errorf("only a non-invocable persona matched; want no suggestion, got %v", only)
	}
}

// TestPromptHookReachesSettingsByConsequence closes the review's third finding.
//
// AC4's own standard is "verify by consequence: deploy and observe — not by
// asserting the file contains a string", and the manifest-declaration test above
// is exactly the form the spec calls insufficient. This runs the real emission
// path — LoadBindTargets -> HookCommands -> MergeHooks — and asserts the hook
// lands in the settings document a deploy would write.
func TestPromptHookReachesSettingsByConsequence(t *testing.T) {
	targets, err := LoadBindTargets(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("LoadBindTargets: %v", err)
	}

	var claude *BindTarget
	for i := range targets {
		if targets[i].Agent == "claude" {
			claude = &targets[i]
			break
		}
	}
	if claude == nil {
		t.Fatal("no claude bind target")
	}

	cmds, err := claude.HookCommands("/opt/bin/dotf")
	if err != nil {
		t.Fatalf("HookCommands: %v", err)
	}

	merged, _, err := MergeHooks(map[string]any{}, cmds)
	if err != nil {
		t.Fatalf("MergeHooks: %v", err)
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	settings := string(raw)
	for _, want := range []string{"UserPromptSubmit", "/opt/bin/dotf harness suggest --from-hook"} {
		if !strings.Contains(settings, want) {
			t.Errorf("emitted settings lack %q — the hook would never fire.\n%s", want, settings)
		}
	}
	// The binary path is absolute in the emitted command for the reason #531
	// records: a harness runs a hook with whatever PATH it inherited.
	if strings.Contains(settings, `"dotf harness suggest`) {
		t.Error("the emitted command is not absolute; a hook that cannot find its binary fails silently")
	}
}
