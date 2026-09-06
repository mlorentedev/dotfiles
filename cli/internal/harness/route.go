package harness

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveTierForPersona and ResolveOne are the resolution half of HARNESS-120:
// turning a task into the persona that should run it and the tier that persona
// declares for itself. Both are pure — they take the loaded roster and the
// loaded map and read no file — so the command layer owns every I/O decision,
// including which root to read from.
//
// See specs/HARNESS-120/proposal.md.

// ResolveTierForPersona answers which tier a persona's own record puts it on.
//
// It is the first reader of Persona.Model. Until HARNESS-120 that field was
// written at persona.go:107 and consumed by nothing: every record declared a
// tier, `dotf agent run` asked its caller for one instead, and the two were
// never connected. A persona that says `model: top` now routes to the top
// chain because it said so.
//
// The legal tiers are whatever the map's `chains` block declares, never a
// constant here. A hardcoded top|mid|low would be a second place the tiers are
// true — the drift model-map.json exists to end (ADR-035 §4) — and it would go
// stale the day a tier is added. The same reasoning forbids a second persona
// parser at roles.go:27-31.
//
// An undeclared tier is refused rather than defaulted. Choosing one here would
// be the defect the gate's loader avoids by applying no default severity: a
// route nobody chose, taken silently, indistinguishable afterwards from a route
// someone did choose.
func ResolveTierForPersona(p *Persona, m map[string]any) (string, error) {
	if p == nil {
		return "", fmt.Errorf("no persona to resolve a tier for")
	}
	declared, err := declaredChainTiers(m)
	if err != nil {
		return "", err
	}

	tier := strings.TrimSpace(p.Model)
	if tier == "" {
		return "", fmt.Errorf(
			"persona %q declares no tier, so there is no chain to walk for it\n\n"+
				"Add one to %s:\n\n    model: mid\n\n"+
				"%s declares these: %s. This is refused rather than defaulted because a "+
				"tier chosen here is a route nobody picked, and afterwards it is "+
				"indistinguishable from one someone did",
			p.Name, p.Path, ModelMapFile, strings.Join(declared, ", "))
	}

	for _, d := range declared {
		if tier == d {
			return tier, nil
		}
	}
	return "", fmt.Errorf(
		"persona %q declares tier %q, which %s does not: it declares %s\n\n"+
			"Either fix `model:` in %s or add the tier to the map — but not by "+
			"guessing here, because a dispatch routed to a tier with no chain has "+
			"nowhere to go",
		p.Name, tier, ModelMapFile, strings.Join(declared, ", "), p.Path)
}

// declaredChainTiers lists the tiers the map can actually dispatch to, sorted.
//
// It reads `chains` and not `tiers`: ADR-035 splits the file into two cadences,
// and this is the run-time half. A persona routed to a tier that renders at
// compile time but has no chain would resolve happily and then fail with
// nowhere to go, which is a worse place to discover it.
func declaredChainTiers(m map[string]any) ([]string, error) {
	chains, ok := m["chains"].(map[string]any)
	if !ok || len(chains) == 0 {
		// C15: an unreadable registry is not an empty one. Refusing here is the
		// fail-closed direction — the alternative routes work with no registry
		// at all, which is worse than not dispatching.
		return nil, fmt.Errorf("%s declares no chains block, so no tier can be resolved", ModelMapFile)
	}
	out := make([]string, 0, len(chains))
	for tier := range chains {
		out = append(out, tier)
	}
	sort.Strings(out)
	return out, nil
}

// AmbiguousRoleError reports that a task's skills are owned by more than one
// persona. It carries the candidates rather than only a sentence, so a caller
// can render them without re-parsing the message.
//
// This is the shape HARNESS-110's decision takes at dispatch time. Advisory
// output may print two personas and let the session's own reasoning settle it;
// a dispatcher has to pick one process, and picking would mean ranking. So it
// refuses, and the refusal IS the deterministic behaviour — determinism here is
// sorted, complete output, never narrowing.
type AmbiguousRoleError struct {
	Candidates []string
	Pattern    string
}

func (e *AmbiguousRoleError) Error() string {
	return fmt.Sprintf(
		"this task resolves to %d personas and a dispatch runs one: %s\n\n"+
			"Choose with --role. They are not ranked, deliberately: each declares a "+
			"skill the task matched, and which one is right is a judgement about the "+
			"work rather than a property of the rules",
		len(e.Candidates), strings.Join(e.Candidates, ", "))
}

// NoRoleError reports that nothing in the task matched a rule that any persona
// owns. It is a distinct type from AmbiguousRoleError because the two need
// opposite responses: one is resolved by choosing, the other cannot be.
type NoRoleError struct {
	// Patterns are the rules that matched but own no persona, if any.
	// `shell-standards` and `powershell-ascii-only` are pattern-only rules with
	// no owner, so "matched something, owned by nobody" is a real state and
	// worth telling apart from "matched nothing at all".
	Patterns []string
}

func (e *NoRoleError) Error() string {
	if len(e.Patterns) > 0 {
		return fmt.Sprintf(
			"this task matched %s, which no persona owns, so there is nobody to dispatch\n\n"+
				"Name one with --role if the work belongs to a phase anyway",
			strings.Join(e.Patterns, ", "))
	}
	return "this task matched no trigger rule, so no persona can be derived from it\n\n" +
		"Name one with --role, or rephrase the task using the vocabulary the rules " +
		"declare — the match is on keywords, not on meaning"
}

// ResolveOne narrows a suggestion to the single persona a dispatch can run,
// and returns the pattern that caused the match so the route can be judged.
//
// The three outcomes are the whole contract: exactly one persona dispatches,
// and the two failures are DIFFERENT errors because an operator reading them
// needs to do different things. Collapsing them into one "could not resolve"
// would be the same defect as the gate's single allow reason, which read
// identically whether enforcement was satisfied or simply switched off.
//
// The join itself is not recomputed here. ResolveRoles owns it, and a second
// implementation is how a solved bug comes back.
func ResolveOne(s Suggestion, personas []*Persona) (*Persona, string, error) {
	roles := ResolveRoles(s, personas)
	pattern := strings.Join(s.Patterns, ", ")

	switch len(roles) {
	case 0:
		return nil, "", &NoRoleError{Patterns: s.Patterns}
	case 1:
		for _, p := range personas {
			if p != nil && p.Name == roles[0] {
				return p, pattern, nil
			}
		}
		// ResolveRoles only ever names a persona it was given, so this is
		// unreachable by construction. It returns an error rather than nil so a
		// future caller that breaks the invariant fails loudly instead of
		// dereferencing nothing.
		return nil, "", fmt.Errorf("resolved role %q is not in the roster it came from", roles[0])
	default:
		return nil, "", &AmbiguousRoleError{Candidates: roles, Pattern: pattern}
	}
}
