package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

const reconcileSecret = "ghp_CMD_PLANTED_never_print"

// fakeVault models the store's SHAPE — which items exist, in which folder, with
// which fields — behind every seam reconcile touches: lister, reader, writer and
// syncer. One model, so the command's own verifying re-plan sees what apply did.
type fakeVault struct {
	folders map[string]string            // id -> name
	items   map[string]string            // item -> folder id
	fields  map[string]map[string]string // item -> field -> value
	writes  int
	syncs   int
	// noop makes every write succeed without changing anything, so the command's
	// convergence check has something to catch.
	noop bool
}

func newFakeVault() *fakeVault {
	return &fakeVault{folders: map[string]string{}, items: map[string]string{}, fields: map[string]map[string]string{}}
}

func (v *fakeVault) Sync() error { v.syncs++; return nil }

func (v *fakeVault) ListFolders() ([]string, error) {
	var out []string
	for _, n := range v.folders {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
}

func (v *fakeVault) ListItems() ([]secrets.ItemSummary, error) {
	var out []secrets.ItemSummary
	for name, fid := range v.items {
		s := secrets.ItemSummary{Name: name, Folder: v.folders[fid]}
		for f := range v.fields[name] {
			s.Fields = append(s.Fields, f)
		}
		sort.Strings(s.Fields)
		out = append(out, s)
	}
	return out, nil
}

func (v *fakeVault) Field(item, field string) (string, error) {
	val, ok := v.fields[item][field]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", secrets.ErrBWItemNotFound, item, field)
	}
	return val, nil
}

func (v *fakeVault) SetField(item, field, value string) error {
	v.writes++
	if !v.noop {
		v.fields[item][field] = value
	}
	return nil
}

func (v *fakeVault) CreateItem(item, field, value, folderID string) error {
	v.writes++
	if !v.noop {
		v.items[item] = folderID
		v.fields[item] = map[string]string{field: value}
	}
	return nil
}

func (v *fakeVault) MoveItem(item, folderID string) error {
	v.writes++
	if !v.noop {
		v.items[item] = folderID
	}
	return nil
}

func (v *fakeVault) ResolveFolder(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	for id, n := range v.folders {
		if n == name {
			return id, nil
		}
	}
	v.writes++
	id := "fid-" + name
	if !v.noop {
		v.folders[id] = name
	}
	return id, nil
}

const reconcileRegistry = `
version: 1
secrets:
  - {id: GITHUB_PERSONAL_ACCESS_TOKEN, plane: app, backend: bw, bw: {item: github-cli-pat, field: GITHUB_PERSONAL_ACCESS_TOKEN, folder: Dotfiles/apps, from: {item: GitHub, field: "Personal Access Token"}}, expose: {env: GITHUB_PERSONAL_ACCESS_TOKEN}}
  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT, folder: Dotfiles/apps}, expose: {env: DOCKERHUB_TOKEN}}
`

// legacyVault is the pre-reconcile shape: the token inside a shared item, and
// dockerhub unfoldered.
func legacyVault() *fakeVault {
	v := newFakeVault()
	v.items["GitHub"], v.items["dockerhub"] = "", ""
	v.fields["GitHub"] = map[string]string{"Personal Access Token": reconcileSecret}
	v.fields["dockerhub"] = map[string]string{"PAT": "d"}
	return v
}

func runReconcile(t *testing.T, v *fakeVault, registry string, args ...string) (string, error) {
	t.Helper()
	useTempRegistry(t, registry)
	oldL, oldR, oldW, oldS := bwLister, bwReader, bwWriter, bwSyncer
	t.Cleanup(func() { bwLister, bwReader, bwWriter, bwSyncer = oldL, oldR, oldW, oldS })
	bwLister, bwReader, bwWriter, bwSyncer = v, v, v, v

	c := newSecretsReconcileCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}

// AC1: without --apply nothing is written, and the plan names every operation.
func TestReconcilePlanWritesNothing(t *testing.T) {
	v := legacyVault()
	out, err := runReconcile(t, v, reconcileRegistry)
	if err != nil {
		t.Fatalf("a plannable plan exits 0: %v\n%s", err, out)
	}
	if v.writes != 0 {
		t.Fatalf("plan mode wrote %d time(s)", v.writes)
	}
	for _, want := range []string{"create-folder", "move-item", "create-item", "github-cli-pat", "Plan: 3 to apply", "--apply"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan output missing %q:\n%s", want, out)
		}
	}
	if v.syncs == 0 {
		t.Error("AC6: the store must be synced before the inventory is read")
	}
}

// AC2 + AC3: --apply converges, the command's own re-plan is empty, and no value
// appears anywhere in what it printed.
func TestReconcileApplyConvergesAndPrintsNoValue(t *testing.T) {
	v := legacyVault()
	out, err := runReconcile(t, v, reconcileRegistry, "--apply")
	if err != nil {
		t.Fatalf("apply: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Converged: 3 operation(s) applied") {
		t.Errorf("apply must confirm convergence:\n%s", out)
	}
	if v.fields["github-cli-pat"]["GITHUB_PERSONAL_ACCESS_TOKEN"] != reconcileSecret || v.folders[v.items["dockerhub"]] != "Dotfiles/apps" {
		t.Errorf("store not converged: items %v fields %v", v.items, v.fields)
	}
	if strings.Contains(out, "PLANTED") {
		t.Fatal("a secret value reached the command's output")
	}

	// And a second invocation is a no-op: nothing written, nothing planned.
	before := v.writes
	out2, err := runReconcile(t, v, reconcileRegistry, "--apply")
	if err != nil || v.writes != before || !strings.Contains(out2, "Plan: 0 to apply") {
		t.Errorf("second apply must change nothing (err %v, writes %d->%d):\n%s", err, before, v.writes, out2)
	}
	if !strings.Contains(out2, "bw.from on GITHUB_PERSONAL_ACCESS_TOKEN is satisfied") {
		t.Errorf("a satisfied from: must be reported as removable:\n%s", out2)
	}
}

// The convergence check is real: writes that change nothing are caught.
func TestReconcileApplyFailsWhenTheStoreDoesNotConverge(t *testing.T) {
	v := legacyVault()
	v.noop = true
	_, err := runReconcile(t, v, reconcileRegistry, "--apply")
	if err == nil || !strings.Contains(err.Error(), "did not converge") {
		t.Fatalf("want a non-convergence failure, got %v", err)
	}
}

// AC5: a blocked finding fails the plan, and --apply writes nothing at all.
func TestReconcileBlockedPlanFailsAndApplyWritesNothing(t *testing.T) {
	reg := reconcileRegistry + "  - {id: LIVE, plane: personal, backend: bw, bw: {item: never-created, field: k}, expose: {env: LIVE}}\n"
	for _, args := range [][]string{nil, {"--apply"}} {
		v := legacyVault()
		out, err := runReconcile(t, v, reg, args...)
		if err == nil || !strings.Contains(err.Error(), "blocked") {
			t.Errorf("%v: want a blocked failure, got %v", args, err)
		}
		if !strings.Contains(out, "dotf secrets set LIVE --yes") {
			t.Errorf("%v: the blocker must name its remedy:\n%s", args, out)
		}
		if v.writes != 0 {
			t.Errorf("%v: a blocked plan wrote %d time(s)", args, v.writes)
		}
	}
}
