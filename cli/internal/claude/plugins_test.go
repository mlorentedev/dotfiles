package claude

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// fakeRunner stands in for the claude CLI: List returns listed, Install fails
// for the ids in failing and records every call.
type fakeRunner struct {
	listed    string
	listErr   error
	failing   map[string]bool
	installed []string
	// Marketplaces: markets is the registry; a removal drops the name unless
	// sticky names it (the CLI exits 0 and the registration survives).
	markets  []string
	marketsJ string // raw list output, when set, instead of markets
	sticky   map[string]bool
	removed  []string
}

func (f *fakeRunner) List() (string, error) { return f.listed, f.listErr }

func (f *fakeRunner) Install(id string) error {
	f.installed = append(f.installed, id)
	if f.failing[id] {
		return errors.New("install failed")
	}
	return nil
}

func (f *fakeRunner) McpGet(string) (string, error)         { return "", errors.New("not registered") }
func (f *fakeRunner) McpAdd(string, string, []string) error { return nil }
func (f *fakeRunner) McpRemove(string) error                { return nil }

func (f *fakeRunner) Marketplaces() (string, error) {
	if f.marketsJ != "" {
		return f.marketsJ, nil
	}
	parts := make([]string, 0, len(f.markets))
	for _, m := range f.markets {
		parts = append(parts, `{"name":"`+m+`","source":"github"}`)
	}
	return "[" + strings.Join(parts, ",") + "]", nil
}

func (f *fakeRunner) RemoveMarketplace(name string) error {
	f.removed = append(f.removed, name)
	if f.sticky[name] {
		return nil
	}
	f.markets = slices.DeleteFunc(f.markets, func(m string) bool { return m == name })
	return nil
}

func newSyncer(r Runner) Syncer {
	return Syncer{Run: r, ClaudeJSON: filepath.Join(os.TempDir(), "absent-claude-json-for-test"), Floor: 10240}
}

func TestSyncInstallsOnlyWhatIsMissing(t *testing.T) {
	r := &fakeRunner{listed: "  ❯ a@m\n    Version: 1.0\n"}
	rep, err := newSyncer(r).Sync([]string{"a@m", "b@m"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.installed, []string{"b@m"}) {
		t.Fatalf("installed %v, want [b@m]", r.installed)
	}
	if !reflect.DeepEqual(rep.Present, []string{"a@m"}) || !reflect.DeepEqual(rep.Added, []string{"b@m"}) {
		t.Fatalf("report %+v", rep)
	}
}

// AC7: a plugin counts as added only when its install succeeded (the Linux
// twin's behaviour; the PowerShell twin counts attempts, #1491).
// A declared id that is a substring of an installed one is still missing: the
// twins' substring grep would report it present and never install it.
func TestSyncMatchesWholeIDsNotSubstrings(t *testing.T) {
	r := &fakeRunner{listed: "Installed plugins:\n\n  ❯ gopls-lsp@m\n    Version: 1.0.0\n"}
	rep, err := newSyncer(r).Sync([]string{"lsp@m", "gopls-lsp@m"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.installed, []string{"lsp@m"}) || !reflect.DeepEqual(rep.Present, []string{"gopls-lsp@m"}) {
		t.Fatalf("installed %v, report %+v; want lsp@m installed, gopls-lsp@m present", r.installed, rep)
	}
}

func TestSyncCountsOnlySuccessfulInstalls(t *testing.T) {
	r := &fakeRunner{failing: map[string]bool{"b@m": true}}
	rep, err := newSyncer(r).Sync([]string{"a@m", "b@m"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep.Added, []string{"a@m"}) || !reflect.DeepEqual(rep.Failed, []string{"b@m"}) {
		t.Fatalf("report %+v, want Added [a@m] Failed [b@m]", rep)
	}
}

func TestSyncDryRunInstallsNothing(t *testing.T) {
	r := &fakeRunner{}
	rep, err := newSyncer(r).Sync([]string{"a@m"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.installed) != 0 || !reflect.DeepEqual(rep.Added, []string{"a@m"}) {
		t.Fatalf("dry run installed %v, report %+v", r.installed, rep)
	}
}

func TestSyncSurfacesAListFailure(t *testing.T) {
	r := &fakeRunner{listErr: errors.New("claude broke")}
	if _, err := newSyncer(r).Sync([]string{"a@m"}, false); err == nil || !strings.Contains(err.Error(), "claude broke") {
		t.Fatalf("err = %v, want the list failure", err)
	}
	if len(r.installed) != 0 {
		t.Fatalf("installed %v after the list failed", r.installed)
	}
}

func TestLoadPlugins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugins.json")
	for _, c := range []struct {
		body    string
		want    []string
		wantErr string
	}{
		{`{"$comment":["x"],"plugins":["a@m","b@m"]}`, []string{"a@m", "b@m"}, ""},
		{`{"plugins":["a@m"],"retired_marketplaces":["old"]}`, []string{"a@m"}, ""},
		{`{"plugins":["a@m"],"retired_marketplaces":["m"]}`, nil, "both retired and the source"},
		{`{"plugins":["a@m"],"retired_marketplaces":["x@y"]}`, nil, "not a marketplace name"},
		{`{"plugins":["a@m"],"extra":1}`, nil, "unknown field"},
		{`{"plugins":["no-marketplace"]}`, nil, "not <plugin>@<marketplace>"},
		{`{"plugins":["a@m","a@m"]}`, nil, "duplicate"},
	} {
		if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
			t.Fatal(err)
		}
		list, err := LoadPlugins(path)
		got := list.Plugins
		if c.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("%s: err = %v, want %q", c.body, err, c.wantErr)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got (%v, %v), want %v", c.body, got, err, c.want)
		}
	}
}

func TestRetireRemovesARegisteredMarketplaceAndVerifiesItIsGone(t *testing.T) {
	r := &fakeRunner{markets: []string{"official", "old"}}
	rep, err := newSyncer(r).Retire([]string{"old", "never-registered"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.removed, []string{"old"}) {
		t.Fatalf("removed %v, want only the registered retired one", r.removed)
	}
	if !reflect.DeepEqual(rep.Removed, []string{"old"}) || len(rep.Failed) != 0 {
		t.Fatalf("report %+v", rep)
	}
}

// The #1431 class: the removal reports success and the registration survives.
// Only the second list can tell, so it is what decides Failed.
func TestRetireReportsARegistrationThatSurvivesTheRemoval(t *testing.T) {
	r := &fakeRunner{markets: []string{"old"}, sticky: map[string]bool{"old": true}}
	rep, err := newSyncer(r).Retire([]string{"old"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep.Failed, []string{"old"}) || len(rep.Removed) != 0 {
		t.Fatalf("report %+v, want old failed", rep)
	}
}

func TestRetireDryRunRemovesNothing(t *testing.T) {
	r := &fakeRunner{markets: []string{"old"}}
	rep, err := newSyncer(r).Retire([]string{"old"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.removed) != 0 || !reflect.DeepEqual(rep.Removed, []string{"old"}) {
		t.Fatalf("removed %v, report %+v", r.removed, rep)
	}
}

// An unreadable list is an error, never an empty registry: read as empty, a
// retired marketplace would be reported as already gone.
func TestRetireRefusesAnUnparseableList(t *testing.T) {
	r := &fakeRunner{marketsJ: "Configured marketplaces:\n  old\n"}
	if _, err := newSyncer(r).Retire([]string{"old"}, false); err == nil {
		t.Fatal("want an error for a list that is not JSON")
	}
	if len(r.removed) != 0 {
		t.Fatalf("removed %v after an unreadable list", r.removed)
	}
}
