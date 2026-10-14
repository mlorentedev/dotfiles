package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// newSecretsUSBCmd is `dotf secrets usb`: the offline DR copy of ADR-033 D2, a
// VeraCrypt stick holding the age identity, the escrow and its manifest (#1770).
// Mounting the volume stays the owner's step: it asks for the volume password.
func newSecretsUSBCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "usb",
		Short: "Refresh and verify the offline DR copy on a mounted VeraCrypt stick",
	}
	c.AddCommand(newSecretsUSBVerifyCmd(), newSecretsUSBBackupCmd())
	return c
}

func newSecretsUSBVerifyCmd() *cobra.Command {
	var mount string
	c := &cobra.Command{
		Use:   "verify",
		Short: "Prove the stick restores this vault, and record the drill",
		Long: "verify checks the mounted stick by consequence, in memory, printing no key\n" +
			"material and no plaintext: its identity has this machine's recipient, it opens\n" +
			"the repo's escrow to exactly what the manifest describes, its own escrow copy\n" +
			"does the same and is the repo's byte for byte, and nothing else is on the\n" +
			"volume. On a pass it writes sensitive/dr/" + secrets.DrillRecordFileName + ", which\n" +
			"`dotf doctor` reads on every machine; commit it.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := usbConfig(mount)
			if err != nil {
				return err
			}
			rep, err := secrets.VerifyUSB(cfg)
			if err != nil {
				return err
			}
			printUSBReport(cmd, cfg, rep)
			return nil
		},
	}
	c.Flags().StringVar(&mount, "mount", "", "where the VeraCrypt volume is mounted (required)")
	_ = c.MarkFlagRequired("mount")
	return c
}

func newSecretsUSBBackupCmd() *cobra.Command {
	var mount string
	var opts secrets.USBBackupOptions
	c := &cobra.Command{
		Use:   "backup",
		Short: "Copy the escrow and its manifest to the stick, then verify it",
		Long: "backup copies sensitive/dr's escrow and manifest onto the mounted stick, then\n" +
			"runs `usb verify`. Run it after every `dotf secrets backup`.\n\n" +
			"--mount must be a mounted volume, never a directory of the disk it sits on: a\n" +
			"mount point that was never mounted would otherwise receive the age identity in\n" +
			"the clear. The stick must already hold this machine's identity; a new, empty\n" +
			"stick gets it with --init-identity. A stick that holds a different identity is\n" +
			"refused before anything is written. Files outside the declared payload fail\n" +
			"the verification; --prune removes them first.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := usbConfig(mount)
			if err != nil {
				return err
			}
			rep, err := secrets.BackupUSB(cfg, opts)
			if err != nil {
				return err
			}
			printUSBReport(cmd, cfg, rep)
			return nil
		},
	}
	c.Flags().StringVar(&mount, "mount", "", "where the VeraCrypt volume is mounted (required)")
	c.Flags().BoolVar(&opts.Prune, "prune", false, "remove files outside the declared payload")
	c.Flags().BoolVar(&opts.InitIdentity, "init-identity", false, "install this machine's age identity on an empty stick")
	_ = c.MarkFlagRequired("mount")
	return c
}

// usbHost names the machine in the drill record; a seam so a test pins it.
var usbHost = os.Hostname

func usbConfig(mount string) (secrets.USBConfig, error) {
	dir, err := repoSensitiveDir()
	if err != nil {
		return secrets.USBConfig{}, fmt.Errorf("usb: %w", err)
	}
	host, err := usbHost()
	if err != nil {
		return secrets.USBConfig{}, fmt.Errorf("usb: hostname: %w", err)
	}
	return secrets.USBConfig{
		Mount:     mount,
		KeyPath:   ageKeyPath(),
		RepoDR:    filepath.Join(dir, "dr"),
		Host:      host,
		Recipient: ageRecipient,
		Decrypt:   ageDecryptor, // nil in prod → AgeDecrypt
	}, nil
}

func printUSBReport(cmd *cobra.Command, cfg secrets.USBConfig, rep secrets.USBReport) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "recipient: %s (this machine's)\n", rep.Recipient)
	_, _ = fmt.Fprintf(out, "repo escrow with the stick's identity: %d items, as its manifest says\n", rep.RepoItems)
	_, _ = fmt.Fprintf(out, "stick escrow: %d items, the repo's byte for byte\n", rep.USBItems)
	_, _ = fmt.Fprintf(out, "drill recorded: %s (commit it)\n", filepath.Join(cfg.RepoDR, secrets.DrillRecordFileName))
}
