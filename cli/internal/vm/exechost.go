package vm

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
)

// ExecHost is the real Host: the QEMU the catalog installs.
type ExecHost struct {
	QEMU    string // qemu-system-aarch64
	QEMUImg string // qemu-img
}

// FindQEMU locates QEMU on PATH and the UEFI firmware and variable template
// its package ships under <prefix>/share/qemu.
func FindQEMU() (h ExecHost, firmware, varsTemplate string, err error) {
	if h.QEMU, err = exec.LookPath("qemu-system-aarch64"); err != nil {
		return h, "", "", fmt.Errorf("qemu-system-aarch64 is not on PATH: run `dotf converge --only tools`")
	}
	if h.QEMUImg, err = exec.LookPath("qemu-img"); err != nil {
		return h, "", "", fmt.Errorf("qemu-img is not on PATH: run `dotf converge --only tools`")
	}
	share := filepath.Join(filepath.Dir(filepath.Dir(h.QEMU)), "share", "qemu")
	firmware = filepath.Join(share, "edk2-aarch64-code.fd")
	varsTemplate = filepath.Join(share, "edk2-arm-vars.fd")
	for _, f := range []string{firmware, varsTemplate} {
		if _, err := os.Stat(f); err != nil {
			return h, "", "", fmt.Errorf("QEMU's UEFI firmware: %w", err)
		}
	}
	return h, firmware, varsTemplate, nil
}

func (h ExecHost) Start(args []string, log io.Writer) (Process, error) {
	c := exec.Command(h.QEMU, args...)
	c.Stdout, c.Stderr = log, log
	if err := c.Start(); err != nil {
		return nil, err
	}
	p := &execProc{c: c, done: make(chan error, 1)}
	go func() { p.done <- c.Wait() }()
	return p, nil
}

func (h ExecHost) Img(args ...string) error {
	out, err := exec.Command(h.QEMUImg, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

func (h ExecHost) Dial(socket string) (net.Conn, error) { return net.Dial("unix", socket) }

type execProc struct {
	c    *exec.Cmd
	done chan error
}

func (p *execProc) Done() <-chan error { return p.done }
func (p *execProc) Kill() error        { return p.c.Process.Kill() }
