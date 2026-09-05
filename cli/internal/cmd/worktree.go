package cmd

import (
	"encoding/json"
	"fmt"
	"os"
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

func newWorktreeListCmd() *cobra.Command {
	var (
		jsonOutput bool
		repoDir    string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all worktrees with their lifecycle status and fail-closed evaluation",
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			root := repoDir
			if root == "" {
				root = env.RepoDir()
			}
			if root == "" {
				return fmt.Errorf("not in a git repository: run from inside a repo or pass --repo")
			}

			infos, err := worktree.List(root)
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

			w := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "PATH\tBRANCH\tSTATUS\tPR\tSTATE\tREASON")
			for _, info := range infos {
				status := "clean"
				if info.Dirty {
					status = "dirty"
				}

				prStatus := "-"
				if info.PRMerged {
					prStatus = "merged"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					info.Path,
					info.Branch,
					status,
					prStatus,
					info.State,
					info.StateReason,
				)
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	cmd.Flags().StringVar(&repoDir, "repo", "", "target repository root (defaults to current repository)")

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
			root := repoDir
			if root == "" {
				root = env.RepoDir()
			}
			if root == "" {
				return fmt.Errorf("not in a git repository: run from inside a repo or pass --repo")
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
			root := repoDir
			if root == "" {
				root = env.RepoDir()
			}
			if root == "" {
				return fmt.Errorf("not in a git repository: run from inside a repo or pass --repo")
			}

			opts := worktree.SweepOptions{
				RepoRoot: root,
				DryRun:   dryRun,
			}

			report, err := worktree.Sweep(opts)
			if err != nil {
				return fmt.Errorf("sweeping worktrees: %w", err)
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
			root := repoDir
			if root == "" {
				root = env.RepoDir()
			}
			if root == "" {
				return fmt.Errorf("not in a git repository: run from inside a repo or pass --repo")
			}

			target := worktreePath
			if len(args) > 0 {
				target = args[0]
			}
			if target == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				target = cwd
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
