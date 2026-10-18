package sshtarget

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/schedule"
)

// The pf pieces on macOS (owner decisions on #2356, 2026-10-13). The anchor is
// a sub-anchor of the `anchor "com.apple/*"` that Apple's /etc/pf.conf already
// evaluates, so pf.conf is never edited. Sub-anchors run in name order, and 150
// sorts before 200.AirDrop and 250.ApplicationFirewall: a `pass quick` of
// theirs can then never reopen port 22 before our block. Whether they hold
// one, and whether the sub-anchor survives Apple's own `pfctl -f
// /etc/pf.conf`, are two of the three root measurements the owner gated this on.
const (
	PFAnchor     = "com.apple/150.dotf-ssh"
	pfLabel      = "com.github.mlorentedev.dotf.pf-ssh"
	pfLaunchd    = "system/" + pfLabel
	pfctl        = "/sbin/pfctl"
	pfConfirmSec = 120
)

// PF is the firewall on macOS: an anchor that lets port 22 in on lo0 and to
// the node's own mesh address only, and a LaunchDaemon that loads it at boot.
// It matches the destination, not the source: a LAN host can spoof a source
// address, it cannot make its packet arrive at the mesh address. The anchor
// blocks port 22 and nothing else, so it cannot break anything else on the
// machine, and its undo flushes the anchor only: never `pfctl -d`, which
// would switch off every other rule pf holds.
//
// Self-connections on macOS cross lo0, so this host can prove that the anchor
// is loaded and pf enabled, not what the LAN sees.
type PF struct {
	Anchor  string // the anchor file, /etc/pf.anchors/dotf-ssh
	Plist   string // the LaunchDaemon, /Library/LaunchDaemons/<pfLabel>.plist
	Run     Runner // launchctl as the user, for the root-free read
	Mesh    func() (net.IP, error)
	Session string // $SSH_CONNECTION of the applying process; "" when local
	Sudoed  bool   // dotf itself runs under sudo, which strips $SSH_CONNECTION

	timer string // pid of the armed commit-confirm timer, "" when none
}

// PFRules renders the anchor for the node's mesh address.
func PFRules(mesh net.IP) string {
	return "# Managed by `dotf converge --only workstation-target` (#2356): port 22 on\n" +
		"# loopback and to this node's mesh address only.\n" +
		"pass in quick on lo0 proto tcp to any port 22\n" +
		"pass in quick proto tcp to " + mesh.String() + " port 22 flags S/SA keep state\n" +
		"block drop in quick proto tcp to any port 22\n"
}

