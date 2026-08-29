package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/orca"
	"github.com/spf13/cobra"
)

func newOrcaCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "orca",
		Short: "Manage Orca ADE configuration, keybindings and baseline tuning",
		Long: "orca provides commands to export and tune Orca ADE configurations.\n\n" +
			"  dotf orca export            # Extract clean settings & keybindings to repo\n" +
			"  dotf orca tune              # Apply recommended baseline tuning to orca-data.json\n" +
			"  dotf orca tune --dry-run    # Show planned tuning changes without writing\n" +
			"  dotf orca tune-hooks        # Repair Orca's generated Copilot hooks (DX-006)\n" +
			"  dotf orca tune-hooks --check # Report hook drift without writing",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	c.AddCommand(newOrcaExportCmd())
	c.AddCommand(newOrcaTuneCmd())
	c.AddCommand(newOrcaTuneHooksCmd())
	return c
}

// newOrcaTuneHooksCmd is CLI-062 (#1338): the DX-006 repair of Orca's
// generated Copilot hooks, ported from scripts/orca-hook-tune.ps1 so that
// setup, doctor --fix and a hand invocation share one implementation.
func newOrcaTuneHooksCmd() *cobra.Command {
	var (
		check      bool
		timeout    int
		hookConfig string
		hookScript string
	)
	c := &cobra.Command{
		Use:   "tune-hooks",
		Short: "Repair Orca's generated Copilot hooks: raise timeoutSec and swap the slow POST (DX-006)",
		Long: "tune-hooks fixes the two things Orca regenerates on every install or upgrade\n" +
			"that make every Copilot tool call fail with \"hook errored\" (DX-006, lesson 111):\n" +
			"  1. ~/.copilot/hooks/orca.json           every hook timeoutSec below --timeout-sec is raised\n" +
			"  2. ~/.orca/agent-hooks/copilot-hook.ps1 the Invoke-WebRequest POST becomes HttpWebRequest\n" +
			"Each file it changes is backed up beside itself first (<file>.bak.<stamp>) and written\n" +
			"atomically. Idempotent: a tuned pair is left alone. Missing files are nothing to do.\n" +
			"--check reports drift without writing and exits non-zero while any remains.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home := env.Home()
			if hookConfig == "" {
				hookConfig = filepath.Join(home, ".copilot", "hooks", "orca.json")
			}
			if hookScript == "" {
				hookScript = filepath.Join(home, ".orca", "agent-hooks", "copilot-hook.ps1")
			}
			return runOrcaTuneHooks(cmd.OutOrStdout(), hookConfig, hookScript, timeout, check)
		},
	}
	c.Flags().BoolVar(&check, "check", false, "report drift without writing; non-zero exit while any remains")
	c.Flags().IntVar(&timeout, "timeout-sec", orca.DefaultHookTimeout, "minimum hook timeoutSec to enforce in orca.json")
	c.Flags().StringVar(&hookConfig, "hook-config", "", "path to Orca's orca.json (default ~/.copilot/hooks/orca.json)")
	c.Flags().StringVar(&hookScript, "hook-script", "", "path to Orca's copilot-hook.ps1 (default ~/.orca/agent-hooks/copilot-hook.ps1)")
	return c
}

func runOrcaTuneHooks(w io.Writer, hookConfig, hookScript string, timeout int, check bool) error {
	rep, err := orca.TuneHooks(hookConfig, hookScript, timeout, check, time.Now)
	if err != nil {
		return err
	}
	if rep.Nothing() {
		_, _ = fmt.Fprintln(w, "nothing to do: Orca's Copilot hooks not found (Orca not installed for this user)")
		return nil
	}
	if check {
		if rep.ConfigExists {
			if rep.ConfigDrift {
				_, _ = fmt.Fprintf(w, "drift: %s has a hook timeoutSec < %d\n", hookConfig, timeout)
			} else {
				_, _ = fmt.Fprintf(w, "ok: orca.json hook timeouts >= %d\n", timeout)
			}
		}
		if rep.ScriptExists {
			if rep.ScriptDrift {
				_, _ = fmt.Fprintf(w, "drift: %s still uses Invoke-WebRequest\n", hookScript)
			} else {
				_, _ = fmt.Fprintln(w, "ok: copilot-hook.ps1 uses HttpWebRequest")
			}
		}
		if rep.Drift() {
			return fmt.Errorf("Orca's Copilot hooks need tuning — run `dotf orca tune-hooks` (DX-006)")
		}
		return nil
	}
	for _, bak := range rep.Backups {
		_, _ = fmt.Fprintf(w, "backup     %s\n", bak)
	}
	if rep.ScriptUnrecognised {
		_, _ = fmt.Fprintf(w, "unchanged  %s has Invoke-WebRequest but the POST line is unrecognised — review it by hand (DX-006)\n", hookScript)
	}
	if rep.Changed == 0 {
		_, _ = fmt.Fprintln(w, "in sync   Orca's Copilot hooks already tuned (DX-006)")
		return nil
	}
	_, _ = fmt.Fprintf(w, "tuned      %d fix(es) applied — restart the Copilot CLI session to pick up the new orca.json timeout\n", rep.Changed)
	return nil
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
