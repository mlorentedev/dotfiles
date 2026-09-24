package cmd

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// stubLister is a fixed inventory for the drift command.
type stubLister struct {
	items   []secrets.ItemSummary
	folders []string
}

func (s stubLister) Sync() error                               { return nil }
func (s stubLister) ListItems() ([]secrets.ItemSummary, error) { return s.items, nil }
func (s stubLister) ListFolders() ([]string, error)            { return s.folders, nil }

// orderedStore records the order the inventory is touched in.
type orderedStore struct {
	stubLister
	syncErr error
	calls   []string
}

func (s *orderedStore) Sync() error {
	s.calls = append(s.calls, "sync")
	return s.syncErr
}

func (s *orderedStore) ListItems() ([]secrets.ItemSummary, error) {
	s.calls = append(s.calls, "items")
	return s.stubLister.ListItems()
}

func (s *orderedStore) ListFolders() ([]string, error) {
	s.calls = append(s.calls, "folders")
	return s.stubLister.ListFolders()
}

func runDrift(t *testing.T, l inventorySource) (string, error) {
	t.Helper()
	useTempRegistry(t, `
version: 1
secrets:
  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT, folder: Dotfiles/apps}, expose: {env: DOCKERHUB_TOKEN}}
`)
	old := bwLister
	bwLister = l
	t.Cleanup(func() { bwLister = old })
	c := newSecretsDriftCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(nil)
	err := c.Execute()
	return out.String(), err
}

// AC8, the half no test held: a finding makes the command exit non-zero, so a
// hook or CI step can gate on it — and each line names the registry id to edit.
func TestDriftExitsNonZeroOnAFindingAndNamesTheSecret(t *testing.T) {
	out, err := runDrift(t, stubLister{
		items:   []secrets.ItemSummary{{Name: "dockerhub", Fields: []string{"PAT"}}},
		folders: []string{"Dotfiles/apps"},
	})
	if err == nil {
		t.Fatalf("a misfiled item must fail the command:\n%s", out)
	}
	if !strings.Contains(out, "item-misfiled") || !strings.Contains(out, "[DOCKERHUB_TOKEN]") {
		t.Errorf("the finding must be printed with its registry id:\n%s", out)
	}
}

func TestDriftExitsZeroOnAMatchingStore(t *testing.T) {
	out, err := runDrift(t, stubLister{
		items:   []secrets.ItemSummary{{Name: "dockerhub", Folder: "Dotfiles/apps", Fields: []string{"PAT"}}},
		folders: []string{"Dotfiles/apps"},
	})
	if err != nil {
		t.Fatalf("a matching store must exit 0: %v\n%s", err, out)
	}
}

// CLI-078 review round 4: drift read the daemon's cache without syncing, on the
// grounds that it only reports, while its exit code is what a gate reads and
// reconcile refused to read the same inventory unsynced. A folder made since the
// last sync then showed up as an item "in (no folder)".
func TestDriftSyncsBeforeItReads(t *testing.T) {
	s := &orderedStore{stubLister: stubLister{
		items:   []secrets.ItemSummary{{Name: "dockerhub", Folder: "Dotfiles/apps", Fields: []string{"PAT"}}},
		folders: []string{"Dotfiles/apps"},
	}}
	if out, err := runDrift(t, s); err != nil {
		t.Fatalf("a matching store must exit 0: %v\n%s", err, out)
	}
	if len(s.calls) == 0 || s.calls[0] != "sync" {
		t.Fatalf("drift must sync before it reads the inventory, calls: %v", s.calls)
	}
}

// A store that cannot sync is refused, not reported on from a stale cache.
func TestDriftRefusesAStoreItCannotSync(t *testing.T) {
	s := &orderedStore{syncErr: errors.New("daemon unreachable")}
	_, err := runDrift(t, s)
	if err == nil || !strings.Contains(err.Error(), "sync") {
		t.Fatalf("want a sync failure, got %v", err)
	}
	if len(s.calls) != 1 {
		t.Errorf("nothing may be read from a store that did not sync, calls: %v", s.calls)
	}
}

