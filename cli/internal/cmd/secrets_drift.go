package cmd

import (
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// bwLister is the inventory seam — nil in production, where it comes from the
// pinned backend alongside the reader and writer, the same shape as bwSyncer in
// secrets_rotate.go. A var so command tests run with no daemon and no vault.
var bwLister secrets.BWLister

// newSecretsDriftCmd reports where the store disagrees with the registry.
//
// It exists because the registry described a store nobody had ever compared it
// to. Measured the first time this ran against the live vault: 169 of 185 items
// unfoldered, every declared folder name absent under that spelling, and two
// declared items missing entirely — one of them holding a credential that had
// been rotated elsewhere, so the read path was serving a token GitHub answers
// 401 to while `dotf secrets verify` reported OK.
//
// READ-ONLY, and not merely by omission: there is no --fix, the command is never
// handed a writer, and the comparison it calls has none. Converging the store is
// `reconcile` (CLI-078); a report that quietly repaired things would make the
// repair unreviewable, and these are credentials.
//
// It prints coordinates, never values. The inventory it reads is projected to
// shapes inside the backend (see bwserve_list.go) precisely so nothing else can.
func newSecretsDriftCmd() *cobra.Command {
	var verbose bool
	c := &cobra.Command{
		Use:   "drift",
		Short: "Report where the Bitwarden store disagrees with the registry (no values, no writes)",
		Long: "drift compares the registry's declared layout against the vault's actual shape\n" +
			"and reports each disagreement, in the order it has to be fixed:\n\n" +
			"  folder-missing   no folder carries the declared name — and `dotf secrets set`\n" +
			"                   CREATES rather than reuses, so this one splits a store\n" +
			"  item-missing     an item the registry names is not in the vault\n" +
			"  item-misfiled    the item exists, in a different folder than declared\n" +
			"  field-missing    the item exists but carries no field of that name\n\n" +
			"Only item names, folder names and field names are read — never a value.\n" +
			"Nothing is written: `reconcile` converges the store, `drift` only reports.\n\n" +
			"Exit is non-zero when any disagreement is found, so CI and a hook can gate\n" +
			"on it.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			lister := bwLister
			if lister == nil {
				lister = secrets.BWServeClient{}
			}

			items, err := lister.ListItems()
			if err != nil {
				return fmt.Errorf("read the vault's inventory: %w\n"+
					"drift compares against the live store; an unreadable store is not an empty one", err)
			}
			folders, err := lister.ListFolders()
			if err != nil {
				return fmt.Errorf("read the vault's folders: %w", err)
			}

			decls := reg.BWDeclarations()
			findings := secrets.LayoutDrift(decls, items, folders)
			out := cmd.OutOrStdout()

			// The registry id closes each line: it is the line of registry.yaml to
			// edit, and with findings deduped per item the item name alone cannot
			// say which of several declarations raised it.
			for _, f := range findings {
				_, _ = fmt.Fprintf(out, "%-14s %-24s %s [%s]\n", f.Kind, f.Item, f.Detail, f.Secret)
			}

			unmanaged := secrets.UnmanagedItems(decls, items)
			if verbose {
				for _, name := range unmanaged {
					_, _ = fmt.Fprintf(out, "unmanaged      %s\n", name)
				}
			}

			_, _ = fmt.Fprintf(out, "\n%d declared target(s) across %d item(s); %d finding(s); "+
				"%d of %d vault items unmanaged by this registry\n",
				len(decls), len(items)-len(unmanaged), len(findings), len(unmanaged), len(items))

			if len(findings) > 0 {
				// A report that exits 0 on findings cannot gate anything, and this
				// one exists to be gated on.
				return fmt.Errorf("%d layout finding(s)", len(findings))
			}
			return nil
		},
	}
	c.Flags().BoolVar(&verbose, "verbose", false, "also list the vault items no registry entry declares (names only)")
	return c
}
