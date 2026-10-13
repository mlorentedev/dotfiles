package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A report that loses a declared plugin is an error, never a summary: the
// counts must add up to the list (PR-Agent on #2343).
func TestAccountedFor_RefusesAReportThatLosesAPlugin(t *testing.T) {
	rep := PluginReport{Present: []string{"a@m"}, Failed: []string{"b@m"}}
	if err := accountedFor(rep, 2); err != nil {
		t.Errorf("a full report was refused: %v", err)
	}
	if err := accountedFor(rep, 3); err == nil || !strings.Contains(err.Error(), "2 of 3") {
		t.Errorf("a report missing one of 3 plugins = %v; want it refused", err)
	}
}

func applyRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	for rel, body := range map[string]string{
		PluginsRel:      `{"plugins":["a@m","b@m"],"retired_marketplaces":[]}`,
		MarketplacesRel: `{"marketplaces":{"m":"o/m"}}`,
	} {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// A caller that prints only the error must still learn why each plugin failed:
// converge reports one line per step, and the rows a bare deploy prints never
// reach it (BUG-121, #2336; #2387).
func TestApplyPlugins_TheErrorCarriesEachFailedPluginsCause(t *testing.T) {
	r := &fakeRunner{markets: []string{"m"}, failing: map[string]bool{"b@m": true}}
	res, err := ApplyPlugins(newSyncer(r), applyRepo(t), false)
	if err == nil {
		t.Fatal("a failed install must fail ApplyPlugins")
	}
	if !strings.Contains(err.Error(), "b@m (install failed)") {
		t.Errorf("the error must name the plugin with its cause: %v", err)
	}
	if len(res.Plugins.Added) != 1 || res.Declared != 2 {
		t.Errorf("result %+v", res)
	}
}

// A plan asks and never changes: under dryRun ApplyPlugins lists and installs
// nothing, so a converge probe that re-plans cannot mutate the machine.
func TestApplyPlugins_APlanMakesNoChange(t *testing.T) {
	r := &fakeRunner{markets: []string{"m"}}
	res, err := ApplyPlugins(newSyncer(r), applyRepo(t), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.installed) != 0 || len(r.removed) != 0 {
		t.Errorf("a plan installed %v and removed %v", r.installed, r.removed)
	}
	for _, call := range r.log {
		if !strings.HasPrefix(call, "list") {
			t.Errorf("a plan made the mutating call %q", call)
		}
	}
	if len(res.Plugins.Added) != 2 {
		t.Errorf("a plan must report both plugins as to add: %+v", res.Plugins)
	}
}
