package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/vault"
)

// newVaultMaintainCmd builds `dotf vault maintain`, the Go port of
// scripts/vault-maintenance-weekly.{sh,ps1} (CLI-021 / #490, increment 3).
// Built BESIDE the twins: the crontab entry at setup-linux.sh:1605 and the Task
// Scheduler entry at setup-windows.ps1:2185 still point at the scripts, and
// repointing them is CLI-023 (#492).
//
// No flags, matching the twins, which take none — this is a scheduled job whose
// only caller is cron. `--vault` and `--verbose` are `vault health`'s, and a
// weekly log wants neither.
func newVaultMaintainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "maintain",
		Short: "Run the weekly vault maintenance pass (crystallize + health) and log it",
		Long: `maintain runs the unattended weekly maintenance pass:

  1. dotf vault crystallize --all  — MEMORY.md dates across every project
  2. dotf vault health             — the 7-section vault report

Both run best-effort: neither failing stops the other, and neither stops the log
being written. The log lands at ~/.local/share/vault-maintenance/latest.log
(%LOCALAPPDATA%\vault-maintenance\latest.log on Windows) and a best-effort
desktop notification reports how many issue lines it holds.

Exit status is 0 whenever the run did its job. Health findings are NOT failures
— they are reported on stdout and in the log, so a scheduled run does not mail
you every week the Obsidian GUI happened to be closed. The only error is being
unable to write the log.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			return vault.RunMaintain(c.OutOrStdout(), vault.MaintainOptions{
				Home:    home,
				Today:   time.Now().Format("2006-01-02"),
				LogFile: vault.DefaultLogFile(home),
				Health:  healthOptions("", false),
				Notify:  vault.NotifyDesktop,
			})
		},
	}
}
