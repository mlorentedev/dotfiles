package claude

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// mcpFake answers the claude mcp subcommands: registered maps a name to its
// `mcp get` output (absent means get exits non-zero), and failing names adds
// that fail; failOnce names whose first add fails and later ones succeed.
// Every add and remove is recorded.
type mcpFake struct {
	fakeRunner
	registered map[string]string
	failing    map[string]bool
	failOnce   map[string]bool
	failRemove bool
	added      []string
	removed    []string
}

func (f *mcpFake) McpGet(name string) (string, error) {
	out, ok := f.registered[name]
	if !ok {
		return "", errors.New("No MCP server found")
	}
	return out, nil
}

func (f *mcpFake) McpAdd(name, transport string, args []string) error {
	f.added = append(f.added, name+" "+transport+" "+strings.Join(args, " "))
	if f.failing[name] {
		return errors.New("add failed")
	}
	if f.failOnce[name] {
		delete(f.failOnce, name)
		return errors.New("add failed")
	}
	return nil
}

func (f *mcpFake) McpRemove(name string) error {
	f.removed = append(f.removed, name)
	if f.failRemove {
		return errors.New("remove failed")
	}
	delete(f.registered, name)
	return nil
}

func mcpSyncer(f *mcpFake, have ...string) Syncer {
	s := newSyncer(f)
	s.HasCommand = func(bin string) bool {
		for _, h := range have {
			if h == bin {
				return true
			}
		}
		return false
	}
	return s
}

// userScope is how `claude mcp get` names the scope dotf registers in.
const userScope = "User config (available in all your projects)"

// getOutput is `claude mcp get` as Claude Code prints it (measured
// 2026-10-13): a stdio server's Command and Args lines, an http server's URL.
func getOutput(name, transport, argv string) string {
	return scopedOutput(name, userScope, transport, argv)
}

// scopedOutput is getOutput answered by the registration in scope.
func scopedOutput(name, scope, transport, argv string) string {
	head := name + ":\n  Scope: " + scope + "\n  Status: ✔ Connected\n  Type: " + transport + "\n"
	tail := "\nTo remove this server, run: claude mcp remove " + name + " -s user\n"
	if transport != "stdio" {
		return head + "  URL: " + argv + "\n" + tail
	}
	cmd, args, _ := strings.Cut(argv, " ")
	return head + "  Command: " + cmd + "\n  Args: " + args + "\n  Environment:\n" + tail
}

var servers = []Server{
	{Name: "context7", Transport: "http", Args: "https://mcp.context7.com/mcp"},
	{Name: "hive", Transport: "stdio", Args: "hive client", PrerequisiteBinary: "uv", PrerequisiteCommand: "uv tool install --upgrade hive-vault"},
}

func TestRegisterAddsOnlyWhatIsMissing(t *testing.T) {
	f := &mcpFake{registered: map[string]string{"context7": getOutput("context7", "http", "https://mcp.context7.com/mcp")}}
	rep := mcpSyncer(f, "uv").Register(servers, false)
	if !reflect.DeepEqual(f.added, []string{"hive stdio hive client"}) {
		t.Fatalf("added %v, want only hive, split on whitespace", f.added)
	}
	if !reflect.DeepEqual(rep.Present, []string{"context7"}) || !reflect.DeepEqual(rep.Added, []string{"hive"}) {
		t.Fatalf("report %+v", rep)
	}
}

// The plan's second oracle: an add that fails is surfaced by name, never
// counted, and never swallowed.
func TestRegisterSurfacesAFailedAdd(t *testing.T) {
	f := &mcpFake{failing: map[string]bool{"context7": true}}
	rep := mcpSyncer(f, "uv").Register(servers, false)
	if !reflect.DeepEqual(rep.Failed, []string{"context7"}) || !reflect.DeepEqual(rep.Added, []string{"hive"}) {
		t.Fatalf("report %+v, want Failed [context7] Added [hive]", rep)
	}
}

