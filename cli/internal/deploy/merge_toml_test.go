package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// herdr writes its own config.toml: onboarding writes `onboarding = false`, and
// its Settings screen saves what the user picks there. The repo owns the keys
// it declares and nothing else, which is the merge strategy's contract, so a
// TOML destination merges the way a JSON one does (#2013 H4).
func herdrConfig() Config {
	return Config{Name: "herdr", Src: "ai/herdr/config.toml", Dst: "{HOME}/.config/herdr/config.toml", Mode: "0644", Strategy: StrategyMerge}
}

func repoWithHerdrConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, "ai", "herdr", "config.toml")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeHerdrDst(t *testing.T, home, body string) string {
	t.Helper()
	dst := filepath.Join(home, ".config", "herdr", "config.toml")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dst
}

func readTOML(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // test fixture
	if err != nil {
		t.Fatalf("destination missing: %v", err)
	}
	var m map[string]any
	if err := toml.Unmarshal(raw, &m); err != nil {
		t.Fatalf("destination is not TOML: %v\n%s", err, raw)
	}
	return m
}

const herdrManaged = `
[keys]
prefix = "ctrl+s"

[update]
version_check = false

[advanced]
scrollback_limit_bytes = 10000000
`

func TestDeploy_TOMLMergeKeepsWhatTheToolWrote(t *testing.T) {
	root := repoWithHerdrConfig(t, herdrManaged)
	home := t.TempDir()
	dst := writeHerdrDst(t, home, `onboarding = false

[keys]
prefix = "ctrl+b"
new_tab = "prefix+t"

[ui]
status_indicators = "symbols"
`)

	res, err := Deploy(herdrConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Fatal("a managed key differed, so the merge must change the destination")
	}
	got := readTOML(t, dst)
	keys, _ := got["keys"].(map[string]any)
	ui, _ := got["ui"].(map[string]any)
	update, _ := got["update"].(map[string]any)
	if keys["prefix"] != "ctrl+s" {
		t.Errorf("managed key not written: keys.prefix = %v", keys["prefix"])
	}
	if keys["new_tab"] != "prefix+t" || ui["status_indicators"] != "symbols" || got["onboarding"] != false {
		t.Errorf("a key the tool wrote was lost: %v", got)
	}
	if update["version_check"] != false {
		t.Errorf("a managed table absent from the destination was not added: %v", got)
	}
}

