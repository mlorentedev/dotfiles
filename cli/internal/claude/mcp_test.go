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
// that fail. Every add and remove is recorded.
type mcpFake struct {
	fakeRunner
	registered map[string]string
	failing    map[string]bool
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
	return nil
}

func (f *mcpFake) McpRemove(name string) error {
	f.removed = append(f.removed, name)
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

var servers = []Server{
	{Name: "context7", Transport: "http", Args: "https://mcp.context7.com/mcp"},
	{Name: "hive", Transport: "stdio", Args: "hive client", PrerequisiteBinary: "uv", PrerequisiteCommand: "uv tool install --upgrade hive-vault"},
}

func TestRegisterAddsOnlyWhatIsMissing(t *testing.T) {
	f := &mcpFake{registered: map[string]string{"context7": "context7:\n  Type: http\n"}}
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

// HIVE-118: a registration still pointing at `uvx hive-vault` is removed, then
// re-added from the SSOT. A current `hive client` entry is left alone.
func TestRegisterMigratesAStaleHiveEntry(t *testing.T) {
	f := &mcpFake{registered: map[string]string{
		"context7": "ok",
		"hive":     "hive:\n  Command: uvx\n  Args: hive-vault\n",
	}}
	rep := mcpSyncer(f, "uv").Register(servers, false)
	if !reflect.DeepEqual(f.removed, []string{"hive"}) || !reflect.DeepEqual(rep.Added, []string{"hive"}) {
		t.Fatalf("removed %v, report %+v; want hive removed then re-added", f.removed, rep)
	}
	if !reflect.DeepEqual(rep.Migrated, []string{"hive"}) {
		t.Fatalf("migration not reported: %+v", rep)
	}

	f = &mcpFake{registered: map[string]string{"context7": "ok", "hive": "hive:\n  Command: hive\n  Args: client\n"}}
	mcpSyncer(f, "uv").Register(servers, false)
	if len(f.removed) != 0 || len(f.added) != 0 {
		t.Fatalf("a current hive entry was touched: removed %v added %v", f.removed, f.added)
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
	f := &mcpFake{registered: map[string]string{"hive": "Command: uvx hive-vault"}}
	rep := mcpSyncer(f, "uv").Register(servers, true)
	if len(f.added) != 0 || len(f.removed) != 0 {
		t.Fatalf("dry run added %v removed %v", f.added, f.removed)
	}
	if !reflect.DeepEqual(rep.Added, []string{"context7", "hive"}) || !reflect.DeepEqual(rep.Migrated, []string{"hive"}) {
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

// The migration is scoped to hive: pdf-modifier legitimately runs under uvx,
// and an unscoped match would remove and re-add it on every deploy.
func TestRegisterLeavesOtherUvxServersAlone(t *testing.T) {
	pdf := []Server{{Name: "pdf-modifier", Transport: "stdio", Args: "uvx pdf-modifier-mcp", PrerequisiteBinary: "uv"}}
	f := &mcpFake{registered: map[string]string{"pdf-modifier": "Command: uvx\n  Args: pdf-modifier-mcp\n"}}
	rep := mcpSyncer(f, "uv").Register(pdf, false)
	if len(f.removed) != 0 || len(f.added) != 0 || !reflect.DeepEqual(rep.Present, []string{"pdf-modifier"}) {
		t.Fatalf("removed %v added %v report %+v; a current uvx server was churned", f.removed, f.added, rep)
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