// writerCalls returns the names in f that the forbidden set holds: any identifier
// (a call to bwWrite, a method named like a writer's) and any os.<name> selector.
// Comments are not in the AST, so a comment that names a writer is not a call.
func writerCalls(f *ast.File, forbidden, osWrites map[string]bool) []string {
	var hits []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if pkg, ok := x.X.(*ast.Ident); ok && pkg.Name == "os" && osWrites[x.Sel.Name] {
				hits = append(hits, "os."+x.Sel.Name)
			}
		case *ast.Ident:
			if forbidden[x.Name] {
				hits = append(hits, x.Name)
			}
		}
		return true
	})
	sort.Strings(hits)
	return hits
}

// AC8, structurally: drift's source names no write, and it has no flag that could
// ask for one. The forbidden set is DERIVED from the writer interface, so it
// cannot fall behind it. The committed check this replaces grepped for four
// literal method names, and the interface grew to five without it noticing:
// `bwWrite().MoveItem(...)` passed it (CLI-078 review round 4).
func TestDriftSourceCallsNoWriter(t *testing.T) {
	forbidden := map[string]bool{"bwWrite": true, "bwWriter": true}
	w := reflect.TypeOf((*secrets.BWWriteClient)(nil)).Elem()
	for i := 0; i < w.NumMethod(); i++ {
		forbidden[w.Method(i).Name] = true
	}
	if !forbidden["MoveItem"] || !forbidden["RemoveField"] {
		t.Fatalf("the writer surface was not derived from BWWriteClient: %v", forbidden)
	}
	osWrites := map[string]bool{"WriteFile": true, "Create": true, "OpenFile": true, "Remove": true, "RemoveAll": true, "Rename": true, "Mkdir": true, "MkdirAll": true}

	// The scanner must see a write when there is one, or a clean result below
	// proves nothing.
	probe, err := parser.ParseFile(token.NewFileSet(), "probe.go", `package p
func f() { bwWrite().MoveItem("a", "b"); _ = os.WriteFile }`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := writerCalls(probe, forbidden, osWrites); len(got) != 3 {
		t.Fatalf("the scanner missed a planted write: %v", got)
	}

	src, err := parser.ParseFile(token.NewFileSet(), "secrets_drift.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := writerCalls(src, forbidden, osWrites); len(got) > 0 {
		t.Errorf("drift must never write, and its source names %v", got)
	}

	// The inventory seam cannot write either: none of its methods is a writer's.
	inv := reflect.TypeOf((*inventorySource)(nil)).Elem()
	for i := 0; i < inv.NumMethod(); i++ {
		if forbidden[inv.Method(i).Name] {
			t.Errorf("the inventory seam carries the writer method %s", inv.Method(i).Name)
		}
	}
	for _, flag := range []string{"fix", "apply"} {
		if newSecretsDriftCmd().Flags().Lookup(flag) != nil {
			t.Errorf("drift must have no --%s: converging the store is reconcile's", flag)
		}
	}
}

// The summary counted vault items that a declaration names and called them the
// items the declarations span, so an absent item made the two numbers disagree
// with no way to tell which was which (CLI-078 review round 4, question 4).
func TestDriftSummaryCountsDeclaredItemsAndThoseInTheVault(t *testing.T) {
	out, _ := runDrift(t, stubLister{folders: []string{"Dotfiles/apps"}})
	if !strings.Contains(out, "naming 1 item(s), 0 of them in the vault") {
		t.Errorf("an absent declared item must still be counted as declared:\n%s", out)
	}
	out, _ = runDrift(t, stubLister{
		items:   []secrets.ItemSummary{{Name: "dockerhub", Folder: "Dotfiles/apps", Fields: []string{"PAT"}}, {Name: "personal-login"}},
		folders: []string{"Dotfiles/apps"},
	})
	if !strings.Contains(out, "naming 1 item(s), 1 of them in the vault") || !strings.Contains(out, "1 of 2 vault items unmanaged") {
		t.Errorf("summary miscounted:\n%s", out)
	}
}
