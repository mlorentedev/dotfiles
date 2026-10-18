package vm

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// guestRunScript runs one converge step in the guest as the test user.
//
//go:embed windows/guestrun.ps1
var guestRunScript []byte

const (
	// guestUser is the account autounattend.xml creates and logs on.
	guestUser = "dotf"
	// guestWork holds what a run copies in and what its steps write back.
	guestWork = `C:\Users\Public\dotf`
	// guestReport is the converge report of the test user's last run.
	guestReport = `C:\Users\` + guestUser + `\.local\state\dotfiles\converge\last.json`
)

// RunOptions is one run of the Windows guest.
type RunOptions struct {
	Machine Machine
	Source  string        // a zip of the ref to converge (git archive), copied into the guest
	Keep    bool          // leave the guest running for inspection
	Agent   time.Duration // until the guest agent answers and the test user is logged on
	Step    time.Duration // one converge
	Poll    time.Duration
	Out     io.Writer
}

// Run boots a throwaway overlay of clean-base, copies in the ref's archive,
// runs install.ps1 from it as the test user, and converges again: the first
// run must end with no failed step and the second must change nothing. The
// overlay is removed at the end unless Keep, and clean-base never changes.
func Run(h Host, o RunOptions) (err error) {
	l := o.Machine.Layout
	if running(h, l) {
		return errors.New("a Windows guest is already running: `dotf vm windows stop` stops it")
	}
	done, err := baseBuilt(l, false)
	if err != nil {
		return err
	}
	if !done {
		return fmt.Errorf("no clean-base at %s: run `dotf vm windows build` first", l.BaseDisk())
	}
	if err := errors.Join(removeIfPresent(l.Disk()), removeIfPresent(l.Vars())); err != nil {
		return err
	}
	defer func() {
		if !o.Keep {
			err = errors.Join(err, removeIfPresent(l.Disk()), removeIfPresent(l.Vars()))
		}
	}()
	if err := h.Img("create", "-q", "-f", "qcow2", "-b", l.BaseDisk(), "-F", "qcow2", l.Disk()); err != nil {
		return fmt.Errorf("qemu-img create: %w", err)
	}
	if err := copyFile(l.BaseVars(), l.Vars()); err != nil {
		return err
	}
	log, err := os.OpenFile(l.Log(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = log.Close() }()

	step(o.Out, "boot a throwaway overlay of clean-base (QEMU's output in %s)", l.Log())
	p, err := start(h, o.Machine, log)
	if err != nil {
		return err
	}
	defer func() {
		if o.Keep {
			step(o.Out, "kept: the guest is still running; `dotf vm windows stop` stops it")
			return
		}
		_ = p.Kill()
		select {
		case <-p.Done():
		case <-time.After(30 * time.Second):
		}
	}()
	q, err := agent(h, l, o.Agent, p)
	if err != nil {
		return err
	}
	defer func() { _ = q.Close() }()
	if err := waitLogon(q, o); err != nil {
		return err
	}

	step(o.Out, "copy the ref's archive into %s", guestWork)
	if _, err := q.Run("cmd.exe", []string{"/c", "mkdir", guestWork}); err != nil {
		return err
	}
	src, err := os.Open(o.Source)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	if err := q.WriteFile(guestWork+`\src.zip`, src); err != nil {
		return fmt.Errorf("copy the archive in: %w", err)
	}
	if err := q.WriteFile(guestWork+`\guestrun.ps1`, bytes.NewReader(guestRunScript)); err != nil {
		return fmt.Errorf("copy guestrun.ps1 in: %w", err)
	}

	if err := runStep(q, o, "first", "first converge: install.ps1 from the ref", Converged); err != nil {
		return err
	}
	if err := runStep(q, o, "second", "second converge: dotf converge", Idempotent); err != nil {
		return err
	}
	step(o.Out, "converged, and the second run changed nothing")
	return nil
}

// Stop stops a guest a run kept and removes its overlay. With no guest
// running it only removes what a kept run left.
func Stop(h Host, l Layout, out io.Writer) error {
	if conn, err := h.Dial(l.QMPSocket()); err == nil {
		// A build boots clean-base itself; stopping it would leave half a base.
		if done, _ := baseBuilt(l, false); !done {
			_ = conn.Close()
			return errors.New("the running guest is a clean-base build, not a run: let it finish")
		}
		q, err := NewQMP(conn)
		if err != nil {
			_ = conn.Close()
			return err
		}
		qerr := q.Quit()
		_ = q.Close()
		if qerr != nil {
			return fmt.Errorf("stop the guest: %w", qerr)
		}
		step(out, "stopped the Windows guest")
	} else {
		step(out, "no Windows guest is running")
	}
	return errors.Join(removeIfPresent(l.Disk()), removeIfPresent(l.Vars()))
}

// running reports whether a QEMU answers on the guest's QMP socket: a kept
// run, or a build.
func running(h Host, l Layout) bool {
	conn, err := h.Dial(l.QMPSocket())
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// waitLogon waits for autologon: a step runs in the test user's session, and
// the agent answers before that session exists.
func waitLogon(q *QGA, o RunOptions) error {
	deadline := time.Now().Add(o.Agent)
	for {
		r, err := q.Run("powershell.exe", []string{"-NoProfile", "-Command", "(Get-CimInstance Win32_ComputerSystem).UserName"})
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(r.Stdout)) != "" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not log on within %s", guestUser, o.Agent)
		}
		time.Sleep(o.Poll)
	}
}

// runStep runs guestrun.ps1 -Step name as the test user, through a scheduled
// task: the agent runs commands as LocalSystem, whose profile is not the one
// a person converges. It relays the step's output, then judges the step on
// its own converge report.
func runStep(q *QGA, o RunOptions, name, label string, check func(converge.RunReport) error) error {
	step(o.Out, "%s", label)
	// Judge this step on its own report: one that writes none must not pass
	// on the previous one's.
	if _, err := q.Run("cmd.exe", []string{"/c", "del", "/f", "/q", guestReport}); err != nil {
		return err
	}
	task := "dotf-" + name
	action := `powershell.exe -NoProfile -ExecutionPolicy Bypass -File ` + guestWork + `\guestrun.ps1 -Step ` + name
	for _, args := range [][]string{
		{"/create", "/tn", task, "/tr", action, "/sc", "once", "/st", "00:00", "/ru", guestUser, "/it", "/f"},
		{"/run", "/tn", task},
	} {
		r, err := q.Run("schtasks.exe", args)
		if err != nil {
			return err
		}
		if r.ExitCode != 0 {
			return fmt.Errorf("schtasks %s: exit %d: %s", args[0], r.ExitCode, strings.TrimSpace(string(r.Stdout)+string(r.Stderr)))
		}
	}

	code, werr := waitExit(q, o, guestWork+`\`+name+".exit")
	if out, err := q.ReadFile(guestWork + `\` + name + ".log"); err == nil {
		_, _ = o.Out.Write(bytes.TrimPrefix(out, []byte("\xef\xbb\xbf")))
	}
	if werr != nil {
		return fmt.Errorf("%s did not finish in %s: %w", label, o.Step, werr)
	}
	var runErr error
	if code != "0" {
		runErr = fmt.Errorf("%s: exit %s", label, code)
	}
	raw, err := q.ReadFile(guestReport)
	if err != nil {
		return errors.Join(runErr, fmt.Errorf("%s left no report: %w", label, err))
	}
	rep, err := converge.ParseRunReport(raw)
	if err != nil {
		return errors.Join(runErr, err)
	}
	return errors.Join(runErr, check(rep))
}

// waitExit polls for a step's exit file, which guestrun.ps1 writes last.
func waitExit(q *QGA, o RunOptions, path string) (string, error) {
	deadline := time.Now().Add(o.Step)
	for {
		b, err := q.ReadFile(path)
		if code := strings.TrimSpace(string(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")))); err == nil && code != "" {
			return code, nil
		}
		if time.Now().After(deadline) {
			if err == nil {
				err = errors.New("its exit file stayed empty")
			}
			return "", err
		}
		time.Sleep(o.Poll)
	}
}
