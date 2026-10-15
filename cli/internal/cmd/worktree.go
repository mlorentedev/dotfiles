package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	cmd.AddCommand(newWorktreeAdoptCmd())
	return cmd
}

func newWorktreeAdoptCmd() *cobra.Command {
	var (
		repoDir string
		issue   int
		ttl     time.Duration
	)
	cmd := &cobra.Command{
		Use:   "adopt [path|slug]",
		Short: "Give a worktree created without dotf (by herdr or git) a lease, so sweep may reap it",
		Long: `adopt writes the .dotf-worktree.json lease add would have written, for a
linked worktree that has none: list shows such a worktree as UNADOPTED, and
sweep never reaps it. With a lease it is reaped like any other once merged,
clean and past the lease. A worktree that already has a lease keeps it.

The argument is a path or a slug, resolved as done resolves it; without one,
the worktree holding the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			root, err := resolveCommandRepoRoot(repoDir)
			if err != nil {
				return err
			}
			target, err := resolveDoneTarget(args, "", repoDir)
			if err != nil {
				return err
			}
			res, err := worktree.Adopt(worktree.AdoptOptions{RepoRoot: root, WorktreePath: target, TTL: ttl, Issue: issue}, time.Now())
			if err != nil {
				return err
			}
			if res.AlreadyAdopted {
				c.Printf("%s already has a lease, until %s; unchanged\n", target, res.Metadata.LeaseExpiresAt.Format("2006-01-02 15:04:05 MST"))
				return nil
			}
			c.Printf("Adopted %s; lease active until %s\n", target, res.Metadata.LeaseExpiresAt.Format("2006-01-02 15:04:05 MST"))
			return nil
		},
	}
	cmd.Flags().StringVar(&repoDir, "repo", "", "target repository root (defaults to current repo)")
	cmd.Flags().IntVar(&issue, "issue", 0, "associated GitHub issue number")
	cmd.Flags().DurationVar(&ttl, "ttl", 24*time.Hour, "lease duration before sweep may reap it")
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
		Short: "Create a worktree under WORKTREE_ROOT with safety validation and lease metadata",
		Long: `add creates a new worktree isolated outside the parent repository, at
<WORKTREE_ROOT>/<repo>/<branch-slug>: the layout herdr creates, so a worktree made by
either tool is found by the other. The branch defaults to feat/<slug>, so add foo
creates <WORKTREE_ROOT>/<repo>/feat-foo. --path overrides the location.
The branch must be <type>/<kebab-slug> with no phase or milestone token, and
starts from origin's default branch unless --base names another.
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
				RepoRoot:     root,
				WorktreeRoot: env.ResolvePath("WORKTREE_ROOT"),
				Slug:         slug,
				CustomPath:   customPath,
				Branch:       branch,
				BaseRef:      base,
				Issue:        issue,
				TTL:          ttl,
			}

			info, err := worktree.Add(opts)
			if err != nil {
				return fmt.Errorf("creating worktree: %w", err)
			}

			c.Printf("Created worktree at %s (branch: %s)\n", info.Path, info.Branch)
			if name := filepath.Base(info.Path); customPath == "" && name != slug && info.Branch != worktree.DefaultBranch(slug) {
				c.Printf("The directory follows the branch: tear down with `dotf worktree done %s`\n", name)
			}
			if info.Metadata != nil {
				c.Printf("Lease active until %s\n", info.Metadata.LeaseExpiresAt.Format("2006-01-02 15:04:05 MST"))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repoDir, "repo", "", "target repository root (defaults to current repo)")
	cmd.Flags().StringVar(&customPath, "path", "", "custom external target path")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "custom branch name (defaults to feat/<slug>)")
	cmd.Flags().StringVar(&base, "base", "", "base commit or branch (defaults to origin's default branch, fetched first: origin/HEAD, origin/main, origin/master)")
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
				RepoRoot:          root,
				DryRun:            dryRun,
				ClaudeProjectsDir: claudeProjectsDir(),
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
			} else if report.UninspectableProcesses > 0 {
				// Gate f's reach, not a warning. This number is non-zero on
				// every Linux machine — reading /proc/<pid>/cwd is gated by
				// ptrace_may_access, and a desktop runs hundreds of processes
				// this user may not read (measured: 364 of 571, 20 of them the
				// caller's own). It said "this scan was partial" before, which
				// is a warning light wired to always-on, and a reader learns to
				// skip those.
				//
				// Still gated on > 0, because 0 is ambiguous rather than good:
				// gateF only scans worktrees that reached StateReapable, so a
				// run with nothing to consider also produces 0, and printing a
				// reach for a scan that never happened would be its own lie.
				c.Printf("Gate f reach: %d process(es) could not be inspected (other users, root, or\n"+
					"              same-user processes that opt out); one of those sitting in a\n"+
					"              worktree is invisible to this scan.\n",
					report.UninspectableProcesses)
			}

			for _, s := range report.SessionSkips {
				c.Printf("  - skipped: %s: %s\n", s.Path, s.Reason)
			}
			if opts.DryRun {
				c.Printf("[DRY-RUN] Found %d reapable worktree(s), %d skipped.\n", len(report.Reaped), report.SkippedCount)
				for _, r := range report.Reaped {
					c.Printf("  - would reap: %s (%s)\n", r.Path, r.Branch)
				}
				return nil
			}

			c.Printf("Sweep complete: reaped %d worktree(s), %d skipped (active, dirty, unmerged or a recent session).\n", len(report.Reaped), report.SkippedCount)
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

