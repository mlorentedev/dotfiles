package secrets

import (
	"fmt"
	"strings"
	"testing"
)

// CLI-083: a retired: entry may name one field, for a legacy value that is dead
// rather than equal, and bw.from reaches a multi-var secret whose variables read
// one field. Values carry PLANTED so a leak is one substring search away.

// AC1: a field entry is accepted, and each invalid form is refused. A field on an
// item a declaration names is fine as long as no declaration reads that field.
func TestRegistryValidatesRetiredFields(t *testing.T) {
	const live = "version: 1\nsecrets:\n" +
		"  - {id: A, plane: app, backend: bw, bw: {item: kept, field: k, folder: Dotfiles/apps, from: {item: legacy, field: old}}, expose: {env: A}}\n" +
		"  - {id: M, plane: app, backend: bw, bw: {item: multi, field: base, folder: Dotfiles/apps}, expose: {env: {M1: {}, M2: {field: own}}}}\n"
	ok := live + "retired:\n" +
		"  - {item: Stripe, field: backup-codes, reason: superseded by a regeneration}\n" +
		"  - {item: kept, field: stale, reason: no declaration reads it}\n" +
		"  - {item: legacy, field: other, reason: the from source is legacy/old, not this}\n" +
		"  - {item: Notes-only, field: notes, reason: a dead note}\n"
	reg, err := ParseRegistry([]byte(ok))
	if err != nil {
		t.Fatalf("well-formed field entries must parse: %v", err)
	}
	if len(reg.Retired) != 4 || reg.Retired[0].Item != "Stripe" || reg.Retired[0].Field != "backup-codes" {
		t.Fatalf("field entries not carried: %+v", reg.Retired)
	}

	bad := map[string]string{
		"blank field":                    "  - {item: Stripe, field: \"  \", reason: why}\n",
		"username":                       "  - {item: Stripe, field: username, reason: why}\n",
		"password":                       "  - {item: Stripe, field: password, reason: why}\n",
		"a destination field":            "  - {item: kept, field: k, reason: why}\n",
		"a multi-var destination field":  "  - {item: multi, field: own, reason: why}\n",
		"a multi-var default field":      "  - {item: multi, field: base, reason: why}\n",
		"a from source field":            "  - {item: legacy, field: old, reason: why}\n",
		"the same field twice":           "  - {item: S, field: f, reason: a}\n  - {item: S, field: f, reason: b}\n",
		"a field of an item gone whole":  "  - {item: S, reason: a}\n  - {item: S, field: f, reason: b}\n",
		"an item gone whole after field": "  - {item: S, field: f, reason: b}\n  - {item: S, reason: a}\n",
		"a blank reason":                 "  - {item: S, field: f, reason: \"  \"}\n",
	}
	for name, tail := range bad {
		if _, err := ParseRegistry([]byte(live + "retired:\n" + tail)); err == nil {
			t.Errorf("%s: the registry must refuse it", name)
		}
	}
}

// AC2 + AC5: a retired field the vault holds is planned last, with its reason and
// what the item keeps; an absent field or item is reported gone; a name several
// items carry blocks.
func TestPlanRetiredFields(t *testing.T) {
	retired := []RetiredItem{
		{Item: "Stripe", Field: "backup-codes", Reason: "superseded by a regeneration"},
		{Item: "Tailscale", Field: "auth-key", Reason: "past the 90-day maximum"},
		{Item: "gone", Field: "f", Reason: "item deleted"},
		{Item: "Hetzner", Field: "key", Reason: "401"},
		{Item: "Notes-only", Field: "notes", Reason: "dead note"},
	}
	items := []ItemSummary{
		{Name: "Stripe", Fields: []string{"backup-codes", "recovery"}, HasLogin: true},
		{Name: "Tailscale", Fields: []string{"other"}, HasLogin: true},
		{Name: "Hetzner", Fields: []string{"key"}}, {Name: "Hetzner"},
		{Name: "Notes-only", HasNotes: true},
	}
	p := ReconcilePlan{Ops: []ReconcileOp{retireOp("R", "d", "k", "s", "k")}}
	PlanRetiredItems(&p, retired, items)

	if len(p.Ops) != 3 || p.Ops[0].Kind != OpRetireSource {
		t.Fatalf("want the retire, then two delete-field ops, got %+v", p.Ops)
	}
	stripe := p.Ops[1]
	if stripe.Kind != OpDeleteField || stripe.Item != "Stripe" || stripe.Field != "backup-codes" || stripe.Reason != "superseded by a regeneration" {
		t.Fatalf("Stripe's dead field must be planned for deletion with its reason: %+v", stripe)
	}
	if stripe.Shape != "fields recovery; login" {
		t.Errorf("the plan must show what the item keeps, got %q", stripe.Shape)
	}
	if notes := p.Ops[2]; notes.Item != "Notes-only" || notes.Field != "notes" || notes.Shape != "empty" {
		t.Errorf("a retired note must be planned, and an emptied item shown as such: %+v", notes)
	}
	if opRank[OpDeleteField] < opRank[OpRetireSource] {
		t.Error("delete-field must rank after every copy and retire")
	}
	wantGone := map[string]bool{"Tailscale/auth-key": true, "gone/f": true}
	if len(p.RetiredGone) != len(wantGone) {
		t.Errorf("an absent field or item must be reported satisfied, got %v", p.RetiredGone)
	}
	for _, g := range p.RetiredGone {
		if !wantGone[g] {
			t.Errorf("unexpected satisfied entry %q", g)
		}
	}
	if len(p.Blocked) != 1 || p.Blocked[0].Item != "Hetzner" || !strings.Contains(p.Blocked[0].Detail, "matches 2 items") {
		t.Errorf("an ambiguous name must block: %+v", p.Blocked)
	}
}

