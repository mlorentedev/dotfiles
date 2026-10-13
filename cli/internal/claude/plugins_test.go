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
	rmFails  map[string]bool // RemoveMarketplace exits non-zero, registry untouched
	removed  []string
	// updFails names marketplaces whose refresh exits non-zero, addFails the
	// sources whose registration does. log holds every listing, registration,
	// refresh and install in call order ("list", "add o/m", "update m",
	// "install a@m").
	updFails map[string]bool
	addFails map[string]bool
	log      []string
}

func (f *fakeRunner) List() (string, error) { return f.listed, f.listErr }

// AddMarketplace registers the source under the name after its last "/", the
// way a GitHub source's marketplace.json names it in the declared case.
func (f *fakeRunner) AddMarketplace(source string) error {
	f.log = append(f.log, "add "+source)
	if f.addFails[source] {
		return errors.New("repository not found")
	}
	f.markets = append(f.markets, source[strings.LastIndex(source, "/")+1:])
	return nil
}

func (f *fakeRunner) UpdateMarketplace(name string) error {
	f.log = append(f.log, "update "+name)
	if f.updFails[name] {
		return errors.New("network unreachable")
	}
	return nil
}

func (f *fakeRunner) Install(id string) error {
	f.installed = append(f.installed, id)
	f.log = append(f.log, "install "+id)
	if f.failing[id] {
		return errors.New("install failed")
	}
	return nil
}

func (f *fakeRunner) McpGet(string) (string, error)         { return "", errors.New("not registered") }
func (f *fakeRunner) McpAdd(string, string, []string) error { return nil }
func (f *fakeRunner) McpRemove(string) error                { return nil }

