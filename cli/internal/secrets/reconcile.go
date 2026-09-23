package secrets

import (
	"crypto/subtle"
	"errors"
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
	// OpRetireSource removes a from: source field once its destination exists and
	// holds the same value. Last, because it is the only operation that deletes.
	OpRetireSource = "retire-source"
)

var opRank = map[string]int{OpCreateFolder: 0, OpMoveItem: 1, OpCreateItem: 2, OpAddField: 3, OpRetireSource: 4}

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

// Target is what the operation acts on: the folder for create-folder, the SOURCE
// field for retire-source (that is what it removes), else the item.
func (op ReconcileOp) Target() string {
	switch op.Kind {
	case OpCreateFolder:
		return op.Folder
	case OpRetireSource:
		return op.FromItem + "/" + op.FromField
	}
	return op.Item
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
	case DriftItemAmbiguous:
		// Every operation would act on an arbitrary one of the same-named items.
		p.seen[d.Secret] = true
		p.plan.Blocked = append(p.plan.Blocked, PlanNote{
			Secret: d.Secret, Item: d.Item, Detail: f.Detail,
			Remedy: "rename or remove the duplicate items in the vault",
		})
	default:
		// A finding kind this planner does not know must not vanish: an unmapped
		// kind silently dropped is a plan that reports the store converged while
		// drift still disagrees.
		p.plan.Blocked = append(p.plan.Blocked, PlanNote{
			Secret: d.Secret, Item: d.Item, Detail: f.Kind + ": " + f.Detail,
			Remedy: "reconcile has no operation for this finding; teach PlanReconcile before applying",
		})
	}
}

