package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ModelMapFile and ModelMapSchemaFile are the repo-relative paths to the routing
// registry and its declarative contract (ADR-035).
//
// Deliberately NOT embedded, unlike triggers.json next door. triggers.go carries
// a //go:embed copy and falls back to it when the repo file is missing; for a
// routing map that fallback is disqualifying. An absent map would resolve to a
// build-time default and every check downstream would report it healthy, which
// is exactly what constraint C15 forbids: a map that cannot be read must fail
// loudly, never as empty and never as a permissive default. See #1137 for the
// drift the embedded copy already carries.
const (
	ModelMapFile       = "harness/model-map.json"
	ModelMapSchemaFile = "harness/model-map.schema.json"
)

// customRules are the cross-block checks no schema language expresses, keyed by
// the `x-` keyword in the schema that switches each one on.
//
// The slice is a CLOSED set, and that is the point. Draft 2020-12 treats an
// unknown keyword as an annotation — the tolerance these `x-` extensions rely on
// to coexist with a conforming library, and the same tolerance that makes a
// misspelling invisible. `x-poolReferenceResolve` (singular) is not a rule name,
// so a lookup finds nothing, the rule never runs, and a document carrying a ghost
// pool validates clean. Review round 2 closed the wrong-TYPE variant of this and
// left the wrong-NAME variant open; round 5 found it.
//
// Standard keywords stay the library's business. The `x-` namespace is ours, so
// it is policed as a closed set rather than tolerated as annotation — which is
// the loudness property the deleted keyword allow-list used to provide for free.
var customRules = []struct {
	name string
	run  func(any) error
}{
	{"x-poolReferencesResolve", checkPoolReferences},
	{"x-tiersHaveChains", checkTiersHaveChains},
}

