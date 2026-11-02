package vm

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// guestExecScript runs one exec in the guest as the test user.
//
//go:embed windows/guestexec.ps1
var guestExecScript []byte

const utf8BOM = "\xef\xbb\xbf"

// guestTimedOut is what guestexec.ps1 writes to exit once it killed a command.
const guestTimedOut = "timeout"

// guestRequest is what guestexec.ps1 reads: the program never travels on a
// command line, so no deny glob or task listing sees it (ADR-055 decision 4).
type guestRequest struct {
	File      string `json:"file"`
	Args      string `json:"args"` // one Windows command line
	TimeoutMS int64  `json:"timeout_ms"`
}

// serving answers whether the guest can take the call, and the refusal if not.
func (s WindowsSessions) serving(name string, start time.Time) (Record, bool) {
	holder, err := s.holder()
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error()), false
	}
	if ValidSessionName(name) != nil || holder != name {
		return s.rec(name, StatusNoSession, start, fmt.Sprintf("no session %s", name)), false
	}
	if !running(s.H, s.Machine.Layout) {
		return s.rec(name, StatusStale, start, fmt.Sprintf("session %s has no running guest: `dotf vm windows down --session %s` clears it", name, name)), false
	}
	return Record{}, true
}

// connect reaches the running guest's agent. The socket takes one client, so a
// second call while one is in flight fails here rather than interleaving.
func (s WindowsSessions) connect() (*QGA, error) {
	conn, err := s.H.Dial(s.Machine.Layout.QGASocket())
	if err != nil {
		return nil, fmt.Errorf("reach the guest agent: %w", err)
	}
	q := NewQGA(conn)
	q.Timeout = 5 * time.Second
	if err := q.Sync(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("the guest agent did not answer (is another call running?): %w", err)
	}
	q.Timeout = time.Minute
	return q, nil
}

