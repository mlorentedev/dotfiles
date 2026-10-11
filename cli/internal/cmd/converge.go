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
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
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
			"the machine from data in the checkout. The checkout comes first, then the\n" +
			"records, so nothing runs before the files it reads exist. On Linux and\n" +
			"Windows the setup script runs last, for what no native step covers yet; it\n" +
			"cannot plan, so it is reported opaque, never as converged.\n\n" +
			"--plan reports what each reconciler would change and writes nothing.\n" +
			"Without it, each reconciler applies and then proves its post-condition; the\n" +
			"first failure stops the run and names the reconciler. A reconciler that does\n" +
			"not apply to this OS is reported as skipped, never as passed.\n\n" +
			"Re-running on a converged machine reports 0 changed.\n\n" +
			"--only runs the named reconcilers, in registry order, and leaves the\n" +
			"persisted report of the last full run alone.\n\n" +
			"A full run that converged ends with the identity converge cannot\n" +
			"produce (age key, Bitwarden, GitHub login, vault): on a terminal it walks\n" +
			"the restore, as `dotf identity restore` does; elsewhere it prints what is\n" +
			"left. Neither changes the exit status.",
		Example:      "  dotf converge --plan\n  dotf converge\n  dotf converge --only git-config",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if repo == "" {
				repo = convergeCheckout(env.Home())
			}
			err := runConverge(cmd.OutOrStdout(), repo, plan, only)
			// The identity converge cannot produce: walked on a terminal,
			// printed elsewhere, after a full run that converged (#2013 D12).
			if err == nil && len(only) == 0 {
				identityEpilogue(cmd.OutOrStdout(), plan)
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&plan, "plan", false, "report what would change and write nothing")
	cmd.Flags().StringVar(&repo, "repo", "", "dotfiles checkout to converge from")
	cmd.Flags().StringSliceVar(&only, "only", nil, "run only these reconcilers (comma-separated names)")
	return cmd
}

// runConverge drives the registry over repo and prints the report. A full
// apply also persists it; a plan or an --only run leaves the last one alone.
// `dotf converge` and `dotf update` both run through here.
func runConverge(w io.Writer, repo string, plan bool, only []string) error {
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
	printConvergeReport(w, rep)
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
	_, _ = fmt.Fprintf(w, "report: %s\n", path)
	return err
}

// convergeOptions wires the registry's side effects; tests replace it.
var convergeOptions = func() converge.Options {
	home := env.Home()
	run, stdout := tools.HomeRunners(home)
	dest := filepath.Join(home, ".local", "bin")
	return converge.Options{
		RunHarnessDeploy: converge.CompileHarnessDeploy,
		MiseRun:          run,
		MiseStdout:       stdout,
		// Install progress goes to stderr, apart from the report on stdout.
		ToolsCatalog:  &tools.Installer{Dest: dest, Out: os.Stderr},
		ToolsBinDirs:  []string{tools.MiseShimsDir(home, runtime.GOOS, os.Getenv), dest},
		GitRun:        gitconfig.ExecRunner,
		GitStoredAuth: gitconfig.StoredLoginRunner,
		CloneURL:      os.Getenv("DOTFILES_REPO"),
		RenderConfigs: strictDeployRenderer,
		ResolvePath:   env.ResolvePath,
		Launchctl:     env.ExecLaunchctl,
		UID:           os.Getuid(),
		RunSetup:      converge.ExecSetup,
		DotfPath:      harness.ResolveDotfPath,
	}
}

// convergeCheckout is the checkout a run converges from when --repo names
// none: the dotfiles checkout the working directory is in (another project's
// repository does not count), else the declared location
// (DOTFILES_REPO_DIR, machine.json, the contract), else the default. The last
// two may not exist yet: the checkout step clones them, which is how a machine
// from zero gets one.
func convergeCheckout(home string) string {
	if r := env.RepoDir(); r != "" && env.IsDotfilesCheckout(r) {
		return r
	}
	if r := env.ResolvePath("DOTFILES_REPO_DIR"); r != "" {
		return r
	}
	return env.DefaultCheckoutDir(home)
}

var convergeTag = map[converge.Status]string{
	converge.StatusOK:      "[ OK ]",
	converge.StatusChange:  "[CHANGE]",
	converge.StatusSkipped: "[SKIP]",
	converge.StatusFailed:  "[FAIL]",
	converge.StatusOpaque:  "[OPAQUE]",
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
	_, _ = fmt.Fprintf(w, "%d %s, %d ok, %d opaque, %d skipped, %d failed\n",
		n[converge.StatusChange], changed, n[converge.StatusOK], n[converge.StatusOpaque], n[converge.StatusSkipped], n[converge.StatusFailed])
}
