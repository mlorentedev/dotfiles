package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/forge"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// errForgeAttention is returned (silently) when a declared repository drifted
// or could not be checked. Same exit contract as errPending: non-zero both
// when something is wrong and when the question could not be answered.
var errForgeAttention = errors.New("forge protection check: at least one repository needs attention")

// errForgeApply is returned (silently) when apply refused or failed on at least
// one repository.
var errForgeApply = errors.New("forge protection apply: at least one repository was not converged")

// forgeTimeout bounds one gh call, so a hung network degrades to an
// unanswerable result instead of a hung check.
const forgeTimeout = 20 * time.Second

// forgeRun runs gh with a deadline; a package var so tests never reach GitHub.
var forgeRun forge.Runner = func(args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), forgeTimeout)
	defer cancel()
	var out, errOut bytes.Buffer
	c := exec.CommandContext(ctx, "gh", args...)
	c.Stdout, c.Stderr = &out, &errOut
	err := c.Run()
	return out.String(), errOut.String(), err
}

func newForgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "forge",
		Short:        "GitHub-side state declared in git (branch protection)",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	protection := &cobra.Command{
		Use:          "protection",
		Short:        "Branch protection declared in " + forge.DeclarationFile,
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	protection.AddCommand(newForgeProtectionCheckCmd(), newForgeProtectionApplyCmd())
	cmd.AddCommand(protection)
	return cmd
}

func newForgeProtectionCheckCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Diff every declared repository's live branch protection against the declaration",
		Long: `Diff the live branch protection of every repository declared in
` + forge.DeclarationFile + ` against its declaration, field by field.

This exists because branch protection leaves no trace in git: a required
context that is dropped or renamed is invisible until a merge that should
have been impossible (GUARD-017, #1451). It READS only; nothing here changes
a repository.

  DRIFT         live differs from the declaration (each field is named)
  STATE         declared unavailable or unprotected, with its reason
  UNANSWERABLE  the forge could not be asked (auth, network, a 403)

Exit status is 0 only when nothing drifted and every question was answered.`,
		Example:       "  dotf forge protection check\n  dotf forge protection check --repo mlorentedev/hive",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			decl, err := loadForgeDecl(c, "check", repo)
			if err != nil {
				return err
			}
			results := forge.CheckAll(decl, forgeRun)
			printForgeCheck(c.OutOrStdout(), results)
			if forge.NeedsAttention(results) {
				return errForgeAttention
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "check one declared owner/name only")
	return cmd
}

// loadForgeDecl loads the declaration of the checkout containing the cwd,
// restricted to one repository when repo is set.
func loadForgeDecl(c *cobra.Command, verb, repo string) (forge.Declaration, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return forge.Declaration{}, err
	}
	root, err := spec.RepoRoot(cwd)
	if err != nil {
		return forge.Declaration{}, err
	}
	decl, err := forge.Load(root)
	if err != nil {
		c.PrintErrln("forge protection "+verb+":", err)
		return forge.Declaration{}, err
	}
	if repo == "" {
		return decl, nil
	}
	one, ok := decl.Repos[repo]
	if !ok {
		err := fmt.Errorf("forge protection %s: %s is not declared in %s", verb, repo, forge.DeclarationFile)
		c.PrintErrln(err)
		return forge.Declaration{}, err
	}
	decl.Repos = map[string]forge.RepoDecl{repo: one}
	return decl, nil
}

