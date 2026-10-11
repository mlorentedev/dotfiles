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
		scope   string
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Post-setup diagnostics for this machine's tools, configs and secrets",
		Long: `doctor runs the post-setup diagnostic sweep: core tools, versioned tool
paths, version-pin match, key symlinks, environment variables + PATH, optional
tools, the env-contract, vault presence, secrets integrity, PAT expiry, tmux,
opencode/pi, harness drift, and the Antigravity CLI.

It is the Go consolidation of scripts/healthcheck.sh (the 12-section sweep) and
scripts/doctor.sh (the env-contract verifier), reading versions.conf and
env-contract.json natively (no jq). Exit 0 when every check passes, 1 when any
fails; advisory WARN/SKIP/INFO never fail the run.

With --fix it reports the exact shell-profile lines for any missing env default
(a subprocess cannot export into your shell) and wires any safe repaired state
(e.g. core.hooksPath).

With --quick it runs ONLY the env-contract sweep (env vars, PATH, required
binaries) and skips the heavy sections — chiefly the ~2.8s compile-harness drift
gate. This is the fast subset the SessionStart hook wires in; --quick is
report-only (it ignores --fix).

With --scope machine it checks what converge produces and skips the identity
checks: the age key, Bitwarden, the gh login and the knowledge vault, which are
the owner's to restore (docs/runbooks/guide-new-machine.md). Each one skipped
prints a SKIP naming its restore step, and a skip never fails the run. It is
the scope for a fresh machine before its identity is restored, and for CI.`,
		Example:       "  dotf doctor\n  dotf doctor --fix\n  dotf doctor --quick\n  dotf doctor --verbose\n  dotf doctor --scope machine",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sc, err := doctor.ParseScope(scope)
			if err != nil {
				return err
			}
			code, err := doctor.Run(doctor.Options{
				Out:     cmd.OutOrStdout(),
				Fix:     fix,
				Verbose: verbose,
				Quick:   quick,
				Scope:   sc,
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

	cmd.Flags().BoolVar(&fix, "fix", false, "report safe env defaults to persist and wire safe repaired state (e.g. core.hooksPath)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "list passing checks too (default summarises them per section)")
	cmd.Flags().BoolVar(&quick, "quick", false, "env-contract sweep only — fast, no compile-harness gate (for the SessionStart hook)")
	cmd.Flags().StringVar(&scope, "scope", "all", "all, or machine: skip the identity checks (age key, Bitwarden, gh login, vault)")
	return cmd
}
