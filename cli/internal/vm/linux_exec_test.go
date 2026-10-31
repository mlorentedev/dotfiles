package vm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A synthetic value: tests never touch a real store (lesson 261).
const fakeSecret = "s3cr3t-synthetic-value"

// guestExit is what a real *exec.ExitError answers.
type guestExit int

func (g guestExit) Error() string { return "exit status" }
func (g guestExit) ExitCode() int { return int(g) }

// heldSession is a store with one running session, x.
func heldSession(t *testing.T, f *listLima) LinuxSessions {
	t.Helper()
	s := newSessions(t, f)
	if f.instances == nil {
		f.instances = map[string]string{}
	}
	if r := s.Up(context.Background(), "x", io.Discard, io.Discard); r.Status != StatusOK {
		t.Fatalf("Up = %+v", r)
	}
	f.calls = nil
	return s
}

func execOpts(argv ...string) ExecOptions {
	return ExecOptions{Session: "x", Argv: argv, Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard}
}

func shellCalls(f *listLima) []string {
	var calls []string
	for _, c := range f.calls {
		if !strings.HasPrefix(c, "list") {
			calls = append(calls, c)
		}
	}
	return calls
}

// AC2: argv reaches limactl as argv, and the record carries both streams while
// the plain writers receive them as they arrive.
func TestExecRunsArgvAndRecordsBothStreams(t *testing.T) {
	f := &listLima{guest: func(_ context.Context, _ io.Reader, out, errw io.Writer, _ []string) error {
		_, _ = io.WriteString(out, "to-stdout\n")
		_, _ = io.WriteString(errw, "to-stderr\n")
		return nil
	}}
	s := heldSession(t, f)
	var out, errw bytes.Buffer
	o := execOpts("printf", "%s", "a b", "$HOME")
	o.Stdout, o.Stderr = &out, &errw
	r := s.Exec(context.Background(), o)
	if r.Status != StatusOK || r.Exit == nil || *r.Exit != 0 || r.ProcessCode() != 0 || r.OS != "linux" || r.Session != "x" {
		t.Fatalf("Exec = %+v, want ok, exit 0", r)
	}
	if r.Stdout != "to-stdout\n" || r.Stderr != "to-stderr\n" || out.String() != r.Stdout || errw.String() != r.Stderr {
		t.Errorf("record %q/%q, plain %q/%q", r.Stdout, r.Stderr, out.String(), errw.String())
	}
	if got := shellCalls(f); len(got) != 1 || got[0] != "shell dotf-x -- printf %s a b $HOME" {
		t.Errorf("calls = %q, want the argv with no shell around it", got)
	}
}

// AC2: a guest command that fails is ok with its own exit; only dotf's failure
// to run it is failed (ADR-055 decision 8).
func TestExecStatusTable(t *testing.T) {
	code := func(n int) *int { return &n }
	for _, c := range []struct {
		name     string
		err      error
		status   Status
		exit     *int
		procCode int
	}{
		{"guest exits 1", guestExit(1), StatusOK, code(1), 1},
		{"guest exits 3", guestExit(3), StatusOK, code(3), 1},
		{"limactl killed by a signal", guestExit(-1), StatusFailed, nil, 1},
		{"limactl cannot start", errors.New("exec: not found"), StatusFailed, nil, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := &listLima{guest: func(context.Context, io.Reader, io.Writer, io.Writer, []string) error { return c.err }}
			r := heldSession(t, f).Exec(context.Background(), execOpts("x"))
			if r.Status != c.status || r.ProcessCode() != c.procCode {
				t.Errorf("Exec = %+v, want %s, process code %d", r, c.status, c.procCode)
			}
			if (r.Exit == nil) != (c.exit == nil) || (r.Exit != nil && *r.Exit != *c.exit) {
				t.Errorf("exit = %v, want %v", r.Exit, c.exit)
			}
		})
	}
}