func newForgeProtectionApplyCmd() *cobra.Command {
	var repo string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Converge every declared repository's live branch protection on the declaration",
		Long: `Write the branch protection declared in ` + forge.DeclarationFile + ` to every
declared repository whose live protection differs (GUARD-017, #1451).

It writes only on a difference, so a second run reports changed=0. The write
is the COMPLETE object, because the endpoint replaces the whole of it and an
omitted field would be silently cleared, and the result is re-read to prove
the fields took effect: the forge accepting a request is not the forge
applying it.

Before making a context newly required, it confirms that context reported on
at least one of the branch's last 5 merged pull requests, and refuses
otherwise: with enforce_admins, a required check that never reports locks the
branch, the owner included. Repositories declared unavailable or unprotected
are skipped; apply never removes protection.

  UNCHANGED  live already matches
  PLANNED    --dry-run: what a real run would change
  APPLIED    written and verified
  REFUSED    the preflight refused; nothing was written
  SKIPPED    a declared state, not an object
  FAILED     a read or write failed, or did not take effect

Exit status is 0 only when every repository is unchanged, planned, applied or
skipped. Recovery from a bad apply is re-applying the previous declaration
from git history: protection changes through the admin API, not a merge.`,
		Example:       "  dotf forge protection apply --dry-run\n  dotf forge protection apply --repo mlorentedev/dotfiles",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			decl, err := loadForgeDecl(c, "apply", repo)
			if err != nil {
				return err
			}
			results := applyAll(decl, dryRun)
			printForgeApply(c.OutOrStdout(), results)
			for _, r := range results {
				if r.Status == forge.ApplyRefused || r.Status == forge.ApplyFailed {
					return errForgeApply
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "apply to one declared owner/name only")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would change, run the preflight, and write nothing")
	return cmd
}

// applyAll applies in name order, one repository at a time: these are writes,
// and a readable log of them matters more than their wall time.
func applyAll(decl forge.Declaration, dryRun bool) []forge.ApplyResult {
	repos := make([]string, 0, len(decl.Repos))
	for r := range decl.Repos {
		repos = append(repos, r)
	}
	sort.Strings(repos)
	results := make([]forge.ApplyResult, 0, len(repos))
	for _, r := range repos {
		results = append(results, forge.ApplyRepo(r, decl.Repos[r], forgeRun, dryRun))
	}
	return results
}

func printForgeApply(w io.Writer, results []forge.ApplyResult) {
	counts := map[forge.ApplyStatus]int{}
	for _, r := range results {
		counts[r.Status]++
		line := fmt.Sprintf("  [%s] %s", strings.ToUpper(string(r.Status)), r.Repo)
		if r.Detail != "" {
			line += ": " + r.Detail
		}
		_, _ = fmt.Fprintln(w, line)
		if r.Status == forge.ApplyPlanned || r.Status == forge.ApplyApplied {
			for _, ch := range r.Changes {
				_, _ = fmt.Fprintf(w, "          %s: %s -> %s\n", ch.Field, ch.Live, ch.Declared)
			}
		}
	}
	changed := counts[forge.ApplyApplied] + counts[forge.ApplyPlanned]
	_, _ = fmt.Fprintf(w, "\n%d repositories: changed=%d (%d applied, %d planned), %d unchanged, %d skipped, %d refused, %d failed\n",
		len(results), changed, counts[forge.ApplyApplied], counts[forge.ApplyPlanned],
		counts[forge.ApplyUnchanged], counts[forge.ApplySkipped], counts[forge.ApplyRefused], counts[forge.ApplyFailed])
}

func printForgeCheck(w io.Writer, results []forge.RepoResult) {
	counts := map[forge.RepoStatus]int{}
	for _, r := range results {
		counts[r.Status]++
		switch r.Status {
		case forge.StatusOK:
			_, _ = fmt.Fprintf(w, "  [OK] %s\n", r.Repo)
		case forge.StatusDrift:
			_, _ = fmt.Fprintf(w, "  [DRIFT] %s\n", r.Repo)
			for _, ch := range r.Changes {
				_, _ = fmt.Fprintf(w, "          %s: declared %s, live %s\n", ch.Field, ch.Declared, ch.Live)
			}
		default:
			_, _ = fmt.Fprintf(w, "  [%s] %s: %s\n", strings.ToUpper(string(r.Status)), r.Repo, r.Detail)
		}
	}
	_, _ = fmt.Fprintf(w, "\n%d repositories: %d ok, %d drift, %d declared state, %d unanswerable\n",
		len(results), counts[forge.StatusOK], counts[forge.StatusDrift], counts[forge.StatusState], counts[forge.StatusUnanswerable])
}
