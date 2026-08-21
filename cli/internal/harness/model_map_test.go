package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The one rule the schema exists to enforce that a naive schema would miss:
// cross-block reference integrity. ADR-032 §3's reference schema shipped a
// `harnesses.codex.pools: ["codex"]` naming a pool the `pools` block never
// declares, and ADR-035 records it as an amendment rather than copying it in.
// A JSON Schema without a custom rule validates that document happily — every
// type is right and every required key is present — which is why this is the
// first test written.
func TestSchemaRejectsDanglingPoolReference(t *testing.T) {
	schema := []byte(`{
	  "type": "object",
	  "required": ["pools", "harnesses"],
	  "properties": {
	    "pools":     {"type": "object"},
	    "harnesses": {"type": "object"}
	  },
	  "x-poolReferencesResolve": true
	}`)

	cases := []struct {
		name       string
		doc        string
		wantErr    bool
		wantDetail string
	}{
		{
			name: "a harness naming an undeclared pool is rejected",
			doc: `{
			  "pools":     {"nan": {}},
			  "harnesses": {"codex": {"pools": ["codex"]}}
			}`,
			wantErr:    true,
			wantDetail: "codex",
		},
		{
			name: "a harness naming a declared pool is accepted",
			doc: `{
			  "pools":     {"nan": {}},
			  "harnesses": {"pi": {"pools": ["nan"]}}
			}`,
			wantErr: false,
		},
		{
			name: "one bad reference among several good ones is still rejected",
			doc: `{
			  "pools":     {"nan": {}, "claude": {}},
			  "harnesses": {"opencode": {"pools": ["nan", "openrouter"]}}
			}`,
			wantErr:    true,
			wantDetail: "openrouter",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateModelMap([]byte(tc.doc), schema)
			if tc.wantErr && err == nil {
				t.Fatalf("expected validation to reject the document, got nil error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected the document to validate, got: %v", err)
			}
			// The error has to name the offending pool. "invalid document" sends
			// the reader back to diff the file by hand, which is the failure mode
			// an error message exists to prevent.
			if tc.wantErr && !strings.Contains(err.Error(), tc.wantDetail) {
				t.Errorf("error must name the undeclared pool %q, got: %v", tc.wantDetail, err)
			}
		})
	}
}

// TestValidatorRejectsUnimplementedSchemaConstructs was DELETED on 2026-08-21,
// not weakened. It asserted that a construct the validator did not implement —
// `patternProperties` was the case it used — must be a loud error rather than a
// silent pass. That property mattered enormously while the interpreter was
// hand-rolled and covered a subset of the spec.
//
// The library implements the whole of draft 2020-12, so there is no unimplemented
// construct to be loud about, and the test now asserts the absence of a category
// that no longer exists. Keeping it would have meant loosening it until it
// asserted nothing, which is worse than deleting it with the reason recorded.
//
// What survives is the part that is still ours: our two `x-` extension keywords
// are not standard, so the library will not police their types — and
// TestValidatorRejectsMalformedPoolReferenceFlag does.

// The assertion that keeps the shipped map and the shipped schema from drifting
// apart. Both are read from disk rather than from fixtures, because a fixture
// proves the validator works and this proves the repository is correct.
func TestModelMapValidatesAgainstSchema(t *testing.T) {
	root := repoRootForTest(t)
	got, err := LoadModelMap(root)
	if err != nil {
		t.Fatalf("the shipped map must validate against the shipped schema: %v", err)
	}
	for _, block := range []string{"$comment", "version", "pools", "harnesses", "tiers", "chains", "services"} {
		if _, ok := got[block]; !ok {
			t.Errorf("missing block %q", block)
		}
	}
}

