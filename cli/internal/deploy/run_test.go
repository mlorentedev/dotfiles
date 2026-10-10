package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// runFixture writes a checkout with three render-free entries: one for every
// OS, one for an OS this run is not on, and one that requires a command.
func runFixture(t *testing.T) (repo, home string, man *Manifest) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	for name, body := range map[string]string{"a.json": `{"a":1}`, "b.json": `{"b":1}`, "c.json": `{"c":1}`} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	man, err := ParseManifest([]byte(`{"version": 4, "configs": [
	  {"name": "every-os", "src": "a.json", "dst": "{HOME}/.a/config.json", "mode": "0644"},
	  {"name": "other-os", "src": "b.json", "dst": "{HOME}/.b/config.json", "mode": "0644", "platforms": ["windows"]},
	  {"name": "needs-tool", "src": "c.json", "dst": "{HOME}/.c/config.json", "mode": "0644", "requires": "absent-tool"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	return repo, home, man
}

func runOptions(repo, home string, dryRun bool) RunOptions {
	return RunOptions{
		RepoRoot:  repo,
		Home:      home,
		GOOS:      "linux",
		Resolve:   func(string) string { return "" },
		Available: func(string) bool { return false },
		DryRun:    dryRun,
	}
}

func TestRun_DeploysWhatAppliesAndSaysWhyTheRestDidNot(t *testing.T) {
	repo, home, man := runFixture(t)

	res, err := Run(man, man.Configs, runOptions(repo, home, false))
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Changed(); len(got) != 1 || got[0] != "every-os" {
		t.Errorf("changed = %v, want [every-os]", got)
	}
	skipped := map[string]string{}
	for _, s := range res.Steps {
		if s.Skipped != "" {
			skipped[s.Name] = s.Skipped
		}
	}
	if skipped["other-os"] != "not for linux" || skipped["needs-tool"] != "absent-tool not installed" {
		t.Errorf("skip reasons = %v", skipped)
	}
	if _, err := os.Stat(filepath.Join(home, ".a", "config.json")); err != nil {
		t.Errorf("the applicable entry was not deployed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".b")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("an entry for another OS touched HOME: %v", err)
	}

	again, err := Run(man, man.Configs, runOptions(repo, home, false))
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Changed(); len(got) != 0 {
		t.Errorf("a second run changed %v; want nothing", got)
	}
}

// A rendered entry stages a copy before it can compare. Under a plan that copy
// must not create the destination directory: `dotf converge --plan` on an
// empty HOME writes nothing (PLAT-001b AC1).
func TestRun_APlanOfARenderedEntryCreatesNoDirectory(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "m.json"), []byte(`{"key":"{env:TOKEN}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	man, err := ParseManifest([]byte(`{"version": 3, "configs": [
	  {"name": "rendered", "src": "m.json", "dst": "{HOME}/.r/models.json", "render": true, "mode": "0600"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	o := runOptions(repo, home, true)
	o.Render = func(string) error { return nil }

	res, err := Run(man, man.Configs, o)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Changed(); len(got) != 1 {
		t.Errorf("the plan should report the missing file as a change, got %v", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".r")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a plan created the destination directory: %v", err)
	}
}

// An unattended run (converge after `dotf update`) cannot unlock the secret
// store. A render that leaves placeholders must not replace a file that holds
// the resolved values: the entry is skipped with the reason, and the run goes on.
func TestRun_AnIncompleteRenderKeepsTheInstalledFile(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "m.json"), []byte(`{"key":"{env:TOKEN}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(home, ".r", "models.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		t.Fatal(err)
	}
	const installed = `{"key":"resolved-value"}`
	if err := os.WriteFile(dst, []byte(installed), 0o600); err != nil {
		t.Fatal(err)
	}
	man, err := ParseManifest([]byte(`{"version": 3, "configs": [
	  {"name": "rendered", "src": "m.json", "dst": "{HOME}/.r/models.json", "render": true, "mode": "0600"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	o := runOptions(repo, home, false)
	o.Render = func(string) error { return errors.Join(ErrRenderIncomplete, errors.New("TOKEN unresolved")) }

	res, err := Run(man, man.Configs, o)
	if err != nil {
		t.Fatalf("an incomplete render failed the run: %v", err)
	}
	if len(res.Steps) != 1 || !res.Steps[0].Kept || len(res.Changed()) != 0 {
		t.Errorf("want the entry skipped with a reason, got %+v", res.Steps)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != installed {
		t.Errorf("the installed file was replaced: %q, %v", got, err)
	}
}
