package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// newSecretsCurateCmd applies a reviewed plan to the items the registry does not
// declare (SEC-006).
//
// It is reconcile's shape turned on the rest of the vault: a plan printed by
// default, applied only with --apply, and verified by planning again. The plan is
// a file the operator keeps outside this public repo, because it names personal
// items; this command holds the rules, never the data.
//
// Two guarantees beyond reconcile's. The dry run prints a digest over the rows,
// the ids they resolved to and those items' revision dates, and --apply refuses
// unless it is given that digest and the store still produces it: what applies is
// exactly what was reviewed. And --apply counts the vault's passkeys before and
// after, failing on any difference, because a lost passkey is the one loss the
// escrow cannot give back.
func newSecretsCurateCmd() *cobra.Command {
	var (
		planPath string
		apply    bool
		digest   string
		cleared  []string
	)
	c := &cobra.Command{
		Use:   "curate --plan <file>",
		Short: "Apply a reviewed plan to the items the registry does not declare (plan by default, --apply to change)",
		Long: "curate reads a plan, one tab-separated row per operation:\n\n" +
			"  op  target  arg  reason  [flags]\n\n" +
			"  folder         <id> <Folder>        file the item (the folder is created if absent)\n" +
			"  delete         <id> [<kept>]        delete the item (Bitwarden trash, 30 days)\n" +
			"  merge-delete   <id> <keeper>        carry the URIs the keeper lacks, delete the duplicate\n" +
			"  delete-field   <id>:<field>         remove one custom field\n" +
			"  copy-username  <id>:<field> <src>   store src's username as a text field\n" +
			"  add-uri        <id> <uri>           append a login URI\n" +
			"  hide           <id>:<field>         turn a text field hidden\n" +
			"  reprompt       <id>                 ask for the master password to view\n\n" +
			"An id may be a unique prefix; `-` is an empty arg. Flags are comma-separated:\n" +
			"alias (a merge whose usernames differ by design), drop=fields|notes|uris (what\n" +
			"a merge's duplicate carries that the keeper will not receive), gate=<name> (a\n" +
			"precondition outside the vault; the row blocks until --cleared <name>).\n\n" +
			"Each row plans apply, done (already converged) or blocked with a reason. A\n" +
			"blocked row makes the plan unappliable, whole. curate never writes to an item\n" +
			"that carries a passkey, never deletes one with attachments, never merges items\n" +
			"whose passwords differ, touches an item the registry declares only to hide a\n" +
			"field or turn reprompt on, and never prints a value: output names ids,\n" +
			"operations and states.\n\n" +
			"--apply requires --digest, the one the dry run printed; it syncs, re-plans,\n" +
			"refuses on a different digest, applies, and plans again. Anything not done\n" +
			"after that, or a change in the vault's passkey count, fails the command.\n" +
			"Take a DR escrow (`dotf secrets backup`) first.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rows, err := readCuratePlan(planPath)
			if err != nil {
				return err
			}
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			gates := map[string]bool{}
			for _, g := range cleared {
				gates[g] = true
			}
			out := cmd.OutOrStdout()

			items, plan, err := planCurateFromStore(rows, registryItems(reg), gates)
			if err != nil {
				return err
			}
			printCuratePlan(out, plan)
			if n := plan.Count(secrets.CurateBlocked); n > 0 {
				return fmt.Errorf("%d blocked row(s): resolve them before anything can apply", n)
			}
			if !apply {
				if plan.Count(secrets.CurateApply) > 0 {
					_, _ = fmt.Fprintf(out, "\nNothing was changed. To perform this plan: --apply --digest %s\n", plan.Digest)
				}
				return nil
			}
			if digest != plan.Digest {
				return fmt.Errorf("--apply needs --digest %s, the digest of the plan as the store stands now; "+
					"got %q. Review the plan above, then pass its digest", plan.Digest, digest)
			}
			if plan.Count(secrets.CurateApply) == 0 {
				return nil
			}
			return applyCurate(out, rows, registryItems(reg), gates, items, plan)
		},
	}
	c.Flags().StringVar(&planPath, "plan", "", "the plan file (required)")
	c.Flags().BoolVar(&apply, "apply", false, "perform the plan (default: print it and change nothing)")
	c.Flags().StringVar(&digest, "digest", "", "the digest the dry run printed; required with --apply")
	c.Flags().StringSliceVar(&cleared, "cleared", nil, "gates whose outside-the-vault precondition holds")
	_ = c.MarkFlagRequired("plan")
	return c
}

func readCuratePlan(path string) ([]secrets.CurateRow, error) {
	f, err := os.Open(path) //nolint:gosec // an operator-named plan file is the input by design
	if err != nil {
		return nil, fmt.Errorf("read the plan: %w", err)
	}
	defer func() { _ = f.Close() }()
	return secrets.ParseCuratePlan(f)
}

