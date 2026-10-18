package vm

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Host is what the Windows guest's lifecycle drives on the host machine.
type Host interface {
	// Start runs qemu-system-aarch64 with args, its output to log.
	Start(args []string, log io.Writer) (Process, error)
	// Img runs qemu-img with args.
	Img(args ...string) error
	// Dial connects to one of QEMU's unix sockets.
	Dial(socket string) (net.Conn, error)
}

// Process is a running QEMU.
type Process interface {
	Done() <-chan error // receives QEMU's exit, once
	Kill() error
}

// BuildOptions is one build of clean-base.
type BuildOptions struct {
	Machine      Machine
	VarsTemplate string // edk2-arm-vars.fd, copied to clean-base's UEFI variables
	Windows      Medium // WindowsISO
	Virtio       Medium // VirtioWin
	ISO          string // the Windows ISO: a file or a download link; empty uses the one already in place
	Client       *http.Client
	Rebuild      bool
	DiskSize     string        // qemu-img size, "64G"
	BootKeys     time.Duration // how long to answer the ISO's "press any key" prompt
	KeyEvery     time.Duration
	Install      time.Duration // the whole unattended install, reboots included
	Agent        time.Duration // until the guest agent answers on the check boot
	Out          io.Writer
}

// Build installs Windows unattended into clean-base, checks the result on a
// second boot through the guest agent, and makes clean-base read-only. A
// read-only clean-base is a complete one, so a second Build changes nothing.
func Build(h Host, o BuildOptions) (err error) {
	l := o.Machine.Layout
	if running(h, l) {
		return errors.New("a Windows guest is running: `dotf vm windows stop` stops it")
	}
	if done, err := baseBuilt(l, o.Rebuild); err != nil || done {
		if done {
			step(o.Out, "clean-base is built at %s; --rebuild replaces it", l.BaseDisk())
		}
		return err
	}
	if err := Ensure(o.Client, l.WindowsISO(), o.Windows, o.ISO, o.Out); err != nil {
		return err
	}
	if err := Ensure(o.Client, l.VirtioISO(), o.Virtio, o.Virtio.URL, o.Out); err != nil {
		return err
	}
	if err := WriteUnattend(l.UnattendDir()); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.QMPSocket()), 0o700); err != nil {
		return err
	}
	log, err := os.OpenFile(l.Log(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = log.Close() }()

	defer func() {
		if err != nil {
			_ = os.Remove(l.BaseDisk())
			_ = os.Remove(l.BaseVars())
		}
	}()
	if err := h.Img("create", "-q", "-f", "qcow2", l.BaseDisk(), o.DiskSize); err != nil {
		return fmt.Errorf("qemu-img create: %w", err)
	}
	if err := copyFile(o.VarsTemplate, l.BaseVars()); err != nil {
		return err
	}

	step(o.Out, "install Windows unattended (up to %s; QEMU's output in %s)", o.Install, l.Log())
	if err := install(h, o, log); err != nil {
		return err
	}
	step(o.Out, "check boot: the guest agent answers and no sshd is installed")
	if err := checkBoot(h, o, log); err != nil {
		return err
	}
	for _, f := range []string{l.BaseDisk(), l.BaseVars()} {
		if err := os.Chmod(f, 0o444); err != nil {
			return err
		}
	}
	step(o.Out, "clean-base ready: %s", l.BaseDisk())
	return nil
}

// baseBuilt reports whether a complete clean-base exists. A writable one is a
// build that stopped partway, which only --rebuild replaces.
func baseBuilt(l Layout, rebuild bool) (bool, error) {
	fi, err := os.Stat(l.BaseDisk())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if rebuild {
		return false, errors.Join(os.Remove(l.BaseDisk()), removeIfPresent(l.BaseVars()))
	}
	if fi.Mode().Perm()&0o222 != 0 {
		return false, fmt.Errorf("%s is from a build that did not finish: rerun with --rebuild", l.BaseDisk())
	}
	return true, nil
}

func install(h Host, o BuildOptions, log io.Writer) error {
	m := o.Machine
	m.Install = true
	p, err := start(h, m, log)
	if err != nil {
		return err
	}
	if err := pressKeys(h, o, p); err != nil {
		kill(p)
		return err
	}
	select {
	case err := <-p.Done():
		if err != nil {
			return fmt.Errorf("QEMU exited during the install: %w (see %s)", err, m.Layout.Log())
		}
		return nil
	case <-time.After(o.Install):
		shot := screenshot(h, m.Layout)
		kill(p)
		return fmt.Errorf("the install did not finish in %s; %s. In the guest, C:\\Windows\\Temp\\dotf-firstlogon.log holds the first logon's transcript", o.Install, shot)
	}
}

