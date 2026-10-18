package sshtarget

import (
	"bytes"
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
)

// Launchd is sshd on macOS: launchd's com.openssh.sshd, socket-activated, so
// each connection starts an sshd that reads the config afresh and a changed
// drop-in needs no reload. It is what System Settings' Remote Login toggles.
type Launchd struct {
	Run Runner // launchctl as the user, for the root-free read
}

const (
	launchdLabel = "system/com.openssh.sshd"
	launchdPlist = "/System/Library/LaunchDaemons/ssh.plist"
)

// Enabled reports whether launchd has loaded sshd, without root. A print that
// fails for another reason than an unknown service is an error, never "not
// loaded": the step would bootstrap as root a service it never read.
func (l Launchd) Enabled() (bool, error) {
	return launchdLoaded(l.Run, launchdLabel)
}

// launchdLoaded asks launchd, through run, whether it knows service.
// schedule.LaunchdLoaded owns the reading: only `launchctl print`'s exit 113
// means it does not (#2731).
func launchdLoaded(run Runner, service string) (bool, error) {
	return schedule.LaunchdLoaded(func(args ...string) (string, error) {
		out, err := run(append([]string{"launchctl"}, args...)...)
		return string(out), err
	}, service)
}

// HostKeys creates the host keys macOS would otherwise create at the first
// connection, so the config can be validated before sshd ever listens.
func (Launchd) HostKeys(root Root) error {
	return asRoot(root, "ssh-keygen", "-A")
}

func (Launchd) Enable(root Root) error {
	if err := asRoot(root, "launchctl", "enable", launchdLabel); err != nil {
		return err
	}
	return asRoot(root, "launchctl", "bootstrap", "system", launchdPlist)
}

func (Launchd) Disable(root Root) error {
	if err := asRoot(root, "launchctl", "bootout", launchdLabel); err != nil {
		return err
	}
	return asRoot(root, "launchctl", "disable", launchdLabel)
}

// Reload is a no-op: launchd starts a fresh sshd per connection.
func (Launchd) Reload(Root) error { return nil }

func (Launchd) Validate(root Root) error {
	return asRoot(root, "/usr/sbin/sshd", "-t")
}

func asRoot(root Root, argv ...string) error {
	out, err := root.Run(argv...)
	if err != nil {
		return fmt.Errorf("%v: %w: %s", argv, err, bytes.TrimSpace(out))
	}
	return nil
}
