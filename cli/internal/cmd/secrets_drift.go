package cmd

import (
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// inventorySource is what reading the store's shape needs: list it, and sync it
// first. One value provides both, so the cache that is refreshed is by
// construction the cache that is read. rotate learned the other way round what
// syncing one subject and reading another costs (see bwSyncer).
type inventorySource interface {
	secrets.BWLister
	secrets.BWSyncer
}

// bwLister is the inventory seam, a var so command tests run with no daemon and
// no vault. Nil in production, where the inventory is read from the daemon
// directly (secrets.BWServeClient{}). The pinned backend in bwbackend.go has no
// lister half on purpose: a list answers with every item's plaintext, and the
// value-free projection (bwserve_list.go) has exactly one implementation to
// audit. A CLI lister would be a second one.
var bwLister inventorySource

// readInventory syncs the store, then reads its shape: every item and every
// folder. Both commands that compare the registry with the store read through it.
//
// The sync is not optional for either. The daemon answers from its own cache,
// independent of the server, and a folder made since its last sync is missing
// from the list while items already filed in it are not. reconcile writes from
// this inventory, and drift's exit code is what a gate reads: drift used to skip
// the sync because it "only reports" (CLI-078 review round 4). A sync pulls the
// server's state into the daemon's cache and writes no item, so drift stays
// read-only (AC8).
func readInventory() ([]secrets.ItemSummary, []string, error) {
	src := bwLister
	if src == nil {
		src = secrets.BWServeClient{}
	}
	if err := src.Sync(); err != nil {
		return nil, nil, fmt.Errorf("sync before reading the inventory: %w\n"+
			"a possibly stale inventory is refused, not compared", err)
	}
	items, err := src.ListItems()
	if err != nil {
		return nil, nil, fmt.Errorf("read the vault's inventory: %w\n"+
			"an unreadable store is not an empty one", err)
	}
	folders, err := src.ListFolders()
	if err != nil {
		return nil, nil, fmt.Errorf("read the vault's folders: %w", err)
	}
	return items, folders, nil
}

// newSecretsDriftCmd reports where the store disagrees with the registry.
//
// It exists because the registry described a store nobody had ever compared it
// to. Measured the first time this ran against the live vault: 169 of 185 items
// unfoldered, every declared folder name absent under that spelling, and three
// declared items missing entirely — one of them holding a credential that had
// been rotated elsewhere, so the read path was serving a token GitHub answers
// 401 to while `dotf secrets verify` reported OK.
//
// READ-ONLY, and not merely by omission: there is no --fix, the command is never
// handed a writer, and the comparison it calls has none. Converging the store is
// `reconcile` (CLI-080); a report that quietly repaired things would make the
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
			"  item-folder-unknown  the item is filed in a folder the folder list does\n" +
			"                   not carry, so its placement cannot be judged\n\n" +
			"Only item names, folder names and field names are read — never a value.\n" +
			"The store is synced first, so the report is the server's and not a stale\n" +
			"cache's. Nothing is written: `reconcile` converges the store, `drift` only\n" +
			"reports.\n\n" +
			"Exit is non-zero when any disagreement is found, so CI and a hook can gate\n" +
			"on it.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			items, folders, err := readInventory()
			if err != nil {
				return err
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

			named, present := declaredItems(decls, items)
			_, _ = fmt.Fprintf(out, "\n%d declared target(s) naming %d item(s), %d of them in the vault; %d finding(s); "+
				"%d of %d vault items unmanaged by this registry\n",
				len(decls), named, present, len(findings), len(unmanaged), len(items))

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

// declaredItems counts the distinct item names the declarations use, and how
// many of those the vault holds. The summary once printed the second number as
// the first, so an absent item shrank "the items the registry declares" instead
// of showing up as the gap between the two (CLI-078 review round 4).
func declaredItems(decls []secrets.BWDecl, items []secrets.ItemSummary) (named, present int) {
	inVault := make(map[string]bool, len(items))
	for _, it := range items {
		inVault[it.Name] = true
	}
	seen := map[string]bool{}
	for _, d := range decls {
		if seen[d.Item] {
			continue
		}
		seen[d.Item] = true
		named++
		if inVault[d.Item] {
			present++
		}
	}
	return named, present
}
