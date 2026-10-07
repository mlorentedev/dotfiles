package cmd

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

func newConvergeCmd() *cobra.Command {
	var (
		plan bool
		repo string
	)
	cmd := &cobra.Command{
		Use:   "converge",
		Short: "Bring this machine to the state the dotfiles checkout declares",
		Long: "converge runs an ordered list of reconcilers, each converging one part of\n" +
			"the machine from data in the checkout. Records come first, so nothing runs\n" +
			"before the files it reads exist.\n\n" +
			"--plan reports what each reconciler would change and writes nothing.\n" +
			"Without it, each reconciler applies and then proves its post-condition; the\n" +
			"first failure stops the run and names the reconciler. A reconciler that does\n" +
			"not apply to this OS is reported as skipped, never as passed.\n\n" +
			"Re-running on a converged machine reports 0 changed.",
		Example:      "  dotf converge --plan\n  dotf converge",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if repo == "" {
				repo = env.RepoDir()
			}
			if repo == "" {
				return fmt.Errorf("cannot locate the dotfiles checkout — pass --repo, set DOTFILES_REPO_DIR, or run from inside it")
			}
			home := env.Home()
			e := converge.Env{RepoRoot: repo, Home: home, DeployDir: env.DotfilesDir(home), GOOS: runtime.GOOS}
			rep, err := converge.Run(converge.Registry(convergeOptions), e, plan)
			printConvergeReport(cmd.OutOrStdout(), rep)
			if plan {
				return err
			}
			path := filepath.Join(env.StateDir(), "converge", "last.json")
			if werr := converge.WriteReport(path, rep, err); werr != nil {
				return errors.Join(err, fmt.Errorf("writing the converge report: %w", werr))
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "report: %s\n", path)
			return err
		},
	}
	cmd.Flags().BoolVar(&plan, "plan", false, "report what would change and write nothing")
	cmd.Flags().StringVar(&repo, "repo", "", "dotfiles checkout to converge from")
	return cmd
}

// convergeOptions wires the registry's side effects; tests replace it.
var convergeOptions = converge.Options{RunHarnessDeploy: converge.CompileHarnessDeploy}

var convergeTag = map[converge.Status]string{
	converge.StatusOK:      "[ OK ]",
	converge.StatusChange:  "[CHANGE]",
	converge.StatusSkipped: "[SKIP]",
	converge.StatusFailed:  "[FAIL]",
}

func printConvergeReport(w io.Writer, rep converge.Report) {
	heading, changed := "converge", "changed"
	if rep.DryRun {
		heading, changed = "converge plan", "to change"
	}
	_, _ = fmt.Fprintf(w, "%s on %s:\n", heading, rep.GOOS)
	for _, e := range rep.Entries {
		_, _ = fmt.Fprintf(w, "  %-8s %-16s %s\n", convergeTag[e.Status], e.Name, e.Detail)
	}
	n := rep.Summary()
	_, _ = fmt.Fprintf(w, "%d %s, %d ok, %d skipped, %d failed\n",
		n[converge.StatusChange], changed, n[converge.StatusOK], n[converge.StatusSkipped], n[converge.StatusFailed])
}
