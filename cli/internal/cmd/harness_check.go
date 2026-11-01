package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// errHarnessDrift is the exit of a check that found drift; its lines are
// already printed.
var errHarnessDrift = errors.New("harness drift detected")

// newHarnessCheckCmd is the offline drift gate: the Go form of
// compile-harness.sh --check, which ci.yml and dotf doctor run (CLI-026).
func newHarnessCheckCmd() *cobra.Command {
	var repoRoot, vault string
	var againstVault bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check the checkout's harness regions and records for drift, offline",
		Long: "check reads only the checkout. Every enforced id in harness/manifest.json must\n" +
			"reach every target file and the doctrine payload, or carry an opt_out reason\n" +
			"for it. Each target's GENERATED region must equal the enforced records it\n" +
			"injects. Each skill and agent record must open with frontmatter holding every\n" +
			"key its schema requires. Any failure is printed on stderr and exits non-zero.\n\n" +
			"--against-vault also compares each record with the vault source it comes from.\n" +
			"A skill or agent record whose generated_sha is behind its source is stale; one\n" +
			"whose sha matches but whose files differ from what a refresh renders was edited\n" +
			"in the checkout, and the next refresh reverts that edit. Each enforced record\n" +
			"is held to its vault section. A source that is gone, or a record with no\n" +
			"provenance, is drift too.\n" +
			"It needs a vault, so it is not part of the offline check CI runs.",
		Example:      "  dotf harness check\n  dotf harness check --repo-root /path/to/checkout\n  dotf harness check --against-vault",
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
			if againstVault {
				if vault == "" {
					vault = env.ResolvePath("VAULT_PATH")
				}
				if fi, err := os.Stat(vault); vault == "" || err != nil || !fi.IsDir() {
					return fmt.Errorf("--against-vault: no vault at %q — pass --vault or set VAULT_PATH", vault)
				}
				report = append(report, harness.CompareWithVault(repoRoot, vault).Lines()...)
			}
			return reportCheck(cmd.OutOrStdout(), cmd.ErrOrStderr(), report)
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "checkout holding harness/manifest.json (default: DOTFILES_REPO_DIR or the cwd walk-up)")
	cmd.Flags().BoolVar(&againstVault, "against-vault", false, "also compare each record with what a refresh renders from the vault")
	cmd.Flags().StringVar(&vault, "vault", "", "vault for --against-vault (default: $VAULT_PATH, then machine.json, then the env-contract default)")
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
