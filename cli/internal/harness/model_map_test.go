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

// C15's teeth. A validator that silently ignores a schema construct it does not
// implement reports health it never established — the exact class this whole
// spec exists to close. Unknown constructs must be loud.
func TestValidatorRejectsUnimplementedSchemaConstructs(t *testing.T) {
	schema := []byte(`{
	  "type": "object",
	  "properties": {"pools": {"type": "object"}},
	  "patternProperties": {"^x-": {"type": "string"}}
	}`)

	err := ValidateModelMap([]byte(`{"pools": {}}`), schema)
	if err == nil {
		t.Fatal("a schema construct the validator does not implement must be a loud error, not a silent pass")
	}
	if !strings.Contains(err.Error(), "patternProperties") {
		t.Errorf("the error must name the unimplemented construct, got: %v", err)
	}
}

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
