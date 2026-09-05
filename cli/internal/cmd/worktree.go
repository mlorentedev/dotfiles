package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
	"github.com/spf13/cobra"
)

func newWorktreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "worktree",
		Aliases: []string{"wt"},
		Short:   "Worktree lifecycle management and fail-closed garbage collection",
		Long: `worktree manages Git worktrees across parallel agent sessions with fail-closed
safety guarantees. It inspects leases, PR merge status, and git state to prevent
workstation clutter without risking data loss.`,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return c.Help()
		},
	}

	cmd.AddCommand(newWorktreeListCmd())
	cmd.AddCommand(newWorktreeAddCmd())
	cmd.AddCommand(newWorktreeSweepCmd())
	cmd.AddCommand(newWorktreeDoneCmd())
	return cmd
}

func resolveCommandRepoRoot(repoDir string) (string, error) {
	if repoDir != "" {
		return repoDir, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		if mainRoot, err := worktree.ResolveMainRepoRoot(cwd); err == nil && mainRoot != "" {
			return mainRoot, nil
		}
	}
	root := env.RepoDir()
	if root == "" {
		return "", fmt.Errorf("not in a git repository: run from inside a repo or pass --repo")
	}
	return root, nil
}

func printWorktreeTable(out io.Writer, infos []worktree.Info) error {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "PATH\tBRANCH\tSTATUS\tPR\tSTATE\tREASON"); err != nil {
		return err
	}
	for _, info := range infos {
		status := "clean"
		if info.Dirty {
			status = "dirty"
		}

		prStatus := "-"
		if info.PRMerged {
			prStatus = "merged"
		}

		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			info.Path,
			info.Branch,
			status,
			prStatus,
			info.State,
			info.StateReason,
		); err != nil {
			return err
		}
	}
	return w.Flush()
}

