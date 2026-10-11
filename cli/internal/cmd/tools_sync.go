package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// toolsSyncRunner runs mise and the installed tools; nil means the real ones,
// run from HOME, with `mise which` read from stdout only. Tests replace it
// (lesson 335).
var toolsSyncRunner tools.Runner

func newToolsSyncCmd() *cobra.Command {
	var (
		dryRun   bool
		versions string
	)
	c := &cobra.Command{
		Use:   "sync",
		Short: "Install the pinned CLIs through mise, from versions.conf",
		Long: "sync renders the versions.conf pins marked \"" + tools.MiseMarker + "\" into\n" +
			"<mise config dir>/conf.d/dotfiles.toml, a file it owns end to end, and runs\n" +
			"`mise install`. Every tool must then run through `mise which` and report a\n" +
			"version at or above its pin, or sync fails naming it. The rest of the mise\n" +
			"config (a hand-written config.toml) is never touched.\n\n" +
			"A second run with nothing to do writes nothing and installs nothing.\n" +
			"--dry-run prints the config and the tools it would install, and changes\n" +
			"nothing.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if versions == "" {
				versions = env.ResolveVersionsPath()
			}
			raw, err := os.ReadFile(versions) //nolint:gosec // the checkout's or the deploy dir's versions.conf
			if err != nil {
				return fmt.Errorf("versions.conf: %w — run from the repo, pass --versions, or set DOTFILES_DIR", err)
			}
			all, err := tools.ParseMisePins(raw)
			if err != nil {
				return err
			}
			pins := all.Tools
			run, stdout := syncRunners()
			s := tools.MiseSync{ConfigDir: tools.MiseConfigDir(env.Home(), os.Getenv), Run: run, Stdout: stdout, PythonPackages: all.PythonPackages, Latest: all.Latest, CheckUpgrades: true}
			if dryRun {
				return printSyncPlan(cmd.OutOrStdout(), s, pins)
			}
			p, err := s.Apply(pins)
			if err != nil {
				return err
			}
			printSyncResult(cmd.OutOrStdout(), s, pins, p)
			return nil
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print what sync would do, and change nothing")
	c.Flags().StringVar(&versions, "versions", "", "versions.conf to read (default: the checkout's, else the deploy dir's)")
	return c
}

// syncRunners returns the test runner when one is set, else tools.HomeRunners.
func syncRunners() (run, stdout tools.Runner) {
	if toolsSyncRunner != nil {
		return toolsSyncRunner, nil
	}
	return tools.HomeRunners(env.Home())
}

func printSyncPlan(w io.Writer, s tools.MiseSync, pins []tools.MiseTool) error {
	p, err := s.Plan(pins)
	if err != nil {
		return err
	}
	verb := "unchanged"
	if p.ConfigChanged {
		verb = "would write"
	}
	_, _ = fmt.Fprintf(w, "%s %s:\n", verb, s.ConfigPath())
	for _, t := range pins {
		if slices.Contains(s.Latest, t.Name) {
			_, _ = fmt.Fprintf(w, "  %s latest (at least %s)\n", t.Name, t.Version)
			continue
		}
		_, _ = fmt.Fprintf(w, "  %s %s\n", t.Name, t.Version)
	}
	for _, pkg := range s.PythonPackages {
		_, _ = fmt.Fprintf(w, "  python package %s %s (%s)\n", pkg.Name, pkg.Version, s.PythonPackagesPath())
	}
	if len(p.Missing) == 0 && len(p.MissingPackages) == 0 && len(p.Outdated) == 0 {
		_, _ = fmt.Fprintf(w, "%d tool(s) at their pin; nothing to install\n", len(pins))
		return nil
	}
	if len(p.Missing) > 0 {
		_, _ = fmt.Fprintf(w, "to install: %s\n", strings.Join(p.Missing, ", "))
	}
	if len(p.Outdated) > 0 {
		_, _ = fmt.Fprintf(w, "to upgrade to their newest release: %s\n", strings.Join(p.Outdated, ", "))
	}
	if len(p.MissingPackages) > 0 {
		_, _ = fmt.Fprintf(w, "python packages to install: %s\n", strings.Join(p.MissingPackages, ", "))
	}
	return nil
}

func printSyncResult(w io.Writer, s tools.MiseSync, pins []tools.MiseTool, p tools.SyncPlan) {
	if !p.Pending() {
		_, _ = fmt.Fprintf(w, "%d tool(s) at their pin; nothing to do\n", len(pins))
		return
	}
	if p.ConfigChanged {
		_, _ = fmt.Fprintf(w, "wrote %s\n", s.ConfigPath())
	}
	if len(p.Missing) > 0 {
		_, _ = fmt.Fprintf(w, "installed: %s\n", strings.Join(p.Missing, ", "))
	}
	if len(p.Outdated) > 0 {
		_, _ = fmt.Fprintf(w, "upgraded to their newest release: %s\n", strings.Join(p.Outdated, ", "))
	}
	if len(p.MissingPackages) > 0 {
		_, _ = fmt.Fprintf(w, "installed python packages: %s\n", strings.Join(p.MissingPackages, ", "))
	}
	_, _ = fmt.Fprintf(w, "%d tool(s) at their pin\n", len(pins))
}
