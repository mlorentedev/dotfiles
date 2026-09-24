package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	protection.AddCommand(newForgeProtectionCheckCmd())
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
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := spec.RepoRoot(cwd)
			if err != nil {
				return err
			}
			decl, err := forge.Load(root)
			if err != nil {
				c.PrintErrln("forge protection check:", err)
				return err
			}
			if repo != "" {
				one, ok := decl.Repos[repo]
				if !ok {
					err := fmt.Errorf("forge protection check: %s is not declared in %s", repo, forge.DeclarationFile)
					c.PrintErrln(err)
					return err
				}
				decl.Repos = map[string]forge.RepoDecl{repo: one}
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
