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

// CapabilityMapFile and CapabilityMapSchemaFile are the repo-relative paths to
// the compile-time capability registry and its declarative contract (ADR-027 §2,
// shaped by ADR-035).
//
// Not embedded, for the same reason model-map.json is not: an absent map that
// resolved to a build-time default would grant or deny tool access silently,
// which is what C15 forbids.
const (
	CapabilityMapFile       = "harness/capability-map.json"
	CapabilityMapSchemaFile = "harness/capability-map.schema.json"
)

// ValidateCapabilityMap checks doc against schema.
//
// Standard keywords are interpreted by santhosh-tekuri/jsonschema, the same
// draft-2020-12 implementation model-map uses. Unlike model-map this declares no
// `x-` cross-block rule: model-map's rule plumbing is a package-level closed set
// bound to that one schema, and generalising it to per-schema rule sets touches a
// file hardened over six adversarial review rounds. The single cross-block
// invariant here runs in checkVocabularyCoverage below instead, so the refactor
// can be its own deliberate change rather than riding along with a new registry.
func ValidateCapabilityMap(doc, schema []byte) error {
	compiler := jsonschema.NewCompiler()

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return fmt.Errorf("parse %s: %w", CapabilityMapSchemaFile, err)
	}
	// Opaque, stable resource id: a repo-relative path would be resolved against
	// the process working directory and leak a per-machine absolute path into
	// every error message.
	const schemaURL = "https://mlorentedev.github.io/dotfiles/capability-map.schema.json"
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("load %s: %w", CapabilityMapSchemaFile, err)
	}
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("%s is not a valid schema: %w", CapabilityMapSchemaFile, err)
	}

	d, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return fmt.Errorf("parse %s: %w", CapabilityMapFile, err)
	}
	if err := compiled.Validate(d); err != nil {
		return fmt.Errorf("%s does not satisfy %s: %w", CapabilityMapFile, CapabilityMapSchemaFile, err)
	}
	return checkVocabularyCoverage(d)
}

// checkVocabularyCoverage is the rule a stock JSON Schema cannot express: every
// harness must map every verb the `vocabulary` block declares.
//
// A partial harness is the dangerous shape, not an obviously broken one. It
// validates cleanly, and a persona declaring a verb the harness happens to omit
// then either fails at deploy or — worse, for a csv allow-list — renders a
// definition missing exactly the tool it needed, with no error anywhere. The
// vocabulary block exists so coverage is checked against a declared set rather
// than against whatever the first harness happened to list.
func checkVocabularyCoverage(doc any) error {
	root, ok := doc.(map[string]any)
	if !ok {
		return fmt.Errorf("%s is not a JSON object", CapabilityMapFile)
	}
	vocab := toStrings(root["vocabulary"])
	if len(vocab) == 0 {
		return fmt.Errorf("%s declares an empty vocabulary", CapabilityMapFile)
	}
	harnesses, ok := root["harnesses"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s declares no harnesses block", CapabilityMapFile)
	}

	for _, name := range sortedKeys(harnesses) {
		h, ok := harnesses[name].(map[string]any)
		if !ok {
			continue
		}
		caps, ok := h["capabilities"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s: harness %q declares no capabilities block", CapabilityMapFile, name)
		}
		if err := checkHarnessCoversVocabulary(name, caps, vocab); err != nil {
			return err
		}
	}
	return nil
}

