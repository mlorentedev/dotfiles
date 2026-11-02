package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// heldWindows is a store with one session up, proof, whose guest plays
// guestexec.ps1 with play.
func heldWindows(t *testing.T, play func(g *fakeGuest, dir string)) (WindowsSessions, *fakeGuest) {
	t.Helper()
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	s.Grace = 20 * time.Millisecond
	if rec := s.Up(context.Background(), "proof", io.Discard, io.Discard); rec.Status != StatusOK {
		t.Fatalf("Up = %+v", rec)
	}
	g.onExec = func(dir string) {
		if play != nil {
			play(g, dir)
		}
	}
	return s, g
}

func finish(g *fakeGuest, dir, code, stdout, stderr string) {
	g.files[dir+`\stdout`] = []byte(stdout)
	g.files[dir+`\stderr`] = []byte(stderr)
	g.files[dir+`\exit`] = []byte(code + "\r\n")
}

func TestWindowsExecReportsTheCommandInOneRecord(t *testing.T) {
	var req guestRequest
	s, g := heldWindows(t, func(g *fakeGuest, dir string) {
		if err := json.Unmarshal(g.files[dir+`\request.json`], &req); err != nil {
			t.Errorf("request.json: %v", err)
		}
		finish(g, dir, "0", "\xef\xbb\xbfhello\r\n", "warn\r\n")
	})
	var out, errOut bytes.Buffer
	rec := s.Exec(context.Background(), ExecOptions{Session: "proof", Argv: []string{"whoami", "a b"}, Stdout: &out, Stderr: &errOut})
	if rec.Status != StatusOK || rec.Exit == nil || *rec.Exit != 0 || rec.OS != "windows" {
		t.Fatalf("Exec = %+v", rec)
	}
	if rec.Stdout != "hello\r\n" || rec.Stderr != "warn\r\n" || out.String() != "hello\r\n" || errOut.String() != "warn\r\n" {
		t.Errorf("record %q %q, plain %q %q", rec.Stdout, rec.Stderr, out.String(), errOut.String())
	}
	if req.File != "whoami" || req.Args != `"a b"` || req.TimeoutMS != 0 {
		t.Errorf("request = %+v", req)
	}
	if got := g.ran("schtasks.exe /create"); len(got) != 1 || !strings.Contains(got[0], "/ru dotf") || strings.Contains(got[0], "a b") {
		t.Errorf("task = %q: it must run as the test user, with no argv on its command line", got)
	}
	if len(g.ran("schtasks.exe /delete")) != 1 || len(g.ran("cmd.exe /c rmdir")) != 1 {
		t.Errorf("the call left its task or directory: %q", g.execs)
	}
}

func TestWindowsExecANonZeroExitIsOk(t *testing.T) {
	s, _ := heldWindows(t, func(g *fakeGuest, dir string) { finish(g, dir, "3", "", "") })
	rec := s.Exec(context.Background(), ExecOptions{Session: "proof", Argv: []string{"x"}, Stdout: io.Discard, Stderr: io.Discard})
	if rec.Status != StatusOK || rec.Exit == nil || *rec.Exit != 3 || rec.ProcessCode() != 1 {
		t.Errorf("Exec = %+v", rec)
	}
}

