package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// errHarnessDrift is the exit of a check that found drift; its lines are
// already printed.
var errHarnessDrift = errors.New("harness drift detected")

// newHarnessCheckCmd is the offline drift gate: the Go form of
// compile-harness.sh --check, which ci.yml and dotf doctor move onto once a
// release carries it (CLI-026).
func newHarnessCheckCmd() *cobra.Command {
	var repoRoot string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check the checkout's harness regions and records for drift, offline",
		Long: "check reads only the checkout. Every enforced id in harness/manifest.json must\n" +
			"reach every target file and the doctrine payload, or carry an opt_out reason\n" +
			"for it. Each target's GENERATED region must equal the enforced records it\n" +
			"injects. Each skill and agent record must open with frontmatter holding every\n" +
			"key its schema requires. Any failure is printed on stderr and exits non-zero.",
		Example:      "  dotf harness check\n  dotf harness check --repo-root /path/to/checkout",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if repoRoot == "" {
				repoRoot = env.RepoDir()
			}
			if repoRoot == "" {
				return fmt.Errorf("cannot locate the dotfiles checkout — pass --repo-root, set DOTFILES_REPO_DIR, or run from inside it")
			}
			report, err := harness.CheckDrift(repoRoot)
			if err != nil {
				return err
			}
			return reportCheck(cmd.OutOrStdout(), cmd.ErrOrStderr(), report)
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/manifest.json (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	return cmd
}

// reportCheck prints each line of the report, problems on stderr, and the
// verdict last.
func reportCheck(w, ew io.Writer, report []harness.CheckLine) error {
	for _, l := range report {
		dst := w
		if l.Problem {
			dst = ew
		}
		_, _ = fmt.Fprintln(dst, l.Text)
		for _, d := range l.Detail {
			_, _ = fmt.Fprintln(dst, d)
		}
	}
	if harness.HasDrift(report) {
		_, _ = fmt.Fprintln(ew, "[check] FAIL: harness drift detected")
		return errHarnessDrift
	}
	_, _ = fmt.Fprintln(w, "[check] OK: no harness drift")
	return nil
}
