package secrets

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewBWServeState_DerivesPathsFromDeployDir(t *testing.T) {
	s := NewBWServeState(filepath.Join("home", ".dotfiles"))
	wantLog := filepath.Join("home", ".dotfiles", "state", "bw-serve.log")
	wantPID := filepath.Join("home", ".dotfiles", "state", "bw-serve.pid")
	if s.LogPath() != wantLog || s.PIDPath() != wantPID {
		t.Fatalf("paths = %q, %q; want %q, %q", s.LogPath(), s.PIDPath(), wantLog, wantPID)
	}
}

// An empty deploy dir must not silently become a relative "state/" in whatever
// the cwd happens to be.
func TestNewBWServeState_EmptyDeployDirIsDisabled(t *testing.T) {
	s := NewBWServeState("")
	if s.LogPath() != "" || s.PIDPath() != "" {
		t.Fatalf("disabled state must have no paths, got %q %q", s.LogPath(), s.PIDPath())
	}
	if _, err := s.ReadPID(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadPID on a disabled state: want ErrNotExist, got %v", err)
	}
	lines, err := s.LastLogLines(3)
	if err != nil || lines != nil {
		t.Fatalf("LastLogLines on a disabled state: want nil, nil; got %v, %v", lines, err)
	}
}

func TestBWServeState_PIDRoundTrip(t *testing.T) {
	s := NewBWServeState(t.TempDir())
	if _, err := s.ReadPID(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("no pid file yet: want ErrNotExist, got %v", err)
	}
	if err := s.WritePID(4242); err != nil {
		t.Fatalf("WritePID: %v", err)
	}
	pid, err := s.ReadPID()
	if err != nil || pid != 4242 {
		t.Fatalf("ReadPID = %d, %v; want 4242", pid, err)
	}
}

func TestBWServeState_ReadPID_RejectsGarbage(t *testing.T) {
	s := NewBWServeState(t.TempDir())
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.PIDPath(), []byte("not-a-pid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.ReadPID()
	if err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("garbage pid file must be an error distinct from absence, got %v", err)
	}
}

func TestBWServeState_LastLogLines(t *testing.T) {
	s := NewBWServeState(t.TempDir())

	// No log at all: no lines, no error — "it left no log" is a finding.
	lines, err := s.LastLogLines(3)
	if err != nil || len(lines) != 0 {
		t.Fatalf("missing log: want no lines and no error, got %v, %v", lines, err)
	}

	f, err := s.openLog()
	if err != nil {
		t.Fatalf("openLog: %v", err)
	}
	long := strings.Repeat("x", bwServeLogLineWidth+50)
	_, _ = f.WriteString("one\n\ntwo\r\nthree\n" + long + "\nfive\n")
	_ = f.Close()

	lines, err = s.LastLogLines(3)
	if err != nil {
		t.Fatalf("LastLogLines: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d: %q", len(lines), lines)
	}
	if lines[0] != "three" || lines[2] != "five" {
		t.Fatalf("want the last three non-empty lines in order, got %q", lines)
	}
	if len(lines[1]) > bwServeLogLineWidth+len("…") || !strings.HasSuffix(lines[1], "…") {
		t.Fatalf("a long line must be cut to the width and marked, got %d bytes", len(lines[1]))
	}
}

// LastLogLines reads only the tail: a log far larger than the tail window
// still yields its final lines, and the fragment the seek lands in is dropped
// rather than reported as a line.
func TestBWServeState_LastLogLines_ReadsOnlyTheTail(t *testing.T) {
	s := NewBWServeState(t.TempDir())
	f, err := s.openLog()
	if err != nil {
		t.Fatal(err)
	}
	filler := strings.Repeat("filler line that is long enough to matter\n", (2*bwServeTailBytes)/40)
	_, _ = f.WriteString(filler + "last\n")
	_ = f.Close()

	lines, err := s.LastLogLines(1)
	if err != nil || len(lines) != 1 || lines[0] != "last" {
		t.Fatalf("want [last], got %v, %v", lines, err)
	}
}

func writeLogOfSize(t *testing.T, s BWServeState, size int) {
	t.Helper()
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.LogPath(), bytes.Repeat([]byte("y"), size), 0o600); err != nil {
		t.Fatal(err)
	}
}

