package cmd

import (
	"fmt"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// bwDaemonAddr is the bw serve lifecycle seam unlock/lock/doctor share,
// overridable so their tests never spawn a real bw process or touch a real
// terminal — the daemon analog of bwReader/bwWriter. The daemon's trace (log
// + pid, #1315) lands under the deploy dir resolved the same way doctor
// resolves cfg.DotfilesDir, so the writer here and the reader there agree.
var bwDaemonAddr = func() *secrets.BWServeDaemon {
	return &secrets.BWServeDaemon{
		Client: secrets.BWServeClient{},
		State:  secrets.NewBWServeState(env.DotfilesDir(env.Home())),
	}
}

// newSecretsUnlockCmd is CLI-024-secrets-bw-serve's AC1/AC6: unlock Bitwarden
// once for every `dotf secrets` call on this machine, never as an ambient
// BW_SESSION. The password crosses exactly one hidden prompt and one local
// POST; it is never exported, written to disk, or logged.
func newSecretsUnlockCmd() *cobra.Command {
	var startTimeout time.Duration
	c := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock Bitwarden once for this machine (bw serve — no ambient BW_SESSION)",
		Long: "unlock prompts for the Bitwarden master password (hidden, like sudo), starts\n" +
			"a dotf-managed bw serve daemon if none is running yet, and unlocks it. The\n" +
			"password is held only for the duration of this command — never exported as\n" +
			"BW_SESSION, never written to disk, never logged. Every `dotf secrets` read\n" +
			"(run/show/verify) then prefers the daemon automatically until `dotf secrets\n" +
			"lock` or the machine restarts — no per-session re-unlock, no per-agent unlock.\n" +
			"Idempotent: running unlock again while already unlocked is a no-op.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d := bwDaemonAddr()
			if err := d.Start(startTimeout); err != nil {
				return fmt.Errorf("start bw serve: %w", err)
			}
			st, err := d.Status()
			if err != nil {
				return err
			}
			if st == "unlocked" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "already unlocked (%s)\n", d.Trace())
				return nil
			}
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Bitwarden master password: ")
			pw, err := readPassword()
			_, _ = fmt.Fprintln(cmd.ErrOrStderr())
			if err != nil {
				return fmt.Errorf("read password: %w", err)
			}
			defer scrubBytes(pw)
			if err := d.Unlock(string(pw)); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "unlocked (%s)\n", d.Trace())
			return nil
		},
	}
	c.Flags().DurationVar(&startTimeout, "start-timeout", 15*time.Second,
		"how long to wait for the daemon to become reachable after starting it")
	return c
}

// newSecretsLockCmd re-locks the daemon (AC6) without stopping the process —
// the next unlock only re-POSTs the password, not a fresh Node cold start.
func newSecretsLockCmd() *cobra.Command {
	c := &cobra.Command{
		Use:          "lock",
		Short:        "Lock the bw serve daemon (keeps it running, re-locks the vault)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d := bwDaemonAddr()
			if !d.Running() {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no daemon running")
				return nil
			}
			if err := d.Lock(); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "locked (%s)\n", d.Trace())
			return nil
		},
	}
	return c
}

// scrubBytes best-effort zeroes a password buffer after use. Go's garbage
// collector may have already copied the underlying bytes elsewhere (string
// conversion, buffer growth) before this runs — a real guarantee would need
// a locked, non-swappable allocation this package does not have. It narrows
// the window; it does not close it, and the doc comments say so rather than
// overclaiming.
func scrubBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
