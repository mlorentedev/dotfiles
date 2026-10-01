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
				switch got := Check(p, raw, qualified, bare); got {
				case VerdictOK:
				case VerdictWrongPool:
					t.Errorf("%s pin %q: %q normalizes to %q — the map knows the model but not under pool %q",
						site.File, p.ID, raw, Normalize(p, raw), p.Pool)
				case VerdictUnknown:
					t.Errorf("%s pin %q: %q normalizes to %q, which harness/model-map.json does not declare",
						site.File, p.ID, raw, Normalize(p, raw))
				case VerdictMisspelled:
					t.Errorf("%s pin %q: %q is not spelled %q…%q, the form this site declares",
						site.File, p.ID, raw, p.Prefix, p.Suffix)
				default:
					// A verdict this switch does not know would otherwise fall
					// through silently — how round-1 finding 4's mutation would
					// have kept passing here after Check learned to catch it.
					t.Errorf("%s pin %q: %q came back verdict %v, which this guard does not handle",
						site.File, p.ID, raw, got)
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
	// A retired provider's model. It does not carry the declared `nan/`, so it
	// fails on spelling before the map is asked (finding 4).
	if got := Check(p, "openrouter/deepseek/deepseek-v4-pro", qualified, bare); got != VerdictMisspelled {
		t.Errorf("a retired-provider id must be VerdictMisspelled, got %v", got)
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
//
// A catalog is declared (`catalog: true`), not inferred from an array locator:
// `.pr_agent.toml`'s `fallback_models` is an array and an ordered routing list,
// so "array means catalog" stopped holding once finding 2 declared it.
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
			if p.Catalog {
				t.Errorf("%s pin %q is a catalog in a repo-scoped site: catalogs are not routing, and checking one would fire on #1244's recorded decision",
					site.File, p.ID)
			}
			last := p.Locator[strings.LastIndex(p.Locator, ".")+1:]
			if last = strings.TrimSuffix(last, "[]"); last == "enabledModels" || last == "models" {
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
		{"catalog on a scalar", `{"version":1,"sites":[{"file":"x","scope":"deployed","pins":[{"id":"a","kind":"json-path","locator":"m","pool":"nan","why":"w","catalog":true}]}]}`, "catalog"},
		{"unknown spelling", `{"version":1,"sites":[{"file":"x","scope":"repo","pins":[{"id":"a","kind":"json-path","locator":"m","pool":"nan","why":"w","spelling":"vibes"}]}]}`, "spelling"},
		{"exclusion with no why", `{"version":1,"sites":[{"file":"x","scope":"repo","pins":[{"id":"a","kind":"json-path","locator":"m","pool":"nan","why":"w"}]}],"excluded":[{"file":"x","locator":"y"}]}`, "no why"},
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

// Round-1 finding 4: Normalize used TrimPrefix, a no-op when the prefix is
// absent, so `.pr_agent.toml` could drop the `openai/` that litellm needs to
// reach NaN and the guard still passed. The declared spelling is now required.
// The mutation is the reviewer's own, run against the real file.
func TestNormalizeRequiresTheDeclaredPrefix(t *testing.T) {
	root := repoRootForTest(t)
	m, err := LoadModelMap(root)
	if err != nil {
		t.Fatalf("model map does not load: %v", err)
	}
	qualified, bare := DeclaredModels(m)
	p := Pin{ID: "pr-agent-primary", Kind: "toml-key", Locator: "model", Prefix: "openai/", Pool: "nan"}

	real, err := os.ReadFile(filepath.Join(root, ".pr_agent.toml"))
	if err != nil {
		t.Fatal(err)
	}
	values, err := Extract(p, real)
	if err != nil {
		t.Fatal(err)
	}
	if got := Check(p, values[0], qualified, bare); got != VerdictOK {
		t.Fatalf("the shipped value %q must resolve, got %v", values[0], got)
	}

	dropped := strings.Replace(string(real), `model = "openai/`, `model = "`, 1)
	values, err = Extract(p, []byte(dropped))
	if err != nil {
		t.Fatal(err)
	}
	if got := Check(p, values[0], qualified, bare); got != VerdictMisspelled {
		t.Errorf("%q dropped the declared prefix %q and came back %v, want VerdictMisspelled", values[0], p.Prefix, got)
	}
}

// A declared suffix is required the same way. Claude Code reads `opus[1m]` as
// the opus alias with the 1M-token context window; the map routes `opus`, so the
// suffix is spelling, and dropping it is a registry edit rather than a pass.
func TestDeclaredSuffixIsSpellingNotPartOfTheID(t *testing.T) {
	m, err := LoadModelMap(repoRootForTest(t))
	if err != nil {
		t.Fatalf("model map does not load: %v", err)
	}
	qualified, bare := DeclaredModels(m)
	p := Pin{ID: "claude-model", Kind: "json-path", Locator: "model", Suffix: "[1m]", Pool: "claude"}

	if got := Normalize(p, "opus[1m]"); got != "claude:opus" {
		t.Errorf("Normalize(opus[1m]) = %q, want claude:opus", got)
	}
	if got := Check(p, "opus[1m]", qualified, bare); got != VerdictOK {
		t.Errorf("opus[1m] must resolve, got %v", got)
	}
	if got := Check(p, "opus", qualified, bare); got != VerdictMisspelled {
		t.Errorf("opus without the declared suffix came back %v, want VerdictMisspelled", got)
	}
}

// agy writes the display name its model picker shows, while `agy --model`
// takes the id. The pairs below are the gemini rows of `agy models`, measured
// 2026-10-01; the rule holds for that family only (`Claude Sonnet 4.6
// (Thinking)` is `claude-sonnet-4-6`), which is why a pin declares it rather
// than the guard applying it everywhere.
func TestDisplayNameSpellingMatchesTheAgyCatalog(t *testing.T) {
	for display, id := range map[string]string{
		"Gemini 3.8 Flash (High)":   "gemini-3.8-flash-high",
		"Gemini 3.8 Flash (Medium)": "gemini-3.8-flash-medium",
		"Gemini 3.8 Flash (Low)":    "gemini-3.8-flash-low",
		"Gemini 3.7 Flash (High)":   "gemini-3.7-flash-high",
		"Gemini 3.7 Flash (Medium)": "gemini-3.7-flash-medium",
		"Gemini 3.7 Flash (Low)":    "gemini-3.7-flash-low",
		"Gemini 3.6 Flash (High)":   "gemini-3.6-flash-high",
		"Gemini 3.1 Pro (High)":     "gemini-3.1-pro-high",
		"Gemini 3.1 Pro (Low)":      "gemini-3.1-pro-low",
	} {
		p := Pin{ID: "agy-model", Kind: "json-path", Locator: "model", Spelling: "display-name", Pool: "gemini"}
		if got := Normalize(p, display); got != "gemini:"+id {
			t.Errorf("Normalize(%q) = %q, want gemini:%s", display, got, id)
		}
	}
}

// Finding 2's pins are nested (opencode's `agent.plan.model`) or arrays in TOML
// (`fallback_models`), which the top-level extractors could not reach.
func TestExtractReachesNestedAndArrayPins(t *testing.T) {
	doc := []byte(`{
  // a comment, as opencode.jsonc carries
  "provider": {"nan": {"options": {"model": "glm5.3-flash"}}},
  "agent": {"plan": {"model": "nan/a"}, "build": {"model": "nan/b"}, "review": {"permission": {}}}
}`)
	for _, tc := range []struct {
		locator string
		want    []string
	}{
		{"provider.nan.options.model", []string{"glm5.3-flash"}},
		// An agent without a model inherits the default: skipped under a
		// wildcard, never an error.
		{"agent.*.model", []string{"nan/b", "nan/a"}},
	} {
		got, err := Extract(Pin{ID: "x", Kind: "json-path", Locator: tc.locator}, doc)
		if err != nil {
			t.Errorf("%s: %v", tc.locator, err)
			continue
		}
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%s = %v, want %v", tc.locator, got, tc.want)
		}
	}
	for _, rotted := range []string{"provider.nan.options.gone", "agent.*.nothing", "nowhere.*.model"} {
		if _, err := Extract(Pin{ID: "x", Kind: "json-path", Locator: rotted}, doc); err == nil {
			t.Errorf("%s matched nothing and did not error", rotted)
		}
	}

	toml := []byte("model = \"openai/a\"\nfallback_models = [\n  \"openai/b\",\n  \"openai/c\",\n]\n")
	got, err := Extract(Pin{ID: "x", Kind: "toml-key", Locator: "fallback_models[]"}, toml)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "openai/b,openai/c" {
		t.Errorf("fallback_models[] = %v", got)
	}
	if _, err := Extract(Pin{ID: "x", Kind: "toml-key", Locator: "fallback_models[]"}, []byte("fallback_models = []\n")); err == nil {
		t.Error("an empty fallback list extracted nothing and did not error")
	}
}
