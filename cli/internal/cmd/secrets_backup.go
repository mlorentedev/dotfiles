package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// Backup seams as overridable vars so command tests inject fakes (no bw, no age key, no
// network) — the same pattern as ageDecryptor/bwReader in secrets.go. Production wires the
// real shell-outs (BWExport, AgeEncrypt, AgeRecipient) and the checkout-resolving dest.
var (
	bwExporter       secrets.BWExporter  = secrets.BWExport{}
	ageEncryptor     secrets.Encryptor   = secrets.AgeEncrypt
	ageRecipient     secrets.RecipientFn = secrets.AgeRecipient
	repoSensitiveDir                     = env.RepoSensitiveDir
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
		Short: "Escrow the whole Bitwarden vault, age-encrypted, to sensitive/dr (ADR-028 §5)",
		Long: "backup runs the disaster-recovery escrow: `bw sync` + `bw export` (the entire\n" +
			"vault — keys, tokens, TOTP seeds) piped in memory into age, encrypted to your own\n" +
			"recipient (`age-keygen -y` of your identity), and written atomically (0600) to\n" +
			"sensitive/dr/bitwarden-export.age in the dotfiles checkout. The plaintext is never\n" +
			"written to disk. The artifact is decrypted back and verified to round-trip before\n" +
			"the command succeeds; a corrupt escrow is removed, never committed. Recover with\n" +
			"the offline age key + a repo clone (docs/runbooks/guide-secrets-governance.md).",
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
			path, manifestWarn, err := secrets.Backup(secrets.BackupConfig{
				Exporter:  bwExporter,
				Recipient: ageRecipient,
				Encrypt:   ageEncryptor,
				Decrypt:   ageDecryptor, // nil in prod → Backup falls back to AgeDecrypt
				KeyPath:   ageKeyPath(),
				DestDir:   destDir,
			})
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
