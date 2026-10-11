package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
			"  dotf orca tune-hooks        # Repair Orca's generated Copilot hooks\n" +
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
		Short: "Repair Orca's generated Copilot hooks: raise timeoutSec and swap the slow POST",
		Long: "tune-hooks fixes the two things Orca regenerates on every install or upgrade\n" +
			"that make every Copilot tool call fail with \"hook errored\":\n" +
			"  1. ~/.copilot/hooks/orca.json           every hook timeoutSec below --timeout-sec is raised\n" +
			"  2. ~/.orca/agent-hooks/copilot-hook.ps1 the Invoke-WebRequest POST becomes HttpWebRequest\n" +
			"Each file it changes is backed up beside itself first (<file>.bak.<stamp>) and written\n" +
			"atomically. Idempotent: a tuned pair is left alone. Missing files are nothing to do.\n" +
			"--check reports drift without writing and exits non-zero while any remains.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			defConfig, defScript := orcaHookPaths(env.Home())
			if hookConfig == "" {
				hookConfig = defConfig
			}
			if hookScript == "" {
				hookScript = defScript
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

// orcaHookPaths is where Orca writes its generated Copilot hooks for home.
func orcaHookPaths(home string) (config, script string) {
	return filepath.Join(home, ".copilot", "hooks", "orca.json"),
		filepath.Join(home, ".orca", "agent-hooks", "copilot-hook.ps1")
}

// deployOrcaHooks is the converge step `dotf deploy` runs after the declared
// configs (CLI-093, #1953). Orca regenerates its hooks on every install, so a
// setup that deploys configs has to re-tune them; doing it here reaches both
// setups through the one call they already make.
func deployOrcaHooks(w io.Writer, home string, dryRun bool) error {
	config, script := orcaHookPaths(home)
	rep, err := orca.TuneHooks(config, script, orca.DefaultHookTimeout, dryRun, time.Now)
	if err != nil {
		return fmt.Errorf("orca-hooks: %w", err)
	}
	switch {
	case rep.Nothing():
		deployRow(w, "skipped", "orca-hooks", "(Orca not installed)")
	case dryRun && rep.Drift():
		deployRow(w, "would tune", "orca-hooks", "%s", config)
	case rep.Changed == 0:
		deployRow(w, "in sync", "orca-hooks", "%s", config)
	default:
		for _, bak := range rep.Backups {
			deployRow(w, "backup", "orca-hooks", "%s", bak)
		}
		deployRow(w, "tuned", "orca-hooks", "%d fix(es) — restart the Copilot CLI session to pick them up", rep.Changed)
	}
	if rep.ScriptUnrecognised {
		deployRow(w, "unchanged", "orca-hooks", "%s has an unrecognised POST line — review it by hand", script)
	}
	return nil
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
		return reportOrcaHookCheck(w, rep, hookConfig, hookScript, timeout)
	}
	reportOrcaHookTune(w, rep, hookScript)
	return nil
}

// reportOrcaHookCheck prints what --check measured and fails on drift, which
// is what makes --check the gate.
func reportOrcaHookCheck(w io.Writer, rep *orca.HookTuneReport, hookConfig, hookScript string, timeout int) error {
	switch {
	case rep.ConfigExists && rep.ConfigDrift:
		_, _ = fmt.Fprintf(w, "drift: %s has a hook timeoutSec < %d\n", hookConfig, timeout)
	case rep.ConfigExists && !rep.TimeoutFloorApplies:
		_, _ = fmt.Fprintln(w, "ok: orca.json registers no PowerShell hook, so the timeoutSec floor does not apply")
	case rep.ConfigExists:
		_, _ = fmt.Fprintf(w, "ok: orca.json hook timeouts >= %d\n", timeout)
	}
	switch {
	case rep.ScriptExists && rep.ScriptDrift:
		_, _ = fmt.Fprintf(w, "drift: %s still uses Invoke-WebRequest\n", hookScript)
	case rep.ScriptExists:
		_, _ = fmt.Fprintln(w, "ok: copilot-hook.ps1 uses HttpWebRequest")
	}
	if rep.Drift() {
		return fmt.Errorf("the Orca Copilot hooks need tuning — run `dotf orca tune-hooks`")
	}
	return nil
}

// reportOrcaHookTune prints what a repair run did. An unrecognised POST exits
// 0, as the retired script did; --check is the gate.
func reportOrcaHookTune(w io.Writer, rep *orca.HookTuneReport, hookScript string) {
	for _, bak := range rep.Backups {
		_, _ = fmt.Fprintf(w, "backup     %s\n", bak)
	}
	if rep.ScriptUnrecognised {
		_, _ = fmt.Fprintf(w, "unchanged  %s has Invoke-WebRequest but the POST line is unrecognised — review it by hand\n", hookScript)
	}
	switch {
	case rep.Changed == 0 && rep.ScriptUnrecognised:
		// Not "in sync": the line above says a file still needs a hand.
	case rep.Changed == 0:
		_, _ = fmt.Fprintln(w, "in sync   Orca's Copilot hooks already tuned")
	default:
		_, _ = fmt.Fprintf(w, "tuned      %d fix(es) applied — restart the Copilot CLI session to pick up the new orca.json timeout\n", rep.Changed)
	}
}

func newOrcaExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Extract keybindings and clean settings from Orca into dotfiles repo",
		Long: "export reads ~/.orca/keybindings.json and orca-data.json from Orca's data\n" +
			"directory (~/.config/orca, ~/Library/Application Support/orca, %APPDATA%\\orca),\n" +
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
	orcaUserDataDir := orca.UserDataDir(home, runtime.GOOS, os.Getenv)
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
		Long: "tune checks orca-data.json in Orca's data directory and ensures recommended baseline settings\n" +
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
	orcaUserDataDir := orca.UserDataDir(env.Home(), runtime.GOOS, os.Getenv)

	rep, err := orca.Tune(orcaUserDataDir, dryRun, orca.RunningIn(orcaUserDataDir))
	if errors.Is(err, orca.ErrOrcaRunning) {
		// Name the directory: a lock a crashed Orca left behind is diagnosed there.
		return fmt.Errorf("%w (Orca's lock lives in %s)", err, orcaUserDataDir)
	}
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
