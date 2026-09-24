package cmd

import (
	"fmt"
	"io"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// newSecretsReconcileCmd converges the store's layout toward the registry (CLI-080).
//
// It is `drift` with a second half: the same comparison, turned into operations,
// shown as a plan, and performed only on --apply. The shape is Terraform's on
// purpose — the declaration is reviewed code, the plan is what a reviewer reads,
// and the apply is the tool's, never a hand in the web vault.
//
// --apply ends by syncing and planning AGAIN, and fails unless a further plan is
// empty. Idempotence is therefore checked by the command on every apply, not
// asserted once in a test and assumed afterwards. The one exception is a
// retire-source, which a pass can only unlock and never perform; it gets a second
// pass (applyUntilConverged).
func newSecretsReconcileCmd() *cobra.Command {
	var apply bool
	c := &cobra.Command{
		Use:   "reconcile",
		Short: "Converge the Bitwarden store's layout toward the registry (plan by default, --apply to change)",
		Long: "reconcile plans the operations that remove the disagreements `drift` reports,\n" +
			"and performs them only with --apply:\n\n" +
			"  create-folder  a declared folder no folder carries\n" +
			"  move-item      an item outside its declared folder\n" +
			"  create-item    an absent item, copied from its declared bw.from source\n" +
			"  add-field      an absent field, copied from its declared bw.from source\n\n" +
			"It never invents a value (an absent item with no bw.from is BLOCKED, with its\n" +
			"remedy), never overwrites one (an existing field is left alone), never moves an\n" +
			"item declared with no folder, and never prints a value.\n\n" +
			"A blocked finding makes the plan unappliable, whole. A dormant declaration\n" +
			"with no source is DEFERRED to `dotf secrets migrate` and blocks nothing.\n\n" +
			"--apply syncs, applies, syncs again and re-plans. A retire-source is planned\n" +
			"only once its copy exists, so a declaration that copies and retires takes a\n" +
			"second pass; anything else left after a pass fails, and so does anything left\n" +
			"after the second.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			decls := reg.BWDeclarations()
			out := cmd.OutOrStdout()

			plan, err := planFromStore(decls)
			if err != nil {
				return err
			}
			printPlan(out, plan)
			if len(plan.Blocked) > 0 {
				return fmt.Errorf("%d blocked finding(s): resolve them before anything can apply", len(plan.Blocked))
			}
			if !apply {
				if len(plan.Ops) > 0 {
					_, _ = fmt.Fprintln(out, "\nNothing was changed. Run with --apply to perform this plan.")
				}
				return nil
			}
			if len(plan.Ops) == 0 {
				return nil
			}

			return applyUntilConverged(out, decls, plan)
		},
	}
	c.Flags().BoolVar(&apply, "apply", false, "perform the plan (default: print it and change nothing)")
	return c
}

// maxApplyPasses bounds --apply. Two passes, because the operation graph is two
// deep: every kind but one is planned from the store as found, and retire-source
// is planned only once the copy it retires exists (reconcile.go, satisfied). A
// third pass would have nothing it could unlock.
const maxApplyPasses = 2