// resolveDoneTarget picks the worktree done removes: the argument, then --path,
// then the current directory.
func resolveDoneTarget(args []string, worktreePath, repoDir string) (string, error) {
	if len(args) > 0 {
		return resolveDoneArg(args[0], repoDir)
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

// resolveDoneArg reads done's argument as the path add printed or the slug add
// was given. A bare name is also a slug, naming any of worktree.SlugCandidates:
// the legacy <repo>-wt-<slug> sibling, or the worktree under WORKTREE_ROOT. A
// name that resolves to more than one directory is refused, naming each, rather
// than guessed. When it names none, the error lists every path it tried.
func resolveDoneArg(arg, repoDir string) (string, error) {
	asPath, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	if strings.ContainsAny(arg, `/\`) {
		if isDirectory(asPath) {
			return worktreeAt(asPath)
		}
		if p := branchUnderRoot(arg, repoDir); p != "" && isDirectory(p) {
			return worktreeAt(p)
		}
		return "", fmt.Errorf("no worktree at %s", asPath)
	}
	root, rootErr := resolveCommandRepoRoot(repoDir)
	if rootErr != nil {
		if isDirectory(asPath) {
			return worktreeAt(asPath)
		}
		return "", fmt.Errorf("no worktree at %s, and no repository to read %q as a slug in: %w", asPath, arg, rootErr)
	}
	wtRoot := env.ResolvePath("WORKTREE_ROOT")
	tried := append([]string{asPath}, worktree.SlugCandidates(root, wtRoot, arg)...)
	var found []string
	for _, p := range tried {
		if isDirectory(p) && !slices.Contains(found, p) {
			found = append(found, p)
		}
	}
	switch len(found) {
	case 0:
		return "", doneNotFound(arg, tried, wtRoot)
	case 1:
		return worktreeAt(found[0])
	}
	return "", fmt.Errorf("%q names %d directories, %s; pass the full path of the one to remove", arg, len(found), strings.Join(found, " and "))
}

// branchUnderRoot reads arg as a branch name and returns where add put its
// worktree under WORKTREE_ROOT, or "" when no repository or root resolves. A
// path that exists always wins over this reading.
func branchUnderRoot(arg, repoDir string) string {
	if strings.Contains(arg, `\`) {
		return ""
	}
	root, err := resolveCommandRepoRoot(repoDir)
	wtRoot := env.ResolvePath("WORKTREE_ROOT")
	if err != nil || wtRoot == "" {
		return ""
	}
	return worktree.ResolveWorktreePath(wtRoot, filepath.Base(filepath.Clean(root)), arg)
}

// doneNotFound names every path done tried, and says when the root itself was
// not searched, so a missing env-contract.json reads as that and not as "the
// worktree does not exist".
func doneNotFound(arg string, tried []string, wtRoot string) error {
	msg := fmt.Sprintf("no worktree %q: none of %s is a directory", arg, strings.Join(tried, ", "))
	if wtRoot == "" {
		msg += "; WORKTREE_ROOT could not be resolved, so the worktree root was not searched (run `dotf doctor`)"
	}
	return errors.New(msg)
}

// worktreeAt returns the top of the git worktree holding dir, or an error that
// names dir when it is not inside one, rather than letting git status fail on
// it later with a bare exit status.
func worktreeAt(dir string) (string, error) {
	top, err := worktree.ResolveWorktreeRoot(dir)
	if err != nil {
		return "", fmt.Errorf("%s is a directory but not a git worktree", dir)
	}
	return top, nil
}

func isDirectory(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
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
		repoDir             string
		worktreePath        string
		force               bool
		ignoreRecentSession bool
	)

	cmd := &cobra.Command{
		Use:   "done [path | slug]",
		Short: "Tear down a completed worktree cleanly",
		Long: `done removes a completed worktree and prunes git metadata. Unless --force is
passed, it refuses on uncommitted changes, and on commits that are neither on
the upstream nor contained in the head of a merged pull request for the branch
(how a squash-merged branch looks once its remote is deleted). When that pull
request cannot be listed (no gh, offline), it refuses and says so.

Whatever --force says, it also refuses a worktree that contains the current
directory or, on Linux and macOS, the working directory of a process it runs under: the
removal would strand that shell or agent, and a Claude Code session launched
from the worktree could not be resumed. Run it from outside the worktree.
It also refuses one whose Claude Code transcript was written in the last 6h,
a session that may still be resumed; --ignore-recent-session overrides that
check, and --force does not.

The argument is the worktree's path, its branch, or its directory name under
the root. The directory follows the branch: add <slug> makes feat/<slug> in
.../feat-<slug>, so the slug works too, but add <slug> -b fix/login makes
.../fix-login, which done takes as fix/login or fix-login (add prints it). A
bare name resolves to <WORKTREE_ROOT>/<repo>/<name> or .../feat-<name>, and to
the legacy <repo>-wt-<name> beside the repository. A name matching more than
one is refused.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			target, err := resolveDoneTarget(args, worktreePath, repoDir)
			if err != nil {
				return err
			}

			root, err := resolveDoneRepoRoot(repoDir, target)
			if err != nil {
				return err
			}

			opts := worktree.DoneOptions{
				RepoRoot:            root,
				WorktreePath:        target,
				Cwd:                 callerCwd(os.Getwd, os.Getenv),
				Force:               force,
				ClaudeProjectsDir:   claudeProjectsDir(),
				IgnoreRecentSession: ignoreRecentSession,
			}

			if err := worktree.Done(opts); err != nil {
				return err
			}

			c.Printf("Cleaned up worktree at %s\n", target)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoDir, "repo", "", "target repository root (defaults to current repo)")
	cmd.Flags().StringVar(&worktreePath, "path", "", "path to worktree to remove (defaults to the one holding the current directory, which done then refuses)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force removal even if there are uncommitted changes")
	cmd.Flags().BoolVar(&ignoreRecentSession, "ignore-recent-session", false, "remove even though a Claude Code session wrote its transcript in the last 6h (--force does not)")

	return cmd
}

// callerCwd is the directory `worktree done` checks it is not removing from
// under its caller. When the cwd cannot be read, the launching shell's $PWD
// stands in, so the check does not silently drop out: on Windows there is no
// ancestor walk behind it.
func callerCwd(getwd func() (string, error), getenv func(string) string) string {
	if cwd, err := getwd(); err == nil {
		return cwd
	}
	return getenv("PWD")
}

// claudeProjectsDir is where Claude Code keeps one transcript directory per
// working directory, read by done and sweep to spot a live session (#1653).
func claudeProjectsDir() string {
	return filepath.Join(claudeConfigDir(env.Home()), "projects")
}
