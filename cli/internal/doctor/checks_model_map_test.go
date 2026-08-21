package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validModelMap = `{
  "$comment": ["why"],
  "version": 1,
  "pools": {"nan": {"auth": "subscription", "probe": "env:NAN_API_KEY"}},
  "harnesses": {"pi": {"pools": ["nan"], "render": "adapter"}},
  "tiers": {"mid": {"pi": "deepseek-v4-flash"}},
  "chains": {"mid": ["nan:deepseek-v4-flash"]},
  "services": {}
}`

const validModelMapSchema = `{
  "type": "object",
  "required": ["$comment", "version", "pools", "harnesses", "tiers", "chains", "services"],
  "x-poolReferencesResolve": true,
  "properties": {
    "$comment":  {"type": "array"},
    "version":   {"type": "integer"},
    "pools":     {"type": "object"},
    "harnesses": {"type": "object"},
    "tiers":     {"type": "object"},
    "chains":    {"type": "object"},
    "services":  {"type": "object"}
  }
}`

func modelMapFixture(t *testing.T, mapJSON, schemaJSON string) *Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "harness"), 0o755); err != nil {
		t.Fatal(err)
	}
	if mapJSON != "" {
		if err := os.WriteFile(filepath.Join(dir, "harness", "model-map.json"), []byte(mapJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if schemaJSON != "" {
		if err := os.WriteFile(filepath.Join(dir, "harness", "model-map.schema.json"), []byte(schemaJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &Config{DotfilesDir: dir}
}

// AC6 and the whole point of constraint C15. Three broken states, three distinct
// loud outcomes, and — the part that matters — none of them may render as an
// empty or permissive map.
//
// The distinction is not pedantry. This repository has now measured the same
// failure four times in two days: a positive-looking signal standing in for a
// check that never ran. A routing map is the worst place for it, because
// "no pools declared" and "the file is unreadable" would produce identical
// downstream behaviour while meaning opposite things.
func TestModelMapCheckThreeBrokenStates(t *testing.T) {
	cases := []struct {
		name        string
		mapJSON     string
		schemaJSON  string
		wantFail    bool
		wantPhrases []string
		bannedWords []string
	}{
		{
			name:        "absent",
			mapJSON:     "",
			schemaJSON:  validModelMapSchema,
			wantFail:    true,
			wantPhrases: []string{"not found", "not an empty routing map"},
			bannedWords: []string{"no pools", "0 pools"},
		},
		{
			name:        "unparseable",
			mapJSON:     `{"pools": {`,
			schemaJSON:  validModelMapSchema,
			wantFail:    true,
			wantPhrases: []string{"could not be parsed"},
			bannedWords: []string{"no pools", "0 pools"},
		},
		{
			name:        "schema-invalid",
			mapJSON:     `{"$comment": ["x"], "version": 1, "pools": {}, "harnesses": {"pi": {"pools": ["ghost"]}}, "tiers": {}, "chains": {}, "services": {}}`,
			schemaJSON:  validModelMapSchema,
			wantFail:    true,
			wantPhrases: []string{"ghost"},
			bannedWords: []string{"no pools", "0 pools"},
		},
		{
			name:        "schema itself missing — the map cannot be trusted without its contract",
			mapJSON:     validModelMap,
			schemaJSON:  "",
			wantFail:    true,
			wantPhrases: []string{"model-map.schema.json"},
		},
		{
			name:       "healthy",
			mapJSON:    validModelMap,
			schemaJSON: validModelMapSchema,
			wantFail:   false,
		},
	}

	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			rep := capture(&buf)
			checkModelMap(modelMapFixture(t, tc.mapJSON, tc.schemaJSON), rep)
			rep.flush()
			out := buf.String()

			failed := strings.Contains(out, "FAIL")
			if failed != tc.wantFail {
				t.Fatalf("wantFail=%v, got output:\n%s", tc.wantFail, out)
			}
			for _, phrase := range tc.wantPhrases {
				if !strings.Contains(out, phrase) {
					t.Errorf("output must contain %q, got:\n%s", phrase, out)
				}
			}
			for _, banned := range tc.bannedWords {
				if strings.Contains(out, banned) {
					t.Errorf("a broken map must never render as %q — that is the permissive default C15 forbids:\n%s", banned, out)
				}
			}
			if tc.wantFail {
				// Each broken state must be distinguishable from the others. Three
				// failures that all say "model-map is broken" send the reader to
				// diff the file by hand, which is what a diagnostic exists to avoid.
				if seen[out] {
					t.Errorf("this broken state produces the same message as an earlier one:\n%s", out)
				}
				seen[out] = true
			}
		})
	}
}