// checkHarnessCoversVocabulary enforces both directions for one harness: it maps
// every declared verb, and it maps nothing else.
//
// The missing direction is the dangerous one — a partial harness validates
// cleanly and then renders a definition missing exactly the tool the persona
// asked for. The extra direction is quieter but still wrong: a mapping for a verb
// no record can ask for reads as coverage while being unreachable.
func checkHarnessCoversVocabulary(name string, caps map[string]any, vocab []string) error {
	var missing []string
	for _, v := range vocab {
		if _, present := caps[v]; !present {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf(
			"%s: harness %q maps no native names for %s — every harness must cover the whole "+
				"declared vocabulary. A partial harness validates cleanly and then renders a "+
				"definition missing exactly the tool the persona asked for",
			CapabilityMapFile, name, strings.Join(missing, ", "))
	}
	var extra []string
	for _, k := range sortedKeys(caps) {
		if !contains(vocab, k) {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		return fmt.Errorf(
			"%s: harness %q maps %s, which the vocabulary does not declare — "+
				"add it to `vocabulary` or drop it, but do not leave a mapping nothing can ask for",
			CapabilityMapFile, name, strings.Join(extra, ", "))
	}
	return nil
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

// LoadCapabilityMap reads the capability registry and validates it against the
// schema beside it. No fallback and no embedded default: where the map cannot be
// read, this errors (C15).
func LoadCapabilityMap(repoRoot string) (map[string]any, error) {
	mapPath := filepath.Join(repoRoot, CapabilityMapFile)
	schemaPath := filepath.Join(repoRoot, CapabilityMapSchemaFile)

	doc, err := os.ReadFile(mapPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w\nthis is not an empty capability map — nothing falls back to a default", CapabilityMapFile, err)
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w\nthe map cannot be trusted without the contract it declares against", CapabilityMapSchemaFile, err)
	}
	if err := ValidateCapabilityMap(doc, schema); err != nil {
		return nil, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(doc, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s: %w", CapabilityMapFile, err)
	}
	return parsed, nil
}

// ResolveCapabilities renders one harness's complete native value for a set of
// neutral capabilities, returning the frontmatter line ready to emit.
//
// It returns a WHOLE line — "tools: Read, Glob, Bash" — rather than a bare value,
// because the field name differs per harness and the caller should not have to
// know it. Both declared forms render on one line and both are valid YAML:
// `csv` produces a comma list, `decision-map` a YAML flow mapping.
//
// The set is resolved as a set, not capability by capability, and that is the
// point. A `csv` field is an ALLOW-LIST — naming a tool grants it and omitting
// one denies it — while a `decision-map` grants without denying. Emitting a
// per-capability token for a caller to concatenate would lose that distinction
// at exactly the layer that has to preserve it.
func ResolveCapabilities(m map[string]any, caps []string, harness string) (string, error) {
	h, err := harnessBlock(m, harness)
	if err != nil {
		return "", err
	}
	table, ok := h["capabilities"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("%s: harness %q declares no capabilities block", CapabilityMapFile, harness)
	}
	natives, err := collectNatives(caps, table, harness)
	if err != nil {
		return "", err
	}
	return renderNatives(h, natives, harness)
}

func harnessBlock(m map[string]any, harness string) (map[string]any, error) {
	harnesses, ok := m["harnesses"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s declares no harnesses block", CapabilityMapFile)
	}
	h, ok := harnesses[harness].(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"harness %q is not declared in %s (declared: %s)\n"+
				"a harness whose native tool vocabulary has not been verified is absent on purpose — "+
				"guessing native names would render a definition granting tools that do not exist",
			harness, CapabilityMapFile, strings.Join(sortedKeys(harnesses), ", "))
	}
	return h, nil
}

// collectNatives flattens the requested verbs into native names in declaration
// order, deduped across the set with the first occurrence winning. Deterministic
// so the rendered file does not churn between deploys.
func collectNatives(caps []string, table map[string]any, harness string) ([]string, error) {
	var natives []string
	seen := map[string]bool{}
	for _, c := range caps {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		raw, present := table[c]
		if !present {
			return nil, fmt.Errorf(
				"capability %q is not mapped for harness %q in %s (mapped: %s)",
				c, harness, CapabilityMapFile, strings.Join(sortedKeys(table), ", "))
		}
		for _, n := range toStrings(raw) {
			if !seen[n] {
				seen[n] = true
				natives = append(natives, n)
			}
		}
	}
	if len(natives) == 0 {
		return nil, fmt.Errorf(
			"no capabilities requested for harness %q — resolving to an empty value would render a "+
				"definition granting nothing, which is not the same as declaring none", harness)
	}
	return natives, nil
}

// renderNatives writes the harness's declared form.
//
// Nothing is escaped here, on purpose: a frontmatter line is not a shell command,
// and the values are held to identifier tokens by the schema instead. That is
// where a comma or a brace has to be stopped, because here it would not appear in
// the output — it would ALTER it. See TestCapabilityMapRejectsRenderBreakingNames.
func renderNatives(h map[string]any, natives []string, harness string) (string, error) {
	field, _ := h["field"].(string)
	switch form, _ := h["form"].(string); form {
	case "csv":
		return fmt.Sprintf("%s: %s", field, strings.Join(natives, ", ")), nil
	case "decision-map":
		grant, _ := h["grant"].(string)
		if strings.TrimSpace(grant) == "" {
			return "", fmt.Errorf(
				"%s: harness %q uses form %q but declares no `grant` value — "+
					"a decision map with no decision to write grants nothing",
				CapabilityMapFile, harness, form)
		}
		// Sorted: a flow mapping has no meaningful order, so a stable one keeps
		// the rendered file from churning on map iteration.
		sorted := append([]string(nil), natives...)
		sort.Strings(sorted)
		pairs := make([]string, 0, len(sorted))
		for _, n := range sorted {
			pairs = append(pairs, fmt.Sprintf("%s: %s", n, grant))
		}
		return fmt.Sprintf("%s: {%s}", field, strings.Join(pairs, ", ")), nil
	default:
		return "", fmt.Errorf(
			"%s: harness %q declares form %q, which this resolver does not render",
			CapabilityMapFile, harness, form)
	}
}