// herdr rewrites the file in its own layout. In sync is a question about
// values, not bytes: comments, key order and inline tables must not read as
// drift, or every setup run would rewrite the file and doctor would never be
// green.
func TestDeploy_TOMLMergeIsInSyncAfterTheToolReformatsIt(t *testing.T) {
	root := repoWithHerdrConfig(t, herdrManaged)
	home := t.TempDir()
	dst := writeHerdrDst(t, home, `# written by herdr
onboarding = false
advanced = { scrollback_limit_bytes = 10000000 }

[update]
version_check = false # checked by dotf

[keys]
new_tab = "prefix+t"
prefix = "ctrl+s"
`)
	before, err := os.ReadFile(dst) //nolint:gosec // test fixture
	if err != nil {
		t.Fatal(err)
	}

	plan, err := PlanConfig(herdrConfig(), root, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changed {
		t.Errorf("every managed value is already there; the plan must report in sync:\n%s", plan.Content)
	}
	res, err := Deploy(herdrConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(dst) //nolint:gosec // test fixture
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed || string(before) != string(after) {
		t.Error("an in-sync merge rewrote the file")
	}
}

func TestDeploy_TOMLMergeCreatesAnAbsentDestination(t *testing.T) {
	root := repoWithHerdrConfig(t, herdrManaged)
	home := t.TempDir()

	res, err := Deploy(herdrConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	got := readTOML(t, res.Dst)
	if keys, _ := got["keys"].(map[string]any); keys["prefix"] != "ctrl+s" {
		t.Errorf("absent destination not created from the source: %v", got)
	}
	second, err := Deploy(herdrConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed {
		t.Error("a second merge of the same source must report no change")
	}
}

func TestDeploy_TOMLMergeRefusesADestinationItCannotRead(t *testing.T) {
	root := repoWithHerdrConfig(t, herdrManaged)
	home := t.TempDir()
	const broken = "prefix = = \"ctrl+b\"\n"
	dst := writeHerdrDst(t, home, broken)

	_, err := Deploy(herdrConfig(), root, home, noResolve, nil, false)
	if err == nil {
		t.Fatal("merging into unreadable TOML must fail, not replace it")
	}
	if !strings.Contains(err.Error(), `"herdr"`) || !strings.Contains(err.Error(), "not a TOML table") {
		t.Errorf("error must name the config and the cause: %v", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != broken { //nolint:gosec // test fixture
		t.Errorf("the destination was touched: %q", got)
	}
}

// `paths` rewrites the string values of the source before it is installed. A
// TOML source needs version 6 (TestParseManifest_PathsOnATOMLSourceNeedsVersion6),
// so a version 3 manifest refuses it. What decides is the source's format: a
// replace installs the source's bytes whatever the destination's extension
// says.
func TestParseManifest_RefusesPathsOnATOMLSource(t *testing.T) {
	for _, pair := range [][2]string{{"a.toml", "{HOME}/b.toml"}, {"a.toml", "{HOME}/b.json"}} {
		_, err := ParseManifest([]byte(`{"version":3,"configs":[{"name":"x","src":"` + pair[0] + `","dst":"` + pair[1] + `","paths":"native"}]}`))
		if err == nil || !strings.Contains(err.Error(), `config "x": paths`) {
			t.Errorf("%s -> %s: want a paths error naming the config, got %v", pair[0], pair[1], err)
		}
	}
	if _, err := ParseManifest([]byte(`{"version":3,"configs":[{"name":"x","src":"a.json","dst":"{HOME}/b.toml","paths":"native"}]}`)); err != nil {
		t.Errorf("paths on a JSON source must not be refused: %v", err)
	}
}

// A merge source that names no key deploys nothing and reports success, so an
// emptied or truncated config would read as in sync forever. JSON refused an
// empty file only by accident (it does not parse); `{}` and an empty or
// comment-only TOML file parse fine, so the refusal is the merge's, not the
// decoder's.
func TestDeploy_MergeRefusesASourceThatManagesNoKey(t *testing.T) {
	for name, src := range map[string]string{"empty TOML": "", "comment-only TOML": "# nothing yet\n", "header-only TOML": "[keys]\n[update]\n"} {
		t.Run(name, func(t *testing.T) {
			root := repoWithHerdrConfig(t, src)
			_, err := Deploy(herdrConfig(), root, t.TempDir(), noResolve, nil, false)
			if err == nil || !strings.Contains(err.Error(), "manages no key") {
				t.Errorf("want a refusal naming the empty source, got %v", err)
			}
		})
	}
	for name, body := range map[string]string{"empty JSON object": "{}\n", "JSON of empty objects": `{"a": {}, "b": {"c": {}}}`} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "s.json"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			c := Config{Name: "x", Src: "s.json", Dst: "{HOME}/d.json", Mode: "0644", Strategy: StrategyMerge}
			_, err := Deploy(c, root, t.TempDir(), noResolve, nil, false)
			if err == nil || !strings.Contains(err.Error(), "manages no key") {
				t.Errorf("want a refusal naming the empty source, got %v", err)
			}
		})
	}
	// An empty list is a value: the source sets it.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "s.json"), []byte(`{"a": {"list": []}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := Config{Name: "x", Src: "s.json", Dst: "{HOME}/d.json", Mode: "0644", Strategy: StrategyMerge}
	if _, err := Deploy(c, root, t.TempDir(), noResolve, nil, false); err != nil {
		t.Errorf("an empty list is a managed value: %v", err)
	}
}

// A merge reads its source in the destination's format, so a source of the
// other format fails on every machine at deploy time with a misleading
// "source is not a TOML table". The manifest is refused instead.
func TestParseManifest_RefusesAMergeWhoseSourceAndDestinationFormatsDiffer(t *testing.T) {
	for _, pair := range [][2]string{{"a.json", "{HOME}/b.toml"}, {"a.toml", "{HOME}/b.json"}} {
		_, err := ParseManifest([]byte(`{"version":3,"configs":[{"name":"x","src":"` + pair[0] + `","dst":"` + pair[1] + `","strategy":"merge"}]}`))
		if err == nil || !strings.Contains(err.Error(), `config "x": merge`) {
			t.Errorf("%s -> %s: want a format mismatch error naming the config, got %v", pair[0], pair[1], err)
		}
	}
	// replace copies bytes and never parses, so it may cross formats.
	if _, err := ParseManifest([]byte(`{"version":3,"configs":[{"name":"x","src":"a.json","dst":"{HOME}/b.toml"}]}`)); err != nil {
		t.Errorf("replace must not be refused: %v", err)
	}
}

// Lists union, so an entry the tool appended inside a managed key survives a
// deploy. The union is written against `[]any`, which is how JSON decodes an
// array; go-toml/v2 decodes into `[]any` as well, and this pins it, because a
// decoder that produced a typed slice would make the repo's list replace the
// tool's without an error.
func TestDeploy_TOMLMergeUnionsAListTheToolAppendedTo(t *testing.T) {
	root := repoWithHerdrConfig(t, "[keys]\nprefix = \"ctrl+s\"\nextra = [\"a\", \"b\"]\n")
	home := t.TempDir()
	dst := writeHerdrDst(t, home, "[keys]\nprefix = \"ctrl+s\"\nextra = [\"a\", \"b\", \"tool\"]\n")
	if _, err := Deploy(herdrConfig(), root, home, noResolve, nil, false); err != nil {
		t.Fatal(err)
	}
	keys, _ := readTOML(t, dst)["keys"].(map[string]any)
	got, _ := keys["extra"].([]any)
	if len(got) != 3 || got[2] != "tool" {
		t.Errorf("the element the tool appended was lost: keys.extra = %#v", keys["extra"])
	}
}

// #2013 H4: the SHIPPED herdr entry and the SHIPPED ai/herdr/config.toml, as
// the agy test above does for its config. The tests before this one prove the
// engine with a fixture; this one fails if the entry stops being a merge or
// its dst stops resolving through HERDR_CONFIG_DIR, and if the file starts
// naming one of the keys the fixture below has herdr write (onboarding,
// ui.accent, keys.zoom). It cannot see every key herdr may write; the file's
// header states the rule for the rest.
func TestDeploy_TheShippedHerdrEntryKeepsHerdrsOwnKeys(t *testing.T) {
	const root = "../../.."
	raw, err := os.ReadFile(filepath.Join(root, ManifestRel))
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	for _, c := range m.Configs {
		if c.Name == "herdr" {
			cfg = c
		}
	}
	if cfg.Name == "" || cfg.Strategy != StrategyMerge || cfg.Requires != "herdr" {
		t.Fatalf("want a merge entry named herdr that requires herdr, got %+v", cfg)
	}
	home := t.TempDir()
	dst := filepath.Join(home, "herdr-config", "config.toml")
	resolve := func(name string) string {
		if name == "HERDR_CONFIG_DIR" {
			return filepath.Dir(dst)
		}
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	// What herdr itself writes: onboarding's answer and a Settings choice.
	if err := os.WriteFile(dst, []byte("onboarding = false\n\n[keys]\nprefix = \"ctrl+b\"\nzoom = \"prefix+z\"\n\n[ui]\naccent = \"magenta\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Deploy(cfg, root, home, resolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Fatal("the prefix differed, so the first deploy must change the file")
	}
	got := readTOML(t, dst)
	keys, _ := got["keys"].(map[string]any)
	ui, _ := got["ui"].(map[string]any)
	worktrees, _ := got["worktrees"].(map[string]any)
	if keys["prefix"] == "ctrl+b" || keys["prefix"] == nil {
		t.Errorf("keys.prefix = %v: D6 wants a prefix that is not tmux's ctrl+b", keys["prefix"])
	}
	if worktrees["directory"] == nil {
		t.Errorf("worktrees.directory not written: %v", got)
	}
	if got["onboarding"] != false || keys["zoom"] != "prefix+z" || ui["accent"] != "magenta" {
		t.Errorf("a key herdr wrote was lost: %v", got)
	}
	// H9: lazygit replaces Orca's diff viewer, as a popup over the focused
	// pane, which herdr opens in that pane's directory. The key is pinned: it
	// was chosen against herdr's default keymap, which `herdr config check`
	// does not compare it with.
	lazygit := false
	cmds, _ := keys["command"].([]any)
	for _, c := range cmds {
		if b, _ := c.(map[string]any); b["command"] == "lazygit" && b["type"] == "popup" && b["key"] == "prefix+d" {
			lazygit = true
		}
	}
	if !lazygit {
		t.Errorf("no lazygit popup on prefix+d in keys.command: %v", keys["command"])
	}

	plan, err := PlanConfig(cfg, root, home, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changed {
		t.Errorf("a second deploy must be in sync:\n%s", plan.Content)
	}
}

// herdr's custom command keys are an array of tables (`[[keys.command]]`), the
// first such value a TOML merge carries (#2013 H9). A binding the user added
// in herdr survives beside the repo's, the merged file still holds tables,
// and a second merge is in sync, which the union only gives if the decoded
// tables compare equal after a round trip.
func TestDeploy_TOMLMergeUnionsAnArrayOfTables(t *testing.T) {
	root := repoWithHerdrConfig(t, `[keys]
prefix = "ctrl+s"

[[keys.command]]
key = "prefix+d"
type = "popup"
command = "lazygit"
`)
	home := t.TempDir()
	dst := writeHerdrDst(t, home, `[keys]
prefix = "ctrl+s"

[[keys.command]]
key = "prefix+alt+t"
type = "pane"
command = "htop"
`)
	if _, err := Deploy(herdrConfig(), root, home, noResolve, nil, false); err != nil {
		t.Fatal(err)
	}
	keys, _ := readTOML(t, dst)["keys"].(map[string]any)
	cmds, _ := keys["command"].([]any)
	var got []string
	for _, c := range cmds {
		table, ok := c.(map[string]any)
		if !ok {
			t.Fatalf("keys.command element is %T, want a table", c)
		}
		got = append(got, table["command"].(string))
	}
	if strings.Join(got, ",") != "htop,lazygit" {
		t.Errorf("keys.command = %v, want the user's htop then the repo's lazygit", got)
	}
	again, err := Deploy(herdrConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed {
		t.Error("a second merge of the same array of tables must report no change")
	}
}
