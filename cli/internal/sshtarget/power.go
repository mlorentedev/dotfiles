package sshtarget

import (
	"bytes"
	"errors"
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
// declares none yet (macOS pmset is PR 7d's).
func PowerFor(goos string, run Runner, paths Paths) Power {
	if goos == "linux" {
		return Logind{Run: run, DropIn: paths.LogindDropIn}
	}
	return nil
}

// LogindPolicy is the drop-in Logind installs. It keeps the machine awake with its lid closed on external power,
// and leaves battery behaviour as it was.
const LogindPolicy = `# Managed by ` + "`dotf converge --only workstation-target`" + ` (#2356): an SSH
# target stays awake with its lid closed on external power. Battery keeps the
# distribution's default.
[Login]
HandleLidSwitchExternalPower=ignore
HandleLidSwitchDocked=ignore
`

// sleepTargets are masked so nothing can suspend the machine. logind's lid
// policy alone is not enough: an open GNOME session takes the lid-switch
// inhibitor and suspends on its own, and only the mask stops it.
var sleepTargets = []string{"sleep.target", "suspend.target", "hibernate.target", "hybrid-sleep.target"}

// Logind is the power policy under systemd: a logind drop-in for the lid, and
// the sleep targets masked.
type Logind struct {
	Run    Runner
	DropIn string
}

func (l Logind) Differs() (bool, error) {
	cur, err := readOptional(l.DropIn)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(cur, []byte(LogindPolicy)) || len(l.unmasked()) > 0, nil
}

// unmasked lists the sleep targets not masked yet. `systemctl is-enabled`
// prints "masked" for a masked unit and needs no root.
func (l Logind) unmasked() []string {
	var todo []string
	for _, u := range sleepTargets {
		out, _ := l.Run("systemctl", "is-enabled", u)
		if strings.TrimSpace(string(out)) != "masked" {
			todo = append(todo, u)
		}
	}
	return todo
}

func (l Logind) Apply(root Root) (func() error, error) {
	prev, err := readOptional(l.DropIn)
	if err != nil {
		return nil, err
	}
	todo := l.unmasked()
	var undo []func() error
	undoAll := func() error {
		var errs []error
		for i := len(undo) - 1; i >= 0; i-- {
			errs = append(errs, undo[i]())
		}
		return errors.Join(errs...)
	}
	if !bytes.Equal(prev, []byte(LogindPolicy)) {
		if err := asRoot(root, "install", "-d", "-m", "0755", filepath.Dir(l.DropIn)); err != nil {
			return undoAll, err
		}
		if err := installRoot(root, l.DropIn, []byte(LogindPolicy)); err != nil {
			return undoAll, err
		}
		undo = append(undo, func() error {
			if err := restoreRoot(root, l.DropIn, prev); err != nil {
				return err
			}
			return reloadLogind(root)
		})
		if err := reloadLogind(root); err != nil {
			return undoAll, err
		}
	}
	if len(todo) > 0 {
		if err := asRoot(root, append([]string{"systemctl", "mask"}, todo...)...); err != nil {
			return undoAll, err
		}
		undo = append(undo, func() error {
			return asRoot(root, append([]string{"systemctl", "unmask"}, todo...)...)
		})
	}
	return undoAll, nil
}

// reloadLogind makes logind read its config again. SIGHUP reloads it without
// ending sessions; a restart would end them, so it is never done here.
func reloadLogind(root Root) error {
	return asRoot(root, "systemctl", "kill", "-s", "HUP", "systemd-logind")
}

// Prove asks logind what it now does with the lid on external power, and
// systemd whether every sleep target is masked.
func (l Logind) Prove() error {
	out, err := l.Run("busctl", "get-property", "org.freedesktop.login1", "/org/freedesktop/login1",
		"org.freedesktop.login1.Manager", "HandleLidSwitchExternalPower")
	if err != nil {
		return fmt.Errorf("ask logind for its lid policy: %w: %s", err, bytes.TrimSpace(out))
	}
	if got := strings.TrimSpace(string(out)); got != `s "ignore"` {
		return fmt.Errorf("logind still answers HandleLidSwitchExternalPower=%s after the reload; it reads the drop-in at its next restart (`sudo systemctl restart systemd-logind` from a local session ends the graphical one)", got)
	}
	if todo := l.unmasked(); len(todo) > 0 {
		return fmt.Errorf("%s not masked: the machine can still suspend", strings.Join(todo, ", "))
	}
	return nil
}
