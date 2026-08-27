package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC1 — the registry is well-formed, every pin justified, ids unique, locators
// compile with exactly one capture.
func TestModelPinsRegistryIsWellFormed(t *testing.T) {
	pins, err := LoadModelPins(repoRootForTest(t))
	if err != nil {
		t.Fatalf("registry does not load: %v", err)
	}
	if len(pins.Sites) == 0 {
		t.Fatal("no sites declared")
	}
	repo := pins.SortedSiteFiles("repo")
	if len(repo) == 0 {
		t.Error("no repo-scoped sites — nothing would be checkable in CI")
	}
	if len(pins.SortedSiteFiles("deployed")) == 0 {
		t.Error("no deployed-scoped sites — the drift that motivated this lives there")
	}
}

// AC2 — the load-bearing one. Every routing pin in a COMMITTED file resolves to
// something harness/model-map.json declares.
//
// It reads the real repository, not a fixture: a guard over a fixture proves the
// guard parses, never that this repository agrees with its own routing registry.
func TestEveryRepoRoutingPinResolvesInTheMap(t *testing.T) {
	root := repoRootForTest(t)
	pins, err := LoadModelPins(root)
	if err != nil {
		t.Fatalf("registry does not load: %v", err)
	}
	m, err := LoadModelMap(root)
	if err != nil {
		t.Fatalf("model map does not load: %v", err)
	}
	qualified, bare := DeclaredModels(m)

	checked := 0
	for _, site := range pins.Sites {
		if site.Scope != "repo" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, site.File))
		if err != nil {
			t.Errorf("%s: declared as a pin site but unreadable: %v", site.File, err)
			continue
		}
		for _, p := range site.Pins {
			values, err := Extract(p, content)
			if err != nil {
				// A locator that matches nothing is the failure this check
				// exists to prevent, not a reason to skip the site.
				t.Errorf("%s: %v", site.File, err)
				continue
			}
			for _, raw := range values {
				checked++
				switch Check(p, raw, qualified, bare) {
				case VerdictOK:
				case VerdictWrongPool:
					t.Errorf("%s pin %q: %q normalizes to %q — the map knows the model but not under pool %q",
						site.File, p.ID, raw, Normalize(p, raw), p.Pool)
				case VerdictUnknown:
					t.Errorf("%s pin %q: %q normalizes to %q, which harness/model-map.json does not declare",
						site.File, p.ID, raw, Normalize(p, raw))
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("checked 0 pins — a sweep that inspects nothing reports clean, which is the defect this guards")
	}
	t.Logf("resolved %d routing pins across %d repo files", checked, len(pins.SortedSiteFiles("repo")))
}

// AC3 — the guard fails on a bad pin, so it cannot pass vacuously. Every
// assertion above is worthless without this one.
func TestGuardRejectsAnUnresolvablePin(t *testing.T) {
	m, err := LoadModelMap(repoRootForTest(t))
	if err != nil {
		t.Fatalf("model map does not load: %v", err)
	}
	qualified, bare := DeclaredModels(m)
	p := Pin{ID: "injected", Kind: "json-path", Locator: "defaultModel", Prefix: "nan/", Pool: "nan"}

	// The real id measured in the deployed settings.json on 2026-08-26.
	if got := Check(p, "nan/deepseek-v4-flash-0731", qualified, bare); got != VerdictUnknown {
		t.Errorf("a dead dated id must be VerdictUnknown, got %v", got)
	}
	// A retired provider's model.
	if got := Check(p, "openrouter/deepseek/deepseek-v4-pro", qualified, bare); got != VerdictUnknown {
		t.Errorf("a retired-provider id must be VerdictUnknown, got %v", got)
	}
	// And the control: a live one still passes, so the check is not simply
	// failing everything.
	if got := Check(p, "nan/qwen3.6", qualified, bare); got != VerdictOK {
		t.Errorf("a live id must be VerdictOK, got %v", got)
	}
}

// AC4 — a CATALOG entry the map does not route is NOT drift.
//
// qwen3.8-flash and glm5.3-flash are in pi's and opencode's pickers deliberately
// and deliberately absent from the map: availability on a promotional allocation
// pending a community vote (#1244). The guard must not fire on a recorded
// decision, and the way it does not is by checking only the pins the registry
// declares — never the catalogs those files also carry.
func TestCatalogEntriesAreNotCheckedAsRouting(t *testing.T) {
	pins, err := LoadModelPins(repoRootForTest(t))
	if err != nil {
		t.Fatalf("registry does not load: %v", err)
	}
	for _, site := range pins.Sites {
		if site.Scope != "repo" {
			continue
		}
		for _, p := range site.Pins {
			if strings.HasSuffix(p.Locator, "[]") {
				t.Errorf("%s pin %q locates an array in a repo-scoped site: catalogs are not routing, and checking one would fire on #1244's recorded decision",
					site.File, p.ID)
			}
			if p.Locator == "enabledModels" || p.Locator == "models" {
				t.Errorf("%s pin %q points at a catalog key", site.File, p.ID)
			}
		}
	}
}

// AC7 — a registry that cannot be read fails loudly and never as "no sites".
func TestModelPinsRefusesToReadAsEmpty(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"absent", "", "not an empty pin registry"},
		{"unparseable", `{"sites": [`, "parse"},
		{"no sites", `{"version":1,"sites":[]}`, "declares no sites"},
		{"site with no pins", `{"version":1,"sites":[{"file":"x","scope":"repo","pins":[]}]}`, "declares no pins"},
		{"pin with no why", `{"version":1,"sites":[{"file":"x","scope":"repo","pins":[{"id":"a","kind":"regex","locator":"(x)","pool":"nan"}]}]}`, "no why"},
		{"bad scope", `{"version":1,"sites":[{"file":"x","scope":"elsewhere","pins":[{"id":"a","kind":"regex","locator":"(x)","pool":"nan","why":"w"}]}]}`, "scope"},
		{"unknown kind", `{"version":1,"sites":[{"file":"x","scope":"repo","pins":[{"id":"a","kind":"telepathy","locator":"x","pool":"nan","why":"w"}]}]}`, "unknown kind"},
		{"locator without a capture", `{"version":1,"sites":[{"file":"x","scope":"repo","pins":[{"id":"a","kind":"regex","locator":"x","pool":"nan","why":"w"}]}]}`, "capture group"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.body != "" {
				if err := os.MkdirAll(filepath.Join(dir, "harness"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ModelPinsFile), []byte(tc.body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := LoadModelPins(dir)
			if err == nil {
				t.Fatalf("%s loaded clean — a broken registry must never read as no drift", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

// A locator that has silently stopped matching must be an error, never an empty
// result — "found no pins" and "found no drift" are different facts.
func TestExtractRefusesAnEmptyMatch(t *testing.T) {
	p := Pin{ID: "rotted", Kind: "regex-all", Locator: `opencode run -m (\S+)`, Pool: "nan"}
	if _, err := Extract(p, []byte("nothing here resembles the pattern\n")); err == nil {
		t.Fatal("a locator matching nothing must error, not return zero values")
	}

	// And a `regex` (exactly one) locator that starts matching twice is a
	// registry error rather than a silent pick-the-first.
	single := Pin{ID: "ambiguous", Kind: "regex", Locator: `model=(\S+)`, Pool: "nan"}
	if _, err := Extract(single, []byte("model=a\nmodel=b\n")); err == nil {
		t.Fatal("kind=regex matching twice must error rather than silently choosing one")
	}
}

// A tier key that is NOT a declared pool contributes to `bare` only, so a pin
// claiming that model under some pool reports VerdictWrongPool rather than OK.
//
// That behaviour is deliberate — without a pool declaration nothing establishes
// where the model lives, and WrongPool is a warning while Unknown would be a
// failure — but it is subtle enough that the PR reviewer on #1256 flagged it as
// a possible flaw. It was latent rather than live: every tier key in the shipped
// map (`claude`, `nan`, `opencode`) IS a declared pool, so the reviewer's own
// example could not occur. This pins the behaviour on a synthetic map so a
// future edit that introduces a non-pool tier key finds a test rather than a
// comment.
func TestTierKeyThatIsNotAPoolStaysUnqualified(t *testing.T) {
	m := map[string]any{
		"pools": map[string]any{"nan": map[string]any{}},
		"tiers": map[string]any{
			"mid": map[string]any{"somewhere-else": "orphan-model"},
		},
		"chains":   map[string]any{},
		"services": map[string]any{},
	}
	qualified, bare := DeclaredModels(m)

	if !bare["orphan-model"] {
		t.Error("a tier's model must always reach the bare set")
	}
	if qualified["somewhere-else:orphan-model"] {
		t.Error("a non-pool tier key must not produce a qualified entry — that would invent an attribution the map never made")
	}
	p := Pin{ID: "x", Kind: "regex", Locator: "(x)", Prefix: "", Pool: "nan"}
	if got := Check(p, "orphan-model", qualified, bare); got != VerdictWrongPool {
		t.Errorf("want VerdictWrongPool (a warning: the model is known, its pool is not), got %v", got)
	}
}

// A `$comment` inside a tier is prose, not a model id. Found 2026-08-27 when
// `tiers.low` acquired one: the whole sentence was entering the declared set.
// Only ever widened it, so nothing could fail wrongly — but a registry that
// treats prose as a model id is one coincidence away from masking real drift.
func TestDeclaredModelsIgnoresAnnotationKeysInTiers(t *testing.T) {
	m := map[string]any{
		"pools": map[string]any{"nan": map[string]any{}},
		"tiers": map[string]any{
			"low": map[string]any{"nan": "qwen3.8-flash", "$comment": "some prose about the promotion"},
		},
		"chains":   map[string]any{},
		"services": map[string]any{},
	}
	_, bare := DeclaredModels(m)
	if !bare["qwen3.8-flash"] {
		t.Error("the real model id must still be declared")
	}
	if bare["some prose about the promotion"] {
		t.Error("a $comment's text was taken for a model id")
	}
}

// DeclaredModels must not invent a pool attribution the map never made: tiers
// are keyed by whatever consumes the id, and `claude`/`opencode` key by harness
// there while `nan` keys by pool.
func TestDeclaredModelsDoesNotInventPoolsFromHarnessKeys(t *testing.T) {
	m, err := LoadModelMap(repoRootForTest(t))
	if err != nil {
		t.Fatalf("model map does not load: %v", err)
	}
	qualified, bare := DeclaredModels(m)
	if len(bare) == 0 || len(qualified) == 0 {
		t.Fatal("declared nothing from a valid map")
	}
	pools, _ := m["pools"].(map[string]any)
	for q := range qualified {
		pool, _, _ := strings.Cut(q, ":")
		if _, ok := pools[pool]; !ok {
			t.Errorf("qualified entry %q names %q, which is not a declared pool", q, pool)
		}
	}
}
