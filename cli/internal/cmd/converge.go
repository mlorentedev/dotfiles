package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

func newConvergeCmd() *cobra.Command {
	var (
		plan bool
		repo string
		only []string
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
			"Re-running on a converged machine reports 0 changed.\n\n" +
			"--only runs the named reconcilers, in registry order, and leaves the\n" +
			"persisted report of the last full run alone.",
		Example:      "  dotf converge --plan\n  dotf converge\n  dotf converge --only git-config",
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
			reg := converge.Registry(convergeOptions())
			if len(only) > 0 {
				var serr error
				if reg, serr = converge.Select(reg, only); serr != nil {
					return serr
				}
			}
			rep, err := converge.Run(reg, e, plan)
			printConvergeReport(cmd.OutOrStdout(), rep)
			if plan || len(only) > 0 {
				return err
			}
			state, serr := env.StateDir()
			if serr != nil {
				return errors.Join(err, fmt.Errorf("writing the converge report: %w", serr))
			}
			path := filepath.Join(state, "converge", "last.json")
			if werr := converge.WriteReport(path, rep, err); werr != nil {
				return errors.Join(err, fmt.Errorf("writing the converge report: %w", werr))
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "report: %s\n", path)
			return err
		},
	}
	cmd.Flags().BoolVar(&plan, "plan", false, "report what would change and write nothing")
	cmd.Flags().StringVar(&repo, "repo", "", "dotfiles checkout to converge from")
	cmd.Flags().StringSliceVar(&only, "only", nil, "run only these reconcilers (comma-separated names)")
	return cmd
}

// convergeOptions wires the registry's side effects; tests replace it.
var convergeOptions = func() converge.Options {
	run, stdout := tools.HomeRunners(env.Home())
	return converge.Options{
		RunHarnessDeploy: converge.CompileHarnessDeploy,
		MiseRun:          run,
		MiseStdout:       stdout,
		GitRun:           gitconfig.ExecRunner,
		RenderConfigs:    strictDeployRenderer,
		ResolvePath:      env.ResolvePath,
		Launchctl:        env.ExecLaunchctl,
		UID:              os.Getuid(),
	}
}

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
