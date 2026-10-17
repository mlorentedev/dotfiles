package sshtarget

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// WindowsService is sshd on Windows: the OpenSSH.Server capability's `sshd`
// service. It is a daemon that reads its config at start, so a changed config
// needs a restart, which keeps the sessions already open.
type WindowsService struct {
	Run    Runner // sc.exe as the user, for the reads
	Config string // the sshd_config `sshd -t` validates
}

// The numeric codes sc.exe prints beside the names. The names are translated
// on a localized Windows; the numbers are not.
const (
	scAutoStart     = 2    // START_TYPE AUTO_START
	scRunning       = 4    // STATE RUNNING
	scNoSuchService = 1060 // sc.exe's exit code: ERROR_SERVICE_DOES_NOT_EXIST
)

const capabilityHint = "install it first, elevated: Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0"

// Enabled reports whether sshd starts at boot and runs now. Neither read needs
// administrator rights.
func (s WindowsService) Enabled() (bool, error) {
	start, err := s.scField("qc", "START_TYPE")
	if err != nil {
		return false, err
	}
	state, err := s.scField("query", "STATE")
	if err != nil {
		return false, err
	}
	return start == scAutoStart && state == scRunning, nil
}

func (s WindowsService) running() (bool, error) {
	state, err := s.scField("query", "STATE")
	return state == scRunning, err
}

// scField reads the number sc.exe prints after "<field> :". sc exits 1060
// (ERROR_SERVICE_DOES_NOT_EXIST) when no such service exists, which is the
// capability not installed. The exit code decides, not the output: sc's prose
// is localized, and a 1060 in it may be anything.
func (s WindowsService) scField(verb, field string) (int, error) {
	out, err := s.Run("sc.exe", verb, "sshd")
	if err != nil {
		var exit interface{ ExitCode() int }
		if errors.As(err, &exit) && exit.ExitCode() == scNoSuchService {
			return 0, errors.New("no sshd service: the OpenSSH server is not installed; " + capabilityHint)
		}
		return 0, fmt.Errorf("sc.exe %s sshd: %w: %s", verb, err, bytes.TrimSpace(out))
	}
	for _, line := range strings.Split(string(out), "\n") {
		name, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(name) != field {
			continue
		}
		if f := strings.Fields(value); len(f) > 0 {
			if n, err := strconv.Atoi(f[0]); err == nil {
				return n, nil
			}
		}
	}
	return 0, fmt.Errorf("sc.exe %s sshd printed no %s: %s", verb, field, bytes.TrimSpace(out))
}

// HostKeys creates the host keys the service would otherwise create at its
// first start, so the config validates before sshd listens.
func (WindowsService) HostKeys(root Root) error {
	return asRoot(root, "ssh-keygen", "-A")
}

func (WindowsService) Enable(root Root) error {
	return powershell(root, "Set-Service -Name sshd -StartupType Automatic; Start-Service -Name sshd")
}

// Disable undoes Enable, back to the capability's own default: Manual, stopped.
func (WindowsService) Disable(root Root) error {
	return powershell(root, "Stop-Service -Name sshd; Set-Service -Name sshd -StartupType Manual")
}

// Reload restarts a running sshd so it reads the changed config, and does
// nothing when none runs: the next start reads it.
func (s WindowsService) Reload(root Root) error {
	running, err := s.running()
	if err != nil || !running {
		return err
	}
	return powershell(root, "Restart-Service -Name sshd -Force")
}

func (s WindowsService) Validate(root Root) error {
	return asRoot(root, "sshd", "-t", "-f", s.Config)
}

// powershell runs a fixed script as root. The scripts are this package's own
// constants and quoted values, never input.
func powershell(root Root, script string) error {
	return asRoot(root, psArgv(script)...)
}

func psArgv(script string) []string {
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "$ErrorActionPreference = 'Stop'; " + script}
}

// psQuote quotes s as a PowerShell single-quoted string.
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
