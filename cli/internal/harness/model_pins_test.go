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