func TestWindowsExecRunsAScriptWithPowerShell(t *testing.T) {
	host := filepath.Join(t.TempDir(), "check")
	if err := os.WriteFile(host, []byte("Write-Output hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	var req guestRequest
	var script []byte
	s, _ := heldWindows(t, func(g *fakeGuest, dir string) {
		_ = json.Unmarshal(g.files[dir+`\request.json`], &req)
		script = g.files[dir+`\script.ps1`]
		finish(g, dir, "0", "hi\r\n", "")
	})
	rec := s.Exec(context.Background(), ExecOptions{Session: "proof", Script: host, Stdout: io.Discard, Stderr: io.Discard})
	if rec.Status != StatusOK || string(script) != utf8BOM+"Write-Output hi" {
		t.Fatalf("Exec = %+v, script %q", rec, script)
	}
	if req.File != "powershell.exe" || !strings.Contains(req.Args, `-File C:\Users\Public\dotf\exec-`) {
		t.Errorf("request = %+v", req)
	}
}

// AC3: the agent has no stdin, so a caller's is refused, never dropped.
func TestWindowsExecRefusesStdinAndSecrets(t *testing.T) {
	s, g := heldWindows(t, nil)
	before := len(g.execs)
	rec := s.Exec(context.Background(), ExecOptions{Session: "proof", Argv: []string{"x"}, Stdin: strings.NewReader("data")})
	if rec.Status != StatusUnsupported || rec.ProcessCode() != 2 {
		t.Errorf("stdin: %+v", rec)
	}
	called := false
	rec = s.Exec(context.Background(), ExecOptions{Session: "proof", Argv: []string{"x"}, Secrets: func() ([]string, error) {
		called = true
		return nil, nil
	}})
	if rec.Status != StatusUnsupported || called {
		t.Errorf("secret: %+v, resolved %v", rec, called)
	}
	if len(g.execs) != before {
		t.Error("a refusal reached the guest")
	}
}

func TestWindowsExecOnNoSessionAndOnStale(t *testing.T) {
	g := newFakeGuest(t, okStep, nothingStep)
	s := windowsStore(t, g)
	o := ExecOptions{Session: "ghost", Argv: []string{"x"}, Stdout: io.Discard, Stderr: io.Discard}
	if rec := s.Exec(context.Background(), o); rec.Status != StatusNoSession || rec.ProcessCode() != 3 {
		t.Errorf("unknown: %+v", rec)
	}
	holdName(t, s, "ghost")
	if rec := s.Exec(context.Background(), o); rec.Status != StatusStale || rec.ProcessCode() != 2 {
		t.Errorf("stale: %+v", rec)
	}
}

func TestWindowsExecTimesOutAndWaitsForTheGuestsKill(t *testing.T) {
	var req guestRequest
	s, g := heldWindows(t, func(g *fakeGuest, dir string) {
		_ = json.Unmarshal(g.files[dir+`\request.json`], &req)
		g.files[dir+`\stdout`] = []byte("partial")
	})
	rec := s.Exec(context.Background(), ExecOptions{Session: "proof", Argv: []string{"x"}, Timeout: 30 * time.Millisecond, Stdout: io.Discard, Stderr: io.Discard})
	if rec.Status != StatusTimeout || rec.Exit != nil || rec.ProcessCode() != 1 {
		t.Fatalf("Exec = %+v", rec)
	}
	if rec.Stdout != "partial" || !strings.Contains(rec.Stderr, "may still be running") {
		t.Errorf("record %q %q", rec.Stdout, rec.Stderr)
	}
	if req.TimeoutMS != 30 {
		t.Errorf("request = %+v: the guest needs its own limit", req)
	}
	if len(g.ran("cmd.exe /c rmdir")) != 0 {
		t.Error("removed the directory of a command that may still be running")
	}
}

// A cancel is no deadline: nothing kills the command, so it may still run.
func TestWindowsExecCancelledKeepsTheCommandsDirectory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s, g := heldWindows(t, func(*fakeGuest, string) { cancel() })
	rec := s.Exec(ctx, ExecOptions{Session: "proof", Argv: []string{"x"}, Stdout: io.Discard, Stderr: io.Discard})
	if rec.Status != StatusFailed || !strings.Contains(rec.Stderr, "may still be running") {
		t.Fatalf("Exec = %+v", rec)
	}
	if len(g.ran("cmd.exe /c rmdir")) != 0 {
		t.Error("removed the directory of a command that may still be running")
	}
}

// An exit read as the deadline passes is the command's outcome, not a timeout.
func TestWindowsExecAnExitAtTheDeadlineWins(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	s, _ := heldWindows(t, func(g *fakeGuest, dir string) { finish(g, dir, "3", "", "") })
	rec := s.Exec(ctx, ExecOptions{Session: "proof", Argv: []string{"x"}, Stdout: io.Discard, Stderr: io.Discard})
	if rec.Status != StatusOK || rec.Exit == nil || *rec.Exit != 3 {
		t.Fatalf("Exec = %+v", rec)
	}
}

func TestWindowsExecTheGuestsTimeoutIsATimeout(t *testing.T) {
	s, _ := heldWindows(t, func(g *fakeGuest, dir string) { finish(g, dir, guestTimedOut, "", "") })
	rec := s.Exec(context.Background(), ExecOptions{Session: "proof", Argv: []string{"x"}, Stdout: io.Discard, Stderr: io.Discard})
	if rec.Status != StatusTimeout || rec.Exit != nil {
		t.Fatalf("Exec = %+v", rec)
	}
}

// A call that never started leaves nothing in the guest.
func TestWindowsExecAMissingScriptLeavesNoDirectory(t *testing.T) {
	s, g := heldWindows(t, nil)
	rec := s.Exec(context.Background(), ExecOptions{Session: "proof", Script: filepath.Join(t.TempDir(), "absent.ps1"), Stdout: io.Discard, Stderr: io.Discard})
	if rec.Status != StatusFailed {
		t.Fatalf("Exec = %+v", rec)
	}
	if len(g.ran("cmd.exe /c rmdir")) != 1 {
		t.Error("the call's directory stayed in the guest")
	}
}

func TestWindowsExecBoundsTheRecord(t *testing.T) {
	s, _ := heldWindows(t, func(g *fakeGuest, dir string) {
		finish(g, dir, "0", strings.Repeat("x", maxRecordOutput+1), "")
	})
	var out bytes.Buffer
	rec := s.Exec(context.Background(), ExecOptions{Session: "proof", Argv: []string{"x"}, Stdout: &out, Stderr: io.Discard})
	if !rec.Truncated || len(rec.Stdout) != maxRecordOutput || out.Len() != maxRecordOutput+1 {
		t.Errorf("truncated %v, record %d, plain %d", rec.Truncated, len(rec.Stdout), out.Len())
	}
}

func TestWindowsCommandLine(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{"a", "b"}, "a b"},
		{[]string{"a b"}, `"a b"`},
		{[]string{""}, `""`},
		{[]string{`say "hi"`}, `"say \"hi\""`},
		{[]string{`C:\dir\`}, `C:\dir\`},
		{[]string{`C:\my dir\`}, `"C:\my dir\\"`},
		{[]string{`a\"b`}, `"a\\\"b"`},
		{[]string{"tab\there"}, "\"tab\there\""},
	} {
		if got := windowsCommandLine(c.args); got != c.want {
			t.Errorf("%q = %s, want %s", c.args, got, c.want)
		}
	}
}

// The script is only ever run on the real guest, so its contract is pinned here:
// the limit kills the program's children too, and exit appears last and whole.
func TestGuestExec_KillsTheTreeAndWritesExitLast(t *testing.T) {
	script := string(guestExecScript)
	for _, want := range []string{"taskkill.exe /T /F /PID", "$p.HasExited) { exit 1 }", "$null = $p.Handle", "RedirectStandardInput"} {
		if !strings.Contains(script, want) {
			t.Errorf("guestexec.ps1 lost %q", want)
		}
	}
	if !strings.HasSuffix(strings.TrimSpace(script), `Move-Item -Path "$exit.tmp" -Destination $exit -Force`) {
		t.Error("guestexec.ps1 must publish exit last, by a rename")
	}
}