func (f *fakeRunner) Marketplaces() (string, error) {
	f.log = append(f.log, "list")
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
	if f.rmFails[name] {
		return errors.New("permission denied")
	}
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

// Every declared id lands in exactly one bucket, and a failure keeps the
// CLI's error: a fresh runner failed all three installs and the summary read
// "0 added, 0 already present" with no cause anywhere (#2336).
func TestSyncAccountsForEveryIDAndKeepsTheCause(t *testing.T) {
	ids := []string{"a@m", "b@m", "c@m"}
	r := &fakeRunner{listed: "a@m", failing: map[string]bool{"c@m": true}}
	rep, err := newSyncer(r).Sync(ids, false)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(rep.Present) + len(rep.Added) + len(rep.Failed); n != len(ids) {
		t.Errorf("report %+v accounts for %d of %d ids", rep, n, len(ids))
	}
	if cause := rep.Causes["c@m"]; cause == nil || !strings.Contains(cause.Error(), "install failed") {
		t.Errorf("cause of c@m = %v; want the install's error", cause)
	}
	if len(rep.Causes) != len(rep.Failed) {
		t.Errorf("causes %v for failed %v", rep.Causes, rep.Failed)
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

// A failed removal keeps the CLI's own message, and the second list still
// judges it: the operator sees why, not only that it is still there.
func TestRetireKeepsTheCLIErrorOfAFailedRemoval(t *testing.T) {
	r := &fakeRunner{markets: []string{"old"}, rmFails: map[string]bool{"old": true}}
	rep, err := newSyncer(r).Retire([]string{"old"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep.Failed, []string{"old"}) {
		t.Fatalf("report %+v, want old failed", rep)
	}
	if cause := rep.Causes["old"]; cause == nil || !strings.Contains(cause.Error(), "permission denied") {
		t.Fatalf("cause %v, want the CLI's error", rep.Causes["old"])
	}
}

// An entry without a name means the list's shape changed. Read leniently it
// would be an empty registry, and every retired marketplace would look gone.
func TestRetireRefusesAListEntryWithoutAName(t *testing.T) {
	r := &fakeRunner{marketsJ: `[{"id":"old","source":"github"}]`}
	if _, err := newSyncer(r).Retire([]string{"old"}, false); err == nil {
		t.Fatal("want an error for an entry without a name")
	}
}

// The shape of `claude plugin marketplace list --json`, captured from Claude
// Code on 2026-10-10 rather than written to match the parser.
func TestRetireReadsTheCapturedListShape(t *testing.T) {
	r := &fakeRunner{marketsJ: `[{"name":"claude-plugins-official","source":"github","repo":"anthropics/claude-plugins-official","installLocation":"/home/u/.claude/plugins/marketplaces/claude-plugins-official"},{"name":"old","source":"github","repo":"someone/old-plugins","installLocation":"/home/u/.claude/plugins/marketplaces/old"}]`}
	rep, err := newSyncer(r).Retire([]string{"old"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep.Removed, []string{"old"}) {
		t.Fatalf("report %+v, want old to be removed", rep)
	}
}

// A fresh Claude Code no longer registers claude-plugins-official, and a
// plugin from a marketplace the box does not register cannot install, so every
// declared plugin failed on a new machine (#2378). Before the first install
// from a marketplace, Sync registers it from its declared source when it is
// missing, or refreshes it when it is there (its copy may predate a plugin
// declared since). Each marketplace is prepared once; the registry is listed
// once, and again after each add to prove the registration.
func TestSyncPreparesEachMarketplaceOnceBeforeItsFirstInstall(t *testing.T) {
	r := &fakeRunner{listed: "  ❯ c@p\n", markets: []string{"n"}}
	s := newSyncer(r)
	s.Sources = map[string]string{"m": "org/m"}
	if _, err := s.Sync([]string{"a@m", "b@m", "x@n", "c@p"}, false); err != nil {
		t.Fatal(err)
	}
	want := []string{"list", "add org/m", "list", "install a@m", "install b@m", "update n", "install x@n"}
	if !reflect.DeepEqual(r.log, want) {
		t.Fatalf("calls %v, want %v", r.log, want)
	}
}

// A converged box installs nothing, so it lists, registers and refreshes
// nothing: each is a network or disk call paid only when there is something to
// install. A dry run changes nothing, the registry included.
func TestSyncPreparesNothingWithoutAnInstall(t *testing.T) {
	r := &fakeRunner{listed: "  ❯ a@m\n"}
	if _, err := newSyncer(r).Sync([]string{"a@m"}, false); err != nil {
		t.Fatal(err)
	}
	r2 := &fakeRunner{}
	if _, err := newSyncer(r2).Sync([]string{"a@m"}, true); err != nil {
		t.Fatal(err)
	}
	if len(r.log)+len(r2.log) != 0 {
		t.Fatalf("calls %v and %v, want none", r.log, r2.log)
	}
}

// A preparation that fails does not stop the install: an offline box whose
// copy already lists the plugin still installs it. When the install fails as
// well, its cause names both, since the preparation is the likelier reason.
// A marketplace the box lacks and no source declares is named as such.
func TestAFailedPreparationIsNamedInTheCauseOfAFailedInstall(t *testing.T) {
	r := &fakeRunner{markets: []string{"m"}, updFails: map[string]bool{"m": true}, failing: map[string]bool{"b@m": true, "z@q": true, "y@k": true}}
	s := newSyncer(r)
	s.Sources = map[string]string{"k": "org/k"}
	r.addFails = map[string]bool{"org/k": true}
	rep, err := s.Sync([]string{"a@m", "b@m", "z@q", "y@k"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep.Added, []string{"a@m"}) || !reflect.DeepEqual(rep.Failed, []string{"b@m", "z@q", "y@k"}) {
		t.Fatalf("report %+v", rep)
	}
	for id, want := range map[string]string{
		"b@m": "network unreachable",
		"z@q": "ai/claude/marketplaces.json declares no source",
		"y@k": "repository not found",
	} {
		if c := rep.Causes[id].Error(); !strings.Contains(c, "install failed") || !strings.Contains(c, want) {
			t.Errorf("%s cause %q, want the install's error and %q", id, c, want)
		}
	}
	want := []string{"list", "update m", "install a@m", "install b@m", "install z@q", "add org/k", "install y@k"}
	if !reflect.DeepEqual(r.log, want) {
		t.Fatalf("calls %v, want %v: a failed preparation is not retried per install", r.log, want)
	}
}

// An add that exits 0 but registers the source under another name leaves the
// install to fail with "not found in marketplace", which names no cause: the
// misleading reading lesson 404 records. The registry is read again after an
// add, and a missing name is the cause of the failed install.
func TestARegistrationUnderAnotherNameIsNamedInTheCause(t *testing.T) {
	r := &fakeRunner{failing: map[string]bool{"a@m": true}}
	s := newSyncer(r)
	s.Sources = map[string]string{"m": "org/renamed"}
	rep, err := s.Sync([]string{"a@m"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if c := rep.Causes["a@m"]; c == nil || !strings.Contains(c.Error(), `the registry still has no marketplace "m"`) {
		t.Fatalf("cause %v, want the missing registration named", c)
	}
	want := []string{"list", "add org/renamed", "list", "install a@m"}
	if !reflect.DeepEqual(r.log, want) {
		t.Fatalf("calls %v, want %v", r.log, want)
	}
}

// The marketplace sources are their own file, so a dotf that predates them
// still reads plugins.json (strict) unchanged. Absent, there are no sources;
// present, it is read as strictly as the plugin list.
func TestLoadMarketplaceSources(t *testing.T) {
	dir := t.TempDir()
	got, err := LoadMarketplaceSources(filepath.Join(dir, "absent.json"))
	if err != nil || len(got) != 0 {
		t.Fatalf("absent file: %v %v, want empty and no error", got, err)
	}
	for name, body := range map[string]string{
		"unknown field": `{"marketplaces":{"m":"o/m"},"extra":1}`,
		"empty source":  `{"marketplaces":{"m":""}}`,
		"id as name":    `{"marketplaces":{"a@m":"o/m"}}`,
	} {
		p := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadMarketplaceSources(p); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// Every plugin the repo declares comes from a marketplace whose source the
// repo declares, so a fresh box can register it. A plugin added from a new
// marketplace without its source fails here, not on the next new machine.
func TestEveryDeclaredPluginsMarketplaceHasASource(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	list, err := LoadPlugins(filepath.Join(root, filepath.FromSlash(PluginsRel)))
	if err != nil {
		t.Fatal(err)
	}
	sources, err := LoadMarketplaceSources(filepath.Join(root, filepath.FromSlash(MarketplacesRel)))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range list.Plugins {
		if m := marketplaceOf(id); sources[m] == "" {
			t.Errorf("%s: marketplace %q has no source in %s", id, m, MarketplacesRel)
		}
	}
}