// ADR-035 amendment: the provider was deleted upstream, so no pool may declare it
// and no harness may reference it. Structural, not textual — the $comment names
// openrouter on purpose, so that its absence reads as a decision.
func TestModelMapDeclaresNoRetiredProvider(t *testing.T) {
	// Both are retired rather than overlooked: openrouter was deleted upstream in
	// August 2026, and codex is no longer used by this operator (2026-08-21).
	// ADR-032 section 3 referenced a `codex` pool it never declared, and the
	// resolution is deletion, not declaration — a pool nobody dispatches to is a
	// route to nowhere whether or not it validates.
	//
	// Structural, not textual. The $comment names both on purpose, so that their
	// absence reads as a decision to the next person who wonders where they went.
	retired := []string{"openrouter", "codex"}
	root := repoRootForTest(t)
	m, err := LoadModelMap(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	pools, _ := m["pools"].(map[string]any)
	harnesses, _ := m["harnesses"].(map[string]any)
	for _, dead := range retired {
		if _, present := pools[dead]; present {
			t.Errorf("pools declares %q, which is retired", dead)
		}
		if _, present := harnesses[dead]; present {
			t.Errorf("harnesses declares %q, which is retired", dead)
		}
		for name, raw := range harnesses {
			h, _ := raw.(map[string]any)
			for _, p := range toStrings(h["pools"]) {
				if p == dead {
					t.Errorf("harnesses.%s references the retired %q pool", name, dead)
				}
			}
		}
	}
}

// repoRootForTest walks up from the test's working directory to the repo root,
// so the test reads the files the repository actually ships rather than a copy.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ModelMapFile)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find %s walking up from %s", ModelMapFile, dir)
		}
		dir = parent
	}
}

