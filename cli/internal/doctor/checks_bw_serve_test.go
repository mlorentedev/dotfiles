package doctor

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// bwServeSys is a System whose daemon reports status and whose recorded pid is
// alive or not, on top of newSys's safe defaults.
func bwServeSys(status string, alive bool) *System {
	sys := newSys(nil, nil, nil)
	sys.BWServeStatus = func() (string, error) { return status, nil }
	sys.ProcessAlive = func(int) bool { return alive }
	return sys
}

// writeBWServeTrace lays down the trace a started daemon leaves under the
// deploy dir: its pid file and, when log is non-empty, its log.
func writeBWServeTrace(t *testing.T, dotfilesDir string, pid int, log string) secrets.BWServeState {
	t.Helper()
	state := secrets.NewBWServeState(dotfilesDir)
	if err := state.WritePID(pid); err != nil {
		t.Fatal(err)
	}
	if log != "" {
		if err := os.WriteFile(state.LogPath(), []byte(log), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return state
}

// TestCheckBWServeDaemon_States is AC5 of CLI-057: one row per observable
// state, asserted by status tag (statusOfLine) rather than prose. The rows
// that matter are the two "absent" ones with a pid file — before #1315 both
// rendered as the same "no daemon running" Info, and a daemon that had died
// was indistinguishable from one never started.
func TestCheckBWServeDaemon_States(t *testing.T) {
	cases := []struct {
		name       string
		status     string
		pid        int // 0 ⇒ no pid file
		alive      bool
		log        string
		wantStatus Status
		needle     string
		wantSubstr []string
	}{
		{
			name: "absent, never started → Info as before", status: "absent",
			wantStatus: StatusInfo, needle: "no daemon running",
		},
		{
			name: "absent, pid recorded and gone → WARN daemon exited with last lines", status: "absent",
			pid: 4242, alive: false, log: "Listening on 127.0.0.1:8087\nTypeError: cannot read properties of undefined\n    at Object.<anonymous>\n",
			wantStatus: StatusWarn, needle: "daemon exited",
			wantSubstr: []string{"pid 4242", "TypeError: cannot read properties of undefined", "bw-serve.log", "dotf secrets unlock"},
		},
		{
			name: "absent, pid recorded and gone, empty log → WARN says the log is empty", status: "absent",
			pid: 4242, alive: false,
			wantStatus: StatusWarn, needle: "daemon exited",
			wantSubstr: []string{"log is empty"},
		},
		{
			name: "absent, pid recorded and alive → WARN alive but not answering", status: "absent",
			pid: 4242, alive: true,
			wantStatus: StatusWarn, needle: "alive but nothing answers",
			wantSubstr: []string{"pid 4242", "reusing the pid"},
		},
		{
			name: "locked → Info", status: "locked",
			pid: 4242, alive: true,
			wantStatus: StatusInfo, needle: "locked",
		},
		{
			name: "unlocked → PASS", status: "unlocked",
			pid: 4242, alive: true,
			wantStatus: StatusPass, needle: "unlocked",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dotfiles := t.TempDir()
			if tc.pid != 0 {
				writeBWServeTrace(t, dotfiles, tc.pid, tc.log)
			}
			var buf bytes.Buffer
			rep := capture(&buf)

			checkBWServeDaemon(bwServeSys(tc.status, tc.alive), &Config{DotfilesDir: dotfiles}, rep)

			out := buf.String()
			if got := statusOfLine(out, tc.needle); got != tc.wantStatus {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.wantStatus), out)
			}
			for _, want := range tc.wantSubstr {
				if !strings.Contains(out, want) {
					t.Errorf("expected %q in:\n%s", want, out)
				}
			}
		})
	}
}

// tagOf renders a Status for a failure message; -1 (statusOfLine's "no such
// line") has no tag.
func tagOf(s Status) string {
	if tag, ok := statusTag[s]; ok {
		return tag
	}
	return "<no line>"
}

// A pid file that cannot be parsed is its own WARN, not silently "never
// started": the trace exists and something wrote garbage into it.
func TestCheckBWServeDaemon_UnreadablePIDFileIsAWarn(t *testing.T) {
	dotfiles := t.TempDir()
	state := secrets.NewBWServeState(dotfiles)
	if err := os.MkdirAll(state.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.PIDPath(), []byte("garbage\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkBWServeDaemon(bwServeSys("absent", false), &Config{DotfilesDir: dotfiles}, rep)

	if got := statusOfLine(buf.String(), "pid file is unreadable"); got != StatusWarn {
		t.Fatalf("want a WARN naming the unreadable pid file, got:\n%s", buf.String())
	}
}

// The absent branch must survive a System with no ProcessAlive seam (older
// callers, hand-built test Systems): an unknown liveness reads as gone, which
// is the honest direction — the port is not answering.
func TestCheckBWServeDaemon_NilProcessAliveReadsAsGone(t *testing.T) {
	dotfiles := t.TempDir()
	writeBWServeTrace(t, dotfiles, 4242, "")
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{BWServeStatus: func() (string, error) { return "absent", nil }}

	checkBWServeDaemon(sys, &Config{DotfilesDir: dotfiles}, rep)

	if got := statusOfLine(buf.String(), "daemon exited"); got != StatusWarn {
		t.Fatalf("want the exited WARN, got:\n%s", buf.String())
	}
}

func TestCheckBWServeDaemon_StatusUnreadable(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{BWServeStatus: func() (string, error) { return "", errors.New("boom") }}

	checkBWServeDaemon(sys, &Config{DotfilesDir: t.TempDir()}, rep)

	if got := buf.String(); !strings.Contains(got, "boom") {
		t.Fatalf("expected the underlying error surfaced, got: %s", got)
	}
}

// CLI-056 (#1316): the daemon's own cache age is reported whenever the daemon
// answers, from the seam that reads ITS /status — not `bw status`, whose cache
// is a different one. One row per branch, asserted by status tag.
func TestCheckBWServeDaemon_CacheAge(t *testing.T) {
	cases := []struct {
		name     string
		status   string
		lastSync time.Time
		err      error
		want     Status
		needle   string
	}{
		{"fresh cache → PASS", "unlocked", fixedTestNow.Add(-2 * 24 * time.Hour), nil, StatusPass, "cache synced 2d ago"},
		{"eight days old → WARN naming unlock", "unlocked", fixedTestNow.Add(-8 * 24 * time.Hour), nil, StatusWarn, "cache is 8d old"},
		{"never synced → WARN", "unlocked", time.Time{}, nil, StatusWarn, "never synced"},
		{"locked daemon still reports its cache", "locked", fixedTestNow.Add(-9 * 24 * time.Hour), nil, StatusWarn, "cache is 9d old"},
		{"future stamp → WARN clock", "unlocked", fixedTestNow.Add(24 * time.Hour), nil, StatusWarn, "in the future"},
		{"status unreadable → WARN", "unlocked", time.Time{}, errors.New("dial tcp: refused"), StatusWarn, "cache age unreadable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dotfiles := t.TempDir()
			writeBWServeTrace(t, dotfiles, 4242, "")
			sys := bwServeSys(tc.status, true)
			sys.BWServeLastSync = func() (time.Time, error) { return tc.lastSync, tc.err }
			var buf bytes.Buffer
			rep := capture(&buf)

			checkBWServeDaemon(sys, &Config{DotfilesDir: dotfiles}, rep)

			if got := statusOfLine(buf.String(), tc.needle); got != tc.want {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.want), buf.String())
			}
		})
	}
}
