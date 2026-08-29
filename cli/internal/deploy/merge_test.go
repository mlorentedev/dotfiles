package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoWithCopilotSettings writes the merge source the way ai/copilot/settings.json
// is shaped: three managed keys, nothing else.
func repoWithCopilotSettings(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, "ai", "copilot", "settings.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func copilotSettingsConfig() Config {
	return Config{Name: "copilot-settings", Src: "ai/copilot/settings.json", Dst: "{HOME}/.copilot/settings.json", Mode: "0644", Strategy: StrategyMerge}
}

func writeDst(t *testing.T, home, body string) string {
	t.Helper()
	dst := filepath.Join(home, ".copilot", "settings.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dst
}

func readObject(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // test fixture
	if err != nil {
		t.Fatalf("destination missing: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("destination is not JSON: %v\n%s", err, raw)
	}
	return m
}

// AC1 (AI-039, #1322): the repo owns three keys of Copilot's settings.json; the
// box owns the rest (allowedUrls, effortLevel, ...). A verbatim copy would wipe
// the box's keys on every setup run, which is why this is a merge and not a
// deploy of the whole file.
func TestDeploy_MergePreservesUnmanagedKeys(t *testing.T) {
	root := repoWithCopilotSettings(t, `{"model":"new","includeCoAuthoredBy":false,"autoUpdate":false}`)
	home := t.TempDir()
	writeDst(t, home, `{"allowedUrls":["https://github.com"],"effortLevel":"max","model":"old","includeCoAuthoredBy":true}`)

	res, err := Deploy(copilotSettingsConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Error("a destination with a differing managed key must report a change")
	}
	got := readObject(t, res.Dst)
	if got["model"] != "new" || got["includeCoAuthoredBy"] != false || got["autoUpdate"] != false {
		t.Errorf("managed keys not written: %v", got)
	}
	if got["effortLevel"] != "max" {
		t.Errorf("unmanaged key effortLevel was lost: %v", got)
	}
	urls, _ := got["allowedUrls"].([]any)
	if len(urls) != 1 || urls[0] != "https://github.com" {
		t.Errorf("unmanaged key allowedUrls was lost: %v", got)
	}
}

func TestDeploy_MergeCreatesAnAbsentDestination(t *testing.T) {
	root := repoWithCopilotSettings(t, `{"model":"m","autoUpdate":false}`)
	home := t.TempDir()

	res, err := Deploy(copilotSettingsConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Error("creating the destination is a change")
	}
	got := readObject(t, res.Dst)
	if len(got) != 2 || got["model"] != "m" || got["autoUpdate"] != false {
		t.Errorf("want exactly the managed keys, got %v", got)
	}
}

// Copilot rewrites config.json with a `// User settings belong in settings.json`
// header. Merge must read past it — and, when the managed keys already match,
// must leave the file byte-identical rather than rewrite it without the header
// and have the CLI put the header back: that loop is a deploy that says
// "deployed" on every run and a doctor row that flaps.
func TestDeploy_MergeToleratesTheCLIHeaderAndDoesNotChurnIt(t *testing.T) {
	root := repoWithCopilotSettings(t, `{"trustedFolders":["/home/u/Projects"]}`)
	home := t.TempDir()
	header := "// User settings belong in settings.json.\n// This file is managed automatically.\n"
	dst := writeDst(t, home, header+`{"trustedFolders":["/home/u/Projects"],"firstLaunchAt":"2026-03-11T00:00:00.000Z"}`)
	before, err := os.ReadFile(dst) //nolint:gosec // test fixture
	if err != nil {
		t.Fatal(err)
	}

	res, err := Deploy(copilotSettingsConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed {
		t.Error("managed keys already equal → no change")
	}
	after, err := os.ReadFile(dst) //nolint:gosec // test fixture
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("an in-sync merge rewrote the file (header lost):\n%s", after)
	}

	// Now a real difference: the header is read past, the unmanaged key kept.
	root2 := repoWithCopilotSettings(t, `{"trustedFolders":["/home/u/Projects","/home/u/Work"]}`)
	res, err = Deploy(copilotSettingsConfig(), root2, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Error("a differing managed key behind the header must be seen")
	}
	got := readObject(t, dst)
	if got["firstLaunchAt"] != "2026-03-11T00:00:00.000Z" {
		t.Errorf("unmanaged key behind the header was lost: %v", got)
	}
	if fs, _ := got["trustedFolders"].([]any); len(fs) != 2 {
		t.Errorf("managed array not replaced whole: %v", got)
	}
}

func TestDeploy_MergeIsIdempotentAndDoesNotRewrite(t *testing.T) {
	root := repoWithCopilotSettings(t, `{"model":"m","includeCoAuthoredBy":false}`)
	home := t.TempDir()
	writeDst(t, home, `{"effortLevel":"max"}`)

	first, err := Deploy(copilotSettingsConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed {
		t.Fatal("first merge must change the destination")
	}
	before, err := os.Stat(first.Dst)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Deploy(copilotSettingsConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed {
		t.Error("a second merge of the same source must report no change")
	}
	after, err := os.Stat(first.Dst)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("an in-sync merge rewrote the file; mtime moved")
	}
}

func TestDeploy_MergeRejectsANonObjectDestinationByName(t *testing.T) {
	// `null` is the case that does not fail at Unmarshal: it yields a nil map,
	// and the first assignment into it would panic rather than error.
	for _, dst := range []string{`[1, 2, 3]`, `null`, `"text"`} {
		t.Run(dst, func(t *testing.T) {
			root := repoWithCopilotSettings(t, `{"model":"m"}`)
			home := t.TempDir()
			writeDst(t, home, dst)

			_, err := Deploy(copilotSettingsConfig(), root, home, noResolve, nil, false)
			if err == nil {
				t.Fatal("merging into a non-object must fail, not replace it")
			}
			if !strings.Contains(err.Error(), `"copilot-settings"`) || !strings.Contains(err.Error(), "not a JSON object") {
				t.Errorf("error must name the config and the cause: %v", err)
			}
		})
	}
	root := repoWithCopilotSettings(t, `null`)
	if _, err := Deploy(copilotSettingsConfig(), root, t.TempDir(), noResolve, nil, false); err == nil || !strings.Contains(err.Error(), "source is not a JSON object") {
		t.Errorf("a null source is not a set of managed keys: %v", err)
	}
}

// AC2: a diagnostic that asks "is it in sync?" must not create ~/.copilot/ as a
// side effect of asking, and `--dry-run` must not either. Before AI-039 the
// staging step ran ahead of the compare, so both did.
func TestPlanConfig_And_DryRun_CreateNoDestinationDirectory(t *testing.T) {
	for _, strategy := range []string{StrategyReplace, StrategyMerge} {
		t.Run(strategy, func(t *testing.T) {
			root := repoWithCopilotSettings(t, `{"model":"m"}`)
			home := t.TempDir()
			c := copilotSettingsConfig()
			c.Strategy = strategy

			p, err := PlanConfig(c, root, home, noResolve)
			if err != nil {
				t.Fatal(err)
			}
			if !p.Changed {
				t.Error("an absent destination is a change")
			}
			if _, err := os.Stat(filepath.Dir(p.Dst)); !os.IsNotExist(err) {
				t.Errorf("PlanConfig created the destination directory %s", filepath.Dir(p.Dst))
			}

			res, err := Deploy(c, root, home, noResolve, nil, true)
			if err != nil {
				t.Fatal(err)
			}
			if !res.Changed {
				t.Error("dry run must still report the change it would make")
			}
			if _, err := os.Stat(filepath.Dir(res.Dst)); !os.IsNotExist(err) {
				t.Errorf("dry run created the destination directory %s", filepath.Dir(res.Dst))
			}
		})
	}
}

func TestPlanConfig_RefusesARenderedConfig(t *testing.T) {
	root := repoWithSource(t, `{"a":"${X}"}`)
	c := piConfig()
	c.Render = true
	_, err := PlanConfig(c, root, t.TempDir(), noResolve)
	if err == nil || !strings.Contains(err.Error(), "cannot be planned") {
		t.Errorf("a rendered config has no plan without rendering: %v", err)
	}
}

func TestParseManifest_ValidatesStrategyByName(t *testing.T) {
	cases := []struct{ name, manifest, want string }{
		{"unknown strategy", `{"version":1,"configs":[{"name":"x","src":"a","dst":"b","strategy":"union"}]}`, `config "x": unknown strategy "union"`},
		{"merge cannot render", `{"version":1,"configs":[{"name":"x","src":"a","dst":"b","strategy":"merge","render":true}]}`, `config "x": strategy merge cannot render`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tc.manifest))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want %q, got %v", tc.want, err)
			}
		})
	}
	m, err := ParseManifest([]byte(`{"version":1,"configs":[{"name":"x","src":"a","dst":"b"},{"name":"y","src":"a","dst":"b","strategy":"merge"}]}`))
	if err != nil {
		t.Fatalf("replace by default and merge must both parse: %v", err)
	}
	if m.Configs[0].strategy() != StrategyReplace || m.Configs[1].strategy() != StrategyMerge {
		t.Errorf("strategies: %q %q", m.Configs[0].strategy(), m.Configs[1].strategy())
	}
}
