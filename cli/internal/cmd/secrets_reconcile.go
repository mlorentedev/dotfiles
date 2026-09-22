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
// --apply ends by syncing and planning AGAIN, and fails unless that second plan is
// empty. Idempotence is therefore checked by the command on every apply, not
// asserted once in a test and assumed afterwards.
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
			"--apply syncs, applies, syncs again and re-plans; it fails unless the second\n" +
			"plan is empty.",
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

			_, _ = fmt.Fprintln(out)
			err = secrets.ApplyReconcile(plan, bwRead(), bwWrite(), func(op secrets.ReconcileOp) {
				_, _ = fmt.Fprintf(out, "applied  %-14s %s\n", op.Kind, op.Target())
			})
			if err != nil {
				return err
			}

			again, err := planFromStore(decls)
			if err != nil {
				return fmt.Errorf("applied, but the verifying re-plan failed: %w", err)
			}
			if len(again.Ops) > 0 {
				_, _ = fmt.Fprintln(out)
				printPlan(out, again)
				return fmt.Errorf("applied, but the store did not converge: a second plan still has %d operation(s)", len(again.Ops))
			}
			_, _ = fmt.Fprintf(out, "\nConverged: %d operation(s) applied, and a second plan is empty.\n", len(plan.Ops))
			printSatisfied(out, again)
			return nil
		},
	}
	c.Flags().BoolVar(&apply, "apply", false, "perform the plan (default: print it and change nothing)")
	return c
}

// planFromStore syncs, reads the inventory and plans.
//
// The sync is not optional. The daemon answers from its cache; a plan computed on a
// stale inventory would create a duplicate of any item made elsewhere since the last
// sync. `drift` tolerates staleness because it only reports; this command writes.
func planFromStore(decls []secrets.BWDecl) (secrets.ReconcilePlan, error) {
	if err := bwSync().Sync(); err != nil {
		return secrets.ReconcilePlan{}, fmt.Errorf("sync before planning: %w\n"+
			"reconcile refuses to plan against a possibly stale inventory", err)
	}
	lister := bwLister
	if lister == nil {
		lister = secrets.BWServeClient{}
	}
	items, err := lister.ListItems()
	if err != nil {
		return secrets.ReconcilePlan{}, fmt.Errorf("read the vault's inventory: %w", err)
	}
	folders, err := lister.ListFolders()
	if err != nil {
		return secrets.ReconcilePlan{}, fmt.Errorf("read the vault's folders: %w", err)
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
