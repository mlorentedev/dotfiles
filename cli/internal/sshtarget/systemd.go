package sshtarget

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// Systemd is sshd on Linux. Debian and Ubuntu name the unit ssh, Fedora and
// Arch name it sshd, and Ubuntu 22.10 and later start it through a socket unit
// by default. Unlike launchd's per-connection sshd, the service is a daemon
// that read its config at start, so a changed drop-in needs a reload, and a
// reload keeps the session that runs the apply.
type Systemd struct {
	Run Runner // systemctl as the user, for the root-free reads
}

// sshUnits are the unit names the distributions use, the Debian one first.
var sshUnits = []string{"ssh", "sshd"}

// units finds the ssh service the machine has, and its socket unit when there
// is one. `systemctl list-unit-files` needs no root.
func (s Systemd) units() (service, socket string, err error) {
	var names []string
	for _, u := range sshUnits {
		names = append(names, u+".service", u+".socket")
	}
	// It exits 1 when no unit matched, so an error with output still lists
	// what did match, and one without is reported with the missing package.
	out, lerr := s.Run(append([]string{"systemctl", "list-unit-files", "--no-legend", "--no-pager"}, names...)...)
	have := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			have[f[0]] = true
		}
	}
	for _, u := range sshUnits {
		if have[u+".service"] {
			if have[u+".socket"] {
				socket = u + ".socket"
			}
			return u + ".service", socket, nil
		}
	}
	if lerr != nil {
		return "", "", fmt.Errorf("no ssh.service or sshd.service unit (systemctl list-unit-files: %w): install openssh-server first", lerr)
	}
	return "", "", errors.New("no ssh.service or sshd.service unit: install openssh-server first")
}

// starter is the unit Enable turns on: the socket when the distribution ships
// one, which is how it starts sshd by default, else the service.
func (s Systemd) starter() (string, error) {
	service, socket, err := s.units()
	if socket != "" {
		return socket, err
	}
	return service, err
}

// Enabled reports whether sshd starts at boot and runs now, through either
// unit: a machine where the owner chose the service over the socket counts.
func (s Systemd) Enabled() (bool, error) {
	service, socket, err := s.units()
	if err != nil {
		return false, err
	}
	for _, u := range []string{socket, service} {
		if u == "" {
			continue
		}
		_, enabled := s.Run("systemctl", "is-enabled", "--quiet", u)
		_, active := s.Run("systemctl", "is-active", "--quiet", u)
		if enabled == nil && active == nil {
			return true, nil
		}
	}
	return false, nil
}

func (Systemd) HostKeys(root Root) error {
	return asRoot(root, "ssh-keygen", "-A")
}

func (s Systemd) Enable(root Root) error {
	u, err := s.starter()
	if err != nil {
		return err
	}
	return asRoot(root, "systemctl", "enable", "--now", u)
}

// Disable undoes Enable. Stopping the socket does not stop the service a
// connection already started through it (the proof's, for one), so that is
// stopped too.
func (s Systemd) Disable(root Root) error {
	service, socket, err := s.units()
	if err != nil {
		return err
	}
	if socket == "" {
		return asRoot(root, "systemctl", "disable", "--now", service)
	}
	if err := asRoot(root, "systemctl", "disable", "--now", socket); err != nil {
		return err
	}
	return asRoot(root, "systemctl", "stop", service)
}

// Reload makes a running sshd read the changed drop-in and does nothing when
// none runs: a socket-started sshd that has not had a connection yet reads it
// when the first one comes.
func (s Systemd) Reload(root Root) error {
	service, _, err := s.units()
	if err != nil {
		return err
	}
	return asRoot(root, "systemctl", "try-reload-or-restart", service)
}

// privsepDir is the directory sshd -t insists on. systemd creates it when it
// starts the service, so under socket activation, before the first
// connection, it can be missing and `sshd -t` fails on a valid config.
const privsepDir = "/run/sshd"

func (Systemd) Validate(root Root) error {
	out, err := root.Run("/usr/sbin/sshd", "-t")
	if err != nil && bytes.Contains(out, []byte("Missing privilege separation directory")) {
		if err := asRoot(root, "install", "-d", "-m", "0755", privsepDir); err != nil {
			return err
		}
		out, err = root.Run("/usr/sbin/sshd", "-t")
	}
	if err != nil {
		return fmt.Errorf("/usr/sbin/sshd -t: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// ServiceFor is sshd's service manager on goos.
func ServiceFor(goos string, run Runner, paths Paths) (Service, error) {
	switch goos {
	case "darwin":
		return Launchd{Run: run}, nil
	case "linux":
		return Systemd{Run: run}, nil
	case "windows":
		return WindowsService{Run: run, Config: paths.SSHDConfig}, nil
	}
	return nil, fmt.Errorf("no sshd service manager for %s", goos)
}