// kill ends QEMU and waits, bounded, for it to exit, so the next command
// never meets a dying QEMU that still holds the disk.
func kill(p Process) {
	_ = p.Kill()
	select {
	case <-p.Done():
	case <-time.After(30 * time.Second):
	}
}

// bootKey answers the ISO's prompt and is inert in Setup, whose window opens
// within the key window with its Cancel button focused: Enter or Space there
// asks to quit the install, an arrow key moves nothing on a lone button.
const bootKey = "down"

// pressKeys answers the Windows ISO's "press any key to boot from CD" prompt,
// which a headless guest otherwise lets time out into an empty disk.
func pressKeys(h Host, o BuildOptions, p Process) error {
	conn, err := dialWithin(h, o.Machine.Layout.QMPSocket(), 10*time.Second)
	if err != nil {
		return err
	}
	q, err := NewQMP(conn)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = q.Close() }()
	stop := time.After(o.BootKeys)
	for {
		if err := q.PressKey(bootKey); err != nil {
			return err
		}
		select {
		case <-stop:
			return nil
		case <-time.After(o.KeyEvery):
		}
	}
}

func screenshot(h Host, l Layout) string {
	conn, err := h.Dial(l.QMPSocket())
	if err != nil {
		return "no screenshot: " + err.Error()
	}
	q, err := NewQMP(conn)
	if err != nil {
		_ = conn.Close()
		return "no screenshot: " + err.Error()
	}
	defer func() { _ = q.Close() }()
	if err := q.Screenshot(l.Stuck()); err != nil {
		return "no screenshot: " + err.Error()
	}
	return "its screen is in " + l.Stuck()
}

// checkBoot boots clean-base once more without the install media and asks the
// guest agent, the only way a run reaches the guest, whether it is up and
// whether sshd is absent: a run verifies that dotf installs sshd (#2672).
func checkBoot(h Host, o BuildOptions, log io.Writer) (err error) {
	m := o.Machine
	m.Base = true
	p, err := start(h, m, log)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			kill(p)
		}
	}()
	q, err := agent(h, m.Layout, o.Agent, p)
	if err != nil {
		return err
	}
	defer func() { _ = q.Close() }()
	ver, err := q.Run("cmd.exe", []string{"/c", "ver"})
	if err != nil {
		return err
	}
	step(o.Out, "guest: %s", strings.TrimSpace(string(ver.Stdout)))
	sshd, err := q.Run("sc.exe", []string{"query", "sshd"})
	if err != nil {
		return err
	}
	// 1060: the service does not exist.
	if sshd.ExitCode != 1060 {
		return fmt.Errorf("clean-base must have no sshd service; sc query sshd exited %d: %s", sshd.ExitCode, strings.TrimSpace(string(sshd.Stdout)))
	}
	if err := q.Shutdown(); err != nil {
		return err
	}
	select {
	case err := <-p.Done():
		return err
	case <-time.After(5 * time.Minute):
		return errors.New("the guest did not power off within 5 minutes of guest-shutdown")
	}
}

// agent connects to the guest agent, retrying until it answers a sync: the
// channel exists from QEMU's start, the agent only once Windows has booted.
func agent(h Host, l Layout, within time.Duration, p Process) (*QGA, error) {
	deadline := time.Now().Add(within)
	for {
		conn, err := h.Dial(l.QGASocket())
		if err == nil {
			q := NewQGA(conn)
			q.Timeout = 5 * time.Second
			if err = q.Sync(); err == nil {
				q.Timeout = time.Minute
				return q, nil
			}
			_ = conn.Close()
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("the guest agent did not answer within %s: %v", within, err)
		}
		select {
		case perr := <-p.Done():
			return nil, fmt.Errorf("QEMU exited before the guest agent answered: %v", perr)
		case <-time.After(time.Second):
		}
	}
}

func start(h Host, m Machine, log io.Writer) (Process, error) {
	args, err := m.Args()
	if err != nil {
		return nil, err
	}
	p, err := h.Start(args, log)
	if err != nil {
		return nil, fmt.Errorf("start QEMU: %w", err)
	}
	return p, nil
}

func dialWithin(h Host, socket string, within time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(within)
	for {
		conn, err := h.Dial(socket)
		if err == nil || time.Now().After(deadline) {
			return conn, err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, cerr := io.Copy(out, in)
	return errors.Join(cerr, out.Close())
}

func removeIfPresent(p string) error {
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
