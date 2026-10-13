package sshtarget

import (
	"bytes"
	"fmt"
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

// Enabled reports whether launchd has loaded sshd. `launchctl print` exits
// non-zero (113) for a service it does not know, and needs no root to ask.
func (l Launchd) Enabled() (bool, error) {
	_, err := l.Run("launchctl", "print", launchdLabel)
	return err == nil, nil
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
