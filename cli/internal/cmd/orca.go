package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/orca"
	"github.com/spf13/cobra"
)

func newOrcaCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "orca",
		Short: "Manage Orca ADE configuration, keybindings and baseline tuning",
		Long: "orca provides commands to export and tune Orca ADE configurations.\n\n" +
			"  dotf orca export        # Extract clean settings & keybindings to repo\n" +
			"  dotf orca tune          # Apply recommended baseline tuning to orca-data.json\n" +
			"  dotf orca tune --dry-run # Show planned tuning changes without writing",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	c.AddCommand(newOrcaExportCmd())
	c.AddCommand(newOrcaTuneCmd())
	return c
}

func newOrcaExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Extract keybindings and clean settings from Orca into dotfiles repo",
		Long: "export reads ~/.orca/keybindings.json and ~/.config/orca/orca-data.json,\n" +
			"extracts non-ephemeral settings, and writes formatted JSON files into\n" +
			"ai/orca/ in the dotfiles checkout.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOrcaExport(cmd.OutOrStdout())
		},
	}
}

func runOrcaExport(w io.Writer) error {
	repoRoot := env.RepoDir()
	if repoRoot == "" {
		return fmt.Errorf("cannot locate dotfiles checkout — set DOTFILES_REPO_DIR or run from inside it")
	}
	home := env.Home()
	orcaUserDataDir := filepath.Join(home, ".config", "orca")
	orcaHomeDir := filepath.Join(home, ".orca")

	rep, err := orca.Export(repoRoot, orcaUserDataDir, orcaHomeDir)
	if err != nil {
		return err
	}

	if rep.KeybindingsCopied {
		_, _ = fmt.Fprintf(w, "exported  keybindings  %s\n", rep.RepoKeybindings)
	} else {
		_, _ = fmt.Fprintf(w, "skipped   keybindings  (not found or invalid in %s)\n", orcaHomeDir)
	}

	if rep.SettingsExported {
		_, _ = fmt.Fprintf(w, "exported  settings     %s (%d keys)\n", rep.RepoSettings, rep.SettingsCount)
	} else {
		_, _ = fmt.Fprintf(w, "skipped   settings     (orca-data.json not found in %s)\n", orcaUserDataDir)
	}
	return nil
}

func newOrcaTuneCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "tune",
		Short: "Apply recommended baseline tuning to orca-data.json",
		Long: "tune checks ~/.config/orca/orca-data.json and ensures recommended baseline settings\n" +
			"(agent hibernation, base ref refresh, telemetry opt-out) are applied.\n" +
			"It guards against running Orca processes and creates timestamped backups before writing.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOrcaTune(cmd.OutOrStdout(), dryRun)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report what would change without modifying orca-data.json")
	return c
}

func runOrcaTune(w io.Writer, dryRun bool) error {
	home := env.Home()
	orcaUserDataDir := filepath.Join(home, ".config", "orca")

	rep, err := orca.Tune(orcaUserDataDir, dryRun, orca.DefaultProcessChecker)
	if err != nil {
		return err
	}

	if len(rep.Changes) == 0 {
		_, _ = fmt.Fprintln(w, "in sync   orca-data.json already matches the tuned baseline")
		return nil
	}

	for _, change := range rep.Changes {
		if dryRun {
			_, _ = fmt.Fprintf(w, "would tune settings.%s: %v -> %v\n", change.Key, change.Old, change.New)
		} else {
			_, _ = fmt.Fprintf(w, "tuned      settings.%s: %v -> %v\n", change.Key, change.Old, change.New)
		}
	}

	if !dryRun && rep.BackupPath != "" {
		_, _ = fmt.Fprintf(w, "backup     created at %s\n", rep.BackupPath)
	}
	return nil
}