// registryItems returns every item name the registry reads or retires: the items
// reconcile owns, which curate refuses to touch.
func registryItems(reg *secrets.Registry) func(string) bool {
	owned := map[string]bool{}
	for _, d := range reg.BWDeclarations() {
		owned[d.Item] = true
		if d.From != nil {
			owned[d.From.Item] = true
		}
	}
	for _, r := range reg.Retired {
		owned[r.Item] = true
	}
	return func(name string) bool { return owned[name] }
}

// planCurateFromStore syncs, reads the inventory and plans. The sync is required
// for the reason it is in reconcile: a plan made from a stale cache describes a
// store that is not there.
func planCurateFromStore(rows []secrets.CurateRow, owned func(string) bool, gates map[string]bool) ([]secrets.ItemSummary, secrets.CuratePlan, error) {
	items, _, err := readInventory()
	if err != nil {
		return nil, secrets.CuratePlan{}, err
	}
	return items, secrets.PlanCurate(rows, secrets.CurateInputs{
		Items: items, Values: bwRead(), Owned: owned, Cleared: gates,
	}), nil
}

// curateStore is the write half the plan applies through: the pinned backend's
// writer, which carries Curate on both implementations, and its reader.
type curateStore struct {
	secrets.BWReader
	bwWriteClient
	secrets.BWCurator
}

func applyCurate(out io.Writer, rows []secrets.CurateRow, owned func(string) bool, gates map[string]bool,
	items []secrets.ItemSummary, plan secrets.CuratePlan) error {
	w := bwWrite()
	curator, ok := w.(secrets.BWCurator)
	if !ok {
		return fmt.Errorf("the Bitwarden backend in use cannot apply a curate edit")
	}
	before := secrets.TotalPasskeys(items)

	_, _ = fmt.Fprintln(out)
	err := secrets.ApplyCurate(plan, curateStore{bwRead(), w, curator}, func(st secrets.CurateStep) {
		_, _ = fmt.Fprintf(out, "applied  %-14s %s\n", st.Row.Op, curateTarget(st.Row))
	})
	if err != nil {
		return err
	}

	after, again, err := planCurateFromStore(rows, owned, gates)
	if err != nil {
		return fmt.Errorf("applied, but the verifying re-plan failed: %w", err)
	}
	if n := secrets.TotalPasskeys(after); n != before {
		return fmt.Errorf("PASSKEY COUNT CHANGED: %d before, %d after. Stop and compare against the escrow", before, n)
	}
	if left := len(again.Steps) - again.Count(secrets.CurateDone); left > 0 {
		_, _ = fmt.Fprintln(out)
		printCuratePlan(out, again)
		return fmt.Errorf("applied, but the store did not converge: %d row(s) are not done", left)
	}
	_, _ = fmt.Fprintf(out, "\nConverged: %d row(s) applied, every row is done, passkeys unchanged (%d).\n",
		plan.Count(secrets.CurateApply), before)
	return nil
}

// printCuratePlan writes one line per row: state, op, target and a value-free
// detail. The row's reason is not printed: it is the plan author's prose, and it
// names personal items.
func printCuratePlan(out io.Writer, p secrets.CuratePlan) {
	for _, st := range p.Steps {
		detail := st.Detail
		if st.State == secrets.CurateApply {
			detail = curateEffect(st)
		}
		_, _ = fmt.Fprintf(out, "%-8s %-14s %-24s %s\n", st.State, st.Row.Op, curateTarget(st.Row), detail)
	}
	_, _ = fmt.Fprintf(out, "\nPlan: %d to apply, %d done, %d blocked. Digest: %s\n",
		p.Count(secrets.CurateApply), p.Count(secrets.CurateDone), p.Count(secrets.CurateBlocked), p.Digest)
}

func curateTarget(r secrets.CurateRow) string {
	if r.Field != "" {
		return r.Target + ":" + r.Field
	}
	return r.Target
}

// curateEffect says what an apply step will do, by ids and counts: a URI can
// carry a token in its query string, so none is printed.
func curateEffect(st secrets.CurateStep) string {
	switch st.Row.Op {
	case secrets.CurFolder:
		return "-> " + st.Row.Arg
	case secrets.CurMergeDelete:
		return fmt.Sprintf("into %s, carrying %d URI(s)", st.Row.Arg, len(st.Carry))
	case secrets.CurCopyUsername:
		return "from " + st.Row.Arg
	case secrets.CurDelete:
		if st.Row.Arg != "" {
			return "kept instead: " + st.Row.Arg
		}
	}
	return ""
}
