package secrets

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

// curate applies a reviewed plan to the items the registry does NOT declare
// (SEC-006). reconcile converges what the registry declares; the ~160 personal
// logins beside it are out of the registry's bounds by design, and until this
// existed the only way to reorganise them was by hand in the web vault.
//
// The plan is data, kept outside this repo: item names are personal, and a public
// repo is no place for them. What lives here is the engine and its rules. Every
// op has a converged predicate, so a second run of the same plan is empty; every
// risky op has a precondition, checked against the store as it is when the plan
// is made; and nothing a precondition reads is ever printed.

// CurateOp is one kind of plan row.
type CurateOp string

// The plan's operations. See the SEC-006 proposal for the table of effects.
const (
	CurFolder       CurateOp = "folder"
	CurDelete       CurateOp = "delete"
	CurMergeDelete  CurateOp = "merge-delete"
	CurDeleteField  CurateOp = "delete-field"
	CurCopyUsername CurateOp = "copy-username"
	CurAddURI       CurateOp = "add-uri"
	CurHide         CurateOp = "hide"
	CurReprompt     CurateOp = "reprompt"
)

// curateOps records, per op, whether the target names a field (`id:field`) and
// whether the third column is required.
var curateOps = map[CurateOp]struct{ field, arg bool }{
	CurFolder:       {arg: true},
	CurDelete:       {},
	CurMergeDelete:  {arg: true},
	CurDeleteField:  {field: true},
	CurCopyUsername: {field: true, arg: true},
	CurAddURI:       {arg: true},
	CurHide:         {field: true},
	CurReprompt:     {},
}

// CurateRow is one parsed plan line.
type CurateRow struct {
	Line   int
	Op     CurateOp
	Target string // an item id or id prefix
	Field  string // for the ops whose target is `id:field`
	Arg    string // folder name, keeper, kept item, username source or URI
	// Alias lets a merge proceed when the usernames differ: the two items are
	// one account reached through an alias, a claim the plan's author proved.
	Alias bool
	// Drop names what a merge's duplicate carries that the keeper will not
	// receive ("fields", "notes", "uris"). Unnamed fields or notes block the
	// merge; unnamed URIs are carried to the keeper.
	Drop []string
	// Gate is a precondition outside the vault (a key revoked at its provider).
	// The row blocks until `--cleared <gate>` says it holds.
	Gate string
}

// flags renders the row's flags canonically, for the digest.
func (r CurateRow) flags() string {
	var f []string
	if r.Alias {
		f = append(f, "alias")
	}
	for _, d := range r.Drop {
		f = append(f, "drop="+d)
	}
	if r.Gate != "" {
		f = append(f, "gate="+r.Gate)
	}
	return strings.Join(f, ",")
}

