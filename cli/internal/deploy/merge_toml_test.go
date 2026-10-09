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

// `paths` rewrites the string values of a JSON document; on a TOML file it
// would fail at deploy time, on every machine. The manifest is refused instead.
func TestParseManifest_RefusesPathsOnATOMLConfig(t *testing.T) {
	_, err := ParseManifest([]byte(`{"version":3,"configs":[{"name":"x","src":"a.toml","dst":"{HOME}/b.toml","paths":"native"}]}`))
	if err == nil || !strings.Contains(err.Error(), `config "x": paths`) {
		t.Errorf("want a paths error naming the config, got %v", err)
	}
}

// A merge source that names no key deploys nothing and reports success, so an
// emptied or truncated config would read as in sync forever. JSON refused an
// empty file only by accident (it does not parse); `{}` and an empty or
// comment-only TOML file parse fine, so the refusal is the merge's, not the
// decoder's.
func TestDeploy_MergeRefusesASourceThatManagesNoKey(t *testing.T) {
	for name, src := range map[string]string{"empty TOML": "", "comment-only TOML": "# nothing yet\n"} {
		t.Run(name, func(t *testing.T) {
			root := repoWithHerdrConfig(t, src)
			_, err := Deploy(herdrConfig(), root, t.TempDir(), noResolve, nil, false)
			if err == nil || !strings.Contains(err.Error(), "manages no key") {
				t.Errorf("want a refusal naming the empty source, got %v", err)
			}
		})
	}
	t.Run("empty JSON object", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "s.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		c := Config{Name: "x", Src: "s.json", Dst: "{HOME}/d.json", Mode: "0644", Strategy: StrategyMerge}
		_, err := Deploy(c, root, t.TempDir(), noResolve, nil, false)
		if err == nil || !strings.Contains(err.Error(), "manages no key") {
			t.Errorf("want a refusal naming the empty source, got %v", err)
		}
	})
}
