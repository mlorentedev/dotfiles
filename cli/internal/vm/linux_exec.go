package vm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/redact"
)

const (
	// maxRecordOutput bounds each stream in the record; the plain stream is not
	// bounded, and Record.Truncated says the record lost the rest.
	maxRecordOutput = 1 << 20
	// minRedactable is the shortest value redact.New keys on.
	minRedactable = 6
	// removeLimit bounds the removal of a temporary script, which runs after the
	// call's own deadline may have passed.
	removeLimit = 10 * time.Second
)

// forwardEnv exports one line of stdin per variable named before `--`, then
// execs the rest of argv, which inherits what is left of stdin. `read` takes a
// pipe byte by byte, so nothing of the caller's stdin is consumed. The names are
// in argv; the values never are, so they are in no process listing and no Lima
// log, and never on the guest's disk (ADR-055 decision 12).
const forwardEnv = `while [ "$1" != -- ]; do IFS= read -r v || true; export "$1=$v"; shift; done; shift; unset v; exec "$@"`

// ExecOptions is one `exec` call. Stdout and Stderr receive the guest's streams
// as they arrive, masked, and must not be nil.
type ExecOptions struct {
	Session string
	Argv    []string  // run with no shell; empty when Script is set
	Script  string    // a host file, copied in and run by sh
	Stdin   io.Reader // the caller's, forwarded as it is
	// Secrets resolves the KEY=value pairs to forward. It is called only once a
	// guest can serve the call, so a refusal never costs a decryption.
	Secrets func() ([]string, error)
	Timeout time.Duration // 0 is none
	Stdout  io.Writer
	Stderr  io.Writer
}

// capped keeps the first maxRecordOutput bytes and swallows the rest, so it
// never stops the writer it sits beside.
type capped struct {
	buf  bytes.Buffer
	over bool
}

func (c *capped) Write(p []byte) (int, error) {
	kept := p
	if room := maxRecordOutput - c.buf.Len(); len(p) > room {
		kept, c.over = p[:room], true
	}
	c.buf.Write(kept)
	return len(p), nil
}

// serving answers whether a guest can take the call, and the refusal if not.
func (s LinuxSessions) serving(ctx context.Context, name string, start time.Time) (Record, bool) {
	if ValidSessionName(name) != nil || !s.held(name) {
		return s.rec(name, StatusNoSession, start, fmt.Sprintf("no session %s", name)), false
	}
	found, err := s.instances(ctx)
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error()), false
	}
	if found[instanceOf(name)].Status != "Running" {
		return s.rec(name, StatusStale, start, fmt.Sprintf("session %s has no running guest: `dotf vm linux down --session %s` clears it", name, name)), false
	}
	return Record{}, true
}

// Exec runs one command in the session and reports it in one record. What the
// guest prints is masked for every value forwarded (ADR-055 decision 13) before
// it reaches the record or the plain writers.
func (s LinuxSessions) Exec(ctx context.Context, o ExecOptions) Record {
	start := time.Now()
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	if refused, ok := s.serving(ctx, o.Session, start); !ok {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			refused.Status = StatusTimeout
		}
		return refused
	}
	pairs, err := forwardable(o.Secrets)
	if err != nil {
		return s.rec(o.Session, StatusFailed, start, err.Error())
	}
	var outRec, errRec capped
	stdout := redact.New(io.MultiWriter(&outRec, o.Stdout), pairs)
	stderr := redact.New(io.MultiWriter(&errRec, o.Stderr), pairs)
	runErr := s.guestRun(ctx, o, pairs, stdout, stderr)
	_ = stdout.Flush()
	_ = stderr.Flush()

	r := s.rec(o.Session, StatusOK, start, errRec.buf.String())
	r.Stdout = outRec.buf.String()
	r.Truncated = outRec.over || errRec.over
	settle(ctx, &r, runErr, o.Timeout)
	return r
}

