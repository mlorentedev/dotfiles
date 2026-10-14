package pi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

const repoSettings = `{
  "defaultProvider": "nan",
  "defaultModel": "deepseek-v4-flash",
  "temperature": 0.2,
  "enabledModels": ["nan/a", "nan/b"]
}
`

func settingsFixture(t *testing.T, dst string) (src, dstPath string) {
	t.Helper()
	dir := t.TempDir()
	src = filepath.Join(dir, "repo-settings.json")
	if err := os.WriteFile(src, []byte(repoSettings), 0o644); err != nil {
		t.Fatal(err)
	}
	dstPath = filepath.Join(dir, "home", ".pi", "agent", "settings.json")
	if dst != "" {
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dstPath, []byte(dst), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return src, dstPath
}

func readSettings(t *testing.T, p string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(p) //nolint:gosec // test fixture
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	return m
}

// An absent file is seeded whole, so defaultModel and defaultProvider exist
// before anything merges into it (#1948: the pi-compaction merge onto a
// missing file created one holding only compaction).
func TestSettings_AnAbsentFileIsSeededWhole(t *testing.T) {
	src, dst := settingsFixture(t, "")
	st, err := InspectSettings(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Missing || st.Changes() != 1 {
		t.Fatalf("state %+v: want one change, the seed", st)
	}
	if err := ApplySettings(src, dst, st); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst) //nolint:gosec // test fixture
	if string(got) != repoSettings {
		t.Errorf("seed is not the repo's file byte for byte:\n%s", got)
	}
	again, err := InspectSettings(src, dst)
	if err != nil || again.Changes() != 0 {
		t.Errorf("after the seed: %+v, %v; want converged", again, err)
	}
}

// pi writes the model picked in its TUI, the theme and lastChangelogVersion
// into this file; only enabledModels is the repo's once the file exists.
// It is made equal to the repo's list, so a model the repo drops leaves it,
// which a merge's union never does.
func TestSettings_APresentFileGetsTheReposModelsAndKeepsTheRest(t *testing.T) {
	src, dst := settingsFixture(t, `{"defaultModel": "user-pick", "theme": "dark", "lastChangelogVersion": "1.2.0", "temperature": 0.7, "enabledModels": ["nan/a", "nan/dropped"]}`)
	st, err := InspectSettings(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if st.Missing || !st.ModelsDiffer || st.Changes() != 1 {
		t.Fatalf("state %+v: want one change, the model list", st)
	}
	if err := ApplySettings(src, dst, st); err != nil {
		t.Fatal(err)
	}
	got := readSettings(t, dst)
	models, _ := got["enabledModels"].([]any)
	if !slices.Equal(models, []any{"nan/a", "nan/b"}) {
		t.Errorf("enabledModels = %v, want the repo's exactly", models)
	}
	if got["defaultModel"] != "user-pick" || got["theme"] != "dark" || got["lastChangelogVersion"] != "1.2.0" || got["temperature"] != 0.7 {
		t.Errorf("a value pi wrote was lost: %v", got)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(dst); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("the file's mode was not kept: %v, %v", fi.Mode(), err)
		}
	}
	again, err := InspectSettings(src, dst)
	if err != nil || again.Changes() != 0 {
		t.Errorf("after the sync: %+v, %v; want converged", again, err)
	}
}

func TestSettings_AFileInSyncIsLeftAlone(t *testing.T) {
	const own = `{"defaultModel": "user-pick", "enabledModels": ["nan/a", "nan/b"]}`
	src, dst := settingsFixture(t, own)
	st, err := InspectSettings(src, dst)
	if err != nil || st.Changes() != 0 {
		t.Fatalf("state %+v, %v; want converged", st, err)
	}
	if err := ApplySettings(src, dst, st); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); string(got) != own { //nolint:gosec // test fixture
		t.Errorf("a converged file was rewritten:\n%s", got)
	}
}

// A file pi cannot read either is reported, never replaced: it may hold the
// user's only copy of their choices.
func TestSettings_AFileThatIsNotAnObjectIsAnError(t *testing.T) {
	for name, body := range map[string]string{"broken": "{not json", "null": "null", "array": "[]"} {
		t.Run(name, func(t *testing.T) {
			src, dst := settingsFixture(t, body)
			if _, err := InspectSettings(src, dst); err == nil {
				t.Error("want an error naming the unreadable file")
			}
		})
	}
}

// A repo copy with no enabledModels manages no list: the file is not touched.
func TestSettings_ARepoCopyWithoutModelsSyncsNothing(t *testing.T) {
	src, dst := settingsFixture(t, `{"enabledModels": ["x"]}`)
	if err := os.WriteFile(src, []byte(`{"defaultModel": "m"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := InspectSettings(src, dst)
	if err != nil || st.Changes() != 0 {
		t.Errorf("state %+v, %v; want nothing to do", st, err)
	}
}
