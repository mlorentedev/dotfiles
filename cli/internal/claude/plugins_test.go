package claude

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
}

func (f *fakeRunner) List() (string, error) { return f.listed, f.listErr }

func (f *fakeRunner) Install(id string) error {
	f.installed = append(f.installed, id)
	if f.failing[id] {
		return errors.New("install failed")
	}
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
		{`{"plugins":["a@m"],"extra":1}`, nil, "unknown field"},
		{`{"plugins":["no-marketplace"]}`, nil, "not <plugin>@<marketplace>"},
		{`{"plugins":["a@m","a@m"]}`, nil, "duplicate"},
	} {
		if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := LoadPlugins(path)
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
