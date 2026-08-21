package harness

import "testing"

// FuzzValidateModelMap answers a question four adversarial review rounds could
// not: not "what does this code do wrong", but "what IS this code" — the
// language-level traps that need someone to already know the trap exists.
//
// It earned its place empirically. `containsValue` compared two `any` values
// with `==`, which panics in Go when both hold the same non-comparable type,
// and JSON decodes arrays to []any and objects to map[string]any. Three review
// rounds mutated the map, mutated the schema and probed negative paths without
// reaching it, because the shipped schema enumerates only strings and the trap
// is unreachable through it. Measured 2026-08-21: this target found it in under
// thirty seconds.
//
//	testing.go:1927: panic: runtime error: comparing uncomparable type []interface {}
//
// The contract is deliberately weak — never panic. Any input may be REJECTED;
// none may crash. A validator that takes down `dotf doctor` on a schema typo is
// worse than one that reports the typo.
func FuzzValidateModelMap(f *testing.F) {
	f.Add(`{"pools":{"nan":{}}}`, `{"type":"object","required":["pools"]}`)
	f.Add(`["x"]`, `{"enum":[["x"]]}`)
	f.Add(`{"a":1}`, `{"type":"object","properties":{"a":{"type":"integer"}}}`)
	f.Add(`"s"`, `{"type":"string","minLength":1,"pattern":"^s$"}`)
	f.Fuzz(func(t *testing.T, doc, schema string) {
		// The only contract: never panic. Any input may be rejected; none may crash.
		_ = ValidateModelMap([]byte(doc), []byte(schema))
	})
}
