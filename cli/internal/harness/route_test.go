package harness

import (
	"path/filepath"
	"strings"
	"testing"
)

// repoModelMap loads the shipped routing registry. Like repoPersonas, this is
// deliberately the real file rather than a literal: the tiers a persona may
// declare are whatever the map declares, so a test against a hand-written map
// would keep passing after the two drifted apart.
func repoModelMap(t *testing.T) map[string]any {
	t.Helper()
	m, err := LoadModelMap(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("LoadModelMap: %v", err)
	}
	return m
}

func personaNamed(t *testing.T, name string) *Persona {
	t.Helper()
	for _, p := range repoPersonas(t) {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("no persona named %q in the roster", name)
	return nil
}

// TestResolveTierForPersonaReadsTheRecord is the first read of Persona.Model.
//
// Until this existed the field was written at persona.go:107 and consumed by
// nothing: every record declared a tier and every dispatch asked its caller for
// one instead. The assertion is against the SHIPPED records rather than a
// fixture, because the point of the field is that the record is the authority.
func TestResolveTierForPersonaReadsTheRecord(t *testing.T) {
	m := repoModelMap(t)

	for _, tc := range []struct{ persona, want string }{
		{"architect", "top"},
		{"curator", "top"},
		{"planner", "mid"},
		{"builder", "mid"},
		{"reviewer", "mid"},
		{"shipper", "mid"},
	} {
		got, err := ResolveTierForPersona(personaNamed(t, tc.persona), m)
		if err != nil {
			t.Errorf("%s: %v", tc.persona, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s resolved tier %q, want %q — the record is the authority, not this test",
				tc.persona, got, tc.want)
		}
	}
}

// TestResolveTierForPersonaRefusesRatherThanDefaulting is AC4.
//
// A default here would be the defect the gate's loader avoids by applying no
// default severity: a route nobody chose, taken silently. Both failures must
// also be USEFUL — an operator who gets "invalid tier" and has to go and look
// up which ones are valid is one who edits the record twice.
func TestResolveTierForPersonaRefusesRatherThanDefaulting(t *testing.T) {
	m := repoModelMap(t)

	for _, tc := range []struct {
		name    string
		persona *Persona
	}{
		{"no tier declared", &Persona{Name: "hollow", Model: ""}},
		{"whitespace only", &Persona{Name: "blank", Model: "   "}},
		{"a tier the map does not declare", &Persona{Name: "greedy", Model: "enormous"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveTierForPersona(tc.persona, m)
			if err == nil {
				t.Fatalf("resolved %q instead of refusing — a route nobody chose is the "+
					"failure this refusal exists to prevent", got)
			}
			if got != "" {
				t.Errorf("returned tier %q alongside an error; a refused resolution must "+
					"yield nothing a caller could accidentally use", got)
			}
			if !strings.Contains(err.Error(), tc.persona.Name) {
				t.Errorf("refusal does not name the persona, so the operator cannot tell "+
					"which record to edit: %v", err)
			}
			// The legal set must be quoted from the map, so the message stays
			// correct if a tier is ever added to it.
			for _, tier := range []string{"top", "mid", "low"} {
				if !strings.Contains(err.Error(), tier) {
					t.Errorf("refusal does not name the declared tier %q; it must list what "+
						"IS legal, not only that this was not: %v", tier, err)
				}
			}
		})
	}
}

// TestResolveTierForPersonaRefusesAMapWithNoChains guards the direction of the
// failure. An unreadable map is not an empty one (C15): resolving to some
// default when the registry cannot be read would route work with no registry at
// all, which is worse than not dispatching.
func TestResolveTierForPersonaRefusesAMapWithNoChains(t *testing.T) {
	got, err := ResolveTierForPersona(&Persona{Name: "planner", Model: "mid"}, map[string]any{})
	if err == nil {
		t.Fatalf("resolved %q from a map declaring no chains", got)
	}
}

// TestResolveOneDispatchesOrRefuses covers AC1, AC2 and AC3 against the shipped
// rules. The three outcomes are the whole contract of the join at dispatch time:
// exactly one persona runs, and the other two cases refuse DIFFERENTLY.
func TestResolveOneDispatchesOrRefuses(t *testing.T) {
	personas := repoPersonas(t)
	cfg := repoTriggers(t)

	resolve := func(prompt string) (*Persona, string, error) {
		s := Suggest(cfg.Triggers, prompt, nil)
		return ResolveOne(s, personas)
	}

	t.Run("one persona dispatches", func(t *testing.T) {
		// The exact text from AC1, so the criterion and the test cannot drift.
		p, pattern, err := resolve("open a ticket for the bitacora")
		if err != nil {
			t.Fatalf("AC1's task refused: %v", err)
		}
		if p.Name != "planner" {
			t.Errorf("resolved %q, want planner", p.Name)
		}
		if pattern == "" {
			t.Error("resolved with no pattern reported; a route that cannot be judged " +
				"is obeyed on its worst day as readily as its best")
		}
	})

	var ambiguous, unmatched error

	t.Run("two personas refuse", func(t *testing.T) {
		// spec-driven-development -> [planner, reviewer]. A dispatcher may not
		// pick one: HARNESS-110 made ambiguity a first-class output, and
		// determinism is sorted output rather than narrowing.
		p, _, err := resolve("write the spec and its acceptance criteria")
		if err == nil {
			t.Fatalf("dispatched %q for an ambiguous match instead of refusing", p.Name)
		}
		ambiguous = err
		for _, want := range []string{"planner", "reviewer"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("refusal omits candidate %q; naming both is what makes it "+
					"actionable rather than merely correct: %v", want, err)
			}
		}
	})

	t.Run("no persona refuses", func(t *testing.T) {
		p, _, err := resolve("zzzz nothing here matches any declared rule zzzz")
		if err == nil {
			t.Fatalf("dispatched %q for a prompt matching no rule", p.Name)
		}
		unmatched = err
	})

	// Asserted as a difference rather than as two fixed strings: the property
	// that matters is that an operator can tell "I must choose" from "there is
	// nothing to choose", and pinning the wording would be a chore on every
	// rephrasing while testing nothing more.
	if ambiguous != nil && unmatched != nil && ambiguous.Error() == unmatched.Error() {
		t.Errorf("an ambiguous match and no match refuse identically (%q) — "+
			"the two need opposite responses from whoever reads them", ambiguous)
	}
}
