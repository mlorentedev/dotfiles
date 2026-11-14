package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/lane"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// newLaneDoctorCmd answers "is this repository's lane configuration
// coherent?": the marker's content, and the marker against the lanes on this
// machine, in both directions. It repairs nothing; every finding names the
// command that does.
func newLaneDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "doctor",
		Short:        "Report drift between .dotf/lanes.json and the lanes on this machine",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		// The findings are the diagnostic: cobra's Error: line would only
		// repeat their count. Failures to audit at all are printed here.
		SilenceErrors: true,
		RunE:          func(c *cobra.Command, _ []string) error { return runLaneDoctor(c) },
	}
}

// runLaneDoctor resolves the main checkout, audits it and prints the findings;
// exit 2 when the audit could not run, 1 when it found something. Warnings
// are printed first and do not count as findings.
func runLaneDoctor(c *cobra.Command) error {
	repoRoot, err := worktree.ResolveMainRepoRoot(".")
	if err != nil {
		_, _ = fmt.Fprintf(c.ErrOrStderr(), "lane doctor: %v\n", err)
		return withExitCode(2, err)
	}
	a, err := lane.AuditLanes(repoRoot, env.ResolvePath("WORKTREE_ROOT"))
	if err != nil {
		_, _ = fmt.Fprintf(c.ErrOrStderr(), "lane doctor: %v\n", err)
		return withExitCode(2, err)
	}
	if a.Managed {
		a.Warnings = append(a.Warnings, lane.GuardDrift(repoRoot, env.DotfilesDir(env.Home()))...)
	}
	out := c.OutOrStdout()
	// Warnings never change the exit code: they are not drift, only what the
	// guard will refuse once the marker arms it.
	for _, w := range a.Warnings {
		_, _ = fmt.Fprintln(out, w.Warning())
	}
	switch {
	case !a.Managed:
		_, _ = fmt.Fprintf(out, "lane doctor: %s is not lane-managed (no %s)\n", repoRoot, lane.LanesMarkerFile)
		return nil
	case !a.Initialised && len(a.Findings) == 0:
		_, _ = fmt.Fprintf(out, "lane doctor: %s declares lanes that are not initialised on this machine; dotf lane init provisions them\n", repoRoot)
		return nil
	case len(a.Findings) == 0:
		return nil
	}
	for _, f := range a.Findings {
		_, _ = fmt.Fprintln(out, f.String())
	}
	return withExitCode(1, fmt.Errorf("lane doctor: %d finding(s) in %s", len(a.Findings), repoRoot))
}
