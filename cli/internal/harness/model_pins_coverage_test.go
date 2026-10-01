package harness

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Round-1 review of HARNESS-067, finding 1 (Blocker): three committed files
// carried routing pins that no site declared, and the guard could not see them
// because it only ever iterates what the registry declares. A guard that reads
// its own declaration cannot notice the declaration is incomplete.
//
// This sweep looks from the other side: every routing-shaped key in a JSON file
// under ai/, at any depth, must be located by a declared pin or named in the
// registry's `excluded` list with a reason. A new pin site then fails here until
// someone declares it or says why not.
var routingKey = regexp.MustCompile(`(?i)^(model|defaultModel|advisorModel|small_model|smallModel)$`)

func TestEveryRoutingKeyUnderAIIsDeclared(t *testing.T) {
	root := repoRootForTest(t)
	pins, err := LoadModelPins(root)
	if err != nil {
		t.Fatalf("registry does not load: %v", err)
	}

	located := map[string]bool{}
	for _, site := range pins.Sites {
		if site.Scope != "repo" {
			continue
		}
		for _, p := range site.Pins {
			if p.Kind != "json-path" {
				continue
			}
			for _, path := range expandLocator(t, root, site.File, p.Locator) {
				located[site.File+" "+path] = true
			}
		}
	}
	for _, x := range pins.Excluded {
		located[x.File+" "+x.Locator] = true
	}

	found := 0
	err = filepath.WalkDir(filepath.Join(root, "ai"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".json") && !strings.HasSuffix(path, ".jsonc") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc any
		if json.Unmarshal(stripJSONComments(b), &doc) != nil {
			// Not every file under ai/ is JSON that this sweep can read; the
			// routing files all are, and the declared ones are parsed for real
			// by TestEveryRepoRoutingPinResolvesInTheMap.
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		for _, key := range routingKeys(doc, "") {
			found++
			if !located[rel+" "+key] {
				t.Errorf("%s: %q is a routing-shaped key that no pin in %s locates.\n"+
					"Declare it as a pin, or add it to `excluded` with the reason it does not route.",
					rel, key, ModelPinsFile)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found == 0 {
		t.Fatal("the sweep found no routing key at all under ai/ — it is reading nothing, not finding nothing")
	}
}

// The TOML half of the same sweep. `.pr_agent.toml` is the only TOML pin site,
// and its `fallback_models` was undeclared in round 1 (finding 2). Any key whose
// name carries "model" and whose value is a string or a list must be a pin or
// an exclusion; numeric budgets such as `max_model_tokens` are not ids.
var tomlModelKey = regexp.MustCompile(`(?m)^\s*(\w*model\w*)\s*=\s*["\[]`)

func TestEveryModelKeyInATOMLSiteIsDeclared(t *testing.T) {
	root := repoRootForTest(t)
	pins, err := LoadModelPins(root)
	if err != nil {
		t.Fatalf("registry does not load: %v", err)
	}
	located := map[string]bool{}
	files := map[string]bool{}
	for _, site := range pins.Sites {
		for _, p := range site.Pins {
			if site.Scope == "repo" && p.Kind == "toml-key" {
				files[site.File] = true
				located[site.File+" "+strings.TrimSuffix(p.Locator, "[]")] = true
			}
		}
	}
	for _, x := range pins.Excluded {
		located[x.File+" "+x.Locator] = true
	}
	if len(files) == 0 {
		t.Fatal("no repo-scoped TOML site declared; this sweep would pass over nothing")
	}
	for file := range files {
		b, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range tomlModelKey.FindAllStringSubmatch(string(b), -1) {
			if !located[file+" "+m[1]] {
				t.Errorf("%s: %q carries a model id that no pin locates; declare it or exclude it with a reason", file, m[1])
			}
		}
	}
}

// routingKeys lists the dotted path of every string-valued routing-shaped key.
// Paths step through objects only: a key inside an array is a catalog entry
// (pi's models.json), and catalogs are not routing (#1244).
func routingKeys(n any, prefix string) []string {
	obj, ok := n.(map[string]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []string
	for _, k := range keys {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if _, isString := obj[k].(string); isString && routingKey.MatchString(k) {
			out = append(out, path)
		}
		out = append(out, routingKeys(obj[k], path)...)
	}
	return out
}

// expandLocator turns a json-path locator with `*` segments into the concrete
// paths it covers in the real file, so `agent.*.model` covers `agent.plan.model`.
func expandLocator(t *testing.T, root, file, locator string) []string {
	t.Helper()
	locator = strings.TrimSuffix(locator, "[]")
	if !strings.Contains(locator, "*") {
		return []string{locator}
	}
	b, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	var doc any
	if err := json.Unmarshal(stripJSONComments(b), &doc); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	paths := []string{""}
	nodes := []any{doc}
	for _, seg := range strings.Split(locator, ".") {
		var nextPaths []string
		var nextNodes []any
		for i, n := range nodes {
			obj, ok := n.(map[string]any)
			if !ok {
				continue
			}
			keys := []string{seg}
			if seg == "*" {
				keys = keys[:0]
				for k := range obj {
					keys = append(keys, k)
				}
			}
			for _, k := range keys {
				if v, ok := obj[k]; ok {
					nextPaths = append(nextPaths, strings.TrimPrefix(paths[i]+"."+k, "."))
					nextNodes = append(nextNodes, v)
				}
			}
		}
		paths, nodes = nextPaths, nextNodes
	}
	return paths
}

// Finding 2's regression test, and the per-pin form of AC3. For EVERY declared
// pin, a dead id written in that pin's own spelling must come back as
// VerdictUnknown. The review injected dead ids into three undeclared pins and
// the guard still passed; deriving the table from the registry means a pin
// added later is covered without anyone remembering to add a row.
//
// The dead id carries the declared spelling on purpose: a dead id with the
// prefix dropped would fail as misspelled, which proves the spelling rule and
// not that the guard sees a dead model behind a correct spelling.
func TestEveryDeclaredPinRejectsADeadID(t *testing.T) {
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

	covered := 0
	for _, site := range pins.Sites {
		for _, p := range site.Pins {
			t.Run(p.ID, func(t *testing.T) {
				dead := deadIDIn(p)
				var content []byte
				if site.Scope == "repo" {
					content = injectDeadID(t, root, site.File, p, dead)
				} else {
					content = deployedShape(t, p, dead)
				}
				values, err := Extract(p, content)
				if err != nil {
					t.Fatalf("the injected file no longer extracts, so this would prove the locator, not the check: %v", err)
				}
				for _, v := range values {
					if v != dead {
						continue
					}
					if got := Check(p, v, qualified, bare); got != VerdictUnknown {
						t.Errorf("a dead id %q in pin %q came back %v, want VerdictUnknown", v, p.ID, got)
					}
					covered++
					return
				}
				t.Errorf("the dead id %q was injected but pin %q did not extract it: %v", dead, p.ID, values)
			})
		}
	}
	if covered == 0 {
		t.Fatal("no pin was injected — a test over nothing passes")
	}
}

// deadIDIn spells a model id that cannot resolve the way the pin's site spells ids.
func deadIDIn(p Pin) string {
	if p.Spelling == "display-name" {
		return "Dead Model 9999 (High)"
	}
	return p.Prefix + "dead-model-9999" + p.Suffix
}

// injectDeadID rewrites the real file with the first value the pin extracts
// replaced by the dead id, everywhere it occurs.
func injectDeadID(t *testing.T, root, file string, p Pin, dead string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	values, err := Extract(p, b)
	if err != nil {
		t.Fatalf("%s: the real file does not extract: %v", file, err)
	}
	return []byte(strings.ReplaceAll(string(b), values[0], dead))
}

// deployedShape builds the smallest document a json-path pin reads, for a
// deployed site the repository holds no copy of. Any other kind in deployed
// scope stops the test, so a new one cannot slip past the injection uncovered.
func deployedShape(t *testing.T, p Pin, dead string) []byte {
	t.Helper()
	if p.Kind != "json-path" {
		t.Fatalf("pin %q: deployed kind %q has no document shape here; extend deployedShape", p.ID, p.Kind)
	}
	locator, isArray := strings.CutSuffix(p.Locator, "[]")
	segs := strings.Split(locator, ".")
	var leaf any = dead
	if isArray {
		leaf = []any{dead}
	}
	for i := len(segs) - 1; i >= 0; i-- {
		key := segs[i]
		if key == "*" {
			key = "x"
		}
		leaf = map[string]any{key: leaf}
	}
	b, err := json.Marshal(leaf)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
