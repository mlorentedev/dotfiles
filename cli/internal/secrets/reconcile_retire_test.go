package secrets

import (
	"encoding/json"
	"strings"
	"testing"
)

func retireDecl(secret, item, field, fromItem, fromField string) BWDecl {
	d := fromDecl(secret, item, field, "Dotfiles/apps", fromItem, fromField)
	d.From.Retire = true
	return d
}

// --- the pure core ----------------------------------------------------------

// Removing a field removes exactly that field: every other key and every other
// custom field survives, byte-for-byte in value.
func TestRemoveItemFieldRemovesOnlyThatField(t *testing.T) {
	orig := []byte(`{"id":"g","name":"GitHub","notes":"n","login":{"username":"u","password":"p"},` +
		`"fields":[{"name":"release-token","value":"r","type":1},{"name":"keep","value":"k","type":1}]}`)
	got, err := removeItemField(orig, "release-token")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	fields := m["fields"].([]any)
	if len(fields) != 1 || fields[0].(map[string]any)["name"] != "keep" {
		t.Errorf("want only the other field left, got %v", fields)
	}
	if m["notes"] != "n" || m["login"].(map[string]any)["password"] != "p" {
		t.Errorf("removing a custom field touched the rest of the item: %s", got)
	}
}

// The native fields are cleared, not deleted: notes and the login block are part
// of the item's type, the same dispatch fieldFromItem and setItemField use.
func TestRemoveItemFieldClearsNativeFields(t *testing.T) {
	const orig = `{"notes":"NOTE_V","login":{"username":"USER_V","password":"PASS_V"}}`
	cleared := map[string]string{"notes": "NOTE_V", "username": "USER_V", "password": "PASS_V"}
	for field := range cleared {
		got, err := removeItemField([]byte(orig), field)
		if err != nil {
			t.Fatalf("%s: %v", field, err)
		}
		for other, v := range cleared {
			if present := strings.Contains(string(got), v); present == (other == field) {
				t.Errorf("clearing %s: %s present=%v in %s", field, other, present, got)
			}
		}
	}
}

// A field that is not there is an error, not a silent success: the plan said it
// was, so absence means the store moved underneath the plan.
func TestRemoveItemFieldRefusesAnAbsentField(t *testing.T) {
	if _, err := removeItemField([]byte(`{"fields":[{"name":"other","value":"x"}]}`), "gone"); err == nil {
		t.Fatal("removing an absent field must fail")
	}
}

// --- the planner --------------------------------------------------------------

// Retire is planned only once the destination exists: a source is never removed in
// the same run that creates its copy, so the copy has been verified by then.
func TestPlanRetiresTheSourceOnlyOnceTheDestinationExists(t *testing.T) {
	d := retireDecl("REL", "github-release-pat", "RELEASE_TOKEN", "GitHub", "release-token")

	before := PlanReconcile([]BWDecl{d}, []ItemSummary{{Name: "GitHub", Fields: []string{"release-token"}}}, []string{"Dotfiles/apps"})
	if got := opKinds(before); len(got) != 1 || got[0] != OpCreateItem+":github-release-pat" {
		t.Fatalf("first run creates and does NOT retire, got %v", got)
	}

	after := PlanReconcile([]BWDecl{d}, []ItemSummary{
		{Name: "GitHub", Fields: []string{"release-token"}},
		{Name: "github-release-pat", Folder: "Dotfiles/apps", Fields: []string{"RELEASE_TOKEN"}},
	}, []string{"Dotfiles/apps"})
	if got := opKinds(after); len(got) != 1 || got[0] != OpRetireSource+":github-release-pat" {
		t.Fatalf("second run retires the source, got %v", got)
	}
	if len(after.Satisfied) != 0 {
		t.Errorf("a from: whose source is still to retire is not yet removable: %v", after.Satisfied)
	}

	done := PlanReconcile([]BWDecl{d}, []ItemSummary{
		{Name: "GitHub", Fields: []string{"other"}},
		{Name: "github-release-pat", Folder: "Dotfiles/apps", Fields: []string{"RELEASE_TOKEN"}},
	}, []string{"Dotfiles/apps"})
	if len(done.Ops) != 0 || len(done.Satisfied) != 1 {
		t.Errorf("once retired: no ops, record removable; got ops %v satisfied %v", opKinds(done), done.Satisfied)
	}
}

// Without retire, a satisfied from: never touches its source.
func TestPlanLeavesTheSourceWithoutRetire(t *testing.T) {
	p := PlanReconcile(
		[]BWDecl{fromDecl("REL", "github-release-pat", "RELEASE_TOKEN", "Dotfiles/apps", "GitHub", "release-token")},
		[]ItemSummary{
			{Name: "GitHub", Fields: []string{"release-token"}},
			{Name: "github-release-pat", Folder: "Dotfiles/apps", Fields: []string{"RELEASE_TOKEN"}},
		}, []string{"Dotfiles/apps"})
	if len(p.Ops) != 0 {
		t.Fatalf("no retire requested, so nothing to do: %v", opKinds(p))
	}
}

