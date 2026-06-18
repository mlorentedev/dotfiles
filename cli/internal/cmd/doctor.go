package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/doctor"
)

// errChecksFailed is returned (silently) when at least one diagnostic check
// failed. main.go maps any non-nil error to exit status 1 — the healthcheck.sh
// / doctor.sh exit contract — and SilenceErrors keeps the human-readable report
// the only output, with no spurious "Error:" line tacked on.
var errChecksFailed = errors.New("doctor: one or more checks failed")

func newDoctorCmd() *cobra.Command {
	var (
		fix     bool
		verbose bool
		quick   bool
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Post-setup diagnostics — the consolidated healthcheck + doctor twins (ADR-021)",
		Long: `doctor runs the post-setup diagnostic sweep: core tools, versioned tool
paths, version-pin match, key symlinks, environment variables + PATH, optional
tools, the env-contract, vault presence, secrets integrity, PAT expiry, tmux,
opencode/pi, harness drift, and the Antigravity CLI.

It is the Go consolidation of scripts/healthcheck.sh (the 12-section sweep) and
scripts/doctor.sh (the env-contract verifier), reading versions.conf and
env-contract.json natively (no jq). Exit 0 when every check passes, 1 when any
fails; advisory WARN/SKIP/INFO never fail the run.

With --fix it reports the exact shell-profile lines for any missing env default
(a subprocess cannot export into your shell) and invokes the known heal scripts
(claude-mem).

With --quick it runs ONLY the env-contract sweep (env vars, PATH, required
binaries) and skips the heavy sections — chiefly the ~2.8s compile-harness drift
gate. This is the fast subset the SessionStart hook wires in; --quick is
report-only (it ignores --fix).`,
		Example:       "  dotf doctor\n  dotf doctor --fix\n  dotf doctor --quick\n  dotf doctor --verbose",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			code, err := doctor.Run(doctor.Options{
				Out:     cmd.OutOrStdout(),
				Fix:     fix,
				Verbose: verbose,
				Quick:   quick,
			})
			if err != nil {
				cmd.PrintErrln("doctor:", err)
				return err
			}
			if code != 0 {
				return errChecksFailed
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "report safe env defaults to persist and run known heals (claude-mem)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "list passing checks too (default summarises them per section)")
	cmd.Flags().BoolVar(&quick, "quick", false, "env-contract sweep only — fast, no compile-harness gate (for the SessionStart hook)")
	return cmd
}
