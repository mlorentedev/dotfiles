package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// CLI-082: the plan says what a retire will do, and items are retired by
// declaration. Every value below carries PLANTED so a leak into any output,
// error or plan is one substring search away.

// mapReader answers item/field lookups from a map; a missing key is not found.
type mapReader map[string]string

func (m mapReader) Field(item, field string) (string, error) {
	v, ok := m[item+"/"+field]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", ErrBWItemNotFound, item, field)
	}
	return v, nil
}

func retireOp(secret, item, field, fromItem, fromField string) ReconcileOp {
	return ReconcileOp{Kind: OpRetireSource, Secret: secret, Item: item, Field: field, FromItem: fromItem, FromField: fromField}
}

// AC1 + AC2: every planned retire gets a verdict. Only an equal one stays an
// operation; the rest become blockers that say how to settle them. Nothing
// derived from a value is printed.
func TestVerifyRetiresGivesEachRetireAVerdict(t *testing.T) {
	add := ReconcileOp{Kind: OpAddField, Secret: "ADD", Item: "a", Field: "f", FromItem: "s", FromField: "g"}
	p := ReconcilePlan{Ops: []ReconcileOp{
		add,
		retireOp("EQ", "eq-dst", "k", "eq-src", "k"),
		retireOp("DIFF", "diff-dst", "k", "diff-src", "k"),
		retireOp("EMPTY", "empty-dst", "k", "empty-src", "k"),
		retireOp("UNREAD", "gone-dst", "k", "unread-src", "k"),
	}}
	r := mapReader{
		"eq-dst/k": "PLANTED-same", "eq-src/k": "PLANTED-same",
		"diff-dst/k": "PLANTED-new", "diff-src/k": "PLANTED-old",
		"empty-dst/k": "", "empty-src/k": "PLANTED-orphan",
		"unread-src/k": "PLANTED-unread",
	}

	VerifyRetires(&p, r)

	if len(p.Ops) != 2 || p.Ops[0].Kind != OpAddField || p.Ops[1].Secret != "EQ" || p.Ops[1].Verdict != RetireEqual {
		t.Fatalf("only the add-field and the equal retire may stay operations, got %+v", p.Ops)
	}
	want := map[string]string{"DIFF": RetireDiffers, "EMPTY": RetireDestinationEmpty, "UNREAD": RetireUnreadable}
	if len(p.Blocked) != len(want) {
		t.Fatalf("want %d blockers, got %+v", len(want), p.Blocked)
	}
	for _, b := range p.Blocked {
		verdict, ok := want[b.Secret]
		if !ok || !strings.Contains(b.Detail, verdict) {
			t.Errorf("%s: blocker must carry its verdict %q: %+v", b.Secret, verdict, b)
		}
	}
	if s := fmt.Sprintf("%+v", p); strings.Contains(s, "PLANTED") {
		t.Fatalf("a value reached the plan: %s", s)
	}
}

// AC2: a blocker says how to settle it. differs and destination empty have the
// same two exits; unreadable names the side the store did not answer for.
func TestVerifyRetiresNamesHowToSettleEachBlocker(t *testing.T) {
	p := ReconcilePlan{Ops: []ReconcileOp{
		retireOp("DIFF", "d", "k", "s", "k"),
		retireOp("UNREAD", "missing", "k", "s2", "k"),
	}}
	VerifyRetires(&p, mapReader{"d/k": "PLANTED-1", "s/k": "PLANTED-2", "s2/k": "PLANTED-3"})
	for _, b := range p.Blocked {
		switch b.Secret {
		case "DIFF":
			if !strings.Contains(b.Remedy, "dotf secrets rotate DIFF") || !strings.Contains(b.Remedy, "retire:") {
				t.Errorf("differs must name both exits: %q", b.Remedy)
			}
		case "UNREAD":
			if !strings.Contains(b.Detail, "missing/") {
				t.Errorf("unreadable must name the side that could not be read: %q", b.Detail)
			}
		}
	}
}

// AC2, the apply half: the refusal names the same exits, so an operator who
// skipped the plan is not left with "settle it" (#1624 item 1).
func TestApplyRetireRefusalNamesTheExits(t *testing.T) {
	p := ReconcilePlan{Ops: []ReconcileOp{retireOp("REL", "d", "k", "s", "k")}}
	err := ApplyReconcile(p, mapReader{"d/k": "PLANTED-a", "s/k": "PLANTED-b"}, nil, func(ReconcileOp) {})
	if err == nil || !strings.Contains(err.Error(), "dotf secrets rotate REL") || !strings.Contains(err.Error(), "retire:") {
		t.Fatalf("the refusal must name both exits, got %v", err)
	}
	if strings.Contains(err.Error(), "PLANTED") {
		t.Fatal("the refusal leaked a value")
	}
}

