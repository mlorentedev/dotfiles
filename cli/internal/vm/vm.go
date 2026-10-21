// Package vm converges a throwaway virtual machine from a git archive of a
// committed ref and checks the result: Linux behaviour verified from a Mac, without a
// Linux box or a CI round trip.
package vm

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// LinuxTemplate is the Lima template of the Linux guest.
//
//go:embed linux.yaml
var LinuxTemplate []byte

// Limactl runs `limactl args...`. A nil stdin reads nothing.
type Limactl func(stdin io.Reader, stdout, stderr io.Writer, args ...string) error

// Options is one verification run.
type Options struct {
	Ref    string    // the committed ref Source was archived from, for the log
	Source string    // a tar.gz of Ref's tree under dotfiles/ (git archive --prefix=dotfiles/)
	Name   string    // the Lima instance, unique per run
	Keep   bool      // leave the instance running for inspection
	Token  string    // GITHUB_TOKEN for the guest's downloads; it reaches the guest on stdin only
	Out    io.Writer // progress and the guest's output
}

// hostShares are the filesystem types through which a guest sees a host
// directory: Lima's virtiofs, 9p and sshfs mounts, and network shares.
var hostShares = map[string]bool{"virtiofs": true, "9p": true, "fuse.sshfs": true, "sshfs": true, "nfs": true, "nfs4": true, "cifs": true, "smb3": true}

// withToken runs its arguments with GITHUB_TOKEN set from the first line of
// stdin, when there is one, and DOTFILES_REPO_DIR set to the working
// directory. The token is never an argument, so it is in no process listing
// and no Lima log, and never on the guest's disk.
const withToken = `IFS= read -r t || true; if [ -n "$t" ]; then export GITHUB_TOKEN="$t"; fi; unset t; export DOTFILES_REPO_DIR="$PWD"; exec "$@"`