func newWorktreeListCmd() *cobra.Command {
	var (
		jsonOutput bool
		repoDir    string
		allRepos   bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all worktrees with their lifecycle status and fail-closed evaluation",
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			root, err := resolveCommandRepoRoot(repoDir)
			if err != nil {
				return err
			}

			var infos []worktree.Info
			if allRepos {
				parentDir := filepath.Dir(root)
				infos, err = worktree.ListAll(parentDir)
			} else {
				infos, err = worktree.List(root)
			}
			if err != nil {
				return fmt.Errorf("listing worktrees: %w", err)
			}

			if jsonOutput {
				enc := json.NewEncoder(c.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(infos)
			}

			if len(infos) == 0 {
				c.Println("No worktrees found.")
				return nil
			}

			return printWorktreeTable(c.OutOrStdout(), infos)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	cmd.Flags().StringVar(&repoDir, "repo", "", "target repository root (defaults to current repository)")
	cmd.Flags().BoolVar(&allRepos, "all", false, "scan all sibling repositories in parent directory")

	return cmd
}

func newWorktreeAddCmd() *cobra.Command {
	var (
		repoDir    string
		customPath string
		branch     string
		base       string
		issue      int
		ttl        time.Duration
	)

	cmd := &cobra.Command{
		Use:   "add <slug>",
		Short: "Create an external sibling worktree with safety validation and lease metadata",
		Long: `add creates a new worktree isolated outside the parent repository (<repo>-wt-<slug>).
It writes .dotf-worktree.json to register lease time and creator metadata, and automatically
adds it to .git/info/exclude.`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			slug := args[0]
			root, err := resolveCommandRepoRoot(repoDir)
			if err != nil {
				return err
			}

			opts := worktree.AddOptions{
				RepoRoot:   root,
				Slug:       slug,
				CustomPath: customPath,
				Branch:     branch,
				BaseRef:    base,
				Issue:      issue,
				TTL:        ttl,
			}

			info, err := worktree.Add(opts)
			if err != nil {
				return fmt.Errorf("creating worktree: %w", err)
			}

			c.Printf("Created worktree at %s (branch: %s)\n", info.Path, info.Branch)
			if info.Metadata != nil {
				c.Printf("Lease active until %s\n", info.Metadata.LeaseExpiresAt.Format("2006-01-02 15:04:05 MST"))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repoDir, "repo", "", "target repository root (defaults to current repo)")
	cmd.Flags().StringVar(&customPath, "path", "", "custom external target path")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "custom branch name (defaults to feat/<slug>)")
	cmd.Flags().StringVar(&base, "base", "", "base commit or branch for checkout")
	cmd.Flags().IntVar(&issue, "issue", 0, "associated GitHub issue number")
	cmd.Flags().DurationVar(&ttl, "ttl", 24*time.Hour, "initial lease duration before eligible for reap")

	return cmd
}

func newWorktreeSweepCmd() *cobra.Command {
	var (
		repoDir string
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "sweep",
		Short: "Reap merged and clean worktrees under fail-closed safety constraints",
		Long: `sweep executes garbage collection across all linked worktrees. It reaps ONLY
worktrees that meet all positive fail-closed gates: explicit reap_ok, expired lease,
clean git status, confirmed merged PR, and minimum age.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			root, err := resolveCommandRepoRoot(repoDir)
			if err != nil {
				return err
			}

			opts := worktree.SweepOptions{
				RepoRoot: root,
				DryRun:   dryRun,
			}

			report, err := worktree.Sweep(opts)
			if err != nil {
				return fmt.Errorf("sweeping worktrees: %w", err)
			}

			// Said before the counts, never after: an inert sweep prints
			// "reaped 0" exactly like a machine with nothing to clean up, and a
			// reader who takes the first for the second concludes the tool ran.
			if !report.ProcessDiscovery {
				c.Printf("note: no process-liveness check on this platform, so nothing can be reaped.\n")
				c.Printf("      Gate f refuses rather than guesses; remove one deliberately with `dotf worktree done`.\n")
			}

			if opts.DryRun {
				c.Printf("[DRY-RUN] Found %d reapable worktree(s), %d skipped.\n", len(report.Reaped), report.SkippedCount)
				for _, r := range report.Reaped {
					c.Printf("  - would reap: %s (%s)\n", r.Path, r.Branch)
				}
				return nil
			}

			c.Printf("Sweep complete: reaped %d worktree(s), %d skipped (active/dirty/unmerged).\n", len(report.Reaped), report.SkippedCount)
			for _, r := range report.Reaped {
				c.Printf("  - reaped: %s (%s)\n", r.Path, r.Branch)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repoDir, "repo", "", "target repository root (defaults to current repo)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "simulate sweep without removing any worktrees or branches")

	return cmd
}

func resolveDoneTarget(args []string, worktreePath string) (string, error) {
	if len(args) > 0 {
		target := args[0]
		if top, err := worktree.ResolveWorktreeRoot(target); err == nil {
			return top, nil
		}
		return target, nil
	}
	if worktreePath != "" {
		if top, err := worktree.ResolveWorktreeRoot(worktreePath); err == nil {
			return top, nil
		}
		return worktreePath, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	top, err := worktree.ResolveWorktreeRoot(cwd)
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	return top, nil
}

func resolveDoneRepoRoot(repoDir, target string) (string, error) {
	if repoDir != "" {
		return repoDir, nil
	}
	mainRoot, err := worktree.ResolveMainRepoRoot(target)
	if err == nil && mainRoot != "" {
		return mainRoot, nil
	}
	root := env.RepoDir()
	if root == "" {
		return "", fmt.Errorf("not in a git repository: run from inside a repo or pass --repo")
	}
	return root, nil
}

func newWorktreeDoneCmd() *cobra.Command {
	var (
		repoDir      string
		worktreePath string
		force        bool
	)

	cmd := &cobra.Command{
		Use:   "done [path]",
		Short: "Tear down a completed worktree cleanly",
		Long: `done removes a completed worktree and prunes git metadata. Refuses if there are
uncommitted changes unless --force is passed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			target, err := resolveDoneTarget(args, worktreePath)
			if err != nil {
				return err
			}

			root, err := resolveDoneRepoRoot(repoDir, target)
			if err != nil {
				return err
			}

			opts := worktree.DoneOptions{
				RepoRoot:     root,
				WorktreePath: target,
				Force:        force,
			}

			if err := worktree.Done(opts); err != nil {
				return err
			}

			c.Printf("Cleaned up worktree at %s\n", target)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoDir, "repo", "", "target repository root (defaults to current repo)")
	cmd.Flags().StringVar(&worktreePath, "path", "", "path to worktree to remove (defaults to current directory)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force removal even if there are uncommitted changes")

	return cmd
}