// AC3: retired entries are declared with a reason, once, and never for an item a
// declaration still uses.
func TestRegistryValidatesRetiredItems(t *testing.T) {
	const live = "version: 1\nsecrets:\n" +
		"  - {id: A, plane: app, backend: bw, bw: {item: kept, field: k, folder: Dotfiles/apps, from: {item: legacy, field: old}}, expose: {env: A}}\n"
	ok := live + "retired:\n  - {item: husk, reason: revoked 2026-09-23}\n"
	reg, err := ParseRegistry([]byte(ok))
	if err != nil {
		t.Fatalf("a well-formed retired list must parse: %v", err)
	}
	if len(reg.Retired) != 1 || reg.Retired[0].Item != "husk" || reg.Retired[0].Reason == "" {
		t.Fatalf("retired entry not carried: %+v", reg.Retired)
	}

	bad := map[string]string{
		"no reason":            "retired:\n  - {item: husk}\n",
		"no item":              "retired:\n  - {reason: why}\n",
		"listed twice":         "retired:\n  - {item: husk, reason: a}\n  - {item: husk, reason: b}\n",
		"still a bw.item":      "retired:\n  - {item: kept, reason: why}\n",
		"still a bw.from.item": "retired:\n  - {item: legacy, reason: why}\n",
	}
	for name, tail := range bad {
		if _, err := ParseRegistry([]byte(live + tail)); err == nil {
			t.Errorf("%s: the registry must refuse it", name)
		}
	}
}

// AC4 + AC5: a retired item the vault holds is planned for deletion, after every
// other operation, with its shape shown; one already gone is reported; an
// ambiguous name blocks.
func TestPlanRetiredItems(t *testing.T) {
	retired := []RetiredItem{
		{Item: "husk", Reason: "emptied by a retire"},
		{Item: "gone", Reason: "deleted by hand"},
		{Item: "twice", Reason: "ambiguous"},
	}
	items := []ItemSummary{
		{Name: "husk", HasNotes: true},
		{Name: "twice"}, {Name: "twice"},
		{Name: "other"},
	}
	p := ReconcilePlan{Ops: []ReconcileOp{retireOp("R", "d", "k", "s", "k")}}
	PlanRetiredItems(&p, retired, items)

	if len(p.Ops) != 2 || p.Ops[1].Kind != OpDeleteItem || p.Ops[1].Item != "husk" {
		t.Fatalf("want the retire then one delete-item for husk, got %+v", p.Ops)
	}
	if !strings.Contains(p.Ops[1].Shape, "notes") || p.Ops[1].Reason != "emptied by a retire" {
		t.Errorf("delete-item must carry the item's shape and the declared reason: %+v", p.Ops[1])
	}
	if opRank[OpDeleteItem] <= opRank[OpRetireSource] {
		t.Error("delete-item must rank after every other operation")
	}
	if len(p.RetiredGone) != 1 || p.RetiredGone[0] != "gone" {
		t.Errorf("an absent retired item must be reported satisfied: %v", p.RetiredGone)
	}
	if len(p.Blocked) != 1 || p.Blocked[0].Item != "twice" {
		t.Errorf("an ambiguous retired item must block: %+v", p.Blocked)
	}
}

// AC4: apply deletes through the writer seam, last.
func TestApplyDeletesARetiredItem(t *testing.T) {
	w := &recordingWriter{}
	p := ReconcilePlan{Ops: []ReconcileOp{{Kind: OpDeleteItem, Item: "husk", Reason: "why"}}}
	var applied []string
	if err := ApplyReconcile(p, mapReader{}, w, func(op ReconcileOp) { applied = append(applied, op.Target()) }); err != nil {
		t.Fatal(err)
	}
	if len(w.deleted) != 1 || w.deleted[0] != "husk" || len(applied) != 1 || applied[0] != "husk" {
		t.Fatalf("want husk deleted and reported, got deleted %v applied %v", w.deleted, applied)
	}
}

// recordingWriter is a BWWriteClient that records deletions and refuses the rest.
type recordingWriter struct{ deleted []string }

func (w *recordingWriter) SetField(string, string, string) error { return fmt.Errorf("unexpected") }
func (w *recordingWriter) CreateItem(string, string, string, string) error {
	return fmt.Errorf("unexpected")
}
func (w *recordingWriter) ResolveFolder(string) (string, error) { return "", fmt.Errorf("unexpected") }
func (w *recordingWriter) MoveItem(string, string) error        { return fmt.Errorf("unexpected") }
func (w *recordingWriter) RemoveField(string, string) error     { return fmt.Errorf("unexpected") }
func (w *recordingWriter) DeleteItem(item string) error {
	w.deleted = append(w.deleted, item)
	return nil
}

// The daemon transport: resolve the name to one id, DELETE it, then sync so the
// next plan does not still list it.
func TestBWServeWriter_DeleteItem(t *testing.T) {
	f, w, closeSrv := newWriterFake(t)
	defer closeSrv()
	syncs := f.syncs
	if err := w.DeleteItem("dockerhub"); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if len(f.deleted) != 1 || f.deleted[0] != "item-id-1" {
		t.Fatalf("want item-id-1 deleted, got %v", f.deleted)
	}
	if f.syncs == syncs {
		t.Error("a delete must sync, or the next plan still lists the item")
	}
	if err := w.DeleteItem("never-there"); !errors.Is(err, ErrBWItemNotFound) {
		t.Errorf("deleting an absent item must say so, got %v", err)
	}
}

// The CLI transport resolves the name the same way and deletes by id, never by
// name: `bw delete item <name>` would act on whichever item bw picked.
func TestBWPut_DeleteItemDeletesByID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake bw is a POSIX shell script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	bin := filepath.Join(dir, "bw")
	script := "#!/bin/sh\necho \"$*\" >> " + log + "\n" +
		"case \"$1 $2\" in \"get item\") echo '{\"id\":\"id-7\",\"name\":\"husk\"}' ;; esac\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := (BWPut{Bin: bin}).DeleteItem("husk"); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	calls, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(calls), "delete item id-7") {
		t.Fatalf("want a delete by id, got calls:\n%s", calls)
	}
}
