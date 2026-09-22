package secrets

import (
	"strings"
	"testing"
)

func fromDecl(secret, item, field, folder, fromItem, fromField string) BWDecl {
	return BWDecl{Secret: secret, Var: secret, Item: item, Field: field, Folder: folder,
		From: &BWFrom{Item: fromItem, Field: fromField}}
}

func opKinds(p ReconcilePlan) []string {
	out := make([]string, 0, len(p.Ops))
	for _, o := range p.Ops {
		out = append(out, o.Kind+":"+o.Item)
	}
	return out
}

// The GitHub case, end to end at the plan level: the destination item is absent,
// its folder is absent, and the value lives as one field of a legacy item.
func TestPlanCreatesAnAbsentItemFromItsDeclaredSource(t *testing.T) {
	p := PlanReconcile(
		[]BWDecl{fromDecl("PAT", "github-cli-pat", "PAT", "Dotfiles/apps", "GitHub", "Personal Access Token")},
		[]ItemSummary{{Name: "GitHub", Fields: []string{"Personal Access Token", "other"}}},
		nil,
	)
	if len(p.Blocked) != 0 {
		t.Fatalf("nothing should block: %+v", p.Blocked)
	}
	want := []string{OpCreateFolder + ":", OpCreateItem + ":github-cli-pat"}
	if got := opKinds(p); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
	op := p.Ops[1]
	if op.Folder != "Dotfiles/apps" || op.Field != "PAT" || op.FromItem != "GitHub" || op.FromField != "Personal Access Token" {
		t.Errorf("create-item carries the wrong coordinates: %+v", op)
	}
}

// Operations come out in the order they can be applied: a folder must exist before
// anything moves into it, and an item must exist before a field is added to it.
func TestPlanOrdersOperationsByDependency(t *testing.T) {
	// Names chosen against alphabetical order: drift sorts findings by item name,
	// so an absent "aa-shared" reports BEFORE a misfiled "zz-dockerhub", and only
	// the dependency sort puts the move first.
	p := PlanReconcile(
		[]BWDecl{
			fromDecl("B", "aa-shared", "b", "", "src", "b"), // same item as A, second field
			fromDecl("A", "aa-shared", "a", "", "src", "a"),
			decl("D", "zz-dockerhub", "PAT", "Dotfiles/apps", false),
		},
		[]ItemSummary{{Name: "src", Fields: []string{"a", "b"}}, {Name: "zz-dockerhub", Fields: []string{"PAT"}}},
		nil,
	)
	var got []string
	for _, o := range p.Ops {
		got = append(got, o.Kind)
	}
	want := []string{OpCreateFolder, OpMoveItem, OpCreateItem, OpAddField}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
	// Two secrets sharing one absent item: the first creates it, the second adds
	// its field. Planning two create-items would make a duplicate item.
	if p.Ops[2].Item != "aa-shared" || p.Ops[3].Item != "aa-shared" || p.Ops[2].Field == p.Ops[3].Field {
		t.Errorf("a shared absent item must be created once and extended once: %+v", p.Ops[2:])
	}
}

// Reconcile never changes a value. A destination field that exists is left alone
// whatever it holds, and the from: that pointed at it is reported as removable.
func TestPlanNeverOverwritesAndReportsASatisfiedFrom(t *testing.T) {
	p := PlanReconcile(
		[]BWDecl{fromDecl("PAT", "github-cli-pat", "PAT", "Dotfiles/apps", "GitHub", "Personal Access Token")},
		[]ItemSummary{
			{Name: "github-cli-pat", Folder: "Dotfiles/apps", Fields: []string{"PAT"}},
			{Name: "GitHub", Fields: []string{"Personal Access Token"}},
		},
		[]string{"Dotfiles/apps"},
	)
	if len(p.Ops) != 0 || len(p.Blocked) != 0 {
		t.Fatalf("a converged store plans nothing: ops %v blocked %+v", opKinds(p), p.Blocked)
	}
	if len(p.Satisfied) != 1 || p.Satisfied[0] != "PAT" {
		t.Errorf("the satisfied from: must be reported as removable, got %v", p.Satisfied)
	}
}

