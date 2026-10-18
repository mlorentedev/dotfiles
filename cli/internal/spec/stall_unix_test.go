//go:build !windows

package spec

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTranscript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "review-transcript.jsonl")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// The file predates the run, as a transcript left by earlier rounds would.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	return p
}

// CLI-094 AC1: a runner that stays alive and writes nothing is stopped at the
// stall timeout, not the cap, with a status of its own and the last call named.
func TestRunWithLimitsStopsARunnerWhoseTranscriptGoesSilent(t *testing.T) {
	p := writeTranscript(t, `{"type":"tool_execution_update","toolName":"bash","args":{"command":"find ~/Projects"}}`+"\n")

	began := time.Now()
	res, err := RunWithLimits(Limits{Deadline: time.Minute, StallTimeout: 400 * time.Millisecond, Transcript: p},
		[]string{"sleep", "60"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stop != StopStall || res.Code != ExitStalled {
		t.Errorf("stop=%v code=%d, want stall and %d", res.Stop, res.Code, ExitStalled)
	}
	if ExitStalled == ExitDeadline {
		t.Errorf("a stall must not read as the cap: both are %d", ExitStalled)
	}
	if res.LastCall != "bash: find ~/Projects" {
		t.Errorf("LastCall = %q", res.LastCall)
	}
	if elapsed := time.Since(began); elapsed > 15*time.Second {
		t.Errorf("a 400ms stall took %s to stop", elapsed)
	}
}

// CLI-094 AC2: a transcript that keeps growing is never a stall, however long
// the run goes on; the stall timeout is shorter than the whole run here.
func TestRunWithLimitsNeverStopsARunnerThatKeepsWriting(t *testing.T) {
	p := writeTranscript(t, "")
	script := `i=0; while [ $i -lt 15 ]; do echo '{"type":"message_end"}' >> "$1"; sleep 0.1; i=$((i+1)); done`
	res, err := RunWithLimits(Limits{Deadline: time.Minute, StallTimeout: 700 * time.Millisecond, Transcript: p},
		[]string{"sh", "-c", script, "sh", p}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stop != StopNone || res.Code != 0 {
		t.Errorf("stop=%v code=%d, want a run that ended on its own", res.Stop, res.Code)
	}
}

// CLI-094 AC2, second half: silence shorter than the timeout is a long tool
// call, not a wedge. The measured longest legitimate call is 97s against a
// 10 minute timeout; this is the same ratio at test scale.
func TestRunWithLimitsToleratesSilenceShorterThanTheTimeout(t *testing.T) {
	p := writeTranscript(t, "")
	res, err := RunWithLimits(Limits{Deadline: time.Minute, StallTimeout: 3 * time.Second, Transcript: p},
		[]string{"sh", "-c", "sleep 0.6; exit 4"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stop != StopNone || res.Code != 4 {
		t.Errorf("stop=%v code=%d, want the runner's own status 4", res.Stop, res.Code)
	}
}

// A zero stall timeout is the detector off: only the cap binds, as before.
func TestRunWithLimitsWithoutAStallTimeoutOnlyTheCapBinds(t *testing.T) {
	p := writeTranscript(t, "")
	res, err := RunWithLimits(Limits{Deadline: 500 * time.Millisecond, Transcript: p},
		[]string{"sleep", "60"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stop != StopDeadline || res.Code != ExitDeadline {
		t.Errorf("stop=%v code=%d, want the cap and %d", res.Stop, res.Code, ExitDeadline)
	}
}

// A transcript that does not exist yet is silence from the start of the run,
// not a reason to wait forever: the sink creates it on the first event.
func TestRunWithLimitsCountsAMissingTranscriptAsSilence(t *testing.T) {
	p := filepath.Join(t.TempDir(), "never-written.jsonl")
	res, err := RunWithLimits(Limits{Deadline: time.Minute, StallTimeout: 400 * time.Millisecond, Transcript: p},
		[]string{"sleep", "60"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stop != StopStall {
		t.Errorf("stop=%v, want a stall", res.Stop)
	}
}

// CLI-094 round 1: the production pipeline is `spec deadline` piped into the
// transcript sink, and the detector reads the file the sink writes. The earlier
// tests pre-wrote the transcript and so never exercised that hand-off, which is
// where a buffering sink left the file empty and the wedge unnamed.
func TestRunWithLimitsNamesTheWedgedCallThroughARealSink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review-transcript.jsonl")
	pr, pw := io.Pipe()
	sunk := make(chan error, 1)
	go func() { sunk <- SinkTranscript(pr, io.Discard, path) }()

	event := `{"type":"tool_execution_start","toolName":"bash","args":{"command":"find ~/Projects"}}`
	res, err := RunWithLimits(Limits{Deadline: time.Minute, StallTimeout: 600 * time.Millisecond, Transcript: path},
		[]string{"sh", "-c", "echo '" + event + "'; sleep 60"}, nil, pw, &bytes.Buffer{})
	_ = pw.Close()
	if serr := <-sunk; serr != nil {
		t.Fatal(serr)
	}
	if err != nil {
		t.Fatal(err)
	}
	if res.Stop != StopStall || res.LastCall != "bash: find ~/Projects" {
		t.Errorf("stop=%v LastCall=%q, want a stall naming the call the sink stored", res.Stop, res.LastCall)
	}
}