// Exec runs one command as the test user and reports it in one record. The
// agent runs commands as LocalSystem, so it goes through a scheduled task, and
// the output arrives when the command ends. No stdin and no credential reach a
// Windows guest (ADR-055 decisions 4 and 12): both are refused, never dropped.
func (s WindowsSessions) Exec(ctx context.Context, o ExecOptions) Record {
	start := time.Now()
	if o.Stdin != nil {
		return s.rec(o.Session, StatusUnsupported, start, "the Windows guest has no stdin: `cp` the data in and read it from there")
	}
	if o.Secrets != nil {
		return s.rec(o.Session, StatusUnsupported, start, "a Windows session receives no credential")
	}
	if refused, ok := s.serving(o.Session, start); !ok {
		return refused
	}
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	q, err := s.connect()
	if err != nil {
		return s.rec(o.Session, StatusFailed, start, err.Error())
	}
	defer func() { _ = q.Close() }()

	id := fmt.Sprintf("exec-%d", s.Now().UnixNano())
	dir := guestWork + `\` + id
	task := "dotf-" + id
	remove := func() {
		_, _ = q.Run("schtasks.exe", []string{"/delete", "/tn", task, "/f"})
		_, _ = q.Run("cmd.exe", []string{"/c", "rmdir", "/s", "/q", dir})
	}
	if err := s.launch(q, o, dir, task); err != nil {
		remove()
		return s.rec(o.Session, StatusFailed, start, err.Error())
	}
	code, werr := pollExit(ctx, q, s.Poll, dir+`\exit`)
	timedOut := code == guestTimedOut || (werr != nil && errors.Is(ctx.Err(), context.DeadlineExceeded))
	settled := werr == nil
	if werr != nil && timedOut {
		// The guest's own limit started after the host's: give it the moment it
		// needs to end the command, so the record holds the output and the
		// directory is not pulled from under a process that still writes it.
		grace, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.Grace)
		defer cancel()
		_, gerr := pollExit(grace, q, s.Poll, dir+`\exit`)
		settled = gerr == nil
	}
	r := s.collect(q, o, dir, start)
	if settled {
		remove()
	}
	switch {
	case timedOut:
		r.Status = StatusTimeout
		r.Stderr += fmt.Sprintf("dotf: timed out after %s\n", o.Timeout)
	case werr != nil:
		r.Status = StatusFailed
		r.Stderr += werr.Error() + "\n"
	default:
		n, err := strconv.Atoi(code)
		if err != nil {
			r.Status = StatusFailed
			r.Stderr += fmt.Sprintf("the guest reported %q, not an exit code\n", code)
			break
		}
		r.Exit = &n
	}
	if !settled {
		r.Stderr += "dotf: the guest command may still be running\n"
	}
	return r
}

// launch writes the request into the call's own directory and starts the task.
func (s WindowsSessions) launch(q *QGA, o ExecOptions, dir, task string) error {
	if _, err := q.Run("cmd.exe", []string{"/c", "mkdir", dir}); err != nil {
		return err
	}
	req := guestRequest{File: "", TimeoutMS: o.Timeout.Milliseconds()}
	if o.Script != "" {
		f, err := os.Open(o.Script)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		guest := dir + `\script.ps1`
		// With no BOM, Windows PowerShell reads a script as the ANSI code page.
		if err := q.WriteFile(guest, io.MultiReader(strings.NewReader(utf8BOM), f)); err != nil {
			return fmt.Errorf("copy the script in: %w", err)
		}
		req.File = "powershell.exe"
		req.Args = windowsCommandLine([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", guest})
	} else {
		req.File, req.Args = o.Argv[0], windowsCommandLine(o.Argv[1:])
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if err := q.WriteFile(dir+`\request.json`, bytes.NewReader(body)); err != nil {
		return fmt.Errorf("copy the request in: %w", err)
	}
	if err := q.WriteFile(guestWork+`\guestexec.ps1`, bytes.NewReader(guestExecScript)); err != nil {
		return fmt.Errorf("copy guestexec.ps1 in: %w", err)
	}
	var flags []string
	if o.Elevated {
		flags = []string{"/rl", "highest"}
	}
	return startTask(q, task, `powershell.exe -NoProfile -ExecutionPolicy Bypass -File `+guestWork+`\guestexec.ps1 -Dir `+dir, flags...)
}

// collect reads what the command left, as far as it got. A stream that never
// appeared is empty: a program that could not start has only stderr.
func (s WindowsSessions) collect(q *QGA, o ExecOptions, dir string, start time.Time) Record {
	var lost error
	stream := func(name string, w io.Writer) *capped {
		c := &capped{}
		plain := &lossy{w: w, err: &lost}
		// A stream that never appeared is empty; one that broke midway is a loss.
		if err := q.CopyFile(dir+`\`+name, &skipBOM{w: io.MultiWriter(c, plain)}); err != nil && c.buf.Len() > 0 {
			c.over, lost = true, err
		}
		return c
	}
	out, errOut := stream("stdout", o.Stdout), stream("stderr", o.Stderr)
	r := s.rec(o.Session, StatusOK, start, errOut.buf.String())
	r.Stdout = out.buf.String()
	r.Truncated = out.over || errOut.over
	if lost != nil {
		r.Stderr += fmt.Sprintf("dotf: writing the guest's output: %v\n", lost)
	}
	return r
}

// lossy keeps a failed writer from stopping the record beside it: it remembers
// the first error and drops the rest.
type lossy struct {
	w   io.Writer
	err *error
}

func (l *lossy) Write(p []byte) (int, error) {
	if *l.err == nil {
		_, *l.err = l.w.Write(p)
	}
	return len(p), nil
}

// skipBOM drops the byte order mark a stream starts with. QGA chunks are far
// larger than the mark, so it is whole in the first write.
type skipBOM struct {
	w       io.Writer
	started bool
}

func (s *skipBOM) Write(p []byte) (int, error) {
	n := len(p)
	if !s.started {
		s.started, p = true, bytes.TrimPrefix(p, []byte(utf8BOM))
	}
	_, err := s.w.Write(p)
	return n, err
}

// windowsCommandLine joins args the way CommandLineToArgvW splits them: a
// backslash is literal unless a quote follows it.
func windowsCommandLine(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = quoteWindowsArg(a)
	}
	return strings.Join(quoted, " ")
}

func quoteWindowsArg(a string) string {
	if a != "" && !strings.ContainsAny(a, " \t\"") {
		return a
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, r := range a {
		switch r {
		case '\\':
			slashes++
			continue
		case '"':
			b.WriteString(strings.Repeat(`\`, 2*slashes+1))
		default:
			b.WriteString(strings.Repeat(`\`, slashes))
		}
		slashes = 0
		b.WriteRune(r)
	}
	b.WriteString(strings.Repeat(`\`, 2*slashes))
	b.WriteByte('"')
	return b.String()
}
