package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// CLI-063 (#1339). Claude Code's settings.json is a file Claude rewrites while
// it runs, and the twins' merge_claude_settings applied a per-key policy to it:
// `env` and `enabledPlugins` merge nested, `permissions.allow` unions. Each case
// below keeps a key only the box has; under the top-level replace the proposal
// was written against, every one of them is lost.
func TestPerKeyMergePolicy(t *testing.T) {
	cases := []struct {
		name, box, tmpl string
		keep, apply     []string // paths that must hold after the merge
	}{
		{
			name:  "env merges nested",
			box:   `{"env":{"BOX_ONLY":"1","SHARED":"old"}}`,
			tmpl:  `{"env":{"SHARED":"new","TMPL":"1"}}`,
			keep:  []string{`"BOX_ONLY": "1"`},
			apply: []string{`"SHARED": "new"`, `"TMPL": "1"`},
		},
		{
			name:  "enabledPlugins merges nested",
			box:   `{"enabledPlugins":{"box@m":true,"shared@m":false}}`,
			tmpl:  `{"enabledPlugins":{"shared@m":true,"tmpl@m":true}}`,
			keep:  []string{`"box@m": true`},
			apply: []string{`"shared@m": true`, `"tmpl@m": true`},
		},
		{
			name:  "permissions.allow unions without duplicates",
			box:   `{"permissions":{"allow":["Bash(box:*)","Bash(ls:*)"]}}`,
			tmpl:  `{"permissions":{"allow":["Bash(ls:*)","WebSearch"]}}`,
			keep:  []string{`"Bash(box:*)"`},
			apply: []string{`"WebSearch"`},
		},
		{
			// The twins replaced attribution whole so a stale trailer could not
			// survive. The merge overwrites each scalar the template declares,
			// which clears it just the same.
			name:  "attribution clears a stale trailer",
			box:   `{"attribution":{"commit":"Co-Authored-By: x","pr":"Generated with x","sessionUrl":true}}`,
			tmpl:  `{"attribution":{"commit":"","pr":"","sessionUrl":false}}`,
			apply: []string{`"commit": ""`, `"pr": ""`, `"sessionUrl": false`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dst := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(dst, []byte(tc.box), 0o644); err != nil {
				t.Fatal(err)
			}
			out, _, err := mergeInto(dst, []byte(tc.tmpl))
			if err != nil {
				t.Fatal(err)
			}
			got := string(out)
			for _, want := range append(tc.keep, tc.apply...) {
				if !strings.Contains(got, want) {
					t.Errorf("%s missing from\n%s", want, got)
				}
			}
			if strings.Count(got, `"Bash(ls:*)"`) > 1 {
				t.Errorf("a shared allow entry was duplicated:\n%s", got)
			}
		})
	}
}

// The twins' policy was an allow-list, so a template key it did not name was a
// silent no-op on every existing box (outputStyle). A merge writes every
// template key, which closes that defect and opens its mirror image: the
// TEMPLATE is now the allow-list. `hooks` belongs to `dotf harness bind`
// (HARNESS-045), so it must never appear in the template a merge writes from.
func TestClaudeSettingsTemplateHasNoHooks(t *testing.T) {
	tmpl := readObject(t, filepath.Join("..", "..", "..", "ai", "claude", "settings.json"))
	if _, ok := tmpl["hooks"]; ok {
		t.Fatal("ai/claude/settings.json carries a hooks key: a merge would make dotf deploy a second hooks writer")
	}
}

// Joins the engine and the shipped data, as the agy test does: the real entry,
// the real template, deployed over a box that holds what Claude and the user
// write at runtime.
func TestClaudeEntriesDeclared(t *testing.T) {
	const root = "../../.."
	data, err := os.ReadFile(filepath.Join(root, ManifestRel))
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	cfg := m.Lookup("claude-settings")
	if cfg == nil {
		t.Fatal("claude-settings is not declared in " + ManifestRel)
	}
	if cfg.strategy() != StrategyMerge || cfg.Dst != "{CLAUDE_CONFIG_DIR}/settings.json" || cfg.Requires != "claude" {
		t.Fatalf("claude-settings = %+v; want merge into {CLAUDE_CONFIG_DIR}/settings.json, requiring claude", *cfg)
	}

	configDir := t.TempDir()
	dst := filepath.Join(configDir, "settings.json")
	hooks := map[string]any{"SessionStart": []any{map[string]any{"matcher": "third-party"}}}
	box := map[string]any{
		"model":       "stale",
		"hooks":       hooks,
		"statusLine":  map[string]any{"type": "command", "command": "box"},
		"env":         map[string]any{"BOX_ONLY": "1"},
		"permissions": map[string]any{"allow": []any{"Bash(box:*)"}},
	}
	raw, _ := json.Marshal(box)
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	resolve := func(name string) string {
		if name == "CLAUDE_CONFIG_DIR" {
			return configDir
		}
		return ""
	}
	if _, err := Deploy(*cfg, root, t.TempDir(), resolve, nil, false); err != nil {
		t.Fatal(err)
	}

	got := readObject(t, dst)
	tmpl := readObject(t, filepath.Join(root, cfg.Src))
	// Every template key reaches the box: the allow-list defect (outputStyle)
	// is closed only if no key, present or future, is skipped.
	for key, want := range tmpl {
		if !contains(got[key], want) {
			t.Errorf("template key %q did not reach the box: got %v", key, got[key])
		}
	}
	if got["model"] != tmpl["model"] {
		t.Errorf("model = %v, the template declares %v", got["model"], tmpl["model"])
	}
	if !reflect.DeepEqual(jsonRoundTrip(t, got["hooks"]), jsonRoundTrip(t, hooks)) {
		t.Errorf("hooks were rewritten: %v", got["hooks"])
	}
	if got["statusLine"] == nil || got["env"].(map[string]any)["BOX_ONLY"] != "1" {
		t.Errorf("a box-only key was lost: %v", got)
	}
	perms := got["permissions"].(map[string]any)
	if !strings.Contains(string(jsonRoundTrip(t, perms["allow"])), "Bash(box:*)") {
		t.Errorf("a box-only allow entry was lost: %v", perms["allow"])
	}
	// Divergence 18: the twins never wrote deny on an existing box; the merge does.
	if deny, _ := perms["deny"].([]any); len(deny) == 0 {
		t.Error("the template's deny rules did not reach an existing box")
	}
	// Divergence 19: attribution must hold the template's values (no trailer).
	if !reflect.DeepEqual(jsonRoundTrip(t, got["attribution"]), jsonRoundTrip(t, tmpl["attribution"])) {
		t.Errorf("attribution = %v, the template declares %v", got["attribution"], tmpl["attribution"])
	}
}

func jsonRoundTrip(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// contains reports whether got holds everything want declares: objects by key,
// recursively; lists by membership; anything else by equality.
func contains(got, want any) bool {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range w {
			if !contains(g[k], v) {
				return false
			}
		}
		return true
	case []any:
		g, ok := got.([]any)
		if !ok {
			return false
		}
		for _, v := range w {
			found := false
			for _, x := range g {
				if jsonEqual(x, v) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default:
		return jsonEqual(got, want)
	}
}
