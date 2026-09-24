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

func (v *fakeVault) RemoveField(item, field string) error {
	v.writes++
	if !v.noop {
		delete(v.fields[item], field)
	}
	return nil
}

func (v *fakeVault) DeleteItem(item string) error {
	v.writes++
	if !v.noop {
		delete(v.items, item)
		delete(v.fields, item)
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

// CLI-080 review round 1, Blocker: a declaration that both copies and retires
// must converge in ONE --apply. The retire is planned only once its destination
// exists, and that is deliberate: it compares the copy's value as the store holds
// it after a sync, never the value this run meant to write. So the copy and its
// retire land in two passes of one command. The old command stopped after the
// first pass and reported its own second pass as non-convergence.
func TestReconcileApplyCopiesAndRetiresInOneRun(t *testing.T) {
	reg := `
version: 1
secrets:
  - {id: GITHUB_PERSONAL_ACCESS_TOKEN, plane: app, backend: bw, bw: {item: github-cli-pat, field: GITHUB_PERSONAL_ACCESS_TOKEN, folder: Dotfiles/apps, from: {item: GitHub, field: "Personal Access Token", retire: true}}, expose: {env: GITHUB_PERSONAL_ACCESS_TOKEN}}
`
	v := legacyVault()
	out, err := runReconcile(t, v, reg, "--apply")
	if err != nil {
		t.Fatalf("copy + retire must converge in one --apply: %v\n%s", err, out)
	}
	if v.fields["github-cli-pat"]["GITHUB_PERSONAL_ACCESS_TOKEN"] != reconcileSecret {
		t.Errorf("the copy did not land: %v", v.fields)
	}
	if _, still := v.fields["GitHub"]["Personal Access Token"]; still {
		t.Errorf("the source field was not retired: %v", v.fields["GitHub"])
	}
	if !strings.Contains(out, "retire-source") || !strings.Contains(out, "Converged") {
		t.Errorf("the second pass must be shown and convergence confirmed:\n%s", out)
	}
	if strings.Contains(out, "PLANTED") {
		t.Fatal("a secret value reached the command's output")
	}
}

// The second pass exists for what the first one unlocks, and only that. An
// operation of any other kind after a pass is a store that did not take the
// write, or two declarations pulling one item two ways, and one more pass would
// only repeat it. It must still fail, in the pass it appears in.
func TestReconcileSecondPassIsOnlyForRetires(t *testing.T) {
	v := legacyVault()
	v.noop = true
	_, err := runReconcile(t, v, reconcileRegistry, "--apply")
	if err == nil || !strings.Contains(err.Error(), "did not converge") {
		t.Fatalf("a no-op store must still fail convergence, got %v", err)
	}
	if v.writes > 4 {
		t.Errorf("a non-retire leftover must not earn another pass: %d writes", v.writes)
	}
}

// CLI-082 AC1 + AC2: the plan compares every retire before anything applies. An
// equal one is shown verified; a differing one blocks and says how to settle it.
// Nothing is written, and no value reaches the output.
func TestReconcilePlanShowsEachRetireVerdict(t *testing.T) {
	reg := `
version: 1
secrets:
  - {id: SAME, plane: app, backend: bw, bw: {item: same-dst, field: k, folder: Dotfiles/apps, from: {item: legacy, field: same, retire: true}}, expose: {env: SAME}}
  - {id: DIFF, plane: app, backend: bw, bw: {item: diff-dst, field: k, folder: Dotfiles/apps, from: {item: legacy, field: diff, retire: true}}, expose: {env: DIFF}}
`
	v := newFakeVault()
	v.folders["f1"] = "Dotfiles/apps"
	v.items["legacy"], v.items["same-dst"], v.items["diff-dst"] = "", "f1", "f1"
	v.fields["legacy"] = map[string]string{"same": "PLANTED-same", "diff": "PLANTED-old"}
	v.fields["same-dst"] = map[string]string{"k": "PLANTED-same"}
	v.fields["diff-dst"] = map[string]string{"k": "PLANTED-new"}

	out, err := runReconcile(t, v, reg)
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("a differing retire must block the plan, got %v\n%s", err, out)
	}
	for _, want := range []string{"legacy/\"same\"", "verified equal", "differs", "dotf secrets rotate DIFF"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan output missing %q:\n%s", want, out)
		}
	}
	if v.writes != 0 || strings.Contains(out, "PLANTED") {
		t.Fatalf("a plan must write nothing and print no value (writes %d):\n%s", v.writes, out)
	}
}

// CLI-082 AC4: a retired item is shown with its shape and reason, deleted by
// --apply, and reported gone on the next run.
func TestReconcileDeletesARetiredItem(t *testing.T) {
	reg := `
version: 1
secrets:
  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT, folder: Dotfiles/apps}, expose: {env: DOCKERHUB_TOKEN}}
retired:
  - {item: github-cli-pat, reason: its registry entry was retired and the token revoked}
`
	v := newFakeVault()
	v.folders["f1"] = "Dotfiles/apps"
	v.items["dockerhub"], v.items["github-cli-pat"] = "f1", "f1"
	v.fields["dockerhub"] = map[string]string{"PAT": "d"}
	v.fields["github-cli-pat"] = map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "PLANTED-revoked"}

	out, err := runReconcile(t, v, reg)
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, out)
	}
	for _, want := range []string{"delete-item", "github-cli-pat", "fields GITHUB_PERSONAL_ACCESS_TOKEN", "token revoked"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan output missing %q:\n%s", want, out)
		}
	}
	out, err = runReconcile(t, v, reg, "--apply")
	if err != nil || !strings.Contains(out, "Converged") {
		t.Fatalf("apply must delete and converge: %v\n%s", err, out)
	}
	if _, still := v.items["github-cli-pat"]; still {
		t.Fatal("the retired item is still in the vault")
	}
	out, _ = runReconcile(t, v, reg)
	if !strings.Contains(out, "retired item github-cli-pat is gone") {
		t.Errorf("a deleted retired item must be reported removable:\n%s", out)
	}
	if strings.Contains(out, "PLANTED") {
		t.Fatal("a value reached the output")
	}
}