// checkCustomRuleNamespace rejects any top-level `x-` keyword that is not a rule
// this validator implements.
//
// Deliberately NOT the mirror rule ("every known rule must be declared"). That
// belongs to the shipped schema, not to this function: a unit test isolating one
// cross-block rule must be able to pass a schema declaring only that rule, and
// forcing every caller to name both would trade a real guard for test noise.
// TestShippedSchemaDeclaresEveryCustomRule asserts the shipped schema declares
// both, which is the only document where an omission means anything.
//
// Keys are sorted so a schema with two bad keywords reports the same one every
// run; map iteration order would make the message a coin flip.
func checkCustomRuleNamespace(schema map[string]any) error {
	known := make(map[string]bool, len(customRules))
	names := make([]string, 0, len(customRules))
	for _, r := range customRules {
		known[r.name] = true
		names = append(names, r.name)
	}

	unknown := make([]string, 0)
	for key := range schema {
		if strings.HasPrefix(key, "x-") && !known[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)

	return fmt.Errorf(
		"%s: %q is not a custom rule this validator implements (known: %s) — "+
			"a misspelled rule name is read as an annotation and the rule silently never runs, "+
			"which is indistinguishable from it passing",
		ModelMapSchemaFile, unknown[0], strings.Join(names, ", "))
}

// ValidateModelMap checks doc against schema.
//
// Standard JSON Schema keywords are interpreted by santhosh-tekuri/jsonschema,
// a draft-2020-12 implementation. The two CUSTOM cross-block rules below are
// ours, because no schema language expresses them.
//
// This replaced a hand-rolled interpreter on 2026-08-21, under a threshold
// declared before the evidence arrived: any NEW interpreter-semantics finding in
// review round 4 meant the no-dependency choice was a defect factory. Round 4
// delivered two — malformed `properties` elements and non-string `required`
// members skipped silently, and `minimum` unimplemented so negative budgets
// validated. Family total across four rounds: eight, every one of them something
// a conforming implementation gets right by construction.
//
// What did NOT change: the schema FILE remains the single source of truth, so
// editors and external validators are unaffected, and both custom rules stay.
// What changed is only who interprets the standard keywords. The keyword
// allow-list died with the interpreter it guarded; unknown keywords — including
// this schema's two `x-` extensions — are annotations under draft 2020-12, which
// is exactly the tolerance the custom rules need.
func ValidateModelMap(doc, schema []byte) error {
	compiler := jsonschema.NewCompiler()

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return fmt.Errorf("parse %s: %w", ModelMapSchemaFile, err)
	}
	// A stable, opaque resource id. Passing the repo-relative path makes the
	// library resolve it against the process working directory and leak an
	// absolute path into every error message — noise that differs per machine.
	const schemaURL = "https://mlorentedev.github.io/dotfiles/model-map.schema.json"
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("load %s: %w", ModelMapSchemaFile, err)
	}
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		// A schema that does not compile is loud, which is the property eight
		// hand-rolled defects were spent failing to guarantee.
		return fmt.Errorf("%s is not a valid schema: %w", ModelMapSchemaFile, err)
	}

	d, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return fmt.Errorf("parse %s: %w", ModelMapFile, err)
	}
	if err := compiled.Validate(d); err != nil {
		return fmt.Errorf("%s does not satisfy %s: %w", ModelMapFile, ModelMapSchemaFile, err)
	}

	// Both custom rules run; neither short-circuits the other. An earlier
	// revision returned from the first, so a schema requesting both got only the
	// first checked — and a rule that is declared and never runs is the same
	// shape as one that passed.
	var s map[string]any
	if err := json.Unmarshal(schema, &s); err != nil {
		return fmt.Errorf("parse %s: %w", ModelMapSchemaFile, err)
	}
	// The two `x-` flags are OURS, so their types are ours to police. Draft
	// 2020-12 treats unknown keywords as annotations, which is the tolerance the
	// custom rules need — and it means the library will not complain if one is
	// written as the string "true". A bare `.(bool)` would then read it as
	// false and silently disable the rule, which is the defect round 2 found and
	// which delegating standard keywords does not fix, because these are not
	// standard keywords.
	if err := checkCustomRuleNamespace(s); err != nil {
		return err
	}
	for _, flag := range customRules {
		raw, present := s[flag.name]
		if !present {
			continue
		}
		want, ok := raw.(bool)
		if !ok {
			return fmt.Errorf(
				"%s: %s is %T, which this validator cannot interpret (expected a boolean) — "+
					"silently disabling a cross-block rule is indistinguishable from passing it",
				ModelMapSchemaFile, flag.name, raw)
		}
		if !want {
			continue
		}
		if err := flag.run(d); err != nil {
			return err
		}
	}
	return nil
}

// checkPoolReferences is the rule a stock JSON Schema cannot express and the one
// ADR-032 section 3's reference block actually violated: harnesses.codex.pools
// named a `codex` pool that the pools block never declared. Every type there is
// correct and every required key is present, so only a cross-block check catches it.
//
// It walks THREE blocks, not one. An earlier version checked only `harnesses`,
// and an adversarial review found the gap: `chains` entries are `pool:model` and
// `services.*.pool` is a pool name, so a ghost pool in either validated cleanly.
// `chains` is the worse of the two — it is what a dispatcher walks at run time,
// so a ghost there is a fallback that fails at the exact moment the primary
// already has.
func checkPoolReferences(doc any) error {
	root, ok := doc.(map[string]any)
	if !ok {
		return nil
	}
	pools, _ := root["pools"].(map[string]any)

	var dangling []string
	note := func(where, pool string) {
		if _, declared := pools[pool]; !declared {
			dangling = append(dangling, fmt.Sprintf("%s names %q", where, pool))
		}
	}

	harnesses, _ := root["harnesses"].(map[string]any)
	for _, hName := range sortedKeys(harnesses) {
		h, ok := harnesses[hName].(map[string]any)
		if !ok {
			continue
		}
		for _, poolName := range toStrings(h["pools"]) {
			note(fmt.Sprintf("harnesses.%s.pools[]", hName), poolName)
		}
	}

	// chains entries are `pool:model`; only the pool half is a cross-reference.
	chains, _ := root["chains"].(map[string]any)
	for _, tier := range sortedKeys(chains) {
		for i, entry := range toStrings(chains[tier]) {
			pool, _, found := strings.Cut(entry, ":")
			if !found {
				dangling = append(dangling, fmt.Sprintf(
					"chains.%s[%d] is %q, which is not in `pool:model` form", tier, i, entry))
				continue
			}
			note(fmt.Sprintf("chains.%s[%d]", tier, i), pool)
		}
	}

	services, _ := root["services"].(map[string]any)
	for _, svc := range sortedKeys(services) {
		entry, ok := services[svc].(map[string]any)
		if !ok {
			continue
		}
		if pool, ok := entry["pool"].(string); ok {
			note(fmt.Sprintf("services.%s.pool", svc), pool)
		}
	}

	if len(dangling) > 0 {
		return fmt.Errorf(
			"%s references pools that the `pools` block does not declare:\n  %s\n"+
				"declare each pool, or remove the reference — a harness pointing at a pool "+
				"that does not exist routes nowhere at dispatch time",
			ModelMapFile, strings.Join(dangling, "\n  "))
	}
	return nil
}

