package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ModelPinsFile is the pin-site registry (HARNESS-067, #902).
const ModelPinsFile = "harness/model-pins.json"

// ModelPins is harness/model-pins.json: where a model id is pinned for ROUTING
// outside the map, and how each site spells it.
type ModelPins struct {
	Version int       `json:"version"`
	Sites   []PinSite `json:"sites"`
}

// PinSite is one file carrying routing pins.
type PinSite struct {
	File  string `json:"file"`
	Scope string `json:"scope"` // "repo" | "deployed"
	Why   string `json:"why"`
	Pins  []Pin  `json:"pins"`
}

// Pin is one routing decision inside a site.
type Pin struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"` // "json-path" | "toml-key" | "regex" | "regex-all"
	Locator string `json:"locator"`
	Prefix  string `json:"prefix"`
	Pool    string `json:"pool"`
	Why     string `json:"why"`
}

// LoadModelPins reads and validates the registry.
//
// Every failure is loud and none of them can render as "no pin sites declared"
// — constraint C15, and for the same reason it applies to the map next door: a
// registry that cannot be read and a registry that declares nothing produce
// identical downstream behaviour (zero drift reported) while meaning opposite
// things. One is a configuration, the other is a broken guard.
func LoadModelPins(repoRoot string) (*ModelPins, error) {
	path := filepath.Join(repoRoot, ModelPinsFile)
	doc, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w\nthis is not an empty pin registry — a guard that cannot read its own declaration reports no drift", ModelPinsFile, err)
	}
	var pins ModelPins
	if err := json.Unmarshal(doc, &pins); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ModelPinsFile, err)
	}
	if len(pins.Sites) == 0 {
		return nil, fmt.Errorf("%s declares no sites — refusing to report a clean sweep over nothing", ModelPinsFile)
	}
	seen := map[string]string{}
	for _, s := range pins.Sites {
		if s.Scope != "repo" && s.Scope != "deployed" {
			return nil, fmt.Errorf("%s: site %q has scope %q, want repo or deployed", ModelPinsFile, s.File, s.Scope)
		}
		if len(s.Pins) == 0 {
			return nil, fmt.Errorf("%s: site %q declares no pins", ModelPinsFile, s.File)
		}
		for _, p := range s.Pins {
			if p.Why == "" {
				return nil, fmt.Errorf("%s: pin %q has no why — a pin nobody can justify is one to delete", ModelPinsFile, p.ID)
			}
			if prev, dup := seen[p.ID]; dup {
				return nil, fmt.Errorf("%s: pin id %q declared twice (%s and %s)", ModelPinsFile, p.ID, prev, s.File)
			}
			seen[p.ID] = s.File
			switch p.Kind {
			case "json-path", "toml-key", "regex", "regex-all":
			default:
				return nil, fmt.Errorf("%s: pin %q has unknown kind %q", ModelPinsFile, p.ID, p.Kind)
			}
			if p.Kind == "regex" || p.Kind == "regex-all" {
				re, err := regexp.Compile(p.Locator)
				if err != nil {
					return nil, fmt.Errorf("%s: pin %q locator does not compile: %w", ModelPinsFile, p.ID, err)
				}
				if re.NumSubexp() != 1 {
					return nil, fmt.Errorf("%s: pin %q locator needs exactly one capture group, has %d", ModelPinsFile, p.ID, re.NumSubexp())
				}
			}
		}
	}
	return &pins, nil
}

// DeclaredModels reports what the map routes, as a set of `pool:id` and a set of
// bare ids.
//
// TWO SETS, NOT ONE, because the map's own $comment records that tiers are keyed
// by whatever CONSUMES the model id, which is not always a pool: `claude` and
// `opencode` key by harness there, while `nan` keys by pool. Collapsing that into
// one pool-qualified set would invent pool attributions the map never made and
// report them as drift.
//
// So `chains` — the one block that is unambiguously `pool:id` — populates the
// qualified set, and every block contributes to the bare set. A pin whose id is
// absent from the bare set names a model the map does not know at all, which is
// real drift. A pin present in the bare set but not under its declared pool is a
// weaker signal, reported as such.
func DeclaredModels(m map[string]any) (qualified map[string]bool, bare map[string]bool) {
	qualified, bare = map[string]bool{}, map[string]bool{}

	if chains, ok := m["chains"].(map[string]any); ok {
		for _, v := range chains {
			for _, entry := range toStrings(v) {
				qualified[entry] = true
				if pool, id, found := strings.Cut(entry, ":"); found {
					bare[id] = true
					qualified[pool+":"+id] = true
				}
			}
		}
	}
	if tiers, ok := m["tiers"].(map[string]any); ok {
		for _, v := range tiers {
			entry, ok := v.(map[string]any)
			if !ok {
				continue
			}
			for consumer, model := range entry {
				id, ok := model.(string)
				if !ok {
					continue
				}
				bare[id] = true
				// Recorded as qualified only when the consumer is itself a
				// declared pool. A harness key is not a pool, and pretending
				// otherwise is the invention this function exists to avoid.
				if pools, ok := m["pools"].(map[string]any); ok {
					if _, isPool := pools[consumer]; isPool {
						qualified[consumer+":"+id] = true
					}
				}
			}
		}
	}
	if services, ok := m["services"].(map[string]any); ok {
		for _, v := range services {
			svc, ok := v.(map[string]any)
			if !ok {
				continue
			}
			id, _ := svc["model"].(string)
			pool, _ := svc["pool"].(string)
			if id == "" {
				continue
			}
			bare[id] = true
			if pool != "" {
				qualified[pool+":"+id] = true
			}
		}
	}
	return qualified, bare
}

// Verdict is what checking one extracted value produced.
type Verdict int

