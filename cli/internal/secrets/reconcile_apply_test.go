package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// plantedValue is distinctive so its absence from every output can be asserted.
const plantedValue = "ghp_PLANTED_c0ffee_never_print_me"

// legacyStore is the live vault's shape on 2026-09-22, reduced: the tokens are two
// of several fields in one unfoldered `GitHub` item, dockerhub sits outside its
// declared folder, and neither folder the declarations need exists.
func legacyStore(t *testing.T) (*fakeBWServe, BWServeClient, func()) {
	t.Helper()
	f := &fakeBWServe{
		status: "unlocked",
		names:  map[string]string{"gh": "GitHub", "dh": "dockerhub"},
		items: map[string]json.RawMessage{
			"gh": json.RawMessage(`{"id":"gh","type":1,"name":"GitHub","folderId":null,"login":{"username":"u","password":"pw"},` +
				`"fields":[{"name":"Personal Access Token","value":"` + plantedValue + `","type":1},` +
				`{"name":"release-token","value":"rel-` + plantedValue + `","type":1},{"name":"unrelated","value":"x","type":1}]}`),
			"dh": json.RawMessage(`{"id":"dh","type":1,"name":"dockerhub","folderId":null,"fields":[{"name":"PAT","value":"d","type":1}]}`),
		},
		folders: map[string]string{},
		// The daemon's cache: a folder created mid-run stays invisible to listings
		// until the next sync, which is what makes resolving it twice dangerous.
		staleFolders: true,
	}
	srv := httptest.NewServer(f.handler())
	return f, BWServeClient{BaseURL: srv.URL}, srv.Close
}

func githubDecls() []BWDecl {
	return []BWDecl{
		fromDecl("GITHUB_PERSONAL_ACCESS_TOKEN", "github-cli-pat", "GITHUB_PERSONAL_ACCESS_TOKEN", "Dotfiles/apps", "GitHub", "Personal Access Token"),
		fromDecl("RELEASE_TOKEN", "github-release-pat", "RELEASE_TOKEN", "Dotfiles/apps", "GitHub", "release-token"),
		decl("DOCKERHUB_TOKEN", "dockerhub", "PAT", "Dotfiles/apps", false),
	}
}

func planAgainst(t *testing.T, c BWServeClient, decls []BWDecl) ReconcilePlan {
	t.Helper()
	items, err := c.ListItems()
	if err != nil {
		t.Fatal(err)
	}
	folders, err := c.ListFolders()
	if err != nil {
		t.Fatal(err)
	}
	return PlanReconcile(decls, items, folders)
}

// THE IDEMPOTENCE PROOF. Plan against the legacy shape, apply, plan again: the
// second plan must be empty. Run against a daemon fake rather than a struct fake,
// so the listing, the reads and the writes go through the real client code and the
// store's state is what the second plan actually sees.
func TestApplyReconcileConvergesAndASecondPlanIsEmpty(t *testing.T) {
	f, c, closeSrv := legacyStore(t)
	defer closeSrv()
	w := BWServeWriter{Client: c}

	plan := planAgainst(t, c, githubDecls())
	if len(plan.Ops) != 4 || len(plan.Blocked) != 0 {
		t.Fatalf("want 4 ops (folder, move, 2 creates), got %v blocked %+v", opKinds(plan), plan.Blocked)
	}
	var applied []string
	if err := ApplyReconcile(plan, BWServeReader(w), w, func(op ReconcileOp) { applied = append(applied, op.Kind) }); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 4 {
		t.Errorf("every op must be reported applied, got %v", applied)
	}

	again := planAgainst(t, c, githubDecls())
	if len(again.Ops) != 0 || len(again.Blocked) != 0 {
		t.Fatalf("not idempotent: second plan %v blocked %+v", opKinds(again), again.Blocked)
	}
	if strings.Join(again.Satisfied, ",") != "GITHUB_PERSONAL_ACCESS_TOKEN,RELEASE_TOKEN" {
		t.Errorf("both from: records must now read as removable, got %v", again.Satisfied)
	}
	// One folder, not one per operation that needed it — against a cache that hides
	// the new folder until a sync, so resolving it a second time would duplicate it.
	if len(f.folders) != 1 {
		t.Errorf("want exactly one folder created, got %v", f.folders)
	}
	// The copy carried the right value to the right place, and left the source alone.
	got, err := BWServeReader(w).Field("github-cli-pat", "GITHUB_PERSONAL_ACCESS_TOKEN")
	if err != nil || got != plantedValue {
		t.Errorf("destination does not hold the source value (err %v)", err)
	}
	if src, _ := BWServeReader(w).Field("GitHub", "Personal Access Token"); src != plantedValue {
		t.Error("the source field must be left in place")
	}
}