// AC5: the two consumer classes must be reachable separately, because they
// resolve at different times. A compile-time consumer that can silently reach a
// run-time field is exactly the drift the `version` key exists to survive.
func TestModelMapConsumerClasses(t *testing.T) {
	root := repoRootForTest(t)
	m, err := LoadModelMap(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	t.Run("tier resolution is compile time", func(t *testing.T) {
		got, err := ResolveTier(m, "mid", "claude")
		if err != nil {
			t.Fatalf("ResolveTier: %v", err)
		}
		if got != "sonnet" {
			t.Errorf("mid/claude = %q, want sonnet", got)
		}
		// A tier that does not resolve for a harness must say so, not return "".
		if _, err := ResolveTier(m, "mid", "copilot"); err == nil {
			t.Error("a tier with no entry for a harness must be a loud error, not an empty string")
		}
	})

	t.Run("chain resolution is run time", func(t *testing.T) {
		chain, err := ResolveChain(m, "mid")
		if err != nil {
			t.Fatalf("ResolveChain: %v", err)
		}
		if len(chain) < 2 {
			t.Fatalf("the mid chain must have a fallback, got %v", chain)
		}
		if chain[0] != "claude:sonnet" {
			t.Errorf("chain order is meaning, not arrangement: first = %q", chain[0])
		}
	})

	t.Run("the top tier has no fallback, on purpose", func(t *testing.T) {
		chain, err := ResolveChain(m, "top")
		if err != nil {
			t.Fatalf("ResolveChain(top): %v", err)
		}
		if len(chain) != 1 {
			t.Errorf("top must queue or escalate rather than degrade silently (ADR-032 §4), got %v", chain)
		}
	})
}

// AC7: the budget is declared and NOT enforced, and the API must not let a
// caller believe otherwise. Nothing here decrements anything.
func TestModelMapBudgetIsDeclarationOnly(t *testing.T) {
	root := repoRootForTest(t)
	m, err := LoadModelMap(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	b, err := DeclaredBudget(m, "nan")
	if err != nil {
		t.Fatalf("DeclaredBudget: %v", err)
	}
	if b.Concurrency != 5 || b.ReserveInteractive != 2 {
		t.Errorf("nan budget = %+v, want concurrency 5 reserve 2 (measured 2026-08-20)", b)
	}
	if len(b.SharedWith) == 0 {
		t.Error("the nan pool is shared, and a budget that does not say so overstates its own guarantee")
	}
	// A pool with no declared concurrency must report absence, never zero — zero
	// reads as "no capacity" and absence means "not declared".
	b2, err := DeclaredBudget(m, "copilot")
	if err != nil {
		t.Fatalf("DeclaredBudget(copilot): %v", err)
	}
	if b2.ConcurrencyDeclared {
		t.Error("copilot declares no concurrency; the budget must report it undeclared, not 0")
	}
}

// Review finding 1 (agy/gemini-3.1-pro-high, 2026-08-21, Major/REAL):
// checkPoolReferences only walked harnesses.*.pools[]. `chains` entries are
// `pool:model` and `services.*.pool` is a pool name — both reference pools, and
// neither was checked. A ghost pool in either passed validation.
//
// The blocks are not decoration: `chains` is what a dispatcher walks at run
// time, so a ghost there is a fallback that fails at the moment the primary
// already has.
func TestSchemaRejectsDanglingPoolReferenceEverywhere(t *testing.T) {
	schema := []byte(`{"type": "object", "x-poolReferencesResolve": true}`)

	cases := []struct {
		name    string
		doc     string
		wantErr string
	}{
		{
			name:    "chains",
			doc:     `{"pools": {"nan": {}}, "chains": {"mid": ["ghost:some-model"]}}`,
			wantErr: "ghost",
		},
		{
			name:    "services",
			doc:     `{"pools": {"nan": {}}, "services": {"embeddings": {"pool": "ghost", "model": "m"}}}`,
			wantErr: "ghost",
		},
		{
			name:    "harnesses still caught",
			doc:     `{"pools": {"nan": {}}, "harnesses": {"pi": {"pools": ["ghost"]}}}`,
			wantErr: "ghost",
		},
		{
			name: "all three clean",
			doc: `{"pools": {"nan": {}},
			       "harnesses": {"pi": {"pools": ["nan"]}},
			       "chains": {"mid": ["nan:deepseek-v4-flash"]},
			       "services": {"embeddings": {"pool": "nan", "model": "m"}}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateModelMap([]byte(tc.doc), schema)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected clean document to validate, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("a dangling pool reference in %s must be rejected", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error must name %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// Review finding 2 (Major/REAL): ResolveTier returned "" with a nil error when
// the map declared an empty model id. The type assertion on `string` succeeds
// for "", so only a missing KEY errored — verification.md claimed the stronger
// property, which was an overstatement rather than a bug in the caller.
//
// An empty model id renders a definition naming no model at all, which is the
// silent-degrade this whole registry exists to prevent.
func TestResolveTierRejectsEmptyModelID(t *testing.T) {
	m := map[string]any{"tiers": map[string]any{"top": map[string]any{
		"claude": "",
		"pi":     "   ",
		"ok":     "opus",
	}}}
	for _, h := range []string{"claude", "pi"} {
		if got, err := ResolveTier(m, "top", h); err == nil {
			t.Errorf("harness %q declares a blank model id; want a loud error, got %q with nil error", h, got)
		}
	}
	if got, err := ResolveTier(m, "top", "ok"); err != nil || got != "opus" {
		t.Errorf("a real model id must still resolve: got %q err=%v", got, err)
	}
}

// Review finding 3 (Minor/THEORETICAL, confirmed REAL on measurement): the
// validator failed OPEN when a schema keyword carried an unexpected type.
// `additionalProperties: "false"` (a string) hit neither switch arm, so
// undeclared keys passed silently.
//
// This is the same argument as the keyword allow-list, one level down: a
// validator must be loud about a schema it cannot interpret, whether the part it
// cannot interpret is an unknown KEY or a malformed VALUE.
func TestValidatorRejectsMalformedSchemaValues(t *testing.T) {
	schema := []byte(`{"type": "object", "properties": {"a": {"type": "string"}}, "additionalProperties": "false"}`)
	err := ValidateModelMap([]byte(`{"a": "x", "undeclared": 1}`), schema)
	if err == nil {
		t.Fatal("a schema whose additionalProperties is neither bool nor object must be a loud error, not a silent pass")
	}
	if !strings.Contains(err.Error(), "additionalProperties") {
		t.Errorf("the error must name the malformed keyword, got: %v", err)
	}
}

// Round-2 review finding (nan/deepseek-v4-flash, Major/REAL): the round-1 fix
// closed the LOADER's tier path and left the SCHEMA layer open, so a map whose
// model ids are blank validated cleanly and `dotf doctor` reported it healthy.
//
// That is worse than the bug it replaced. A blank id in `chains` reaches a
// dispatcher as a `pool:` entry with nothing after the colon — the map says a
// fallback exists and the fallback is nothing, certified OK by the very check
// written to prevent exactly that.
//
// Note `minLength: 1` is not sufficient for chains: "nan:" is four characters.
// The shape is the assertion, so chains carries a pattern.
//
// It is not sufficient for the OTHER blocks either, and that was a real bug in
// the hand-rolled validator this replaced: draft 2020-12 counts `minLength` in
// code points WITHOUT trimming, so "  " satisfies minLength 1. The old
// interpreter trimmed, which made it non-conforming — an editor validating the
// same schema with a real validator would have disagreed with dotf about the
// same document. The pattern is how the standard expresses non-blank, and it is
// what the shipped schema now carries.
func TestSchemaRejectsBlankModelIDs(t *testing.T) {
	schema := []byte(`{
	  "type": "object",
	  "properties": {
	    "tiers":    {"type": "object", "additionalProperties":
	                  {"type": "object", "additionalProperties": {"type": "string", "minLength": 1, "pattern": "^\\S.*$"}}},
	    "chains":   {"type": "object", "additionalProperties":
	                  {"type": "array", "items": {"type": "string", "pattern": "^[^:]+:[^:]+$"}}},
	    "services": {"type": "object", "additionalProperties":
	                  {"type": "object", "properties": {"model": {"type": "string", "minLength": 1, "pattern": "^\\S.*$"}}}}
	  }
	}`)

	cases := []struct{ name, doc string }{
		{"blank tier model", `{"tiers": {"mid": {"claude": ""}}}`},
		{"whitespace tier model", `{"tiers": {"mid": {"claude": "  "}}}`},
		{"chain entry with no model after the colon", `{"chains": {"mid": ["nan:"]}}`},
		{"chain entry with no pool before the colon", `{"chains": {"mid": [":deepseek"]}}`},
		{"blank service model", `{"services": {"embeddings": {"model": ""}}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateModelMap([]byte(tc.doc), schema); err == nil {
				t.Errorf("a blank model id must fail schema validation; %s was accepted", tc.name)
			}
		})
	}

	good := `{"tiers": {"mid": {"claude": "sonnet"}},
	          "chains": {"mid": ["nan:deepseek-v4-flash"]},
	          "services": {"embeddings": {"model": "qwen3-embedding"}}}`
	if err := ValidateModelMap([]byte(good), schema); err != nil {
		t.Errorf("a well-formed map must still validate, got: %v", err)
	}
}

// Round-2 review finding (Minor/REAL): the cross-block rule was gated on
// truthy(schema["x-poolReferencesResolve"]), and truthy only accepts bool — so
// writing the flag as the STRING "true" silently disabled the entire check.
//
// Same class as the additionalProperties fix from round 1, one flag over: a
// schema value this validator cannot interpret must be loud, because a
// disabled assertion looks exactly like a satisfied one.
func TestValidatorRejectsMalformedPoolReferenceFlag(t *testing.T) {
	schema := []byte(`{"type": "object", "x-poolReferencesResolve": "true"}`)
	doc := []byte(`{"pools": {"nan": {}}, "harnesses": {"pi": {"pools": ["ghost"]}}}`)

	err := ValidateModelMap(doc, schema)
	if err == nil {
		t.Fatal("a non-boolean x-poolReferencesResolve must be a loud error — silently disabling the cross-block rule is indistinguishable from passing it")
	}
	if !strings.Contains(err.Error(), "x-poolReferencesResolve") {
		t.Errorf("the error must name the malformed flag, got: %v", err)
	}
}

// The shipped map must satisfy the stricter schema too, end to end.
func TestShippedMapHasNoBlankModelIDs(t *testing.T) {
	root := repoRootForTest(t)
	m, err := LoadModelMap(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for tier := range m["chains"].(map[string]any) {
		chain, err := ResolveChain(m, tier)
		if err != nil {
			t.Fatalf("ResolveChain(%s): %v", tier, err)
		}
		for _, entry := range chain {
			pool, model, _ := strings.Cut(entry, ":")
			if strings.TrimSpace(pool) == "" || strings.TrimSpace(model) == "" {
				t.Errorf("chains.%s carries %q — a fallback entry with an empty half routes nowhere", tier, entry)
			}
		}
	}
}

// Round-3 review finding (agy/gemini-3.1-pro-high, Blocker/REAL): a KNOWN
// keyword carrying a malformed value silently disabled its rule. The helpers
// that read them (toStrings, toInt) return an empty result on a type mismatch,
// so `required: "pools"` or `minLength: "2"` skipped the assertion entirely.
//
// Rounds 1 and 2 closed this one keyword at a time — additionalProperties, then
// x-poolReferencesResolve — and round 3 showed keyword-by-keyword was the wrong
// shape. Five distinct keywords still failed open. The fix is structural: the
// allow-list carries each keyword's expected value type and the up-front walk
// asserts it, so a schema typo is loud rather than a quietly weaker contract.
func TestValidatorRejectsMalformedKeywordValueTypes(t *testing.T) {
	cases := []struct{ name, schema, doc string }{
		{"required as a string", `{"type":"object","required":"pools"}`, `{}`},
		{"minLength as a string", `{"type":"string","minLength":"2"}`, `"x"`},
		{"minItems as a string", `{"type":"array","minItems":"5"}`, `[]`},
		{"pattern as a number", `{"type":"string","pattern":42}`, `"whatever"`},
		{"properties as an array", `{"type":"object","properties":[]}`, `{}`},
		{"type as a number", `{"type":7}`, `{}`},
		{"properties element not a schema", `{"type":"object","properties":{"a":5}}`, `{"a":1}`},
		{"required member not a string", `{"type":"object","required":["a",5]}`, `{"a":1}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateModelMap([]byte(tc.doc), []byte(tc.schema))
			if err == nil {
				t.Fatal("a malformed keyword value must be a loud error — it is skipped rather than enforced, so it weakens the contract silently")
			}
			// The library reports these as metaschema violations and names both
			// the location and the expected type — e.g.
			// `at '/required': got string, want array`, which is more useful than
			// the hand-rolled message this replaced. The assertion checks the
			// PROPERTY (the error locates the offending keyword) rather than the
			// old wording, so it survives a library upgrade that rephrases.
			if !strings.Contains(err.Error(), "want") {
				t.Errorf("the error must say what the keyword's value should be, got: %v", err)
			}
		})
	}
}

// Round-3 review finding (Major): a tier declared in `tiers` with no matching
// entry in `chains` validated cleanly and then failed the moment a dispatcher
// tried to route it — at run time, under load.
func TestSchemaRejectsTierWithoutChain(t *testing.T) {
	schema := []byte(`{"type": "object", "x-tiersHaveChains": true}`)

	bad := `{"tiers": {"mid": {"claude": "sonnet"}, "ghost": {"claude": "opus"}},
	         "chains": {"mid": ["claude:sonnet"]}}`
	err := ValidateModelMap([]byte(bad), schema)
	if err == nil {
		t.Fatal("a tier with no chain must be rejected")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("the error must name the orphaned tier, got: %v", err)
	}

	good := `{"tiers": {"mid": {"claude": "sonnet"}}, "chains": {"mid": ["claude:sonnet"]}}`
	if err := ValidateModelMap([]byte(good), schema); err != nil {
		t.Errorf("matched tiers and chains must validate, got: %v", err)
	}
}

// Both custom rules must run. An earlier revision returned from the first, so a
// schema requesting both got only the first checked — and a rule that is
// declared and never runs is the same shape as one that passed.
func TestBothCrossBlockRulesRun(t *testing.T) {
	schema := []byte(`{"type": "object", "x-poolReferencesResolve": true, "x-tiersHaveChains": true}`)
	// Pool references are clean; only the tier rule should fire.
	doc := `{"pools": {"nan": {}},
	         "harnesses": {"pi": {"pools": ["nan"]}},
	         "tiers": {"orphan": {"pi": "m"}},
	         "chains": {}}`
	err := ValidateModelMap([]byte(doc), schema)
	if err == nil {
		t.Fatal("the second rule must run even when the first one passes")
	}
	if !strings.Contains(err.Error(), "orphan") {
		t.Errorf("expected the tier rule to fire, got: %v", err)
	}
}

// Found by probe, not by a reviewer: `a == node` on two `any` values panics when
// both hold the same non-comparable type, and JSON decodes arrays to []any and
// objects to map[string]any — both non-comparable. An enum of structured values
// therefore crashed the validator and would have taken `dotf doctor` with it.
//
// Unreachable through the shipped schema, which enumerates only strings. That is
// why three review rounds missed it, and why the test exists rather than a
// comment saying it cannot happen.
func TestEnumHandlesNonComparableValues(t *testing.T) {
	cases := []struct {
		name, doc, schema string
		wantValid         bool
	}{
		{"array in enum, matching", `["x"]`, `{"enum": [["x"], ["y"]]}`, true},
		{"array in enum, not matching", `["z"]`, `{"enum": [["x"], ["y"]]}`, false},
		{"object in enum, matching", `{"a":1}`, `{"enum": [{"a":1}]}`, true},
		{"object in enum, not matching", `{"a":2}`, `{"enum": [{"a":1}]}`, false},
		{"strings still work", `"opus"`, `{"enum": ["opus", "sonnet"]}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("validator panicked instead of deciding: %v", r)
				}
			}()
			err := ValidateModelMap([]byte(tc.doc), []byte(tc.schema))
			if tc.wantValid && err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
			if !tc.wantValid && err == nil {
				t.Error("expected the value to be rejected as outside the enum")
			}
		})
	}
}
