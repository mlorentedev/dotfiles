package secrets

import (
	"fmt"
	"sort"
)

// Reconcile: converge the store's LAYOUT toward the registry (CLI-080).
//
// Drift reports where the store disagrees with the declaration; this turns each of
// those findings into the operation that removes it. The planner reads no value and
// performs no write — it maps findings to operations, and ApplyReconcile is the only
// code that touches the store. Plan-by-default is therefore a property of the call
// graph, not of a flag someone checks.
//
// Three rules shape everything below:
//
//   - Reconcile never invents a value. An absent item or field can be created only
//     by copying from the source its declaration names (`bw.from`). Without one, the
//     finding is reported with its remedy and the plan is blocked.
//   - Reconcile never changes a value. An existing field is left alone whatever it
//     holds; changing a value is `set`'s and `rotate`'s job.
//   - Reconcile never touches an undeclared placement. A declaration with no folder
//     governs none (see LayoutDrift), so nothing it names is ever moved.

// Operation kinds, in the order they apply: a folder must exist before an item is
// moved or created into it, and an item before a field is added to it.
const (
	OpCreateFolder = "create-folder"
	OpMoveItem     = "move-item"
	OpCreateItem   = "create-item"
	OpAddField     = "add-field"
)

var opRank = map[string]int{OpCreateFolder: 0, OpMoveItem: 1, OpCreateItem: 2, OpAddField: 3}

// ReconcileOp is one planned change. Coordinates only: the value a create-item or
// add-field copies is read at apply time and never stored here, so a plan can be
// printed, logged and reviewed without holding a secret.
type ReconcileOp struct {
	Kind   string
	Secret string
	Item   string
	Field  string
	Folder string // destination folder name ("" = unfoldered)
	// FromItem/FromField name the source of the value for create-item and add-field.
	FromItem, FromField string
	// Current is where a move-item finds the item now, for the plan line.
	Current string
}

// PlanNote is a finding the plan does not turn into an operation, with what to do
// about it.
type PlanNote struct {
	Secret string
	Item   string
	Detail string
	Remedy string
}

// ReconcilePlan is everything one comparison concluded.
type ReconcilePlan struct {
	Ops []ReconcileOp
	// Blocked findings cannot be resolved without a value or a registry fix. Any of
	// them makes the plan unappliable, whole: applying the rest would leave a store
	// that is still wrong in a way the operator was told about and skipped past.
	Blocked []PlanNote
	// Deferred findings belong to another command — a dormant declaration with no
	// source is `migrate`'s job. Reported, and blocking nothing: nothing reads a
	// dormant declaration, so nothing is broken by leaving it.
	Deferred []PlanNote
	// Satisfied lists secrets whose `from:` has done its job — the destination
	// exists — so the record can be deleted from the registry.
	Satisfied []string
}

// PlanReconcile maps drift findings to operations. Pure: no store, no values.
func PlanReconcile(decls []BWDecl, items []ItemSummary, folders []string) ReconcilePlan {
	p := &planner{
		decls: decls, byName: map[string]ItemSummary{}, count: map[string]int{},
		created: map[string]bool{}, seen: map[string]bool{},
	}
	for _, it := range items {
		p.byName[it.Name] = it
		p.count[it.Name]++
	}
	for _, f := range LayoutDrift(decls, items, folders) {
		p.finding(f)
	}
	p.satisfied()
	sort.SliceStable(p.plan.Ops, func(i, j int) bool {
		return opRank[p.plan.Ops[i].Kind] < opRank[p.plan.Ops[j].Kind]
	})
	return p.plan
}

type planner struct {
	decls   []BWDecl
	byName  map[string]ItemSummary
	count   map[string]int
	created map[string]bool // absent items a create-item already covers
	seen    map[string]bool // secrets already planned or noted
	plan    ReconcilePlan
}

func (p *planner) finding(f LayoutFinding) {
	d := f.Decl
	switch f.Kind {
	case DriftFolderMissing:
		p.plan.Ops = append(p.plan.Ops, ReconcileOp{Kind: OpCreateFolder, Secret: d.Secret, Folder: d.Folder})
	case DriftItemMisfiled:
		p.plan.Ops = append(p.plan.Ops, ReconcileOp{
			Kind: OpMoveItem, Secret: d.Secret, Item: d.Item, Folder: d.Folder,
			Current: p.byName[d.Item].Folder,
		})
	case DriftItemMissing:
		// Drift reports an absent item once, but every declaration naming it needs
		// a field in it: the first sourced one creates the item, the rest add to it.
		for _, other := range p.decls {
			if other.Item == d.Item {
				p.absent(other)
			}
		}
	case DriftFieldMissing:
		p.absent(d)
	}
}

// absent plans one declaration whose item or field does not exist.
func (p *planner) absent(d BWDecl) {
	if p.seen[d.Secret] {
		return
	}
	p.seen[d.Secret] = true

	if d.From == nil {
		note := PlanNote{Secret: d.Secret, Item: d.Item, Detail: fmt.Sprintf("field %q is declared but absent, and no bw.from names a source", d.Field)}
		if d.Dormant {
			note.Remedy = "dotf secrets migrate " + d.Secret
			p.plan.Deferred = append(p.plan.Deferred, note)
			return
		}
		note.Remedy = "dotf secrets set " + d.Secret + " --yes"
		p.plan.Blocked = append(p.plan.Blocked, note)
		return
	}
	if problem := p.sourceProblem(*d.From); problem != "" {
		p.plan.Blocked = append(p.plan.Blocked, PlanNote{
			Secret: d.Secret, Item: d.Item, Detail: problem,
			Remedy: "fix bw.from on " + d.Secret + " in secrets/registry.yaml",
		})
		return
	}

	op := ReconcileOp{
		Kind: OpAddField, Secret: d.Secret, Item: d.Item, Field: d.Field, Folder: d.Folder,
		FromItem: d.From.Item, FromField: d.From.Field,
	}
	if _, exists := p.byName[d.Item]; !exists && !p.created[d.Item] {
		op.Kind = OpCreateItem
		p.created[d.Item] = true
	}
	p.plan.Ops = append(p.plan.Ops, op)
}

// sourceProblem says why a from: cannot be copied from, or "" when it can.
//
// Presence only: the plan reads no value, so an EMPTY source passes here and is
// refused at apply time, before anything is written for it.
func (p *planner) sourceProblem(f BWFrom) string {
	switch n := p.count[f.Item]; {
	case n == 0:
		return fmt.Sprintf("source item %q is not in the vault", f.Item)
	case n > 1:
		return fmt.Sprintf("source name %q matches %d items, so a copy could read either", f.Item, n)
	}
	if !hasField(p.byName[f.Item], f.Field) {
		return fmt.Sprintf("source item %q carries no field %q", f.Item, f.Field)
	}
	return ""
}

// satisfied records every from: whose destination field already exists.
func (p *planner) satisfied() {
	for _, d := range p.decls {
		if d.From == nil || p.seen[d.Secret] {
			continue
		}
		if it, ok := p.byName[d.Item]; ok && hasField(it, d.Field) {
			p.seen[d.Secret] = true
			p.plan.Satisfied = append(p.plan.Satisfied, d.Secret)
		}
	}
	sort.Strings(p.plan.Satisfied)
}
