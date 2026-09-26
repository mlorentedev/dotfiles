package secrets

import (
	"os"
	"strings"
	"testing"
)

// traceWith writes pidFile (unless empty) into a fresh state dir and returns
// Trace's line with the liveness check pinned to alive.
func traceWith(t *testing.T, pidFile string, alive bool) string {
	t.Helper()
	d := &BWServeDaemon{State: BWServeState{Dir: t.TempDir()}}
	if pidFile != "" {
		if err := os.WriteFile(d.State.PIDPath(), []byte(pidFile), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	orig := traceAlive
	t.Cleanup(func() { traceAlive = orig })
	traceAlive = func(int) bool { return alive }
	return d.Trace()
}

// CLI-057's retroactive review (round 1) reproduced the defect: a pid file
// naming a dead process was printed as the answering daemon's pid.
func TestTrace_AStalePIDIsSaidNotGuessed(t *testing.T) {
	got := traceWith(t, "4194304", false)
	if strings.HasPrefix(got, "pid 4194304,") || !strings.Contains(got, "that process is gone") {
		t.Fatalf("a dead recorded pid must not be presented as the daemon's: %q", got)
	}
}

func TestTrace_ALivePIDIsReported(t *testing.T) {
	if got := traceWith(t, "4242", true); !strings.HasPrefix(got, "pid 4242, log ") {
		t.Fatalf("got %q", got)
	}
}

// An unparseable pid file is unreadable, as doctor says; it is not evidence
// that something else started the daemon.
func TestTrace_AnUnreadablePIDFileIsNamedAsSuch(t *testing.T) {
	got := traceWith(t, "garbage", true)
	if !strings.Contains(got, "unreadable") || strings.Contains(got, "not started by this dotf") {
		t.Fatalf("got %q", got)
	}
}

func TestTrace_NoPIDFileMeansNotStartedByThisDotf(t *testing.T) {
	if got := traceWith(t, "", true); !strings.Contains(got, "not started by this dotf") {
		t.Fatalf("got %q", got)
	}
}

func TestTrace_NoStateDirRecordsNothing(t *testing.T) {
	d := &BWServeDaemon{}
	if got := d.Trace(); !strings.Contains(got, "no state dir") {
		t.Fatalf("got %q", got)
	}
}
