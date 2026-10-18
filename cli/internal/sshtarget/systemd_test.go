package sshtarget

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// linux is a fake systemd host: which ssh units exist, which are enabled and
// active, and the commands run as root.
type linux struct {
	units     []string // unit files list-unit-files knows
	enabled   map[string]bool
	active    map[string]bool
	privsep   bool // whether /run/sshd exists
	rootCalls []string
}

func (l *linux) user(argv ...string) ([]byte, error) {
	switch {
	case slices.Equal(argv[:2], []string{"systemctl", "list-unit-files"}):
		var out []string
		for _, u := range l.units {
			if slices.Contains(argv, u) {
				out = append(out, u+" enabled enabled")
			}
		}
		if len(out) == 0 {
			// systemctl exits 1 when no unit matched.
			return nil, errors.New("exit status 1")
		}
		return []byte(strings.Join(out, "\n") + "\n"), nil
	case argv[1] == "is-enabled" && l.enabled[argv[3]], argv[1] == "is-active" && l.active[argv[3]]:
		return nil, nil
	}
	return nil, errors.New("exit status 1")
}

func (l *linux) Ready() error { return nil }

func (l *linux) Run(argv ...string) ([]byte, error) {
	l.rootCalls = append(l.rootCalls, strings.Join(argv, " "))
	switch {
	case argv[0] == "/usr/sbin/sshd" && !l.privsep:
		return []byte("Missing privilege separation directory: /run/sshd"), errors.New("exit status 255")
	case argv[0] == "install" && slices.Contains(argv, privsepDir):
		l.privsep = true
	}
	return nil, nil
}

func TestSystemdFindsTheDistributionsUnit(t *testing.T) {
	for _, tc := range []struct {
		name          string
		units         []string
		service, sock string
		wantErr       string
	}{
		{name: "Ubuntu, socket-activated", units: []string{"ssh.service", "ssh.socket"}, service: "ssh.service", sock: "ssh.socket"},
		{name: "Debian without the socket", units: []string{"ssh.service"}, service: "ssh.service"},
		{name: "Fedora", units: []string{"sshd.service", "sshd.socket"}, service: "sshd.service", sock: "sshd.socket"},
		{name: "no openssh-server", wantErr: "install openssh-server"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := &linux{units: tc.units}
			service, sock, err := Systemd{Run: l.user}.units()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want it to name %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || service != tc.service || sock != tc.sock {
				t.Errorf("units() = %q, %q, %v; want %q, %q", service, sock, err, tc.service, tc.sock)
			}
		})
	}
}