// ParseCuratePlan reads a plan: tab-separated `op target arg reason [flags]`,
// `-` for an empty arg, `#` for a comment. A reason is required on every row,
// as it is on a registry retired: entry — a row that deletes something has to
// say what made that safe.
func ParseCuratePlan(r io.Reader) ([]CurateRow, error) {
	var rows []CurateRow
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		row, err := parseCurateRow(n, strings.Split(line, "\t"))
		if err != nil {
			return nil, fmt.Errorf("plan line %d: %w", n, err)
		}
		rows = append(rows, row)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func parseCurateRow(n int, cols []string) (CurateRow, error) {
	if len(cols) < 4 || len(cols) > 5 {
		return CurateRow{}, fmt.Errorf("want 4 or 5 tab-separated columns (op, target, arg, reason[, flags]), got %d", len(cols))
	}
	row := CurateRow{Line: n, Op: CurateOp(cols[0]), Target: cols[1]}
	spec, known := curateOps[row.Op]
	if !known {
		return row, fmt.Errorf("unknown op %q", cols[0])
	}
	if strings.TrimSpace(cols[3]) == "" {
		return row, fmt.Errorf("%s %s: a reason is required", row.Op, row.Target)
	}
	if spec.field {
		id, field, ok := strings.Cut(row.Target, ":")
		if !ok || id == "" || field == "" {
			return row, fmt.Errorf("%s needs a target of the form id:field, got %q", row.Op, row.Target)
		}
		row.Target, row.Field = id, field
		if field == "password" || field == "username" || field == "notes" {
			return row, fmt.Errorf("%s acts on custom fields only, not %q", row.Op, field)
		}
	}
	if row.Target == "" || strings.Contains(row.Target, ":") {
		return row, fmt.Errorf("%s: bad target %q", row.Op, cols[1])
	}
	if cols[2] != "-" {
		row.Arg = cols[2]
	}
	if spec.arg && row.Arg == "" {
		return row, fmt.Errorf("%s %s: the third column is required", row.Op, row.Target)
	}
	if len(cols) == 5 {
		for _, f := range strings.Split(cols[4], ",") {
			switch k, v, _ := strings.Cut(strings.TrimSpace(f), "="); {
			case k == "alias":
				row.Alias = true
			case k == "drop" && (v == "fields" || v == "notes" || v == "uris"):
				row.Drop = append(row.Drop, v)
			case k == "gate" && v != "":
				row.Gate = v
			case k == "":
			default:
				return row, fmt.Errorf("unknown flag %q", f)
			}
		}
	}
	if (row.Alias || len(row.Drop) > 0) && row.Op != CurMergeDelete {
		return row, fmt.Errorf("alias and drop= apply to merge-delete only")
	}
	return row, nil
}

// CurateState is what a plan says about one row.
type CurateState string

// A row is applied, already converged, or refused with a reason.
const (
	CurateApply   CurateState = "apply"
	CurateDone    CurateState = "done"
	CurateBlocked CurateState = "blocked"
)

// CurateStep is one row planned against the store.
type CurateStep struct {
	Row    CurateRow
	ID     string // the target's full id, "" when absent
	ArgID  string // the keeper's, kept item's or username source's full id
	State  CurateState
	Detail string   // why, value-free
	Carry  []string // the URIs a merge carries to its keeper
}

// CuratePlan is the whole plan against one inventory.
type CuratePlan struct {
	Steps []CurateStep
	// Digest identifies the plan and the store state it was made against: every
	// row, the full ids it resolved to, those items' revision dates and the
	// planned states. A plan applied later must reproduce it, so an item edited
	// between review and apply refuses the apply (Terraform's saved-plan
	// guarantee, without a plan file).
	Digest string
}

// Count returns how many steps are in state s.
func (p CuratePlan) Count(s CurateState) int {
	n := 0
	for _, st := range p.Steps {
		if st.State == s {
			n++
		}
	}
	return n
}

// CurateInputs is what planning reads besides the rows.
type CurateInputs struct {
	Items []ItemSummary
	// Values compares what a precondition must compare: a merge's passwords and
	// usernames, a copied username. Read into memory, never returned.
	Values BWReader
	// Owned reports whether the registry declares an item name. reconcile owns
	// those, and curate refuses to touch them.
	Owned func(name string) bool
	// Cleared holds the gates the operator declared satisfied.
	Cleared map[string]bool
}

// PlanCurate plans every row against the inventory in order.
func PlanCurate(rows []CurateRow, in CurateInputs) CuratePlan {
	var p CuratePlan
	h := sha256.New()
	for _, row := range rows {
		st := planCurateRow(row, in)
		p.Steps = append(p.Steps, st)
		_, _ = fmt.Fprintf(h, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Op, row.Target, row.Field, row.Arg, row.flags(),
			st.ID, revision(in.Items, st.ID), st.ArgID, revision(in.Items, st.ArgID), st.State)
	}
	p.Digest = hex.EncodeToString(h.Sum(nil))[:16]
	return p
}

func revision(items []ItemSummary, id string) string {
	for _, it := range items {
		if id != "" && it.ID == id {
			return it.Revised.UTC().Format(time.RFC3339Nano)
		}
	}
	return ""
}

// resolvePrefix finds the one item whose id starts with prefix. ok is false when more
// than one does: an ambiguous prefix must never pick one.
func resolvePrefix(items []ItemSummary, prefix string) (it *ItemSummary, ok bool) {
	for i := range items {
		if strings.HasPrefix(items[i].ID, prefix) {
			if it != nil {
				return nil, false
			}
			it = &items[i]
		}
	}
	return it, true
}

func blocked(st CurateStep, format string, a ...any) CurateStep {
	st.State, st.Detail = CurateBlocked, fmt.Sprintf(format, a...)
	return st
}

func done(st CurateStep, detail string) CurateStep {
	st.State, st.Detail = CurateDone, detail
	return st
}

func planCurateRow(row CurateRow, in CurateInputs) CurateStep {
	st := CurateStep{Row: row}
	t, ok := resolvePrefix(in.Items, row.Target)
	if !ok {
		return blocked(st, "id prefix %s matches more than one item", row.Target)
	}
	if t != nil {
		st.ID = t.ID
		if in.Owned != nil && in.Owned(t.Name) && !presentation(row.Op) {
			return blocked(st, "the registry declares this item; reconcile owns it")
		}
	}
	var a *ItemSummary
	if row.Arg != "" && (row.Op == CurMergeDelete || row.Op == CurCopyUsername || row.Op == CurDelete) {
		if a, ok = resolvePrefix(in.Items, row.Arg); !ok {
			return blocked(st, "id prefix %s matches more than one item", row.Arg)
		}
		if a != nil {
			st.ArgID = a.ID
		}
	}

	st = planOp(st, t, a, in)
	if st.State == CurateApply && row.Gate != "" && !in.Cleared[row.Gate] {
		return blocked(st, "gate %q is not cleared (--cleared %s once it holds)", row.Gate, row.Gate)
	}
	return st
}

// presentation reports whether op changes only how an item is shown, which the
// registry does not govern: reconcile converges where an item lives and which
// fields it has, never whether a field is masked or a view asks for the master
// password. So hide and reprompt may act on a registry-declared item, whose
// fields `dotf secrets set` already creates hidden, and the rest may not.
func presentation(op CurateOp) bool { return op == CurHide || op == CurReprompt }

// planOp decides one row's state. Absence reads as converged only for the ops
// that remove an item; for any edit, an absent item blocks, so a typo in an id
// can never pass as done.
func planOp(st CurateStep, t, a *ItemSummary, in CurateInputs) CurateStep {
	row := st.Row
	switch row.Op {
	case CurDelete:
		if t == nil {
			return done(st, "absent")
		}
		if row.Arg != "" && a == nil {
			return blocked(st, "the item kept instead, %s, is absent", row.Arg)
		}
	case CurMergeDelete:
		return planMerge(st, t, a, in)
	default:
		if t == nil {
			return blocked(st, "the item is absent")
		}
	}

	switch row.Op {
	case CurFolder:
		if strings.HasPrefix(row.Arg, "Dotfiles/") {
			return blocked(st, "the Dotfiles/* folders belong to the registry")
		}
		if t.FolderUnresolved {
			return blocked(st, "the item's folder is unknown to the folder list; sync and re-plan")
		}
		if t.Folder == row.Arg {
			return done(st, "in "+row.Arg)
		}
	case CurDeleteField:
		if _, has := t.FieldTypes[row.Field]; !has {
			return done(st, "field absent")
		}
	case CurHide:
		kind, has := t.FieldTypes[row.Field]
		switch {
		case !has:
			return blocked(st, "field %q is absent", row.Field)
		case kind == 1:
			return done(st, "hidden")
		case kind != 0:
			return blocked(st, "field %q is not a text field", row.Field)
		}
	case CurReprompt:
		if t.Reprompt {
			return done(st, "reprompt on")
		}
	case CurAddURI:
		if slices.Contains(t.URIs, row.Arg) {
			return done(st, "uri present")
		}
		if !t.HasLogin {
			return blocked(st, "the item has no login to carry a URI")
		}
	case CurCopyUsername:
		if st2, settled := planCopyUsername(st, t, a, in); settled {
			return st2
		}
	}

	if t.Passkeys > 0 {
		return blocked(st, "the item carries %d passkey(s); curate never writes to one", t.Passkeys)
	}
	if row.Op == CurDelete && t.Attachments > 0 {
		return blocked(st, "the item carries %d attachment(s)", t.Attachments)
	}
	st.State = CurateApply
	return st
}

// planCopyUsername settles a copy-username that needs no write, or blocks it.
// settled is false when the copy is still to be made.
func planCopyUsername(st CurateStep, t, a *ItemSummary, in CurateInputs) (CurateStep, bool) {
	_, has := t.FieldTypes[st.Row.Field]
	if a == nil {
		if has {
			// The source's usual fate is the merge that follows the copy.
			return done(st, "field present; source gone"), true
		}
		return blocked(st, "the username source %s is absent", st.Row.Arg), true
	}
	src, err := in.Values.Field(a.ID, "username")
	if err != nil || src == "" {
		return blocked(st, "the username source has no readable username"), true
	}
	if !has {
		return st, false
	}
	cur, err := in.Values.Field(t.ID, st.Row.Field)
	switch {
	case err != nil:
		return blocked(st, "field %q is unreadable", st.Row.Field), true
	case cur == src:
		return done(st, "field holds the username"), true
	default:
		return blocked(st, "field %q exists and holds something else; curate never overwrites", st.Row.Field), true
	}
}

// planMerge plans a merge-delete: the duplicate t goes, the keeper a stays and
// receives the URIs it lacks. Everything else t carries must be the same as the
// keeper's, or named by the row as dropped.
func planMerge(st CurateStep, t, a *ItemSummary, in CurateInputs) CurateStep {
	row := st.Row
	if a == nil {
		return blocked(st, "the keeper %s is absent", row.Arg)
	}
	if t == nil {
		return done(st, "absent")
	}
	if t.ID == a.ID {
		return blocked(st, "an item cannot be merged into itself")
	}
	switch {
	case t.Passkeys > 0:
		return blocked(st, "the duplicate carries %d passkey(s); a passkey is never deleted", t.Passkeys)
	case t.Attachments > 0:
		return blocked(st, "the duplicate carries %d attachment(s)", t.Attachments)
	case t.HasTOTP:
		return blocked(st, "the duplicate carries a TOTP seed")
	case len(t.Fields) > 0 && !slices.Contains(row.Drop, "fields"):
		return blocked(st, "the duplicate carries %d custom field(s) the row does not drop", len(t.Fields))
	case t.HasNotes && !slices.Contains(row.Drop, "notes"):
		return blocked(st, "the duplicate carries notes the row does not drop")
	}
	if v := compare(in.Values, t.ID, a.ID, "password"); v != "equal" {
		return blocked(st, "passwords %s", v)
	}
	if !row.Alias {
		if v := compare(in.Values, t.ID, a.ID, "username"); v != "equal" {
			return blocked(st, "usernames %s (flag alias if they are one account)", v)
		}
	}
	if !slices.Contains(row.Drop, "uris") {
		for _, u := range t.URIs {
			if !slices.Contains(a.URIs, u) && !slices.Contains(st.Carry, u) {
				st.Carry = append(st.Carry, u)
			}
		}
	}
	if len(st.Carry) > 0 && a.Passkeys > 0 {
		return blocked(st, "the keeper carries a passkey and would need an edit to take %d URI(s)", len(st.Carry))
	}
	st.State = CurateApply
	return st
}

// compare reads one field of two items into memory and returns a value-free
// verdict: "equal", "differ", or "unreadable".
func compare(r BWReader, a, b, field string) string {
	va, errA := r.Field(a, field)
	vb, errB := r.Field(b, field)
	switch {
	case errA != nil || errB != nil:
		return "unreadable"
	case va == vb:
		return "equal"
	default:
		return "differ"
	}
}

// CurateStore is everything an apply writes through, plus the reader a copy
// needs to fetch the value it stores.
type CurateStore interface {
	BWReader
	BWCurator
	BWItemDeleter
	BWFolderResolver
}

// ApplyCurate performs every step in state apply, in plan order. It refuses a
// plan with any blocked step, whole: a partial curation is a state nobody
// reviewed. onApplied is called after each step lands.
func ApplyCurate(p CuratePlan, s CurateStore, onApplied func(CurateStep)) error {
	if n := p.Count(CurateBlocked); n > 0 {
		return fmt.Errorf("%d blocked step(s): nothing is applied", n)
	}
	for _, st := range p.Steps {
		if st.State != CurateApply {
			continue
		}
		if err := applyCurateStep(st, s); err != nil {
			return fmt.Errorf("line %d %s %s: %w", st.Row.Line, st.Row.Op, st.Row.Target, err)
		}
		if onApplied != nil {
			onApplied(st)
		}
	}
	return nil
}

func applyCurateStep(st CurateStep, s CurateStore) error {
	row := st.Row
	switch row.Op {
	case CurFolder:
		fid, err := s.ResolveFolder(row.Arg)
		if err != nil {
			return err
		}
		return s.Curate(st.ID, Mutation{Kind: MutMove, Value: fid})
	case CurDelete:
		return s.DeleteItem(st.ID)
	case CurMergeDelete:
		for _, u := range st.Carry {
			if err := s.Curate(st.ArgID, Mutation{Kind: MutAddURI, Value: u}); err != nil {
				return fmt.Errorf("carry a URI to the keeper: %w", err)
			}
		}
		return s.DeleteItem(st.ID)
	case CurDeleteField:
		return s.Curate(st.ID, Mutation{Kind: MutRemoveField, Field: row.Field})
	case CurCopyUsername:
		v, err := s.Field(st.ArgID, "username")
		if err != nil {
			return fmt.Errorf("read the username source: %w", err)
		}
		return s.Curate(st.ID, Mutation{Kind: MutSetText, Field: row.Field, Value: v})
	case CurAddURI:
		return s.Curate(st.ID, Mutation{Kind: MutAddURI, Value: row.Arg})
	case CurHide:
		return s.Curate(st.ID, Mutation{Kind: MutHide, Field: row.Field})
	case CurReprompt:
		return s.Curate(st.ID, Mutation{Kind: MutReprompt})
	}
	return fmt.Errorf("unknown op %q", row.Op)
}

// TotalPasskeys counts every passkey in the inventory. An apply compares it
// before and after: the guards above make a loss impossible by construction, and
// this is the consequence check that says so on the live store.
func TotalPasskeys(items []ItemSummary) int {
	n := 0
	for _, it := range items {
		n += it.Passkeys
	}
	return n
}