// PFLaunchDaemon is the plist of the job that loads anchor at boot. It runs
// one command, which loads the anchor and enables pf. `-a` matters: a plain
// `pfctl -E -f` would replace Apple's main ruleset.
func PFLaunchDaemon(anchor string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + pfLabel + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + pfctl + `</string>
		<string>-E</string>
		<string>-a</string>
		<string>` + PFAnchor + `</string>
		<string>-f</string>
		<string>` + anchor + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`
}

func (f *PF) Differs() (bool, error) {
	mesh, err := f.Mesh()
	if err != nil {
		return false, err
	}
	for path, want := range map[string]string{f.Anchor: PFRules(mesh), f.Plist: PFLaunchDaemon(f.Anchor)} {
		got, err := readOptional(path)
		if err != nil {
			return false, err
		}
		if string(got) != want {
			return true, nil
		}
	}
	loaded, err := f.jobLoaded()
	return !loaded, err
}

// jobLoaded asks launchd, without root, whether it knows the boot job. Only
// `launchctl print`'s exit 113 means it does not; schedule.LaunchdLoaded owns
// that reading (#2731).
func (f *PF) jobLoaded() (bool, error) {
	return schedule.LaunchdLoaded(func(args ...string) (string, error) {
		out, err := f.Run(append([]string{"launchctl"}, args...)...)
		return string(out), err
	}, pfLaunchd)
}

// Apply refuses before changing anything when the session it runs in would be
// cut. Then it arms a root timer that flushes the anchor, loads the anchor,
// and installs the LaunchDaemon; Commit disarms the timer once the target's
// whole proof has passed. A run cut off half way, its session included, is
// undone by the timer.
func (f *PF) Apply(root Root) (func() error, error) {
	var undos []func() error
	undo := func() error {
		var errs []error
		for i := len(undos) - 1; i >= 0; i-- {
			errs = append(errs, undos[i]())
		}
		return errors.Join(errs...)
	}
	mesh, err := f.Mesh()
	if err != nil {
		return undo, err
	}
	if err := refuseCutSession(f.Session, f.Sudoed, mesh); err != nil {
		return undo, err
	}
	prevAnchor, err := readOptional(f.Anchor)
	if err != nil {
		return undo, err
	}
	prevPlist, err := readOptional(f.Plist)
	if err != nil {
		return undo, err
	}
	wasLoaded, err := f.jobLoaded()
	if err != nil {
		return undo, err
	}

	if err := f.arm(root); err != nil {
		return undo, err
	}
	undos = append(undos, func() error { return f.disarm(root) })

	if err := installRoot(root, f.Anchor, []byte(PFRules(mesh))); err != nil {
		return undo, err
	}
	undos = append(undos, func() error { return restoreRoot(root, f.Anchor, prevAnchor) })
	if err := asRoot(root, pfctl, "-E", "-a", PFAnchor, "-f", f.Anchor); err != nil {
		return undo, err
	}
	// Flush only our anchor: pf stays enabled for every other rule it holds.
	undos = append(undos, func() error { return asRoot(root, pfctl, "-a", PFAnchor, "-F", "all") })
	if err := pfLoaded(root); err != nil {
		return undo, err
	}

	if err := installRoot(root, f.Plist, []byte(PFLaunchDaemon(f.Anchor))); err != nil {
		return undo, err
	}
	undos = append(undos, func() error { return restoreRoot(root, f.Plist, prevPlist) })
	if wasLoaded {
		// A loaded job keeps the plist it was bootstrapped with.
		if err := asRoot(root, "launchctl", "bootout", pfLaunchd); err != nil {
			return undo, err
		}
	}
	if err := asRoot(root, "launchctl", "enable", pfLaunchd); err != nil {
		return undo, err
	}
	if err := asRoot(root, "launchctl", "bootstrap", "system", f.Plist); err != nil {
		return undo, err
	}
	if !wasLoaded {
		undos = append(undos, func() error {
			if err := asRoot(root, "launchctl", "bootout", pfLaunchd); err != nil {
				return err
			}
			return asRoot(root, "launchctl", "disable", pfLaunchd)
		})
	}
	return undo, nil
}

// Commit disarms the timer; Target calls it once the whole proof has passed.
// The timer flushes the anchor before anything else, so a kill that fails, or
// an anchor gone once the kill succeeded, means the proof outlived the timer:
// port 22 is unfenced and the run did not converge. Nothing bounds the proof
// in advance (ssh's ConnectTimeout bounds the connect, not the handshake), so
// Commit reads the outcome instead.
func (f *PF) Commit(root Root) error {
	if f.timer == "" {
		return nil
	}
	if out, err := root.Run("kill", f.timer); err != nil {
		return fmt.Errorf("the pf commit-confirm timer %s could not be stopped, so it fired before the proof finished or still will: %w: %s", f.timer, err, bytes.TrimSpace(out))
	}
	f.timer = ""
	if err := pfLoaded(root); err != nil {
		return fmt.Errorf("the pf commit-confirm timer fired before the proof finished: %w", err)
	}
	return nil
}

// Prove reads what it can without root: both files as rendered and the job
// loaded. The anchor's rules and pf's switch were read as root by Apply.
func (f *PF) Prove() error {
	differs, err := f.Differs()
	if err != nil {
		return err
	}
	if differs {
		return fmt.Errorf("pf does not read as converged after the apply (%s, %s, %s)", f.Anchor, f.Plist, pfLaunchd)
	}
	return nil
}

// arm starts, as root and detached from this process, the timer that flushes
// the anchor after pfConfirmSec seconds and disables the LaunchDaemon, so the
// next boot does not load it again. A run killed with its session, the one
// case nothing else could undo, is undone by it: the worst it leaves is port
// 22 fenced by sshd's AllowUsers alone, never a machine nobody can reach.
func (f *PF) arm(root Root) error {
	script := fmt.Sprintf("nohup /bin/sh -c 'sleep %d; %s -a %s -F all; launchctl bootout %s; launchctl disable %s' >/dev/null 2>&1 & echo $!",
		pfConfirmSec, pfctl, PFAnchor, pfLaunchd, pfLaunchd)
	out, err := root.Run("/bin/sh", "-c", script)
	if err != nil {
		return fmt.Errorf("arm the pf commit-confirm timer: %w: %s", err, bytes.TrimSpace(out))
	}
	pid := strings.TrimSpace(string(out))
	if _, err := strconv.Atoi(pid); err != nil {
		return fmt.Errorf("arm the pf commit-confirm timer: no pid in %q", pid)
	}
	f.timer = pid
	return nil
}

func (f *PF) disarm(root Root) error {
	if f.timer == "" {
		return nil
	}
	pid := f.timer
	f.timer = ""
	// The timer may have fired already; its flush is then the state we want.
	_, _ = root.Run("kill", pid)
	return nil
}

// pfLoaded asks pf, as root, whether it is enabled and holds our block rule.
func pfLoaded(root Root) error {
	info, err := root.Run(pfctl, "-s", "info")
	if err != nil {
		return fmt.Errorf("pfctl -s info: %w: %s", err, bytes.TrimSpace(info))
	}
	if !bytes.Contains(info, []byte("Status: Enabled")) {
		return fmt.Errorf("pf is not enabled after loading %s", PFAnchor)
	}
	rules, err := root.Run(pfctl, "-a", PFAnchor, "-s", "rules")
	if err != nil {
		return fmt.Errorf("pfctl -a %s -s rules: %w: %s", PFAnchor, err, bytes.TrimSpace(rules))
	}
	if !bytes.Contains(rules, []byte("block drop in quick proto tcp from any to any port = 22")) {
		return fmt.Errorf("anchor %s does not hold the port 22 block after loading:\n%s", PFAnchor, rules)
	}
	return nil
}

// refuseCutSession refuses an apply whose own ssh session arrived on an
// address the anchor blocks: loading it would cut the session, and on a
// machine reached only over ssh, the owner with it. $SSH_CONNECTION is
// "client-ip client-port server-ip server-port".
//
// A session the guard cannot read is refused, never waved through: sudo strips
// the variable, and an address that does not parse (a link-local one with its
// %zone among them) says nothing about where the session arrived.
func refuseCutSession(session string, sudoed bool, mesh net.IP) error {
	if session == "" {
		if sudoed {
			return errors.New("dotf runs under sudo, which strips $SSH_CONNECTION, so whether the pf anchor would cut this session is unknown: run dotf as your user; it asks for sudo itself")
		}
		return nil
	}
	f := strings.Fields(session)
	var server net.IP
	if len(f) == 4 {
		server = net.ParseIP(f[2])
	}
	if server == nil {
		return fmt.Errorf("could not parse the server address in $SSH_CONNECTION %q, so whether the pf anchor would cut this session is unknown: run the converge from the mesh (ssh to %s) or at the machine", session, mesh)
	}
	if server.IsLoopback() || server.Equal(mesh) {
		return nil
	}
	return fmt.Errorf("this session arrived on %s, which the pf anchor closes to port 22: run the converge from the mesh (ssh to %s) or at the machine", server, mesh)
}
