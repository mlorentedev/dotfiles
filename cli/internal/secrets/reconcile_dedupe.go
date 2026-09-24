package secrets

import (
	"crypto/subtle"
	"fmt"
	"strings"
)

// CLI-082: a plan that says what a retire will do, and items retired by
// declaration. Both exist so duplicates can be removed one reviewed plan at a
// time, instead of one refused apply at a time.

// Retire verdicts: the outcome of comparing a retire's source and destination.
// Only equality is ever reported; no value, length or digest leaves the comparison.
const (
	RetireEqual            = "equal"
	RetireDiffers          = "differs"
	RetireDestinationEmpty = "destination empty"
	RetireUnreadable       = "unreadable"
)

type retireVerdict struct {
	verdict, detail, remedy string
}

// compareRetire reads both sides of a retire into memory and classifies them.
// Compared in constant time out of habit: nothing here is a remote oracle, but it
// costs nothing to not be the example someone copies into one.
func compareRetire(r BWReader, op ReconcileOp) retireVerdict {
	unreadable := func(item, field string, err error) retireVerdict {
		return retireVerdict{
			verdict: RetireUnreadable,
			detail:  fmt.Sprintf("%s: %s/%q could not be read (%v)", RetireUnreadable, item, field, err),
			remedy:  "re-run once the store answers for it; nothing can be compared until then",
		}
	}
	dst, err := r.Field(op.Item, op.Field)
	if err != nil {
		return unreadable(op.Item, op.Field, err)
	}
	src, err := r.Field(op.FromItem, op.FromField)
	if err != nil {
		return unreadable(op.FromItem, op.FromField, err)
	}
	exits := fmt.Sprintf("if the source is current, `dotf secrets rotate %s` with its value; "+
		"to keep both while it is settled, drop retire: from bw.from on %s", op.Secret, op.Secret)
	switch {
	case dst == "":
		return retireVerdict{RetireDestinationEmpty,
			fmt.Sprintf("%s: %s/%q holds nothing, so removing the source would lose the credential", RetireDestinationEmpty, op.Item, op.Field),
			exits}
	case subtle.ConstantTimeCompare([]byte(dst), []byte(src)) != 1:
		return retireVerdict{RetireDiffers,
			fmt.Sprintf("%s: %s/%q and %s/%q hold different values, so one was rotated after the copy and the tool cannot tell which is current",
				RetireDiffers, op.FromItem, op.FromField, op.Item, op.Field),
			exits}
	}
	return retireVerdict{verdict: RetireEqual}
}

// VerifyRetires gives every planned retire-source a verdict, at plan time.
//
// It is the one place a plan reads secret values, and it reads them only to
// compare: the values stay in this function's frame and only the verdict leaves.
// An equal retire stays an operation. Any other becomes a blocker, so the plan is
// unappliable as a whole and every mismatch is visible in one plan, where before
// the apply refused them one run at a time.
func VerifyRetires(p *ReconcilePlan, r BWReader) {
	kept := p.Ops[:0]
	for _, op := range p.Ops {
		if op.Kind != OpRetireSource {
			kept = append(kept, op)
			continue
		}
		v := compareRetire(r, op)
		if v.verdict == RetireEqual {
			op.Verdict = RetireEqual
			kept = append(kept, op)
			continue
		}
		p.Blocked = append(p.Blocked, PlanNote{Secret: op.Secret, Item: op.FromItem, Detail: v.detail, Remedy: v.remedy})
	}
	p.Ops = kept
}

// PlanRetiredItems plans a delete-item for every retired: item the vault holds,
// and a delete-field for every retired field it holds (CLI-083).
//
// The registry has already refused an entry that a declaration still names or
// reads (checkRetired), because that is static. What depends on the store is
// decided here: an item or field already gone is reported so its entry can go
// too, and a name several items carry blocks, because acting on an arbitrary one
// of them could delete the wrong credential. Both kinds rank last, so appending
// keeps order.
func PlanRetiredItems(p *ReconcilePlan, retired []RetiredItem, items []ItemSummary) {
	byName := map[string][]ItemSummary{}
	for _, it := range items {
		byName[it.Name] = append(byName[it.Name], it)
	}
	for _, ri := range retired {
		found := byName[ri.Item]
		switch {
		case len(found) == 0:
			p.RetiredGone = append(p.RetiredGone, ri.label())
		case len(found) > 1:
			p.Blocked = append(p.Blocked, PlanNote{
				Item:   ri.Item,
				Detail: fmt.Sprintf("cannot delete %q: the name matches %d items, and acting on an arbitrary one could delete the wrong credential", ri.label(), len(found)),
				Remedy: "rename or remove the duplicate items in the vault",
			})
		case ri.Field == "":
			p.Ops = append(p.Ops, ReconcileOp{Kind: OpDeleteItem, Item: ri.Item, Shape: itemShape(found[0]), Reason: ri.Reason})
		case !hasField(found[0], ri.Field):
			p.RetiredGone = append(p.RetiredGone, ri.label())
		default:
			p.Ops = append(p.Ops, ReconcileOp{
				Kind: OpDeleteField, Item: ri.Item, Field: ri.Field,
				Shape: itemShape(withoutField(found[0], ri.Field)), Reason: ri.Reason,
			})
		}
	}
}

// withoutField is the item as a delete-field would leave it, so the plan shows
// what it keeps. The registry refuses username and password, so only a custom
// field or the notes can be removed.
func withoutField(it ItemSummary, field string) ItemSummary {
	if field == "notes" {
		it.HasNotes = false
		return it
	}
	kept := make([]string, 0, len(it.Fields))
	for _, f := range it.Fields {
		if f != field {
			kept = append(kept, f)
		}
	}
	it.Fields = kept
	return it
}

// itemShape describes what an item holds by name only: field names, whether it
// has notes, whether it has a login. The value-free projection is all it reads.
func itemShape(it ItemSummary) string {
	var parts []string
	if len(it.Fields) > 0 {
		parts = append(parts, "fields "+strings.Join(it.Fields, ", "))
	}
	if it.HasNotes {
		parts = append(parts, "notes")
	}
	if it.HasLogin {
		parts = append(parts, "login")
	}
	if len(parts) == 0 {
		return "empty"
	}
	return strings.Join(parts, "; ")
}