// A copied value appears nowhere a human or a log can see it: not in the plan, not
// in an applied-op report, not in an error.
func TestApplyReconcileNeverPutsAValueInItsOutput(t *testing.T) {
	_, c, closeSrv := legacyStore(t)
	defer closeSrv()
	w := BWServeWriter{Client: c}
	plan := planAgainst(t, c, githubDecls())

	var seen strings.Builder
	fmt.Fprintf(&seen, "%+v", plan)
	err := ApplyReconcile(plan, BWServeReader(w), w, func(op ReconcileOp) { fmt.Fprintf(&seen, "%+v", op) })
	fmt.Fprintf(&seen, "%v", err)
	if strings.Contains(seen.String(), "PLANTED") {
		t.Fatal("a secret value reached reconcile's output")
	}
}

// Fail closed, whole: a plan carrying a blocker applies nothing at all.
func TestApplyReconcileRefusesABlockedPlan(t *testing.T) {
	f, c, closeSrv := legacyStore(t)
	defer closeSrv()
	w := BWServeWriter{Client: c}
	plan := planAgainst(t, c, append(githubDecls(), decl("LIVE", "never-created", "k", "", false)))
	if len(plan.Blocked) == 0 {
		t.Fatal("fixture should block")
	}
	err := ApplyReconcile(plan, BWServeReader(w), w, func(ReconcileOp) {})
	if !errors.Is(err, ErrPlanBlocked) {
		t.Fatalf("want ErrPlanBlocked, got %v", err)
	}
	if len(f.created) != 0 || len(f.folders) != 0 {
		t.Error("a blocked plan must write nothing")
	}
}

// An empty source is refused before ANY write — every source is read before the
// first mutation, so a bad one leaves the store exactly as it was. The plan cannot
// see emptiness (it reads no value), so apply is where it is caught; creating the
// item would produce a field `verify` reports as present.
func TestApplyReconcileRefusesAnEmptySource(t *testing.T) {
	f, c, closeSrv := legacyStore(t)
	defer closeSrv()
	f.items["gh"] = json.RawMessage(`{"id":"gh","type":1,"name":"GitHub","fields":[{"name":"Personal Access Token","value":"","type":1}]}`)
	w := BWServeWriter{Client: c}
	plan := planAgainst(t, c, githubDecls()[:1])
	err := ApplyReconcile(plan, BWServeReader(w), w, func(ReconcileOp) {})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("want an empty-source refusal, got %v", err)
	}
	if len(f.created) != 0 || len(f.folders) != 0 {
		t.Error("an empty source must stop the apply before its first write, folder included")
	}
}

// add-field: the destination item exists but lacks the declared field — the shape
// every field rename will take. The value lands in the field, the item's other
// fields survive, and a second plan is empty.
func TestApplyReconcileAddsAFieldToAnExistingItem(t *testing.T) {
	_, c, closeSrv := legacyStore(t)
	defer closeSrv()
	w := BWServeWriter{Client: c}
	decls := []BWDecl{fromDecl("GITHUB_PERSONAL_ACCESS_TOKEN", "GitHub", "GITHUB_PERSONAL_ACCESS_TOKEN", "", "GitHub", "Personal Access Token")}

	plan := planAgainst(t, c, decls)
	if got := opKinds(plan); len(got) != 1 || got[0] != OpAddField+":GitHub" {
		t.Fatalf("want one add-field, got %v", got)
	}
	if err := ApplyReconcile(plan, BWServeReader(w), w, func(ReconcileOp) {}); err != nil {
		t.Fatal(err)
	}
	if v, _ := BWServeReader(w).Field("GitHub", "GITHUB_PERSONAL_ACCESS_TOKEN"); v != plantedValue {
		t.Error("the added field does not hold the source value")
	}
	if v, _ := BWServeReader(w).Field("GitHub", "unrelated"); v != "x" {
		t.Error("adding a field must leave the item's other fields intact")
	}
	if again := planAgainst(t, c, decls); len(again.Ops) != 0 {
		t.Errorf("not idempotent: %v", opKinds(again))
	}
}