// AC9: a call no guest can serve never reaches `limactl shell`, and never
// resolves a secret.
func TestExecRefusesBeforeTheGuest(t *testing.T) {
	boom := func(context.Context, io.Reader, io.Writer, io.Writer, []string) error {
		t.Error("a refusal reached the guest")
		return nil
	}
	secrets := func() ([]string, error) {
		t.Error("a refusal resolved a secret")
		return nil, nil
	}
	t.Run("unknown name", func(t *testing.T) {
		o := execOpts("true")
		o.Session, o.Secrets = "ghost", secrets
		r := newSessions(t, &listLima{instances: map[string]string{}, guest: boom}).Exec(context.Background(), o)
		if r.Status != StatusNoSession || r.ProcessCode() != 3 {
			t.Errorf("Exec = %+v, want no-session, exit 3", r)
		}
	})
	for name, gone := range map[string]func(*listLima){
		"guest gone":    func(f *listLima) { delete(f.instances, "dotf-x") },
		"guest stopped": func(f *listLima) { f.instances["dotf-x"] = "Stopped" },
	} {
		t.Run(name, func(t *testing.T) {
			f := &listLima{guest: boom}
			s := heldSession(t, f)
			gone(f)
			o := execOpts("true")
			o.Secrets = secrets
			r := s.Exec(context.Background(), o)
			if r.Status != StatusStale || r.ProcessCode() != 2 || !strings.Contains(r.Stderr, "down --session x") {
				t.Errorf("Exec = %+v, want stale, exit 2, naming down", r)
			}
		})
	}
}