// applyUntilConverged applies plan, syncs, re-plans, and repeats while the new
// plan holds only what the previous pass unlocked.
//
// A retire never rides in the pass that makes its copy. Its safety check compares
// the copy as the STORE holds it after a sync, never the value this run meant to
// write, so it needs the copy to exist first. A declaration that copies and
// retires therefore takes two passes by construction (CLI-080 review round 1,
// Blocker). Stopping after one reported that second pass as non-convergence.
//
// Anything but a retire after a pass is a real failure: a store that did not take
// the write, or two declarations pulling one item two ways. Another pass would
// only repeat it, so it fails where it appears.
//
// The bound lives in the loop header on purpose. It sat in the body once, and a
// mutation that deleted that one condition turned a store that never converges
// into an infinite loop: the test binary grew to 9 GB and the OOM killer took the
// terminal host with it (lesson 286). A header bound survives any edit to the body.
func applyUntilConverged(out io.Writer, decls []secrets.BWDecl, plan secrets.ReconcilePlan) error {
	applied := 0
	for pass := 1; pass <= maxApplyPasses; pass++ {
		_, _ = fmt.Fprintln(out)
		err := secrets.ApplyReconcile(plan, bwRead(), bwWrite(), func(op secrets.ReconcileOp) {
			_, _ = fmt.Fprintf(out, "applied  %-14s %s\n", op.Kind, op.Target())
		})
		if err != nil {
			return err
		}
		applied += len(plan.Ops)

		again, err := planFromStore(decls)
		if err != nil {
			return fmt.Errorf("applied, but the verifying re-plan failed: %w", err)
		}
		if len(again.Ops) == 0 {
			_, _ = fmt.Fprintf(out, "\nConverged: %d operation(s) applied in %d pass(es), and a further plan is empty.\n", applied, pass)
			printSatisfied(out, again)
			return nil
		}
		_, _ = fmt.Fprintln(out)
		printPlan(out, again)
		if !onlyRetires(again) || len(again.Blocked) > 0 {
			return fmt.Errorf("applied, but the store did not converge: pass %d left %d operation(s)", pass, len(again.Ops))
		}
		plan = again
	}
	return fmt.Errorf("applied, but the store did not converge: %d pass(es) left %d operation(s)", maxApplyPasses, len(plan.Ops))
}

// onlyRetires reports whether every operation in p is one that a previous pass
// can have unlocked.
func onlyRetires(p secrets.ReconcilePlan) bool {
	for _, op := range p.Ops {
		if op.Kind != secrets.OpRetireSource {
			return false
		}
	}
	return true
}

// planFromStore syncs, reads the inventory and plans.
//
// The sync (readInventory) is not optional: a plan computed on a stale inventory
// would create a duplicate of any item made elsewhere since the last sync.
func planFromStore(decls []secrets.BWDecl) (secrets.ReconcilePlan, error) {
	items, folders, err := readInventory()
	if err != nil {
		return secrets.ReconcilePlan{}, err
	}
	return secrets.PlanReconcile(decls, items, folders), nil
}

// printPlan writes one line per operation and per note. Coordinates only: a plan
// holds no value, so there is none to print.
func printPlan(out io.Writer, p secrets.ReconcilePlan) {
	for _, op := range p.Ops {
		switch op.Kind {
		case secrets.OpCreateFolder:
			_, _ = fmt.Fprintf(out, "+ %-14s %s\n", op.Kind, op.Folder)
		case secrets.OpMoveItem:
			_, _ = fmt.Fprintf(out, "~ %-14s %-24s %s -> %s\n", op.Kind, op.Item, folderLabel(op.Current), op.Folder)
		case secrets.OpRetireSource:
			_, _ = fmt.Fprintf(out, "- %-14s %-24s remove %s/%q, once verified equal to %s/%q\n",
				op.Kind, op.FromItem, op.FromItem, op.FromField, op.Item, op.Field)
		default:
			_, _ = fmt.Fprintf(out, "+ %-14s %-24s field %q in %s, copied from %s/%q\n",
				op.Kind, op.Item, op.Field, folderLabel(op.Folder), op.FromItem, op.FromField)
		}
	}
	for _, n := range p.Blocked {
		_, _ = fmt.Fprintf(out, "! %-14s %-24s %s -> %s\n", "blocked", n.Item, n.Detail, n.Remedy)
	}
	for _, n := range p.Deferred {
		_, _ = fmt.Fprintf(out, "- %-14s %-24s %s -> %s\n", "deferred", n.Item, n.Detail, n.Remedy)
	}
	_, _ = fmt.Fprintf(out, "\nPlan: %d to apply, %d blocked, %d deferred.\n", len(p.Ops), len(p.Blocked), len(p.Deferred))
	printSatisfied(out, p)
}

func printSatisfied(out io.Writer, p secrets.ReconcilePlan) {
	for _, id := range p.Satisfied {
		_, _ = fmt.Fprintf(out, "note: bw.from on %s is satisfied; it can be removed from the registry\n", id)
	}
}

func folderLabel(name string) string {
	if name == "" {
		return "(no folder)"
	}
	return name
}
