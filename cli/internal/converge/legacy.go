package converge

import (
	"os"
	"os/exec"
	"path/filepath"
)

// LegacySetupGuardEnv marks a setup script that converge started. A converge
// run inside it skips its own legacy step, so a script that calls
// `dotf converge` cannot start itself again.
const LegacySetupGuardEnv = "DOTF_CONVERGE_LEGACY_SETUP"

// setupOverrideEnv names another setup command to run instead of the
// checkout's script; the scheduled self-update has honoured it since CLI-027.
const setupOverrideEnv = "DOTFILES_SELFUPDATE_SETUP_CMD"

// legacySetup runs the setup script of this OS after every native step, for
// the blocks no reconciler has ported yet (ADR-045). The script cannot plan
// and reports nothing but its exit status, so the step is opaque: a plan says
// it would run, and an apply never counts it as converged or changed. macOS
// has no setup script, so the step is reported skipped there.
type legacySetup struct {
	run func(Env) error // ExecSetup in production
}

func (legacySetup) Name() string        { return "legacy-setup" }
func (legacySetup) Platforms() []string { return []string{"linux", "windows"} }

func (l legacySetup) Reconcile(e Env, dryRun bool) (Result, error) {
	switch {
	case l.run == nil:
		return Result{Skip: "no setup runner is wired into this registry"}, nil
	case os.Getenv(LegacySetupGuardEnv) != "":
		return Result{Skip: "already inside a setup run that converge started"}, nil
	}
	script := setupScript(e.GOOS)
	if dryRun {
		return Result{Opaque: true, Detail: script + " cannot plan; an apply runs it"}, nil
	}
	if err := l.run(e); err != nil {
		return Result{}, err
	}
	return Result{Opaque: true, Detail: "ran " + script}, nil
}

// Probe has nothing to hold: the script's exit status is its only post-condition.
func (legacySetup) Probe(Env) error { return nil }

func setupScript(goos string) string {
	if goos == "windows" {
		return "setup-windows.ps1"
	}
	return "setup-linux.sh"
}

// ExecSetup runs the checkout's setup script for e.GOOS, or the command
// DOTFILES_SELFUPDATE_SETUP_CMD names, from the checkout and with the
// terminal's output, marked with LegacySetupGuardEnv. Its stdin stays closed:
// the script runs unattended under a timer and under `curl … | bash` alike,
// so a prompt fails fast instead of reading the pipe (sudo reads the tty).
func ExecSetup(e Env) error {
	cmd := os.Getenv(setupOverrideEnv)
	if cmd == "" {
		cmd = filepath.Join(e.RepoRoot, setupScript(e.GOOS))
	}
	var c *exec.Cmd
	if e.GOOS == "windows" {
		c = exec.Command("pwsh", "-NoProfile", "-File", cmd)
	} else {
		c = exec.Command(cmd) //nolint:gosec // the checkout's setup script, or the owner's override
	}
	c.Dir = e.RepoRoot
	c.Env = append(os.Environ(), LegacySetupGuardEnv+"=1")
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}