// sshd counts as enabled through either unit, but only when it both starts at
// boot and runs now: enabled-but-stopped would refuse the proof, and
// running-but-disabled would not survive a reboot.
func TestSystemdEnabledNeedsBootAndNow(t *testing.T) {
	for _, tc := range []struct {
		name            string
		enabled, active []string
		want            bool
	}{
		{name: "socket enabled and listening", enabled: []string{"ssh.socket"}, active: []string{"ssh.socket"}, want: true},
		{name: "the owner chose the service", enabled: []string{"ssh.service"}, active: []string{"ssh.service"}, want: true},
		{name: "enabled, not running", enabled: []string{"ssh.socket"}},
		{name: "running, not enabled at boot", active: []string{"ssh.service"}},
		{name: "off", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := &linux{units: []string{"ssh.service", "ssh.socket"}, enabled: set(tc.enabled), active: set(tc.active)}
			got, err := Systemd{Run: l.user}.Enabled()
			if err != nil || got != tc.want {
				t.Errorf("Enabled() = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

// Serving is either half: an sshd that runs now, or that starts at the next
// boot, would serve its stock config once the drop-in goes.
func TestSystemdServingIsRunningOrStartingAtBoot(t *testing.T) {
	for _, tc := range []struct {
		enabled, active []string
		want            bool
	}{{[]string{"ssh.socket"}, nil, true}, {nil, []string{"ssh.service"}, true}, {nil, nil, false}} {
		l := &linux{units: []string{"ssh.service", "ssh.socket"}, enabled: set(tc.enabled), active: set(tc.active)}
		if got, err := (Systemd{Run: l.user}).Serving(); err != nil || got != tc.want {
			t.Errorf("enabled %v, active %v: Serving() = %v, %v; want %v", tc.enabled, tc.active, got, err, tc.want)
		}
	}
}

// Enable turns on what the distribution starts sshd with, and Disable, the
// undo, turns off the same unit, and the service the socket started.
func TestSystemdEnableAndItsUndoUseTheSameUnit(t *testing.T) {
	for _, tc := range []struct {
		units []string
		want  []string
	}{
		{units: []string{"ssh.service", "ssh.socket"}, want: []string{"systemctl enable --now ssh.socket", "systemctl disable --now ssh.socket", "systemctl stop ssh.service"}},
		{units: []string{"sshd.service"}, want: []string{"systemctl enable --now sshd.service", "systemctl disable --now sshd.service"}},
	} {
		l := &linux{units: tc.units}
		s := Systemd{Run: l.user}
		if err := s.Enable(l); err != nil {
			t.Fatal(err)
		}
		if err := s.Disable(l); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(l.rootCalls, tc.want) {
			t.Errorf("root ran %q, want %q", l.rootCalls, tc.want)
		}
	}
}

// A reload must reach the service, never the socket, and must not start an
// sshd that is not running: try-reload-or-restart does neither.
func TestSystemdReloadsTheServiceOnlyIfItRuns(t *testing.T) {
	l := &linux{units: []string{"ssh.service", "ssh.socket"}}
	if err := (Systemd{Run: l.user}).Reload(l); err != nil {
		t.Fatal(err)
	}
	if want := []string{"systemctl try-reload-or-restart ssh.service"}; !slices.Equal(l.rootCalls, want) {
		t.Errorf("root ran %q, want %q", l.rootCalls, want)
	}
}

// Under socket activation nothing has created /run/sshd before the first
// connection, and `sshd -t` refuses a valid config for that alone.
func TestSystemdValidateCreatesTheMissingPrivsepDir(t *testing.T) {
	l := &linux{}
	if err := (Systemd{Run: l.user}).Validate(l); err != nil {
		t.Fatalf("Validate = %v on a valid config with no /run/sshd", err)
	}
	want := []string{"/usr/sbin/sshd -t", "install -d -m 0755 /run/sshd", "/usr/sbin/sshd -t"}
	if !slices.Equal(l.rootCalls, want) {
		t.Errorf("root ran %q, want %q", l.rootCalls, want)
	}
}

func TestSystemdValidateReportsARejectedConfig(t *testing.T) {
	l := &linux{privsep: true}
	reject := func(argv ...string) ([]byte, error) {
		return []byte("/etc/ssh/sshd_config.d/01-workstation.conf line 4: Bad configuration option"), errors.New("exit status 255")
	}
	err := Systemd{Run: l.user}.Validate(rootFunc(reject))
	if err == nil || !strings.Contains(err.Error(), "Bad configuration option") {
		t.Fatalf("Validate = %v, want sshd's own message", err)
	}
}

func TestServiceForPicksTheOSManager(t *testing.T) {
	if s, _ := ServiceFor("darwin", nil, Paths{}); !isType[Launchd](s) {
		t.Errorf("darwin → %T", s)
	}
	if s, _ := ServiceFor("linux", nil, Paths{}); !isType[Systemd](s) {
		t.Errorf("linux → %T", s)
	}
	paths := WindowsPaths(`C:\ProgramData`, `C:\Users\manu`)
	if s, _ := ServiceFor("windows", nil, paths); !isType[WindowsService](s) || s.(WindowsService).Config != paths.SSHDConfig {
		t.Errorf("windows → %#v, want WindowsService validating %s", s, paths.SSHDConfig)
	}
	if _, err := ServiceFor("plan9", nil, Paths{}); err == nil {
		t.Error("an OS with no service manager must say so")
	}
}

type rootFunc func(argv ...string) ([]byte, error)

func (rootFunc) Ready() error                         { return nil }
func (f rootFunc) Run(argv ...string) ([]byte, error) { return f(argv...) }

func isType[T any](v any) bool {
	_, ok := v.(T)
	return ok
}

func set(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}