const (
	// VerdictOK — the map routes this exact pool and model.
	VerdictOK Verdict = iota
	// VerdictWrongPool — the map knows the model, but not under this pin's pool.
	VerdictWrongPool
	// VerdictUnknown — the map does not declare this model at all. Real drift:
	// `nan/deepseek-v4-flash-0731` and the retired `openrouter/*` ids both land
	// here.
	VerdictUnknown
)

// Finding is one pin value that did not resolve cleanly.
type Finding struct {
	Site    string
	PinID   string
	Raw     string
	Norm    string
	Verdict Verdict
}

// Normalize turns a site's spelling into `pool:id` using the pin's declared
// rule. Declared, never inferred: one model has three correct spellings across
// these files (`nan:mimo-v2.5`, `openai/mimo-v2.5`, bare `mimo-v2.5`), so a
// guess would be wrong for two of them.
func Normalize(p Pin, raw string) string {
	id := raw
	if p.Prefix != "" {
		id = strings.TrimPrefix(id, p.Prefix)
	}
	// A site may still qualify with some other provider prefix — a retired one,
	// for instance. Keep it in the id so the diagnostic names what is actually
	// written rather than a silently trimmed version of it.
	return p.Pool + ":" + id
}

// Check resolves one raw value against the map.
func Check(p Pin, raw string, qualified, bare map[string]bool) Verdict {
	norm := Normalize(p, raw)
	if qualified[norm] {
		return VerdictOK
	}
	_, id, _ := strings.Cut(norm, ":")
	if bare[id] {
		return VerdictWrongPool
	}
	return VerdictUnknown
}

// Extract pulls every value a pin locates out of the file's content.
//
// It returns an error rather than an empty slice when a locator matches nothing,
// and that distinction is the whole guard: a locator that has silently stopped
// matching produces zero values, and zero values check clean. "No pins found" and
// "no drift" must never be the same outcome.
func Extract(p Pin, content []byte) ([]string, error) {
	switch p.Kind {
	case "json-path":
		return extractJSONPath(p, content)
	case "toml-key":
		return extractTOMLKey(p, content)
	case "regex", "regex-all":
		re, err := regexp.Compile(p.Locator)
		if err != nil {
			return nil, fmt.Errorf("pin %q: %w", p.ID, err)
		}
		matches := re.FindAllStringSubmatch(string(content), -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("pin %q: locator matched nothing — the file changed shape, or the pattern rotted", p.ID)
		}
		if p.Kind == "regex" && len(matches) > 1 {
			return nil, fmt.Errorf("pin %q: locator matched %d times but kind is regex (exactly one); use regex-all", p.ID, len(matches))
		}
		out := make([]string, 0, len(matches))
		for _, m := range matches {
			out = append(out, m[1])
		}
		return out, nil
	}
	return nil, fmt.Errorf("pin %q: unknown kind %q", p.ID, p.Kind)
}

// extractJSONPath supports a top-level key, and `key[]` for a top-level array of
// strings. Deliberately not a JSONPath engine: every pin site here is a flat
// top-level key, and a general query language would be more surface than the
// registry needs.
//
// The content is read as JSONC-tolerant — opencode.jsonc carries comments.
func extractJSONPath(p Pin, content []byte) ([]string, error) {
	var doc map[string]any
	if err := json.Unmarshal(stripJSONComments(content), &doc); err != nil {
		return nil, fmt.Errorf("pin %q: %w", p.ID, err)
	}
	key, isArray := strings.CutSuffix(p.Locator, "[]")
	v, ok := doc[key]
	if !ok {
		return nil, fmt.Errorf("pin %q: key %q not present — the file changed shape", p.ID, key)
	}
	if isArray {
		out := toStrings(v)
		if len(out) == 0 {
			return nil, fmt.Errorf("pin %q: %q is empty or not a string array", p.ID, key)
		}
		return out, nil
	}
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("pin %q: %q is not a string", p.ID, key)
	}
	return []string{s}, nil
}

var tomlKeyLine = `(?m)^\s*%s\s*=\s*"([^"]+)"`

// extractTOMLKey reads a bare `key = "value"` line. `.pr_agent.toml` puts these
// at section scope, and pulling in a TOML parser to read two string keys would
// be a dependency this check does not need.
func extractTOMLKey(p Pin, content []byte) ([]string, error) {
	re, err := regexp.Compile(fmt.Sprintf(tomlKeyLine, regexp.QuoteMeta(p.Locator)))
	if err != nil {
		return nil, fmt.Errorf("pin %q: %w", p.ID, err)
	}
	m := re.FindStringSubmatch(string(content))
	if m == nil {
		return nil, fmt.Errorf("pin %q: key %q not found", p.ID, p.Locator)
	}
	return []string{m[1]}, nil
}

var (
	jsoncBlock = regexp.MustCompile(`(?s)/\*.*?\*/`)
	jsoncLine  = regexp.MustCompile(`(?m)^\s*//.*$`)
)

// stripJSONComments removes the comment forms opencode.jsonc uses. Only
// whole-line `//` comments are stripped, never trailing ones, because a `//`
// inside a string value ("https://...") is not a comment and removing it would
// corrupt the very ids being checked.
func stripJSONComments(b []byte) []byte {
	b = jsoncBlock.ReplaceAll(b, nil)
	return jsoncLine.ReplaceAll(b, nil)
}

// SortedSiteFiles lists declared files for a scope, for stable diagnostics.
func (mp *ModelPins) SortedSiteFiles(scope string) []string {
	var out []string
	for _, s := range mp.Sites {
		if s.Scope == scope {
			out = append(out, s.File)
		}
	}
	sort.Strings(out)
	return out
}
