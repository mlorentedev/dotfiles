package cmd

import (
	"fmt"
	"io"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// ciWorkflowSource and ciFleet are the gh seams of `secrets ci-check`, vars so
// command tests run on fixtures with no gh and no network.
var (
	ciWorkflowSource secrets.WorkflowSource = secrets.GHWorkflows{}
	ciFleet                                 = secrets.GHWorkflows{}.Fleet
)

// newSecretsCICheckCmd implements `dotf secrets ci-check [--repo OWNER/REPO]...`:
// each repository's workflow reads against the secrets the registry declares
// for it. Names only, never values.
func newSecretsCICheckCmd() *cobra.Command {
	var repos []string
	c := &cobra.Command{
		Use:   "ci-check",
		Short: "Compare the CI secrets each repository's workflows read with the ones the registry declares",
		Long: "ci-check reads each repository's .github/workflows (and, for a reusable workflow\n" +
			"called with `secrets: inherit`, the callee's on.workflow_call.secrets at the pinned\n" +
			"ref) and compares the secret names they read with what `secrets sync ci` would\n" +
			"upload. It reports missing, excess and unknown names, never values, and exits\n" +
			"non-zero when any repository has a missing or excess name. A secret a callee\n" +
			"marks `required: false` still counts as missing, marked optional: the run does\n" +
			"not fail without it, it degrades. With no --repo it checks every non-archived,\n" +
			"non-fork repository you own.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			if len(repos) == 0 {
				if repos, err = ciFleet(); err != nil {
					return err
				}
			}
			return runCICheck(cmd.OutOrStdout(), secrets.NewCIChecker(ciWorkflowSource, reg), repos)
		},
	}
	c.Flags().StringSliceVar(&repos, "repo", nil, "repository to check, as owner/name (repeatable; default: every repository you own)")
	return c
}

func runCICheck(w io.Writer, chk *secrets.CIChecker, repos []string) error {
	bad := 0
	for _, repo := range repos {
		rep, err := chk.CheckRepo(repo)
		if err != nil {
			bad++
			_, _ = fmt.Fprintf(w, "%s: ERROR %v\n", repo, err)
			continue
		}
		if rep.Failing() {
			bad++
		}
		printCIReport(w, rep)
	}
	if bad > 0 {
		return fmt.Errorf("%d of %d repositories disagree with the registry or could not be read", bad, len(repos))
	}
	return nil
}

func printCIReport(w io.Writer, rep secrets.CIReport) {
	switch {
	case rep.Failing():
		_, _ = fmt.Fprintf(w, "%s: FAIL\n", rep.Repo)
	case len(rep.Unknown) > 0:
		_, _ = fmt.Fprintf(w, "%s: UNKNOWN\n", rep.Repo)
	default:
		_, _ = fmt.Fprintf(w, "%s: ok\n", rep.Repo)
	}
	for _, m := range rep.Missing {
		if m.OptionalIn != "" {
			_, _ = fmt.Fprintf(w, "  missing  %s (optional in %s)\n", m.Name, m.OptionalIn)
			continue
		}
		_, _ = fmt.Fprintf(w, "  missing  %s\n", m.Name)
	}
	for _, e := range rep.Excess {
		_, _ = fmt.Fprintf(w, "  excess   %s\n", e)
	}
	for _, u := range rep.Unknown {
		_, _ = fmt.Fprintf(w, "  unknown  %s: %s\n", u.File, u.Expr)
	}
}