// checkTiersHaveChains is the second cross-block rule a stock schema cannot
// express. `tiers` resolves at compile time and `chains` at run time, so a tier
// declared without a chain validates cleanly and then fails the moment a
// dispatcher tries to route it — at run time, under load, which is the worst
// place to discover a missing declaration.
func checkTiersHaveChains(doc any) error {
	root, ok := doc.(map[string]any)
	if !ok {
		return nil
	}
	tiers, _ := root["tiers"].(map[string]any)
	chains, _ := root["chains"].(map[string]any)

	var orphans []string
	for _, tier := range sortedKeys(tiers) {
		if _, has := chains[tier]; !has {
			orphans = append(orphans, tier)
		}
	}
	if len(orphans) > 0 {
		return fmt.Errorf(
			"%s declares tiers with no chain: %s\n"+
				"every tier needs a chain, or a dispatcher resolving that tier at run time "+
				"has nowhere to route — give each one a chain, or remove the tier",
			ModelMapFile, strings.Join(orphans, ", "))
	}
	return nil
}

// LoadModelMap reads the routing registry and validates it against the schema
// beside it. There is no fallback and no embedded default: where the map cannot
// be read, this errors (C15).
func LoadModelMap(repoRoot string) (map[string]any, error) {
	mapPath := filepath.Join(repoRoot, ModelMapFile)
	schemaPath := filepath.Join(repoRoot, ModelMapSchemaFile)

	doc, err := os.ReadFile(mapPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w\nthis is not an empty routing map — nothing falls back to a default", ModelMapFile, err)
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w\nthe map cannot be trusted without the contract it declares against", ModelMapSchemaFile, err)
	}
	if err := ValidateModelMap(doc, schema); err != nil {
		return nil, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(doc, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ModelMapFile, err)
	}
	return parsed, nil
}