// AC2: over the cap the log is moved to .log.1 and a fresh one opened; the
// previous .log.1 is replaced, so the bound is two generations.
func TestBWServeState_RotateOverCap(t *testing.T) {
	s := NewBWServeState(t.TempDir())
	rotated := s.LogPath() + ".1"
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rotated, []byte("stale generation"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeLogOfSize(t, s, bwServeLogCap+1)

	f, err := s.openLog()
	if err != nil {
		t.Fatalf("openLog: %v", err)
	}
	_ = f.Close()

	fi, err := os.Stat(s.LogPath())
	if err != nil || fi.Size() != 0 {
		t.Fatalf("fresh log after rotation: size %d, err %v", fi.Size(), err)
	}
	old, err := os.Stat(rotated)
	if err != nil || old.Size() != bwServeLogCap+1 {
		t.Fatalf(".log.1 must hold the previous generation (%d bytes), got %d, %v", bwServeLogCap+1, old.Size(), err)
	}
}

// At or under the cap nothing moves: a log that rotated on every start would
// throw away exactly the lines a fresh death is diagnosed from.
func TestBWServeState_RotateLeavesLogUnderCap(t *testing.T) {
	s := NewBWServeState(t.TempDir())
	writeLogOfSize(t, s, bwServeLogCap)

	f, err := s.openLog()
	if err != nil {
		t.Fatalf("openLog: %v", err)
	}
	_ = f.Close()

	if _, err := os.Stat(s.LogPath() + ".1"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("no rotation expected at the cap, got %v", err)
	}
	fi, _ := os.Stat(s.LogPath())
	if fi.Size() != bwServeLogCap {
		t.Fatalf("log must be untouched, size %d", fi.Size())
	}
}

// When the rename cannot happen the cap still holds: a directory squatting on
// the .log.1 name makes the rename fail on every OS, and the log is truncated.
func TestBWServeState_RotateFallsBackToTruncate(t *testing.T) {
	s := NewBWServeState(t.TempDir())
	writeLogOfSize(t, s, bwServeLogCap+1)
	if err := os.MkdirAll(filepath.Join(s.LogPath()+".1", "occupied"), 0o700); err != nil {
		t.Fatal(err)
	}

	f, err := s.openLog()
	if err != nil {
		t.Fatalf("openLog: %v", err)
	}
	_ = f.Close()

	fi, err := os.Stat(s.LogPath())
	if err != nil || fi.Size() != 0 {
		t.Fatalf("log must be truncated when it cannot be renamed: size %d, err %v", fi.Size(), err)
	}
}

// gatedFakeServe answers /status only once the fake daemon has been spawned:
// Start() must see "unreachable" first (so it spawns) and "reachable" after
// (so it returns), with the transition tied to the spawn itself.
func gatedFakeServe(spawned *atomic.Bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !spawned.Load() {
			http.Error(w, "not yet", http.StatusServiceUnavailable)
			return
		}
		writeEnvelope(w, true, "", map[string]any{
			"object":   "template",
			"template": map[string]string{"status": "locked"},
		})
	}
}

// waitForLog polls until the log contains needle or the deadline passes; the
// child writes asynchronously and a detached child on Windows has no console
// to flush through.
func waitForLog(t *testing.T, path, needle string) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		raw, _ := os.ReadFile(path)
		if strings.Contains(string(raw), needle) || time.Now().After(deadline) {
			return string(raw)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestBWServeDaemon_Start_WritesLogAndPID is AC1 and — run on Windows — AC4: a
// real child, started through Start() with the production detach attributes,
// gets its stdout AND stderr into the log and its pid into the pid file. The
// child is a re-exec of this test binary (TestHelperFakeDaemon), not bw.
func TestBWServeDaemon_Start_WritesLogAndPID(t *testing.T) {
	var spawned atomic.Bool
	srv := httptest.NewServer(gatedFakeServe(&spawned))
	defer srv.Close()

	state := NewBWServeState(t.TempDir())
	var child *exec.Cmd
	d := &BWServeDaemon{
		Client: BWServeClient{BaseURL: srv.URL, HTTPClient: &http.Client{Timeout: time.Second}},
		State:  state,
		newCmd: func(string, int) *exec.Cmd {
			child = fakeDaemonCmd(t)
			spawned.Store(true)
			return child
		},
	}
	t.Cleanup(func() { reap(t, child) })

	if err := d.Start(5 * time.Second); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if child == nil || child.Process == nil {
		t.Fatal("Start did not spawn the child")
	}

	pid, err := state.ReadPID()
	if err != nil {
		t.Fatalf("pid file after Start: %v", err)
	}
	if pid != child.Process.Pid {
		t.Fatalf("pid file holds %d, child is %d", pid, child.Process.Pid)
	}
	if !ProcessAlive(pid) {
		t.Fatalf("recorded pid %d is not alive", pid)
	}

	log := waitForLog(t, state.LogPath(), "fake daemon: stderr line")
	for _, want := range []string{"fake daemon: stdout line", "fake daemon: stderr line", "started bw serve pid " + strconv.Itoa(pid)} {
		if !strings.Contains(log, want) {
			t.Errorf("log lacks %q:\n%s", want, log)
		}
	}
	if got := d.Trace(); !strings.Contains(got, "pid "+strconv.Itoa(pid)) || !strings.Contains(got, state.LogPath()) {
		t.Fatalf("Trace() = %q; want the pid and the log path", got)
	}
}

// The zero State keeps the pre-#1315 contract: Start spawns and records
// nothing, and Trace says so instead of inventing a path.
func TestBWServeDaemon_Start_NoStateDirWritesNothing(t *testing.T) {
	var spawned atomic.Bool
	srv := httptest.NewServer(gatedFakeServe(&spawned))
	defer srv.Close()

	var child *exec.Cmd
	d := &BWServeDaemon{
		Client: BWServeClient{BaseURL: srv.URL, HTTPClient: &http.Client{Timeout: time.Second}},
		newCmd: func(string, int) *exec.Cmd {
			child = fakeDaemonCmd(t)
			spawned.Store(true)
			return child
		},
	}
	t.Cleanup(func() { reap(t, child) })

	if err := d.Start(5 * time.Second); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if child.Stdout != nil || child.Stderr != nil {
		t.Fatal("with no state dir the child's stdio must stay unset")
	}
	if got := d.Trace(); !strings.Contains(got, "not recorded") {
		t.Fatalf("Trace() with no state dir = %q", got)
	}
}

func TestBWServeDaemon_Trace_NoPIDFileIsSaidNotGuessed(t *testing.T) {
	d := &BWServeDaemon{State: NewBWServeState(t.TempDir())}
	got := d.Trace()
	if !strings.Contains(got, "pid unknown") || !strings.Contains(got, d.State.LogPath()) {
		t.Fatalf("Trace() = %q; want 'pid unknown' and the log path", got)
	}
}