// AC11: a timeout is status timeout, exit null, process code 1; and the guest
// runs under its own timeout, because killing limactl leaves the command
// running (measured on Lima 2.2.1).
func TestExecTimeout(t *testing.T) {
	f := &listLima{guest: func(ctx context.Context, _ io.Reader, _, _ io.Writer, _ []string) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	s := heldSession(t, f)
	o := execOpts("sleep", "60")
	o.Timeout = 20 * time.Millisecond
	r := s.Exec(context.Background(), o)
	if r.Status != StatusTimeout || r.Exit != nil || r.ProcessCode() != 1 {
		t.Fatalf("Exec = %+v, want timeout, exit null, process code 1", r)
	}
	if got := shellCalls(f); len(got) != 1 || got[0] != "shell dotf-x -- timeout -k 5 0.020 sleep 60" {
		t.Errorf("calls = %q, want the command under the guest's timeout", got)
	}
}

// The limit covers the whole call: a `limactl list` that stalls is a timeout too.
func TestExecTimeoutCoversTheLivenessCheck(t *testing.T) {
	s := heldSession(t, &listLima{})
	s.L = func(ctx context.Context, _ io.Reader, _, _ io.Writer, _ ...string) error {
		<-ctx.Done()
		return ctx.Err()
	}
	o := execOpts("true")
	o.Timeout = 20 * time.Millisecond
	if r := s.Exec(context.Background(), o); r.Status != StatusTimeout || r.ProcessCode() != 1 {
		t.Errorf("Exec = %+v, want timeout", r)
	}
}

// The temporary script is removed after a timeout, and that removal has its
// own bound: a guest that stopped answering must not hang the call.
func TestExecScriptRemovalOutlivesTheTimeoutAndIsBounded(t *testing.T) {
	removed := false
	f := &listLima{guest: func(ctx context.Context, _ io.Reader, out, _ io.Writer, args []string) error {
		call := strings.Join(args, " ")
		switch {
		case strings.Contains(call, "mktemp"):
			_, _ = io.WriteString(out, "/tmp/dotf-exec-abc\n")
		case strings.Contains(call, " sh "):
			<-ctx.Done()
			return ctx.Err()
		case strings.Contains(call, "rm -f"):
			_, bounded := ctx.Deadline()
			removed = bounded && ctx.Err() == nil
		}
		return nil
	}}
	o := execOpts()
	o.Script, o.Timeout = "s.sh", 20*time.Millisecond
	if r := heldSession(t, f).Exec(context.Background(), o); r.Status != StatusTimeout {
		t.Fatalf("Exec = %+v, want timeout", r)
	}
	if !removed {
		t.Error("the removal did not run with a live, bounded context")
	}
}

// A command that ends on its own inside the limit is its own result.
func TestExecInsideTheTimeoutIsNotATimeout(t *testing.T) {
	f := &listLima{guest: func(context.Context, io.Reader, io.Writer, io.Writer, []string) error { return guestExit(2) }}
	o := execOpts("false")
	o.Timeout = time.Minute
	r := heldSession(t, f).Exec(context.Background(), o)
	if r.Status != StatusOK || r.Exit == nil || *r.Exit != 2 {
		t.Errorf("Exec = %+v, want ok, exit 2", r)
	}
}

// AC3: the caller's stdin reaches the guest.
func TestExecForwardsStdin(t *testing.T) {
	f := &listLima{guest: func(_ context.Context, in io.Reader, out, _ io.Writer, _ []string) error {
		_, err := io.Copy(out, in)
		return err
	}}
	o := execOpts("cat")
	o.Stdin = strings.NewReader("piped\n")
	if r := heldSession(t, f).Exec(context.Background(), o); r.Stdout != "piped\n" {
		t.Errorf("Exec = %+v, want the guest to read the caller's stdin", r)
	}
}

// AC7: the value travels on stdin, ahead of the caller's stdin, never in argv;
// and a guest that echoes it yields a masked record and masked plain output,
// even when the value arrives split across writes.
func TestExecForwardsASecretAndMasksIt(t *testing.T) {
	var argv string
	f := &listLima{guest: func(_ context.Context, in io.Reader, out, errw io.Writer, args []string) error {
		argv = strings.Join(args, " ")
		b, err := io.ReadAll(in)
		if err != nil {
			return err
		}
		if want := fakeSecret + "\npiped\n"; string(b) != want {
			t.Errorf("guest stdin = %q, want %q", b, want)
		}
		_, _ = io.WriteString(out, "got "+fakeSecret[:7])
		_, _ = io.WriteString(out, fakeSecret[7:]+"\n")
		_, _ = io.WriteString(errw, fakeSecret)
		return nil
	}}
	var plain, plainErr bytes.Buffer
	o := execOpts("env")
	o.Stdin, o.Stdout, o.Stderr = strings.NewReader("piped\n"), &plain, &plainErr
	o.Secrets = func() ([]string, error) { return []string{"API_TOKEN=" + fakeSecret}, nil }
	r := heldSession(t, f).Exec(context.Background(), o)
	if r.Status != StatusOK {
		t.Fatalf("Exec = %+v", r)
	}
	for name, text := range map[string]string{"record stdout": r.Stdout, "record stderr": r.Stderr, "plain stdout": plain.String(), "plain stderr": plainErr.String(), "argv": argv} {
		if strings.Contains(text, fakeSecret) || strings.Contains(text, fakeSecret[:7]) {
			t.Errorf("%s leaks the value: %q", name, text)
		}
	}
	if r.Stdout != "got [REDACTED:API_TOKEN]\n" || r.Stderr != "[REDACTED:API_TOKEN]" {
		t.Errorf("record = %q / %q", r.Stdout, r.Stderr)
	}
	if !strings.Contains(argv, "API_TOKEN -- env") {
		t.Errorf("argv = %q, want the variable's name ahead of the command", argv)
	}
}

// AC7: piped stdin stays the caller's: with no secret forwarded, nothing is masked.
func TestExecDoesNotRedactPlainStdin(t *testing.T) {
	f := &listLima{guest: func(_ context.Context, in io.Reader, out, _ io.Writer, _ []string) error {
		_, err := io.Copy(out, in)
		return err
	}}
	o := execOpts("cat")
	o.Stdin = strings.NewReader(fakeSecret)
	if r := heldSession(t, f).Exec(context.Background(), o); r.Stdout != fakeSecret {
		t.Errorf("Stdout = %q, want the caller's bytes untouched", r.Stdout)
	}
}

// A value redaction cannot key on, or that would not fit on stdin's one line,
// is refused: forwarding it would break AC7 in silence.
func TestExecRefusesASecretItCouldNotMask(t *testing.T) {
	for _, kv := range []string{"SHORT=abc", "MULTI=first-line\nsecond-line"} {
		f := &listLima{guest: func(context.Context, io.Reader, io.Writer, io.Writer, []string) error {
			t.Error("the value reached the guest")
			return nil
		}}
		o := execOpts("true")
		o.Secrets = func() ([]string, error) { return []string{kv}, nil }
		r := heldSession(t, f).Exec(context.Background(), o)
		if r.Status != StatusFailed || strings.Contains(r.Stderr, "abc") || strings.Contains(r.Stderr, "first-line") {
			t.Errorf("Exec(%q) = %+v, want failed, naming no value", kv, r)
		}
	}
}

func TestExecFailsWhenASecretDoesNotResolve(t *testing.T) {
	f := &listLima{guest: func(context.Context, io.Reader, io.Writer, io.Writer, []string) error {
		t.Error("ran without its secret")
		return nil
	}}
	o := execOpts("true")
	o.Secrets = func() ([]string, error) { return nil, errors.New("store locked") }
	if r := heldSession(t, f).Exec(context.Background(), o); r.Status != StatusFailed || !strings.Contains(r.Stderr, "store locked") {
		t.Errorf("Exec = %+v, want failed naming the cause", r)
	}
}

func scriptLima(t *testing.T, fail string) *listLima {
	t.Helper()
	return &listLima{guest: func(_ context.Context, _ io.Reader, out, _ io.Writer, args []string) error {
		call := strings.Join(args, " ")
		switch {
		case strings.Contains(call, "mktemp"):
			_, _ = io.WriteString(out, "/tmp/dotf-exec-abc\n")
		case fail != "" && strings.HasPrefix(call, fail):
			return guestExit(1)
		}
		return nil
	}}
}

// AC2: a script is copied in and run by sh, stdin stays the caller's, and it
// does not outlive the call.
func TestExecScript(t *testing.T) {
	script := filepath.Join(t.TempDir(), "s.sh")
	if err := os.WriteFile(script, []byte("true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := scriptLima(t, "")
	o := execOpts()
	o.Script = script
	if r := heldSession(t, f).Exec(context.Background(), o); r.Status != StatusOK {
		t.Fatalf("Exec = %+v", r)
	}
	want := []string{
		"shell dotf-x -- mktemp /tmp/dotf-exec-XXXXXX",
		"copy --backend=scp " + script + " dotf-x:/tmp/dotf-exec-abc",
		"shell dotf-x -- sh /tmp/dotf-exec-abc",
		"shell dotf-x -- rm -f /tmp/dotf-exec-abc",
	}
	if got := shellCalls(f); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The wrappers nest in one order: the guest's timeout outside, so it stops
// everything; the secret's bash next, so the script inherits the variable.
func TestExecWrapsScriptSecretAndTimeoutInOrder(t *testing.T) {
	f := scriptLima(t, "")
	o := execOpts()
	o.Script, o.Timeout = "s.sh", time.Minute
	o.Secrets = func() ([]string, error) { return []string{"API_TOKEN=" + fakeSecret}, nil }
	if r := heldSession(t, f).Exec(context.Background(), o); r.Status != StatusOK {
		t.Fatalf("Exec = %+v", r)
	}
	want := "shell dotf-x -- timeout -k 5 60.000 bash -c " + forwardEnv + " _ API_TOKEN -- sh /tmp/dotf-exec-abc"
	if got := shellCalls(f); len(got) != 4 || got[2] != want {
		t.Errorf("calls = %q, want third %q", got, want)
	}
}

// A copy that fails is dotf's failure: its exit status must not read as the
// script's.
func TestExecScriptCopyFailureIsNotAGuestExit(t *testing.T) {
	f := scriptLima(t, "copy")
	o := execOpts()
	o.Script = "s.sh"
	r := heldSession(t, f).Exec(context.Background(), o)
	if r.Status != StatusFailed || r.Exit != nil {
		t.Errorf("Exec = %+v, want failed with no exit", r)
	}
	if got := shellCalls(f); !strings.HasPrefix(got[len(got)-1], "shell dotf-x -- rm -f") {
		t.Errorf("calls = %q, want the temporary file removed", got)
	}
}

// AC2: the record is bounded; the plain stream is not.
func TestExecTruncatesTheRecordOnly(t *testing.T) {
	big := strings.Repeat("x", maxRecordOutput+1)
	f := &listLima{guest: func(_ context.Context, _ io.Reader, out, _ io.Writer, _ []string) error {
		_, _ = io.WriteString(out, big)
		return nil
	}}
	var plain bytes.Buffer
	o := execOpts("yes")
	o.Stdout = &plain
	r := heldSession(t, f).Exec(context.Background(), o)
	if !r.Truncated || len(r.Stdout) != maxRecordOutput || plain.Len() != len(big) {
		t.Errorf("truncated=%v record=%d plain=%d", r.Truncated, len(r.Stdout), plain.Len())
	}
}

func TestSummaryNamesTheGuestExitNotItsOutput(t *testing.T) {
	code := 7
	r := NewRecord("x", linuxOS, StatusOK)
	r.Exit, r.Stderr = &code, "a long guest stderr"
	if got := r.Summary(); got != "x: the guest command exited 7" {
		t.Errorf("Summary = %q", got)
	}
	r = NewRecord("x", linuxOS, StatusTimeout)
	r.Stderr = "a long guest stderr"
	if got := r.Summary(); got != "x: timed out" {
		t.Errorf("Summary = %q", got)
	}
}