func toStrings(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func toInt(v any) (int, bool) {
	f, ok := v.(float64)
	if !ok {
		return 0, false
	}
	return int(f), true
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// The two consumer classes, kept apart at the API because they resolve at
// different times: `tiers` feeds each harness's render at compile time, `chains`
// is read by a dispatcher at run time. One file, two cadences — which is what
// the `version` key exists to survive.

// ResolveTier answers which model id a neutral tier means for one harness.
// Compile-time consumer.
func ResolveTier(m map[string]any, tier, harness string) (string, error) {
	tiers, ok := m["tiers"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("%s declares no tiers block", ModelMapFile)
	}
	entry, ok := tiers[tier].(map[string]any)
	if !ok {
		return "", fmt.Errorf("tier %q is not declared in %s", tier, ModelMapFile)
	}
	model, ok := entry[harness].(string)
	if ok && strings.TrimSpace(model) == "" {
		// The type assertion succeeds for "", so only a MISSING key errored
		// before. A blank model id renders a definition naming no model at all,
		// which is the silent degrade this registry exists to prevent.
		return "", fmt.Errorf(
			"tier %q declares a blank model id for harness %q in %s — "+
				"remove the entry or give it a model; an empty id is not a default",
			tier, harness, ModelMapFile)
	}
	if !ok {
		return "", fmt.Errorf(
			"tier %q declares no model for harness %q in %s\n"+
				"declare one, or route this harness to a tier that has one — resolving to an "+
				"empty model id would render a definition that names no model at all",
			tier, harness, ModelMapFile)
	}
	return model, nil
}

// ResolveChain answers the ordered fallback for a tier, as `pool:model`.
// Run-time consumer, read only by a level-2 dispatcher.
//
// Order is meaning, not arrangement. And a single-entry chain is a statement:
// the `top` tier has no fallback on purpose (ADR-032 §4), because a top tier
// that quietly degrades converts a gate into a green checkmark. It queues or
// escalates instead.
func ResolveChain(m map[string]any, tier string) ([]string, error) {
	chains, ok := m["chains"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s declares no chains block", ModelMapFile)
	}
	raw, ok := chains[tier]
	if !ok {
		// The legal tiers are listed, not just the rejected one. A dictated
		// --tier lands here rather than in ResolveTierForPersona, which is the
		// only path that enumerated them, so the two refusals told an operator
		// different amounts about the same map (HARNESS-120 review, Minor).
		//
		// Reusing ResolveTierForPersona's wording here would be wrong: it blames
		// a PERSONA's record for the value, and a dictated tier has no record
		// behind it — a human typed it.
		declared, dErr := declaredChainTiers(m)
		if dErr != nil {
			return nil, dErr
		}
		return nil, fmt.Errorf("tier %q has no chain in %s: it declares %s",
			tier, ModelMapFile, strings.Join(declared, ", "))
	}
	chain := toStrings(raw)
	if len(chain) == 0 {
		return nil, fmt.Errorf(
			"tier %q declares an empty chain in %s — an empty chain is not 'no fallback', "+
				"it is a dispatch with nowhere to go", tier, ModelMapFile)
	}
	return chain, nil
}

// Budget is what a pool DECLARES. Nothing in this package decrements any of it.
//
// The naming is deliberate: ADR-035 ships level 1 (declaration) and defers level
// 2 (enforcement where dotf is the launcher) until a dispatcher exists to do the
// decrementing. A type called `Semaphore` or a method called `Acquire` would
// imply a guarantee this repo cannot keep — NaN's slots are shared with the pi
// TUI, qq, hive embeddings and CI, none of which route through dotf.
type Budget struct {
	// ConcurrencyDeclared distinguishes "not declared" from "zero". Zero reads as
	// no capacity; absent means the pool does not state one, which is the honest
	// answer for a seat-based pool where concurrency is a fleet property.
	ConcurrencyDeclared bool
	Concurrency         int
	ReserveInteractive  int
	RPM                 int
	// SharedWith names the consumers that draw on the same quota and that dotf
	// cannot see. It is the reason the eventual guarantee is "dotf alone will
	// never be the cause of exhaustion" rather than "exhaustion will not happen".
	SharedWith []string
}

// DeclaredBudget reads one pool's declared budget. It enforces nothing.
func DeclaredBudget(m map[string]any, pool string) (Budget, error) {
	pools, ok := m["pools"].(map[string]any)
	if !ok {
		return Budget{}, fmt.Errorf("%s declares no pools block", ModelMapFile)
	}
	p, ok := pools[pool].(map[string]any)
	if !ok {
		return Budget{}, fmt.Errorf("pool %q is not declared in %s", pool, ModelMapFile)
	}
	var b Budget
	if c, ok := toInt(p["concurrency"]); ok {
		b.Concurrency, b.ConcurrencyDeclared = c, true
	}
	b.ReserveInteractive, _ = toInt(p["reserve_interactive"])
	b.RPM, _ = toInt(p["rpm"])
	b.SharedWith = toStrings(p["shared_with"])
	return b, nil
}
