package harness

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveRoles and FormatSuggestion implement HARNESS-110: the role a prompt
// implies is DERIVED from `trigger.skills ∩ persona.skills`, never declared.
// See specs/HARNESS-110/proposal.md.

// ResolveRoles returns the persona names whose skill rosters intersect the
// suggestion's skills, sorted, with no I/O.
//
// Ambiguity is a first-class result and is returned in full: two rules resolve
// to two personas each (`code-complexity-and-refactor` → builder+reviewer,
// `spec-driven-development` → planner+reviewer). Do not rank and do not
// tie-break — in advisory mode the honest output is "this is builder-or-reviewer
// work", and the session's own reasoning resolves it. Determinism comes from
// sorted output, not from narrowing.
//
// A suggestion carrying no skills returns an empty slice, and empty is not an
// error: `shell-standards` and `powershell-ascii-only` are pattern-only rules
// with no owner, which is a correct answer rather than a gap.
//
// Personas MUST be supplied by LoadPersona/LoadPersonas. Do not re-parse
// AGENT.md here: the correct parse handles both `skills:` forms, is documented
// at persona.go:71-80, and fails loud under C15. A second reader is how a solved
// bug comes back — `check-roster-consistency.py` had to be repaired once for
// exactly that.
func ResolveRoles(s Suggestion, personas []*Persona) []string {
	if len(s.Skills) == 0 || len(personas) == 0 {
		return nil
	}

	wanted := make(map[string]struct{}, len(s.Skills))
	for _, skill := range s.Skills {
		wanted[skill] = struct{}{}
	}

	roles := make([]string, 0, len(personas))
	for _, p := range personas {
		if p == nil {
			continue
		}
		// Only invocable personas can be suggested. `hermes-nan` is
		// `kind: autonomous` — an externally scheduled steward, not something a
		// session can adopt. Today its only skill (`agent-lifecycle`) is absent
		// from triggers.json so it never surfaces, which is precisely why this
		// filter goes in now: the bug would appear the day someone adds that
		// skill to a rule, far from any change that looks related.
		if p.Kind != "invocable" {
			continue
		}
		for _, binding := range p.Skills {
			if _, ok := wanted[binding.ID]; ok {
				roles = append(roles, p.Name)
				break
			}
		}
	}

	sort.Strings(roles)
	return roles
}

// FormatSuggestion renders what the UserPromptSubmit hook writes to stdout, which
// Claude Code adds verbatim as context the session can see and act on.
//
// Shape chosen by the owner (proposal.md, Decisions): role, the rule and skills
// that caused the match, and the action to consider. The derivation is shown so a
// session can dismiss a bad match instead of obeying it — a suggestion that
// cannot be judged is obeyed on its worst day as readily as its best.
//
// Zero roles prints nothing. Two of the 18 rules are pattern-only and own no
// persona; a suggestion naming nobody would be pure noise charged to every
// prompt.
func FormatSuggestion(roles []string, rule string, skills []string) string {
	if len(roles) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[persona] %s", strings.Join(roles, " | "))
	if rule != "" {
		fmt.Fprintf(&b, "  ← rule: %s", rule)
	}
	b.WriteString("\n")

	if len(skills) > 0 {
		fmt.Fprintf(&b, "  skills: %s\n", strings.Join(skills, ", "))
	}

	// The ambiguous case must read as two live paths, never as indecision, and
	// must never present one role as the answer.
	if len(roles) > 1 {
		b.WriteString("  → each of these declares the matched skill; the work decides which\n")
	} else if entry := entrySkill(skills); entry != "" {
		fmt.Fprintf(&b, "  → consider adopting `%s` and invoking %s\n", roles[0], entry)
	} else {
		fmt.Fprintf(&b, "  → consider adopting `%s`\n", roles[0])
	}

	return b.String()
}

// entrySkill picks which of the matched skills to name as the entry point.
//
// Naming skills[0] would name whichever sorts first alphabetically, which for
// `testing-standards` is `test` — the prerequisite — rather than
// `test-driven-development`, the composite that pulls it in. Prefer a skill whose
// declared dependency closure covers another matched skill, so the suggestion
// names the thing worth invoking rather than its ingredient.
func entrySkill(skills []string) string {
	if len(skills) == 0 {
		return ""
	}
	matched := make(map[string]struct{}, len(skills))
	for _, s := range skills {
		matched[s] = struct{}{}
	}
	for _, s := range skills {
		for _, dep := range DefaultSkillDependencies[s] {
			if _, ok := matched[dep]; ok {
				return s
			}
		}
	}
	return skills[0]
}