// A registration that runs something other than the declaration is removed,
// then re-added from the SSOT, and reported with both (BUG-128): the Mac's
// pdf-modifier still ran the package name mcp-servers.json had dropped. A
// current entry is left alone. HIVE-118's `uvx hive-vault` is one such drift.
func TestRegisterReplacesADriftedRegistration(t *testing.T) {
	pdf := Server{Name: "pdf-modifier", Transport: "stdio", Args: "uvx pdf-modifier", PrerequisiteBinary: "uv"}
	for name, c := range map[string]struct {
		srv Server
		out string
		was string
	}{
		"renamed package": {pdf, getOutput("pdf-modifier", "stdio", "uvx pdf-modifier-mcp"), "stdio uvx pdf-modifier-mcp"},
		"HIVE-118 hive":   {servers[1], getOutput("hive", "stdio", "uvx hive-vault"), "stdio uvx hive-vault"},
		"moved URL":       {servers[0], getOutput("context7", "http", "https://old.example/mcp"), "http https://old.example/mcp"},
		"transport":       {servers[0], getOutput("context7", "sse", "https://mcp.context7.com/mcp"), "sse https://mcp.context7.com/mcp"},
	} {
		f := &mcpFake{registered: map[string]string{c.srv.Name: c.out}}
		rep := mcpSyncer(f, "uv").Register([]Server{c.srv}, false)
		want := []Drift{{Name: c.srv.Name, Was: c.was, Now: declared(c.srv)}}
		if !reflect.DeepEqual(f.removed, []string{c.srv.Name}) || !reflect.DeepEqual(rep.Added, []string{c.srv.Name}) || !reflect.DeepEqual(rep.Migrated, want) {
			t.Errorf("%s: removed %v, report %+v; want it replaced and reported as %+v", name, f.removed, rep, want)
		}
	}

	f := &mcpFake{registered: map[string]string{
		"context7": getOutput("context7", "http", "https://mcp.context7.com/mcp"),
		"hive":     getOutput("hive", "stdio", "hive client"),
	}}
	rep := mcpSyncer(f, "uv").Register(servers, false)
	if len(f.removed) != 0 || len(f.added) != 0 || !reflect.DeepEqual(rep.Present, []string{"context7", "hive"}) {
		t.Fatalf("current entries were touched: removed %v added %v report %+v", f.removed, f.added, rep)
	}
}

// Output that does not parse is never drift: the day Claude Code changes its
// format, treating it as drift would re-register every server on every run and
// converge would never report zero changes. It is reported as untouched.
func TestRegisterLeavesAnUnreadableRegistrationAlone(t *testing.T) {
	for name, out := range map[string]string{
		"empty":      "",
		"no type":    "hive:\n  Command: hive\n  Args: client\n",
		"no scope":   "hive:\n  Type: stdio\n  Command: uvx\n  Args: hive-vault\n",
		"no command": "hive:\n  Type: stdio\n",
		"no URL":     "context7:\n  Type: http\n",
		"new format": `{"name":"hive","command":"hive"}`,
	} {
		srv := servers[1]
		if strings.Contains(name, "URL") {
			srv = servers[0]
		}
		f := &mcpFake{registered: map[string]string{srv.Name: out}}
		rep := mcpSyncer(f, "uv").Register([]Server{srv}, false)
		if len(f.removed) != 0 || len(f.added) != 0 || len(rep.Untouched) != 1 || rep.Untouched[0].Name != srv.Name || !strings.Contains(rep.Untouched[0].Why, "did not parse") || len(rep.Present) != 0 {
			t.Errorf("%s: removed %v added %v report %+v; want it left alone and untouched", name, f.removed, f.added, rep)
		}
	}
}

// `claude mcp get` answers with the winning scope, local before project before
// user. dotf adds and removes at user scope only, so a drifted local or project
// registration is reported and left alone; one that matches is present.
func TestRegisterLeavesAnotherScopesRegistrationAlone(t *testing.T) {
	const local = "Local config (private to you in this project)"
	f := &mcpFake{registered: map[string]string{
		"context7": scopedOutput("context7", local, "http", "https://mcp.context7.com/mcp"),
		"hive":     scopedOutput("hive", "Project config (shared via .mcp.json)", "stdio", "uvx hive-vault"),
	}}
	rep := mcpSyncer(f, "uv").Register(servers, false)
	if len(f.removed)+len(f.added) != 0 {
		t.Fatalf("removed %v added %v; another scope's registration must not be touched", f.removed, f.added)
	}
	if !reflect.DeepEqual(rep.Present, []string{"context7"}) || len(rep.Untouched) != 1 || rep.Untouched[0].Name != "hive" || !strings.Contains(rep.Untouched[0].Why, "Project config") {
		t.Errorf("report %+v; want context7 present and hive untouched naming its scope", rep)
	}
}

// A stdio server with no arguments: Claude Code may print an empty Args line
// or none at all; both read as the bare command.
func TestRegisterReadsACommandWithoutArgs(t *testing.T) {
	solo := Server{Name: "solo", Transport: "stdio", Args: "solo"}
	for name, out := range map[string]string{
		"empty Args": "solo:\n  Scope: " + userScope + "\n  Type: stdio\n  Command: solo\n  Args: \n",
		"no Args":    "solo:\n  Scope: " + userScope + "\n  Type: stdio\n  Command: solo\n",
	} {
		f := &mcpFake{registered: map[string]string{"solo": out}}
		if rep := mcpSyncer(f).Register([]Server{solo}, false); !reflect.DeepEqual(rep.Present, []string{"solo"}) {
			t.Errorf("%s: report %+v, want solo present", name, rep)
		}
	}
}