// forwardable resolves the secrets and refuses a value that could not be masked
// or would not fit on its line of stdin: forwarding it would break the masking
// in silence. The error names the variable, never the value.
func forwardable(resolve func() ([]string, error)) ([]string, error) {
	if resolve == nil {
		return nil, nil
	}
	pairs, err := resolve()
	if err != nil {
		return nil, err
	}
	for _, kv := range pairs {
		name, value, _ := strings.Cut(kv, "=")
		if len(value) < minRedactable || strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("secret %s cannot be forwarded: it must be one line of at least %d bytes, or its redaction would not hold", name, minRedactable)
		}
	}
	return pairs, nil
}

// settle turns how the call ended into status and exit. The deadline is read
// first because a killed limactl also reports an exit status; a code below zero
// is a signal to limactl, not a code of the guest's.
func settle(ctx context.Context, r *Record, err error, limit time.Duration) {
	var coded interface{ ExitCode() int }
	switch {
	case err == nil:
		code := 0
		r.Exit = &code
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		r.Status = StatusTimeout
		r.Stderr += fmt.Sprintf("dotf: timed out after %s\n", limit)
	case errors.As(err, &coded) && coded.ExitCode() >= 0:
		code := coded.ExitCode()
		r.Exit = &code
	default:
		r.Status = StatusFailed
		r.Stderr += err.Error() + "\n"
	}
}

// guestRun builds the guest's command line and runs it. The outermost layer is
// the guest's own timeout, because killing limactl leaves the command running
// in a session that outlives the call. The host deadline starts first, so it is
// the one that fires, and a command that exits 124 by itself is not a timeout.
func (s LinuxSessions) guestRun(ctx context.Context, o ExecOptions, pairs []string, stdout, stderr io.Writer) error {
	instance := instanceOf(o.Session)
	argv := o.Argv
	if o.Script != "" {
		guest, remove, err := s.pushScript(ctx, instance, o.Script, stderr)
		if err != nil {
			return err
		}
		defer remove()
		argv = []string{"sh", guest}
	}
	stdin := o.Stdin
	if len(pairs) > 0 {
		var names []string
		var lines strings.Builder
		for _, kv := range pairs {
			name, value, _ := strings.Cut(kv, "=")
			names = append(names, name)
			lines.WriteString(value + "\n")
		}
		argv = append(append(append([]string{"bash", "-c", forwardEnv, "_"}, names...), "--"), argv...)
		stdin = io.MultiReader(strings.NewReader(lines.String()), stdin)
	}
	if o.Timeout > 0 {
		argv = append([]string{"timeout", "-k", "5", strconv.FormatFloat(o.Timeout.Seconds(), 'f', 3, 64)}, argv...)
	}
	return s.L(ctx, stdin, stdout, stderr, append([]string{"shell", instance, "--"}, argv...)...)
}

// pushScript copies a host file to a temporary guest path. Its errors are flat
// strings, so limactl's exit status can never read as the script's. The removal
// survives the call's deadline with one of its own; what it fails on, limactl
// says on stderr.
func (s LinuxSessions) pushScript(ctx context.Context, instance, host string, stderr io.Writer) (string, func(), error) {
	var name bytes.Buffer
	if err := s.L(ctx, nil, &name, stderr, "shell", instance, "--", "mktemp", "/tmp/dotf-exec-XXXXXX"); err != nil {
		return "", nil, fmt.Errorf("mktemp in the guest: %v", err)
	}
	guest := strings.TrimSpace(name.String())
	remove := func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), removeLimit)
		defer cancel()
		_ = s.L(ctx, nil, io.Discard, stderr, "shell", instance, "--", "rm", "-f", guest)
	}
	if err := s.L(ctx, nil, io.Discard, stderr, "copy", copyBackend, host, instance+":"+guest); err != nil {
		remove()
		return "", nil, fmt.Errorf("limactl copy %s: %v", host, err)
	}
	return guest, remove, nil
}