// absent plans one declaration whose item or field does not exist.
func (p *planner) absent(d BWDecl) {
	if p.seen[d.Secret] {
		return
	}
	p.seen[d.Secret] = true

	if d.From == nil {
		missing := fmt.Sprintf("field %q is declared but absent", d.Field)
		if _, exists := p.byName[d.Item]; !exists {
			missing = "item is declared but absent"
		}
		note := PlanNote{Secret: d.Secret, Item: d.Item, Detail: missing + ", and no bw.from names a source"}
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

// satisfied handles every from: whose destination field already exists: it is
// removable, unless it asks to retire a source that is still there, in which case
// that removal is planned and the record stays until it lands.
//
// Retire is planned only here, so only once the destination exists: a source is
// never removed in the run that creates its copy.
func (p *planner) satisfied() {
	for _, d := range p.decls {
		if d.From == nil || p.seen[d.Secret] {
			continue
		}
		it, ok := p.byName[d.Item]
		if !ok || !hasField(it, d.Field) {
			continue
		}
		p.seen[d.Secret] = true
		if d.From.Retire && p.sourceStillThere(d) {
			continue
		}
		p.plan.Satisfied = append(p.plan.Satisfied, d.Secret)
	}
	sort.Strings(p.plan.Satisfied)
}

// sourceStillThere plans the retirement of d's source when it is present, blocks
// when it is ambiguous, and reports false when there is nothing left to retire.
func (p *planner) sourceStillThere(d BWDecl) bool {
	src := *d.From
	switch n := p.count[src.Item]; {
	case n == 0:
		return false
	case n > 1:
		p.plan.Blocked = append(p.plan.Blocked, PlanNote{
			Secret: d.Secret, Item: src.Item,
			Detail: fmt.Sprintf("cannot retire %s/%q: the name matches %d items, and removing a field from an arbitrary one could delete the wrong credential", src.Item, src.Field, n),
			Remedy: "rename or remove the duplicate items in the vault",
		})
		return true
	}
	if !hasField(p.byName[src.Item], src.Field) {
		return false
	}
	p.plan.Ops = append(p.plan.Ops, ReconcileOp{
		Kind: OpRetireSource, Secret: d.Secret, Item: d.Item, Field: d.Field,
		FromItem: src.Item, FromField: src.Field,
	})
	return true
}

// ErrPlanBlocked is returned when asked to apply a plan that carries a blocker.
var ErrPlanBlocked = errors.New("the plan has blocked findings; nothing was applied")

// ApplyReconcile performs a plan's operations in order, reporting each as it lands.
//
// Two phases, so a bad source costs nothing. Every value a copy needs is read
// first, into memory only; an unreadable or EMPTY source aborts before the first
// write. Only then are the operations performed. A failure mid-way stops at the
// failing operation and says how far it got — every operation is additive and the
// plan is recomputed from the store on the next run, so re-running converges.
//
// Each folder is resolved once and its id reused. ResolveFolder creates on miss and
// the daemon answers from a cache, so resolving the same new folder twice in one run
// could read it absent the second time and create a duplicate.
func ApplyReconcile(p ReconcilePlan, r BWReader, w BWWriteClient, applied func(ReconcileOp)) error {
	if len(p.Blocked) > 0 {
		return fmt.Errorf("%w (%d blocked)", ErrPlanBlocked, len(p.Blocked))
	}
	values := make([]string, len(p.Ops))
	for i, op := range p.Ops {
		if op.Kind == OpRetireSource {
			if err := sameValue(r, op); err != nil {
				return err
			}
			continue
		}
		if op.Kind != OpCreateItem && op.Kind != OpAddField {
			continue
		}
		v, err := r.Field(op.FromItem, op.FromField)
		if err != nil {
			return fmt.Errorf("read source %s/%s for %s (nothing written): %w", op.FromItem, op.FromField, op.Item, err)
		}
		if v == "" {
			return fmt.Errorf("source %s/%s for %s is empty; refusing to create a field `verify` would report as present (nothing written)",
				op.FromItem, op.FromField, op.Item)
		}
		values[i] = v
	}

	folderIDs := map[string]string{}
	folderID := func(name string) (string, error) {
		if id, ok := folderIDs[name]; ok {
			return id, nil
		}
		id, err := w.ResolveFolder(name)
		if err == nil {
			folderIDs[name] = id
		}
		return id, err
	}
	for i, op := range p.Ops {
		if err := applyOp(op, values[i], w, folderID); err != nil {
			return fmt.Errorf("%s %s: %w (%d of %d operations applied; re-run to converge)",
				op.Kind, op.Target(), err, i, len(p.Ops))
		}
		applied(op)
	}
	return nil
}

func applyOp(op ReconcileOp, value string, w BWWriteClient, folderID func(string) (string, error)) error {
	switch op.Kind {
	case OpCreateFolder:
		_, err := folderID(op.Folder)
		return err
	case OpMoveItem:
		id, err := folderID(op.Folder)
		if err != nil {
			return err
		}
		return w.MoveItem(op.Item, id)
	case OpCreateItem:
		id, err := folderID(op.Folder)
		if err != nil {
			return err
		}
		return w.CreateItem(op.Item, op.Field, value, id)
	case OpAddField:
		return w.SetField(op.Item, op.Field, value)
	case OpRetireSource:
		return w.RemoveField(op.FromItem, op.FromField)
	}
	return fmt.Errorf("unknown operation %q", op.Kind)
}

// sameValue verifies, before any write, that a retire's destination holds exactly
// its source's value. If they differ, one side was rotated after the copy and the
// tool cannot know which is the truth, so it refuses — naming the fields, never the
// values. Compared in constant time out of habit: nothing here is a remote oracle,
// but it costs nothing to not be the example someone copies into one.
func sameValue(r BWReader, op ReconcileOp) error {
	dst, err := r.Field(op.Item, op.Field)
	if err != nil {
		return fmt.Errorf("read %s/%s to verify before retiring its source (nothing written): %w", op.Item, op.Field, err)
	}
	src, err := r.Field(op.FromItem, op.FromField)
	if err != nil {
		return fmt.Errorf("read %s/%s to verify before retiring it (nothing written): %w", op.FromItem, op.FromField, err)
	}
	if dst == "" || subtle.ConstantTimeCompare([]byte(dst), []byte(src)) != 1 {
		return fmt.Errorf("refusing to retire %s/%q: its value and %s/%q's differ, so one was rotated after the copy "+
			"and the tool cannot tell which is current (nothing written) — settle it, then re-run",
			op.FromItem, op.FromField, op.Item, op.Field)
	}
	return nil
}