// A server whose prerequisite binary is absent is skipped and says which one;
// its prerequisite command is never run by the Go path (#1993).
func TestRegisterSkipsAServerWhosePrerequisiteIsMissing(t *testing.T) {
	f := &mcpFake{}
	rep := mcpSyncer(f).Register(servers, false)
	if !reflect.DeepEqual(rep.Skipped, []Skip{{Name: "hive", Missing: "uv"}}) {
		t.Fatalf("skipped %+v, want hive for uv", rep.Skipped)
	}
	if !reflect.DeepEqual(rep.Added, []string{"context7"}) {
		t.Fatalf("report %+v", rep)
	}
}

func TestRegisterDryRunChangesNothing(t *testing.T) {
	f := &mcpFake{registered: map[string]string{"hive": getOutput("hive", "stdio", "uvx hive-vault")}}
	rep := mcpSyncer(f, "uv").Register(servers, true)
	if len(f.added) != 0 || len(f.removed) != 0 {
		t.Fatalf("dry run added %v removed %v", f.added, f.removed)
	}
	want := []Drift{{Name: "hive", Was: "stdio uvx hive-vault", Now: "stdio hive client"}}
	if !reflect.DeepEqual(rep.Added, []string{"context7", "hive"}) || !reflect.DeepEqual(rep.Migrated, want) {
		t.Fatalf("dry-run report %+v", rep)
	}
}

func TestLoadServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-servers.json")
	for _, c := range []struct {
		body, wantErr string
	}{
		{`{"_comment":"x","_history":"y","servers":[{"name":"a","transport":"stdio","args":"a b","prerequisite_binary":"uv","prerequisite_command":"uv x"}]}`, ""},
		{`{"servers":[{"name":"a","transport":"stdio","args":"a","env":{}}]}`, "unknown field"},
		{`{"servers":[{"name":"a","transport":"sse","args":"a"}]}`, "transport"},
		{`{"servers":[{"name":"","transport":"stdio","args":"a"}]}`, "empty name"},
		{`{"servers":[{"name":"a","transport":"stdio","args":"  "}]}`, "empty args"},
		{`{"servers":[{"name":"a","transport":"http","args":"u"},{"name":"a","transport":"http","args":"u"}]}`, "duplicate"},
	} {
		if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadServers(path)
		if c.wantErr == "" && err != nil || c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
			t.Errorf("%s: err = %v, want %q", c.body, err, c.wantErr)
		}
	}
}

// The repo's own lists decode under the strict readers: a field added to either
// file without the Go struct would otherwise fail every bare `dotf deploy`.
func TestTheRepoListsLoad(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if _, err := LoadServers(filepath.Join(root, ServersRel)); err != nil {
		t.Error(err)
	}
	if _, err := LoadPlugins(filepath.Join(root, filepath.FromSlash(PluginsRel))); err != nil {
		t.Error(err)
	}
}

// A migration is claimed only when the replacement landed. A failed remove
// leaves the stale entry; a failed re-add puts it back. Either way hive is in
// Failed and not in Migrated.
func TestRegisterReportsAMigrationOnlyWhenItCompleted(t *testing.T) {
	current := getOutput("context7", "http", "https://mcp.context7.com/mcp")
	stale := map[string]string{"context7": current, "hive": getOutput("hive", "stdio", "uvx hive-vault")}
	for name, f := range map[string]*mcpFake{
		"remove fails": {registered: stale, failRemove: true},
		"re-add fails": {registered: map[string]string{"context7": current, "hive": getOutput("hive", "stdio", "uvx hive-vault")}, failing: map[string]bool{"hive": true}},
	} {
		rep := mcpSyncer(f, "uv").Register(servers, false)
		if len(rep.Migrated) != 0 || !reflect.DeepEqual(rep.Failed, []string{"hive"}) {
			t.Errorf("%s: report %+v, want hive Failed and not Migrated", name, rep)
		}
	}
}

// A re-add that fails after the remove restores the old registration, so the
// server is still there, stale, rather than gone; the restore is reported.
func TestRegisterRestoresTheOldRegistrationWhenTheReAddFails(t *testing.T) {
	f := &mcpFake{registered: map[string]string{"hive": getOutput("hive", "stdio", "uvx hive-vault")}, failOnce: map[string]bool{"hive": true}}
	rep := mcpSyncer(f, "uv").Register(servers[1:], false)
	want := []string{"hive stdio hive client", "hive stdio uvx hive-vault"}
	if !reflect.DeepEqual(f.added, want) || !reflect.DeepEqual(rep.RolledBack, []string{"hive"}) || !reflect.DeepEqual(rep.Failed, []string{"hive"}) {
		t.Errorf("added %v report %+v; want the declared add, then the old one back", f.added, rep)
	}
}
