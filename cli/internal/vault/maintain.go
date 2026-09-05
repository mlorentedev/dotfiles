package vault

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Increment 3 of CLI-021 (#490): the Go port of
// scripts/vault-maintenance-weekly.{sh,ps1}. Built BESIDE the twins — nothing
// repoints at this yet (the crontab entry at setup-linux.sh:1605 and the Task
// Scheduler entry at setup-windows.ps1:2185 are CLI-023 / #492's to flip).
//
// Unlike increments 1 and 2 this is NOT golden-characterized, deliberately.
// The twin's output is a timestamped log wrapping two subcommands whose
// byte-parity is already proven by tests/knowledge-crystallize-go-parity.bats
// and tests/vault-health-go-parity.bats. What is left to characterize is the
// wrapper itself — log path, section framing, the issue-count regex, the
// notification threshold, the exit code — and those are behaviours, not bytes.
// They are covered by table tests here plus the behavioural cases in
// tests/vault-maintenance-weekly.bats.
//
// LINUX IS THE REFERENCE. The .ps1 twin runs no health step at all despite its
// own header claiming it does; that divergence is recorded in
// specs/CLI-021-dotf-vault-build-knowledge/divergences.md §Increment 2 and is
// not re-decided here. Porting the Linux behaviour means `maintain` runs both
// steps on every OS, which is the ADR-020 precedent CLI-024 set (reconstruct
// the Linux superset, not the .ps1 subset).
//
// Composition is IN-PROCESS, not `exec dotf`. The shell had no choice; a Go
// binary that already owns both steps does. This deletes the twin's documented
// failure mode outright — cron's minimal PATH excludes ~/.local/bin, so a bare
// `dotf` there silently no-ops under `|| true` every Sunday
// (scripts/vault-maintenance-weekly.sh:12-16, and the regression guard at
// tests/vault-maintenance-weekly.bats:147). There is no PATH to harden when
// there is no subprocess.

// issueRE mirrors the twins' issue counter: `grep -ciE "warning|fail|action|stale"`
// on the .sh, `Select-String -Pattern "WARNING|FAIL|ACTION|STALE" -CaseSensitive:$false`
// on the .ps1. Both count matching LINES, not matches, and both are substring
// matches rather than word-boundary ones — "failed" and "stalemate" score.
// Reproduced faithfully; over-counting is the oracle's behaviour, not a defect
// introduced here.
var issueRE = regexp.MustCompile(`(?i)warning|fail|action|stale`)

// Urgency is the notification level the issue count selects. The names are
// notify-send's own (`-u low|normal`), which the .sh passes through directly.
type Urgency string

const (
	UrgencyLow    Urgency = "low"
	UrgencyNormal Urgency = "normal"
)

// Notifier fires the best-effort desktop notification. It returns nothing
// because both twins discard every failure: notify-send is guarded behind
// `command -v` and then `|| true`, and the .ps1 wraps the whole toast in a
// try/catch. A maintenance run must not fail because a desktop bus is absent.
type Notifier func(urgency Urgency, title, body string)

// MaintainOptions configures RunMaintain. Every seam the tests need to replace
// is a field, so no test touches a real vault, a real log location, or a real
// desktop bus.
type MaintainOptions struct {
	Home    string        // for CrystallizeAll's project discovery
	Today   string        // YYYY-MM-DD, as CrystallizeAll expects
	LogFile string        // absolute; DefaultLogFile derives the twins' location
	Health  HealthOptions // forwarded verbatim to RunHealth
	Now     func() time.Time
	Notify  Notifier

	// The two steps, replaceable only from inside this package — i.e. by tests.
	// Unexported rather than package-level vars so parallel tests cannot race,
	// and unexported rather than public so no caller can substitute a step and
	// still call the result a maintenance run.
	crystallizeStep func(io.Writer) error
	healthStep      func(io.Writer) (int, error)
}

// DefaultLogFile returns the log location the twins write, per OS:
// $HOME/.local/share/vault-maintenance/latest.log on Linux (.sh:19-20) and
// %LOCALAPPDATA%\vault-maintenance\latest.log on Windows (.ps1:16-17).
//
// A GOOS switch rather than os.UserCacheDir(), which resolves ~/.cache on Linux
// — a different directory from the ~/.local/share the twin has been writing to,
// and the log is an artefact a human goes looking for by path.
func DefaultLogFile(home string) string {
	return logFileFor(runtime.GOOS, home, os.Getenv("LOCALAPPDATA"))
}

// logFileFor is DefaultLogFile with its two ambient inputs passed in, so BOTH
// OS branches are testable from either host. Reading runtime.GOOS inline would
// leave whichever branch you are not running on covered only by the compiler.
func logFileFor(goos, home, localAppData string) string {
	if goos == "windows" {
		if localAppData != "" {
			return filepath.Join(localAppData, "vault-maintenance", "latest.log")
		}
		return filepath.Join(home, "AppData", "Local", "vault-maintenance", "latest.log")
	}
	return filepath.Join(home, ".local", "share", "vault-maintenance", "latest.log")
}

// CountIssues reports how many LINES of the log match the twins' issue pattern.
func CountIssues(log string) int {
	n := 0
	for _, line := range strings.Split(log, "\n") {
		if issueRE.MatchString(line) {
			n++
		}
	}
	return n
}

// notificationFor maps an issue count onto the notification the twins send.
// Both agree on the wording and on the threshold being `> 0`.
func notificationFor(issues int) (Urgency, string, string) {
	if issues > 0 {
		return UrgencyNormal, "Vault Maintenance",
			fmt.Sprintf("%d potential issues. Run /insights on active projects.", issues)
	}
	return UrgencyLow, "Vault Maintenance", "All clean. No action needed."
}

// RunMaintain composes crystallize + health into the weekly log, fires the
// best-effort notification, and reports the log path on w.
//
// The timestamp format is Go's time.UnixDate, which matches GNU date(1)'s
// default output — what `$(date)` produces in the .sh. Get-Date's default on
// the .ps1 is a different, locale-dependent shape; recorded as a divergence
// rather than reproduced, since Linux is the reference.
//
// # It returns an error, never an exit code, and that is a decision
//
// A FINDING IS NOT A FAILURE. RunHealth's 0/1/2 answers "what did the report
// find"; maintain's status answers "did the run do its job". Health reporting
// orphans, or degrading because the Obsidian GUI is closed, means maintain
// worked exactly as designed — so those do not become a non-zero status.
//
// Propagating health's code would make the status depend on whether a desktop
// GUI happened to be running, which from cron's point of view is
// nondeterministic: every Sunday the laptop was shut would mail the owner a
// failed-job notice for a healthy run. False alarms on a weekly channel train
// the owner to ignore it, and would take the desktop notification — the signal
// this command was actually built around — down with it.
//
// The finding is not dropped, it is routed: the report body goes to the log,
// the count picks the notification urgency, and the verdict is summarised on w
// for whoever ran this by hand. Informative, not contractual.
//
// The only non-nil error is a genuine failure of the run itself — the log
// directory or the log file could not be written. Everything else is a finding.
func RunMaintain(w io.Writer, opts MaintainOptions) error {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	crystallize := opts.crystallizeStep
	if crystallize == nil {
		crystallize = func(lw io.Writer) error { return CrystallizeAll(lw, opts.Home, opts.Today) }
	}
	health := opts.healthStep
	if health == nil {
		health = func(lw io.Writer) (int, error) { return RunHealth(lw, opts.Health) }
	}

	var log bytes.Buffer
	fmt.Fprintf(&log, "=== Vault Maintenance: %s ===\n\n", now().Format(time.UnixDate))

	// Both steps are best-effort, exactly as the twin's `|| true`: a crystallize
	// that refuses one project must not stop the health report from running, and
	// neither failure stops the log being written.
	fmt.Fprintf(&log, "--- dotf vault crystallize --all ---\n")
	fmt.Fprintf(&log, "[INFO] Date: %s\n", opts.Today)
	if err := crystallize(&log); err != nil {
		fmt.Fprintf(&log, "crystallize: %v\n", err)
	}
	fmt.Fprintf(&log, "\n")

	fmt.Fprintf(&log, "--- vault-health ---\n")
	healthCode, err := health(&log)
	if err != nil {
		fmt.Fprintf(&log, "health: %v\n", err)
	}
	fmt.Fprintf(&log, "\n")

	fmt.Fprintf(&log, "=== Done: %s ===\n", now().Format(time.UnixDate))

	// The log is the artefact. Failing to write it is the one way this run
	// genuinely did not do its job, so it is the one thing that errors.
	if err := os.MkdirAll(filepath.Dir(opts.LogFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(opts.LogFile, log.Bytes(), 0o644); err != nil {
		return err
	}

	issues := CountIssues(log.String())
	if opts.Notify != nil {
		opts.Notify(notificationFor(issues))
	}

	// The verdict, on stdout, for whoever ran this by hand. Silence means pass,
	// as everywhere else in this CLI. The issue count is printed unconditionally
	// because the notification that normally carries it is the FIRST thing to
	// vanish on a headless box — which is exactly where an unattended weekly run
	// lives, and where the log is the only thing anyone will ever read.
	switch healthCode {
	case 1:
		emit(w, "Vault health: FAILED — one or more checks\n")
	case 2:
		emit(w, "Vault health: SKIPPED — the Obsidian GUI was unreachable\n")
	}
	emit(w, "%d issue line(s) in the log\n", issues)
	emit(w, "Log written to %s\n", opts.LogFile)

	return nil
}

// NotifyDesktop is the default Notifier: notify-send on Linux, a NotifyIcon
// balloon via powershell on Windows. Both are fire-and-forget — every error is
// discarded, matching the twins.
func NotifyDesktop(urgency Urgency, title, body string) {
	switch runtime.GOOS {
	case "windows":
		// The balloon dies with the process that owns it, so the .ps1's
		// Start-Sleep 11 (one second past the 10s ShowBalloonTip) is load-bearing
		// and is kept here rather than dropped.
		script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms
$b = New-Object System.Windows.Forms.NotifyIcon
$b.Icon = [System.Drawing.SystemIcons]::Information
$b.BalloonTipTitle = %q
$b.BalloonTipText = %q
$b.Visible = $true
$b.ShowBalloonTip(10000)
Start-Sleep -Seconds 11
$b.Dispose()`, title, body)
		_ = exec.Command("powershell", "-NoProfile", "-Command", script).Run()
	default:
		if _, err := exec.LookPath("notify-send"); err != nil {
			return
		}
		cmd := exec.Command("notify-send", "-u", string(urgency), title, body)
		// The .sh defaults both when unset because cron inherits neither
		// (scripts/vault-maintenance-weekly.sh:43-44).
		cmd.Env = os.Environ()
		if os.Getenv("DISPLAY") == "" {
			cmd.Env = append(cmd.Env, "DISPLAY=:0")
		}
		if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
			cmd.Env = append(cmd.Env,
				fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/%d/bus", os.Getuid()))
		}
		_ = cmd.Run()
	}
}