// Verify boots the guest, checks it shares nothing with the host, converges a
// copy of o.Source with install.sh, and converges again: the first run must
// end with no failed step and the second must change nothing.
func Verify(l Limactl, o Options) (err error) {
	if o.Source == "" {
		return errors.New("no source archive to converge")
	}
	tmpl, err := os.CreateTemp("", "dotf-vm-*.yaml")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpl.Name()) }()
	if _, err := tmpl.Write(LinuxTemplate); err != nil {
		_ = tmpl.Close()
		return err
	}
	if err := tmpl.Close(); err != nil {
		return err
	}

	// Registered before the start: a start that fails partway (an image
	// download, provisioning) can leave the instance and its disk behind.
	defer func() {
		if o.Keep {
			step(o.Out, "kept: limactl shell %s; remove with limactl delete --force %s", o.Name, o.Name)
			return
		}
		if derr := l(nil, o.Out, o.Out, "delete", "--force", o.Name); derr != nil {
			err = errors.Join(err, fmt.Errorf("limactl delete %s: %w", o.Name, derr))
		}
	}()
	step(o.Out, "boot %s (Ubuntu 24.04, nothing shared with the host)", o.Name)
	if err := l(nil, o.Out, o.Out, "start", "--tty=false", "--name", o.Name, tmpl.Name()); err != nil {
		return fmt.Errorf("limactl start: %w", err)
	}

	capture := func(args ...string) (string, error) {
		var out bytes.Buffer
		err := l(nil, &out, o.Out, append([]string{"shell", o.Name, "--"}, args...)...)
		return strings.TrimSpace(out.String()), err
	}
	home, err := capture("printenv", "HOME")
	if err != nil || !path.IsAbs(home) {
		return fmt.Errorf("guest HOME %q: %v", home, err)
	}
	pid1, err := capture("ps", "-p", "1", "-o", "comm=")
	if err != nil {
		return err
	}
	mounts, err := capture("findmnt", "-rn", "-o", "TARGET,FSTYPE")
	if err != nil {
		return err
	}
	if err := Isolated(pid1, mounts); err != nil {
		return err
	}
	step(o.Out, "isolated: systemd is PID 1, no host directory mounted, HOME=%s", home)

	// limactl copy goes over ssh, so plain mode's no-mount isolation holds.
	// The archive extracts to dotfiles/ under ~/Projects.
	repo := home + "/Projects/dotfiles"
	archive := home + "/dotf-src.tar.gz"
	step(o.Out, "copy a git archive of %s into the guest", o.Ref)
	if err := l(nil, o.Out, o.Out, "copy", o.Source, o.Name+":"+archive); err != nil {
		return fmt.Errorf("limactl copy: %w", err)
	}
	for _, args := range [][]string{
		{"mkdir", "-p", home + "/Projects"},
		{"tar", "-xzf", archive, "-C", home + "/Projects"},
		{"rm", "-f", archive},
	} {
		if _, err := capture(args...); err != nil {
			return fmt.Errorf("%s: %w", strings.Join(args, " "), err)
		}
	}

	report := home + "/.local/state/dotfiles/converge/last.json"
	run := func(label string, check func(converge.RunReport) error, args ...string) error {
		step(o.Out, "%s", label)
		// Judge this run on its own report: a run that writes none must not
		// pass on the previous one's.
		if _, err := capture("rm", "-f", report); err != nil {
			return err
		}
		cmd := append([]string{"shell", "--workdir", repo, o.Name, "--", "bash", "-c", withToken, "_"}, args...)
		runErr := l(strings.NewReader(o.Token+"\n"), o.Out, o.Out, cmd...)
		raw, rerr := capture("cat", report)
		if rerr != nil {
			return errors.Join(runErr, fmt.Errorf("%s left no report: %w", label, rerr))
		}
		rep, perr := converge.ParseRunReport([]byte(raw))
		if perr != nil {
			return errors.Join(runErr, perr)
		}
		return errors.Join(runErr, check(rep))
	}
	if err := run("first converge: ./install.sh", Converged, "./install.sh"); err != nil {
		return err
	}
	if err := run("second converge: dotf converge", Idempotent, home+"/.local/bin/dotf", "converge"); err != nil {
		return err
	}
	step(o.Out, "converged, and the second run changed nothing")
	return nil
}

// Isolated checks the guest is a real init system with no host directory
// mounted, from `ps -p 1 -o comm=` and `findmnt -rn -o TARGET,FSTYPE`.
func Isolated(pid1, mounts string) error {
	if pid1 != "systemd" {
		return fmt.Errorf("PID 1 is %q, not systemd", pid1)
	}
	var shared []string
	for _, line := range strings.Split(mounts, "\n") {
		if f := strings.Fields(line); len(f) == 2 && hostShares[f[1]] {
			shared = append(shared, f[0]+" ("+f[1]+")")
		}
	}
	if len(shared) > 0 {
		return fmt.Errorf("the guest mounts a host directory: %s", strings.Join(shared, ", "))
	}
	return nil
}

// Converged accepts a first run that ended ok with no failed step.
func Converged(r converge.RunReport) error {
	var failed []string
	for _, e := range r.Entries {
		if e.Status == "failed" {
			failed = append(failed, e.Name)
		}
	}
	if r.Result != "ok" || len(failed) > 0 {
		return fmt.Errorf("the first converge did not succeed (result %s; failed: %s)", r.Result, strings.Join(failed, ", "))
	}
	return nil
}

// Idempotent accepts a second run that ended ok and changed nothing, naming
// the steps that did change: they are the non-idempotent ones.
func Idempotent(r converge.RunReport) error {
	if err := Converged(r); err != nil {
		return err
	}
	if r.Changed == 0 {
		return nil
	}
	var changing []string
	for _, e := range r.Entries {
		if e.Changes > 0 {
			changing = append(changing, fmt.Sprintf("%s (%d)", e.Name, e.Changes))
		}
	}
	return fmt.Errorf("the second converge changed %d thing(s): %s", r.Changed, strings.Join(changing, ", "))
}

func step(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, "==> "+format+"\n", a...)
}