// An absent field on an existing item is added, from its source.
func TestPlanAddsAnAbsentFieldFromItsSource(t *testing.T) {
	p := PlanReconcile(
		[]BWDecl{fromDecl("S", "svc", "NEW_NAME", "", "svc", "old-name")},
		[]ItemSummary{{Name: "svc", Fields: []string{"old-name"}}},
		nil,
	)
	if got := opKinds(p); len(got) != 1 || got[0] != OpAddField+":svc" {
		t.Fatalf("got %v", got)
	}
}

// Fail closed. Each of these is a finding reconcile cannot turn into an operation
// without inventing something, and each must name its remedy.
func TestPlanBlocksWhatItCannotSource(t *testing.T) {
	cases := map[string]struct {
		decl   BWDecl
		items  []ItemSummary
		remedy string
	}{
		"live item absent, no source": {
			decl("L", "live-item", "k", "", false), nil, "dotf secrets set L",
		},
		"live field absent, no source": {
			decl("L", "live-item", "k", "", false), []ItemSummary{{Name: "live-item"}}, "dotf secrets set L",
		},
		"source item absent": {
			fromDecl("S", "dest", "k", "", "gone", "f"), nil, "gone",
		},
		"source field absent": {
			fromDecl("S", "dest", "k", "", "src", "f"), []ItemSummary{{Name: "src", Fields: []string{"other"}}}, "src",
		},
		// Two items share the source's name, so a copy could read either one.
		"source ambiguous": {
			fromDecl("S", "dest", "k", "", "dup", "f"),
			[]ItemSummary{{Name: "dup", Fields: []string{"f"}}, {Name: "dup", Fields: []string{"f"}}},
			"2 items",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := PlanReconcile([]BWDecl{tc.decl}, tc.items, nil)
			if len(p.Blocked) != 1 {
				t.Fatalf("want one blocker, got %+v (ops %v)", p.Blocked, opKinds(p))
			}
			if !strings.Contains(p.Blocked[0].Remedy+" "+p.Blocked[0].Detail, tc.remedy) {
				t.Errorf("blocker must name %q: %+v", tc.remedy, p.Blocked[0])
			}
		})
	}
}

// A dormant declaration (age-backed secret) with no source is migrate's job, not a
// defect in the plan: nothing reads it yet, so nothing is broken by leaving it.
// Deferred rather than blocked, or one pending migration would stop every other
// operation from ever applying.
func TestPlanDefersADormantItemWithNoSource(t *testing.T) {
	p := PlanReconcile([]BWDecl{decl("Z", "zoho", "k", "", true)}, nil, nil)
	if len(p.Blocked) != 0 {
		t.Fatalf("a dormant absent item must not block: %+v", p.Blocked)
	}
	if len(p.Deferred) != 1 || !strings.Contains(p.Deferred[0].Remedy, "dotf secrets migrate Z") {
		t.Errorf("must be deferred to migrate: %+v", p.Deferred)
	}
}

// A declaration with no folder governs no placement (see LayoutDrift), so reconcile
// never plans a move for it — it would unfile what the operator filed by hand.
func TestPlanNeverMovesAnItemDeclaredWithNoFolder(t *testing.T) {
	p := PlanReconcile(
		[]BWDecl{decl("G", "gmail-backup-code", "notes", "", false)},
		[]ItemSummary{{Name: "gmail-backup-code", Folder: "Personal", HasNotes: true}},
		[]string{"Personal"},
	)
	if len(p.Ops) != 0 {
		t.Fatalf("no operation for an ungoverned placement, got %v", opKinds(p))
	}
}

// A note names what is actually missing: the item when the item is absent, the
// field only when the item exists without it.
func TestPlanNotesNameWhatIsMissing(t *testing.T) {
	absentItem := PlanReconcile([]BWDecl{decl("L", "gone", "k", "", false)}, nil, nil)
	absentField := PlanReconcile([]BWDecl{decl("L", "here", "k", "", false)}, []ItemSummary{{Name: "here"}}, nil)
	if !strings.Contains(absentItem.Blocked[0].Detail, "item is declared but absent") {
		t.Errorf("absent item: %q", absentItem.Blocked[0].Detail)
	}
	if !strings.Contains(absentField.Blocked[0].Detail, `field "k"`) {
		t.Errorf("absent field: %q", absentField.Blocked[0].Detail)
	}
}
