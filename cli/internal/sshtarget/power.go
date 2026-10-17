package sshtarget

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

// Power keeps an SSH target awake: a target that sleeps cannot be reached.
// Differs reads without root; Apply returns the undo of what it changed, even
// when it fails part way; Prove reads the live state back, without root.
type Power interface {
	Differs() (bool, error)
	Apply(root Root) (undo func() error, err error)
	Prove() error
}

// PowerFor is the power policy the target declares on goos, nil where it
// declares none yet.
func PowerFor(goos string, run Runner, paths Paths) Power {
	switch goos {
	case "linux":
		return Logind{Run: run, DropIn: paths.LogindDropIn}
	case "darwin":
		return Pmset{Run: run}
	case "windows":
		return Powercfg{Run: run}
	}
	return nil
}

// LogindPolicy is the drop-in Logind installs. It keeps the machine awake with
// its lid closed on external power, and leaves battery behaviour as it was.
const LogindPolicy = `# Managed by ` + "`dotf converge --only workstation-target`" + ` (#2356): an SSH
# target stays awake with its lid closed on external power. Battery keeps the
# distribution's default.
[Login]
HandleLidSwitchExternalPower=ignore
HandleLidSwitchDocked=ignore
`

// Logind is the power policy under systemd: a logind drop-in for the lid. It
// masks no sleep target, because a mask holds on battery too, and a closed lid
// would then drain the battery to a hard power-off. A desktop session that
// takes the lid-switch inhibitor (GNOME with an external monitor) takes it to
// keep the machine awake, which HandleLidSwitchDocked already declares.
type Logind struct {
	Run    Runner
	DropIn string
}

func (l Logind) Differs() (bool, error) {
	cur, err := readOptional(l.DropIn)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(cur, []byte(LogindPolicy)), nil
}

func (l Logind) Apply(root Root) (func() error, error) {
	noop := func() error { return nil }
	prev, err := readOptional(l.DropIn)
	if err != nil {
		return noop, err
	}
	if bytes.Equal(prev, []byte(LogindPolicy)) {
		return noop, nil
	}
	reload := l.canReload()
	if err := asRoot(root, "install", "-d", "-m", "0755", filepath.Dir(l.DropIn)); err != nil {
		return noop, err
	}
	if err := installRoot(root, l.DropIn, []byte(LogindPolicy)); err != nil {
		return noop, err
	}
	undo := func() error {
		if err := restoreRoot(root, l.DropIn, prev); err != nil || !reload {
			return err
		}
		return reloadLogind(root)
	}
	if reload {
		if err := reloadLogind(root); err != nil {
			return undo, err
		}
	}
	return undo, nil
}

// canReload reports whether this logind reads its config again on SIGHUP.
// One that does not would die of the signal, so it is never sent one; the
// drop-in then applies at its next start, the next boot at the latest.
func (l Logind) canReload() bool {
	out, _ := l.Run("systemctl", "show", "-p", "CanReload", "--value", "systemd-logind.service")
	return strings.TrimSpace(string(out)) == "yes"
}

// reloadLogind makes logind read its config again. SIGHUP reloads it without
// ending sessions; a restart would end them, so it is never done here.
func reloadLogind(root Root) error {
	return asRoot(root, "systemctl", "kill", "-s", "HUP", "systemd-logind")
}

// Prove asks logind, without root, what it now does with the lid on external
// power. A logind that cannot reload has nothing to prove before its next
// start, the next boot at the latest, and is not failed for it: the drop-in
// is on disk, and the restart that would prove it now ends the graphical
// session.
func (l Logind) Prove() error {
	if !l.canReload() {
		return nil
	}
	out, err := l.Run("busctl", "get-property", "org.freedesktop.login1", "/org/freedesktop/login1",
		"org.freedesktop.login1.Manager", "HandleLidSwitchExternalPower")
	if err != nil {
		return fmt.Errorf("ask logind for its lid policy: %w: %s", err, bytes.TrimSpace(out))
	}
	if got := strings.TrimSpace(string(out)); got != `s "ignore"` {
		return fmt.Errorf("logind still answers HandleLidSwitchExternalPower=%s after the reload; it reads the drop-in at its next restart (`sudo systemctl restart systemd-logind` from a local session ends the graphical one)", got)
	}
	return nil
}
