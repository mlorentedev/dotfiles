package converge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
	"github.com/mlorentedev/dotfiles/cli/internal/pi"
)

// piEnv is the real checkout and an empty home.
func piEnv(t *testing.T) Env {
	t.Helper()
	return Env{RepoRoot: "../../..", Home: t.TempDir()}
}

func onlyPi(name string) bool { return name == "pi" }

// #1948, measured on from-zero run 38516177380: with pi installed by the
// tools step, configs-deploy's pi-compaction merge ran onto a missing
// settings.json and created one holding only compaction; the setup script's
// seed then found a file and kept it, and doctor failed on the absent
// defaultModel. The step seeds before configs-deploy, so the merge lands in
// the seeded file.
func TestPiSettings_SeedsBeforeTheCompactionMerge(t *testing.T) {
	e := piEnv(t)
	step := piSettings{has: onlyPi}
	plan, err := step.Reconcile(e, true)
	if err != nil || plan.Changes != 1 {
		t.Fatalf("plan: %+v, %v; want the seed", plan, err)
	}
	if _, err := os.Stat(pi.SettingsPath(e.Home)); !os.IsNotExist(err) {
		t.Fatal("a plan wrote the file")
	}
	if _, err := step.Reconcile(e, false); err != nil {
		t.Fatal(err)
	}
	if err := step.Probe(e); err != nil {
		t.Errorf("probe after the seed: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(e.RepoRoot, deploy.ManifestRel))
	if err != nil {
		t.Fatal(err)
	}
	man, err := deploy.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	var compaction deploy.Config
	for _, c := range man.Configs {
		if c.Name == "pi-compaction" {
			compaction = c
		}
	}
	if compaction.Name == "" {
		t.Fatal("ai/deploy.json has no pi-compaction entry; this test pins its order against the seed")
	}
	if _, err := deploy.Deploy(compaction, e.RepoRoot, e.Home, func(string) string { return "" }, nil, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(pi.SettingsPath(e.Home)) //nolint:gosec // test home
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if m["defaultModel"] == nil || m["defaultProvider"] == nil || m["compaction"] == nil {
		t.Errorf("want the seed and the compaction merge in one file, got %v", m)
	}

	again, err := step.Reconcile(e, false)
	if err != nil || again.Changes != 0 {
		t.Errorf("second run: %+v, %v; want no change", again, err)
	}
}

// Until #2452 the setup script installs pi, after every native step, so on
// most machines today pi is absent when this step runs. It is skipped, never
// a seed nobody reads.
func TestPiSettings_SkipsWhilePiIsNotInstalled(t *testing.T) {
	e := piEnv(t)
	res, err := piSettings{has: func(string) bool { return false }}.Reconcile(e, false)
	if err != nil || res.Skip == "" || res.Changes != 0 {
		t.Errorf("got %+v, %v; want a skip", res, err)
	}
	if _, err := os.Stat(pi.SettingsPath(e.Home)); !os.IsNotExist(err) {
		t.Error("the file was written although pi is not installed")
	}
}

// A file pi already wrote keeps its choices; only the model list is the repo's.
func TestPiSettings_KeepsWhatPiWrote(t *testing.T) {
	e := piEnv(t)
	dst := pi.SettingsPath(e.Home)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(`{"defaultModel": "user-pick", "theme": "dark", "enabledModels": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	step := piSettings{has: onlyPi}
	if err := step.Probe(e); err == nil {
		t.Error("probe passed with the repo's models missing")
	}
	res, err := step.Reconcile(e, false)
	if err != nil || res.Changes != 1 {
		t.Fatalf("got %+v, %v; want the model list synced", res, err)
	}
	var m map[string]any
	got, _ := os.ReadFile(dst) //nolint:gosec // test home
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if m["defaultModel"] != "user-pick" || m["theme"] != "dark" {
		t.Errorf("a choice pi wrote was lost: %v", m)
	}
	if models, _ := m["enabledModels"].([]any); len(models) == 0 {
		t.Error("enabledModels was not synced")
	}
	if err := step.Probe(e); err != nil {
		t.Errorf("probe after the sync: %v", err)
	}
}

// A checkout without the repo's copy (a test fixture, a sparse checkout) has
// nothing to seed from; the step says so instead of failing the run.
func TestPiSettings_SkipsWithoutTheReposCopy(t *testing.T) {
	e := Env{RepoRoot: t.TempDir(), Home: t.TempDir()}
	res, err := piSettings{has: onlyPi}.Reconcile(e, false)
	if err != nil || res.Skip == "" {
		t.Errorf("got %+v, %v; want a skip naming the missing source", res, err)
	}
}
