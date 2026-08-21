package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// implementedKeywords is the JSON Schema subset this validator understands.
//
// The allow-list is the point, not the contents. A validator that skips a
// keyword it does not implement reports a document valid without having checked
// what the schema asked for — health it never established, which is the failure
// class this registry exists to close. Adding a keyword to the schema therefore
// requires adding it here, and forgetting is a loud test failure rather than a
// quiet weakening of the contract.
var implementedKeywords = map[string]bool{
	// Assertions.
	"type": true, "required": true, "properties": true,
	"additionalProperties": true, "items": true, "enum": true,
	"minItems": true, "minProperties": true,
	// Custom: every name in any harnesses.<h>.pools[] must be declared in pools.
	"x-poolReferencesResolve": true,
	// Annotations — carried for humans and editors, asserted on by nobody.
	"$schema": true, "$id": true, "title": true,
	"description": true, "$comment": true,
}

// ValidateModelMap checks doc against schema, which is read as data rather than
// re-expressed as Go literals — so harness/model-map.schema.json stays the single
// source of truth and this file is only its interpreter.
func ValidateModelMap(doc, schema []byte) error {
	var s map[string]any
	if err := json.Unmarshal(schema, &s); err != nil {
		return fmt.Errorf("parse %s: %w", ModelMapSchemaFile, err)
	}
	var d any
	if err := json.Unmarshal(doc, &d); err != nil {
		return fmt.Errorf("parse %s: %w", ModelMapFile, err)
	}
	if err := assertKeywordsImplemented(s, "(root)"); err != nil {
		return err
	}
	if err := validateNode(d, s, "(root)"); err != nil {
		return err
	}
	if truthy(s["x-poolReferencesResolve"]) {
		return checkPoolReferences(d)
	}
	return nil
}

// assertKeywordsImplemented walks the whole schema up front, so an unimplemented
// keyword fails even when it sits under a branch this document never reaches.
func assertKeywordsImplemented(schema map[string]any, path string) error {
	for k, v := range schema {
		if !implementedKeywords[k] {
			return fmt.Errorf(
				"%s declares %q at %s, which this validator does not implement\n"+
					"implement it in %s or remove it from the schema — silently ignoring it "+
					"would report the document valid without checking what the schema asked for",
				ModelMapSchemaFile, k, path, "cli/internal/harness/model_map.go")
		}
		switch k {
		case "properties":
			if props, ok := v.(map[string]any); ok {
				for name, sub := range props {
					if subSchema, ok := sub.(map[string]any); ok {
						if err := assertKeywordsImplemented(subSchema, path+"."+name); err != nil {
							return err
						}
					}
				}
			}
		case "items", "additionalProperties":
			if subSchema, ok := v.(map[string]any); ok {
				if err := assertKeywordsImplemented(subSchema, path+"."+k); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateNode(node any, schema map[string]any, path string) error {
	if want, ok := schema["type"].(string); ok {
		if err := checkType(node, want, path); err != nil {
			return err
		}
	}
	if allowed, ok := schema["enum"].([]any); ok {
		if !containsValue(allowed, node) {
			return fmt.Errorf("%s: %v is not one of the permitted values %v", path, node, allowed)
		}
	}

	switch typed := node.(type) {
	case map[string]any:
		return validateObject(typed, schema, path)
	case []any:
		return validateArray(typed, schema, path)
	}
	return nil
}

func validateObject(obj map[string]any, schema map[string]any, path string) error {
	for _, r := range toStrings(schema["required"]) {
		if _, present := obj[r]; !present {
			return fmt.Errorf("%s: missing required key %q", path, r)
		}
	}
	if minProps, ok := toInt(schema["minProperties"]); ok && len(obj) < minProps {
		return fmt.Errorf("%s: has %d properties, schema requires at least %d", path, len(obj), minProps)
	}

	props, _ := schema["properties"].(map[string]any)
	for _, key := range sortedKeys(obj) {
		sub, declared := props[key]
		if !declared {
			// additionalProperties as a schema constrains every undeclared key;
			// as `false` it forbids them outright.
			switch extra := schema["additionalProperties"].(type) {
			case bool:
				if !extra {
					return fmt.Errorf("%s: key %q is not permitted by the schema", path, key)
				}
			case map[string]any:
				if err := validateNode(obj[key], extra, path+"."+key); err != nil {
					return err
				}
			}
			continue
		}
		if subSchema, ok := sub.(map[string]any); ok {
			if err := validateNode(obj[key], subSchema, path+"."+key); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateArray(arr []any, schema map[string]any, path string) error {
	if minItems, ok := toInt(schema["minItems"]); ok && len(arr) < minItems {
		return fmt.Errorf("%s: has %d items, schema requires at least %d", path, len(arr), minItems)
	}
	itemSchema, ok := schema["items"].(map[string]any)
	if !ok {
		return nil
	}
	for i, item := range arr {
		if err := validateNode(item, itemSchema, fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
	}
	return nil
}

// checkPoolReferences is the rule a stock JSON Schema cannot express and the one
// ADR-032 §3's reference block actually violated: harnesses.codex.pools named a
// `codex` pool that the pools block never declared. Every type there is correct
// and every required key is present, so only a cross-block check catches it.
func checkPoolReferences(doc any) error {
	root, ok := doc.(map[string]any)
	if !ok {
		return nil
	}
	pools, _ := root["pools"].(map[string]any)
	harnesses, _ := root["harnesses"].(map[string]any)

	var dangling []string
	for _, hName := range sortedKeys(harnesses) {
		h, ok := harnesses[hName].(map[string]any)
		if !ok {
			continue
		}
		for _, poolName := range toStrings(h["pools"]) {
			if _, declared := pools[poolName]; !declared {
				dangling = append(dangling, fmt.Sprintf("harnesses.%s.pools[] names %q", hName, poolName))
			}
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

func checkType(node any, want, path string) error {
	got := jsonTypeOf(node)
	if got == want {
		return nil
	}
	// JSON has one number type; "integer" is a narrowing of it.
	if want == "integer" && got == "number" {
		if f, ok := node.(float64); ok && f == float64(int64(f)) {
			return nil
		}
	}
	return fmt.Errorf("%s: expected %s, got %s", path, want, got)
}

func jsonTypeOf(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "unknown"
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

func truthy(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func containsValue(allowed []any, node any) bool {
	for _, a := range allowed {
		if a == node {
			return true
		}
	}
	return false
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
		return nil, fmt.Errorf("tier %q has no chain in %s", tier, ModelMapFile)
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