// AC2: apply removes the field through the writer seam and deletes nothing else.
func TestApplyDeletesARetiredField(t *testing.T) {
	w := &fieldRecordingWriter{}
	p := ReconcilePlan{Ops: []ReconcileOp{{Kind: OpDeleteField, Item: "Stripe", Field: "backup-codes", Reason: "dead"}}}
	var applied []string
	if err := ApplyReconcile(p, mapReader{}, w, func(op ReconcileOp) { applied = append(applied, op.Target()) }); err != nil {
		t.Fatal(err)
	}
	if len(w.removed) != 1 || w.removed[0] != "Stripe/backup-codes" {
		t.Fatalf("want Stripe/backup-codes removed, got %v", w.removed)
	}
	if len(applied) != 1 || applied[0] != "Stripe/backup-codes" {
		t.Fatalf("the applied line must name the item and field, got %v", applied)
	}
}

// fieldRecordingWriter records field removals and refuses every other write.
type fieldRecordingWriter struct{ removed []string }

func (w *fieldRecordingWriter) SetField(string, string, string) error {
	return fmt.Errorf("unexpected")
}
func (w *fieldRecordingWriter) CreateItem(string, string, string, string) error {
	return fmt.Errorf("unexpected")
}
func (w *fieldRecordingWriter) ResolveFolder(string) (string, error) {
	return "", fmt.Errorf("unexpected")
}
func (w *fieldRecordingWriter) MoveItem(string, string) error { return fmt.Errorf("unexpected") }
func (w *fieldRecordingWriter) DeleteItem(string) error       { return fmt.Errorf("unexpected") }
func (w *fieldRecordingWriter) RemoveField(item, field string) error {
	w.removed = append(w.removed, item+"/"+field)
	return nil
}

// AC3: bw.from is accepted on a multi-var secret whose variables all read one
// field, and still refused when they read different fields.
func TestBWFromOnAMultiVarSecret(t *testing.T) {
	const same = "version: 1\nsecrets:\n" +
		"  - {id: NAN, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key, folder: Dotfiles/apps, from: {item: cloud.nan.builders, field: api-key, retire: true}}, expose: {env: [NAN_API_KEY, HIVE_WORKER_API_KEY]}}\n"
	if _, err := ParseRegistry([]byte(same)); err != nil {
		t.Fatalf("two variables over one field have one source to copy: %v", err)
	}
	const differ = "version: 1\nsecrets:\n" +
		"  - {id: X, plane: app, backend: bw, bw: {item: x, field: a, folder: Dotfiles/apps, from: {item: legacy, field: a}}, expose: {env: {X1: {}, X2: {field: b}}}}\n"
	if _, err := ParseRegistry([]byte(differ)); err == nil || !strings.Contains(err.Error(), "multi-var") {
		t.Fatalf("variables over different fields must still refuse bw.from, got %v", err)
	}
	// Every var overriding to one field: that field is the destination, so a
	// source naming it is the secret's own destination.
	const self = "version: 1\nsecrets:\n" +
		"  - {id: S, plane: app, backend: bw, bw: {item: s, field: base, folder: Dotfiles/apps, from: {item: s, field: x}}, expose: {env: {S1: {field: x}, S2: {field: x}}}}\n"
	if _, err := ParseRegistry([]byte(self)); err == nil || !strings.Contains(err.Error(), "own destination") {
		t.Fatalf("a source that is the shared destination must be refused, got %v", err)
	}
}

// AC3: the planner plans one operation for a multi-var secret over one field,
// not one per variable.
func TestPlanMultiVarFromPlansOnce(t *testing.T) {
	const reg = "version: 1\nsecrets:\n" +
		"  - {id: NAN, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key, folder: Dotfiles/apps, from: {item: cloud.nan.builders, field: api-key, retire: true}}, expose: {env: [NAN_API_KEY, HIVE_WORKER_API_KEY]}}\n"
	r, err := ParseRegistry([]byte(reg))
	if err != nil {
		t.Fatal(err)
	}
	items := []ItemSummary{
		{Name: "nan-api-key", Folder: "Dotfiles/apps", Fields: []string{"api-key"}},
		{Name: "cloud.nan.builders", Fields: []string{"api-key", "chat_id"}, HasLogin: true},
	}
	p := PlanReconcile(r.BWDeclarations(), items, []string{"Dotfiles/apps"})
	if len(p.Ops) != 1 || p.Ops[0].Kind != OpRetireSource || p.Ops[0].FromItem != "cloud.nan.builders" {
		t.Fatalf("want exactly one retire-source, got ops %+v blocked %+v", p.Ops, p.Blocked)
	}

	// With the source already retired, the record is satisfied once, not twice.
	items[1].Fields = []string{"chat_id"}
	p = PlanReconcile(r.BWDeclarations(), items, []string{"Dotfiles/apps"})
	if len(p.Ops) != 0 || len(p.Satisfied) != 1 {
		t.Fatalf("want one satisfied record and no ops, got ops %+v satisfied %v", p.Ops, p.Satisfied)
	}
}