// An ambiguous source cannot be retired: removing a field from an arbitrary one of
// several same-named items could delete the wrong credential.
func TestPlanBlocksRetiringAnAmbiguousSource(t *testing.T) {
	p := PlanReconcile(
		[]BWDecl{retireDecl("REL", "github-release-pat", "RELEASE_TOKEN", "GitHub", "release-token")},
		[]ItemSummary{
			{Name: "GitHub", Fields: []string{"release-token"}},
			{Name: "GitHub", Fields: []string{"release-token"}},
			{Name: "github-release-pat", Folder: "Dotfiles/apps", Fields: []string{"RELEASE_TOKEN"}},
		}, []string{"Dotfiles/apps"})
	if len(p.Ops) != 0 || len(p.Blocked) != 1 {
		t.Fatalf("want a blocker, got ops %v blocked %+v", opKinds(p), p.Blocked)
	}
}

// --- apply, against the daemon fake -----------------------------------------

func convergedStore(t *testing.T) (*fakeBWServe, BWServeClient, func()) {
	t.Helper()
	f, c, closeSrv := legacyStore(t)
	w := BWServeWriter{Client: c}
	if err := ApplyReconcile(planAgainst(t, c, githubDecls()), BWServeReader(w), w, func(ReconcileOp) {}); err != nil {
		t.Fatal(err)
	}
	return f, c, closeSrv
}

func retireDecls() []BWDecl {
	return []BWDecl{
		retireDecl("GITHUB_PERSONAL_ACCESS_TOKEN", "github-cli-pat", "GITHUB_PERSONAL_ACCESS_TOKEN", "GitHub", "Personal Access Token"),
		retireDecl("RELEASE_TOKEN", "github-release-pat", "RELEASE_TOKEN", "GitHub", "release-token"),
	}
}

// Retire removes exactly the migrated fields, leaves the item's other credentials,
// and a third plan is empty with both records removable.
func TestApplyRetireRemovesOnlyTheMigratedFieldsAndConverges(t *testing.T) {
	_, c, closeSrv := convergedStore(t)
	defer closeSrv()
	w := BWServeWriter{Client: c}

	plan := planAgainst(t, c, retireDecls())
	if len(plan.Ops) != 2 {
		t.Fatalf("want two retires, got %v", opKinds(plan))
	}
	if err := ApplyReconcile(plan, BWServeReader(w), w, func(ReconcileOp) {}); err != nil {
		t.Fatal(err)
	}
	r := BWServeReader(w)
	if _, err := r.Field("GitHub", "release-token"); err == nil {
		t.Error("the retired source field is still there")
	}
	if v, _ := r.Field("GitHub", "unrelated"); v != "x" {
		t.Error("retire removed a credential it was not asked to")
	}
	if v, _ := r.Field("github-release-pat", "RELEASE_TOKEN"); v != "rel-"+plantedValue {
		t.Error("the destination must be untouched by retiring its source")
	}
	again := planAgainst(t, c, retireDecls())
	if len(again.Ops) != 0 || len(again.Satisfied) != 2 {
		t.Errorf("not converged: ops %v satisfied %v", opKinds(again), again.Satisfied)
	}
}

// If the two sides differ, one of them was rotated and the tool cannot know which
// is the truth. Nothing is removed — not the differing pair, not any other.
func TestApplyRetireRefusesWhenTheValuesDiffer(t *testing.T) {
	f, c, closeSrv := convergedStore(t)
	defer closeSrv()
	w := BWServeWriter{Client: c}
	if err := w.SetField("github-release-pat", "RELEASE_TOKEN", "rotated-since"); err != nil {
		t.Fatal(err)
	}
	before := string(f.items["gh"])

	err := ApplyReconcile(planAgainst(t, c, retireDecls()), BWServeReader(w), w, func(ReconcileOp) {})
	if err == nil || !strings.Contains(err.Error(), "differ") {
		t.Fatalf("want a refusal naming the difference, got %v", err)
	}
	if strings.Contains(err.Error(), "rotated-since") || strings.Contains(err.Error(), "PLANTED") {
		t.Fatal("the refusal leaked a value")
	}
	if string(f.items["gh"]) != before {
		t.Error("a refused retire must remove nothing, from either field")
	}
}

// Parity: the daemon stores exactly what the pure core produces.
func TestBWServeWriter_RemoveField_MatchesTheCore(t *testing.T) {
	f, w, closeSrv := newWriterFake(t)
	defer closeSrv()
	original := append(json.RawMessage(nil), f.items["item-id-1"]...)
	if err := w.RemoveField("dockerhub", "PAT"); err != nil {
		t.Fatal(err)
	}
	want, err := removeItemField(original, "PAT")
	if err != nil {
		t.Fatal(err)
	}
	if string(f.items["item-id-1"]) != string(want) {
		t.Fatalf("daemon diverged from the core:\n got: %s\nwant: %s", f.items["item-id-1"], want)
	}
}

// Retire runs last in any plan. It is the only operation that deletes, so if any
// earlier write fails mid-apply, nothing has been removed yet.
func TestPlanPutsRetireAfterEveryOtherOperation(t *testing.T) {
	p := PlanReconcile(
		[]BWDecl{
			retireDecl("REL", "github-release-pat", "RELEASE_TOKEN", "GitHub", "release-token"),
			fromDecl("NEW", "zz-new", "K", "Dotfiles/apps", "src", "k"),
		},
		[]ItemSummary{
			{Name: "GitHub", Fields: []string{"release-token"}},
			{Name: "github-release-pat", Folder: "Dotfiles/apps", Fields: []string{"RELEASE_TOKEN"}},
			{Name: "src", Fields: []string{"k"}},
		}, []string{"Dotfiles/apps"})
	if n := len(p.Ops); n != 2 || p.Ops[n-1].Kind != OpRetireSource {
		t.Fatalf("retire must be the last operation, got %v", opKinds(p))
	}
}
