package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The sink has two jobs that pull in opposite directions: the pane must see
// everything, the file must stay auditable. Both are asserted on one run.
func TestSinkTranscriptPassesEverythingThroughButStoresOnlyEvents(t *testing.T) {
	in := strings.NewReader(strings.Join([]string{
		`{"type":"message_start"}`,
		`{"type":"message_update","text":"partial"}`,
		`{"type":"thinking_delta","text":"more"}`,
		`{"type":"message_end","text":"settled"}`,
	}, "\n") + "\n")

	var pane bytes.Buffer
	path := filepath.Join(t.TempDir(), "t.jsonl")
	if err := SinkTranscript(in, &pane, path); err != nil {
		t.Fatalf("SinkTranscript: %v", err)
	}

	for _, want := range []string{"message_start", "message_update", "thinking_delta", "message_end"} {
		if !strings.Contains(pane.String(), want) {
			t.Errorf("the live pane must see every frame, %q is missing:\n%s", want, pane.String())
		}
	}

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, dropped := range []string{"message_update", "thinking_delta"} {
		if strings.Contains(string(stored), dropped) {
			t.Errorf("%q is an incremental frame and must not be stored:\n%s", dropped, stored)
		}
	}
	for _, keep := range []string{"message_start", "message_end"} {
		if !strings.Contains(string(stored), keep) {
			t.Errorf("%q is a settled event and must be stored:\n%s", keep, stored)
		}
	}
}

// A line the filter cannot parse is kept. The filter must never be the reason a
// failure left no trace — dropping the unrecognised is how a diagnostic
// disappears exactly when it matters.
func TestSinkTranscriptKeepsUnparseableLines(t *testing.T) {
	in := strings.NewReader("Error: bw resolve dockerhub/password: not found\n{\"type\":\"message_update\"}\n")

	var pane bytes.Buffer
	path := filepath.Join(t.TempDir(), "t.jsonl")
	if err := SinkTranscript(in, &pane, path); err != nil {
		t.Fatalf("SinkTranscript: %v", err)
	}

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stored), "bw resolve dockerhub/password") {
		t.Errorf("a non-JSON line must survive into the transcript:\n%s", stored)
	}
}

// A settled message can be larger than any token cap a scanner would pick; the
// sink must not truncate the stream on one.
func TestSinkTranscriptHandlesAVeryLongLine(t *testing.T) {
	huge := `{"type":"message_end","text":"` + strings.Repeat("x", 4<<20) + `"}`
	in := strings.NewReader(huge + "\n")

	var pane bytes.Buffer
	path := filepath.Join(t.TempDir(), "t.jsonl")
	if err := SinkTranscript(in, &pane, path); err != nil {
		t.Fatalf("SinkTranscript on a 4 MB line: %v", err)
	}

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) < 4<<20 {
		t.Errorf("the long settled event was truncated: stored %d bytes", len(stored))
	}
}

// TmuxWrap falls back to plain tee when no sink is available: an oversized
// transcript is a defect, a broken pipeline is an outage.
func TestTmuxWrapFallsBackToTeeWithoutASink(t *testing.T) {
	withSink := TmuxWrap("s", "/repo", []string{"pi"}, "/tmp/t.jsonl", []string{"/usr/bin/dotf", "spec", "transcript-sink", "/tmp/t.jsonl"})
	if !strings.Contains(withSink[len(withSink)-1], "transcript-sink") {
		t.Errorf("a supplied sink must be used:\n%s", withSink[len(withSink)-1])
	}
	fallback := TmuxWrap("s", "/repo", []string{"pi"}, "/tmp/t.jsonl", nil)
	if !strings.Contains(fallback[len(fallback)-1], "tee ") {
		t.Errorf("without a sink the pipeline must still store the transcript:\n%s", fallback[len(fallback)-1])
	}
}
