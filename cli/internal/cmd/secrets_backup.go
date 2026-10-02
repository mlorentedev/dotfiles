package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/mlorentedev/dotfiles/cli/internal/shellsafe"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Backup seams as overridable vars so command tests inject fakes (no bw, no age key, no
// network) — the same pattern as ageDecryptor/bwReader in secrets.go. Production wires the
// real shell-outs (BWExport, AgeEncrypt, AgeRecipient) and the checkout-resolving dest.
var (
	bwExporter secrets.BWExporter = secrets.BWExport{}
	// bwSessionExporter and bwUnlock are the #1008 seams: the exporter bound to a
	// session the command acquired, and the unlock that acquires it.
	bwSessionExporter                     = func(session string) secrets.BWExporter { return secrets.BWExport{Session: session} }
	bwUnlock                              = secrets.BWUnlockSession
	ageEncryptor      secrets.Encryptor   = secrets.AgeEncrypt
	ageRecipient      secrets.RecipientFn = secrets.AgeRecipient
	repoSensitiveDir                      = env.RepoSensitiveDir
)

// newSecretsBackupCmd is `dotf secrets backup`: the disaster-recovery escrow of ADR-028
// §5. It exports the entire Bitwarden vault, encrypts it to the operator's own age
// recipient, and writes the verified ciphertext to sensitive/dr/ in the checkout — the
// account-independent recovery floor (recover with the offline age key + a repo clone,
// no Bitwarden account needed). The command is the automatable unit a scheduler invokes.
func newSecretsBackupCmd() *cobra.Command {
	var out string
	c := &cobra.Command{
		Use:   "backup",
		Short: "Escrow the whole Bitwarden vault, age-encrypted, to sensitive/dr",
		Long: "backup runs the disaster-recovery escrow: `bw sync` + `bw export` (the entire\n" +
			"vault — keys, tokens, TOTP seeds) piped in memory into age, encrypted to your own\n" +
			"recipient (`age-keygen -y` of your identity), and written atomically (0600) to\n" +
			"sensitive/dr/bitwarden-export.age in the dotfiles checkout. The plaintext is never\n" +
			"written to disk. The artifact is decrypted back and verified to round-trip before\n" +
			"the command succeeds; a corrupt escrow is removed, never committed. Recover with\n" +
			"the offline age key + a repo clone (docs/runbooks/guide-secrets-governance.md).\n\n" +
			"bw serve has no export route, so the export needs the bw CLI's own session. On a\n" +
			"terminal with the CLI locked, backup prompts once for the master password and\n" +
			"keeps the session in memory for its own bw children only. Without a terminal\n" +
			"(a scheduler), prefix the run with BW_SESSION=\"$(bw unlock --raw)\".",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			destDir := out
			if destDir == "" {
				dir, err := repoSensitiveDir()
				if err != nil {
					return fmt.Errorf("backup: %w", err)
				}
				destDir = filepath.Join(dir, "dr")
			}
			cfg := secrets.BackupConfig{
				Exporter:  bwExporter,
				Recipient: ageRecipient,
				Encrypt:   ageEncryptor,
				Decrypt:   ageDecryptor, // nil in prod → Backup falls back to AgeDecrypt
				KeyPath:   ageKeyPath(),
				DestDir:   destDir,
			}
			path, manifestWarn, err := secrets.Backup(cfg)
			if errors.Is(err, secrets.ErrBWVaultLocked) && stdinIsTerminal() {
				session, uerr := promptBWSession(cmd)
				if uerr != nil {
					return uerr
				}
				cfg.Exporter = bwSessionExporter(session)
				path, manifestWarn, err = secrets.Backup(cfg)
			}
			if errors.Is(err, secrets.ErrBWVaultLocked) {
				return fmt.Errorf("%w\nRun:\n    BW_SESSION=\"$(bw unlock --raw)\" %s", err, rerunLine(cmd))
			}
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "escrow written and verified: %s\n", path)
			// The escrow is what recovers the account; the manifest is bookkeeping
			// about it. Exiting non-zero here would tell a script the DR backup
			// failed when it succeeded — a message and an exit code disagreeing,
			// which is the defect class this repository spends its time on.
			if manifestWarn != "" {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", manifestWarn)
			}
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "destination dir for the escrow (default: <checkout>/sensitive/dr)")
	return c
}

// promptBWSession acquires the bw CLI session the escrow needs (#1008): one
// hidden prompt, as `dotf secrets unlock` does, and the session kept in memory
// for this process's bw children. bw serve has no export route, so the daemon
// that unlock starts cannot serve this command.
func promptBWSession(cmd *cobra.Command) (string, error) {
	_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Bitwarden master password (the escrow exports through the bw CLI): ")
	pw, err := readPassword()
	_, _ = fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	defer scrubBytes(pw)
	return bwUnlock(pw)
}

// rerunLine renders the invocation that failed as a line to paste: the command path
// and every flag the operator set, each value shell-quoted. A remedy is copied
// verbatim, so it has to reproduce the command rather than a default one; a fixed
// line dropped --out and put an escrow in the wrong checkout (#1647).
func rerunLine(cmd *cobra.Command) string {
	parts := []string{cmd.CommandPath()}
	cmd.Flags().Visit(func(f *pflag.Flag) {
		values := []string{f.Value.String()}
		if s, ok := f.Value.(pflag.SliceValue); ok {
			values = s.GetSlice() // String() renders "[a,b]", which does not parse back
		}
		for _, v := range values {
			parts = append(parts, "--"+f.Name+"="+shellsafe.Bash(v))
		}
	})
	return strings.Join(parts, " ")
}
