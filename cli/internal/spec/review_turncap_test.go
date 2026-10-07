package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// piTranscript writes a pi --mode json transcript of the given number of turns
// whose last assistant message produced lastOutput tokens, in the event shape
// the transcript sink stores.
func piTranscript(t *testing.T, turns, lastOutput int) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"type":"session"}` + "\n")
	b.WriteString(`{"type":"agent_start"}` + "\n")
	for i := 1; i <= turns; i++ {
		output := 120
		text := "Reading the diff."
		if i == turns {
			output = lastOutput
			text = "Comprehensive named tests present. Let me run the full harness and confirm all finding"
		}
		msg := fmt.Sprintf(`{"role":"assistant","content":[{"type":"text","text":%q}],"usage":{"input":58743,"output":%d},"stopReason":"stop"}`, text, output)
		b.WriteString(`{"type":"turn_start"}` + "\n")
		b.WriteString(`{"type":"message_start","message":{"role":"assistant"}}` + "\n")
		b.WriteString(`{"type":"message_end","message":` + msg + "}\n")
		b.WriteString(`{"type":"message_end","message":{"role":"toolResult","content":[]}}` + "\n")
		b.WriteString(`{"type":"turn_end","message":` + msg + `,"toolResults":[]}` + "\n")
	}
	b.WriteString(`{"type":"agent_end","willRetry":false}` + "\n")
	b.WriteString(`{"type":"agent_settled"}` + "\n")
	path := filepath.Join(t.TempDir(), TranscriptFile)
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The run measured in #1157: 25 turns, agent_end, zero output on the last
// message. The hint must name the cap and the turn count.
func TestTurnCapHintNamesACappedPiRun(t *testing.T) {
	hint := TurnCapHint("pi", piTranscript(t, 25, 0))
	for _, want := range []string{"after 25 turns", "turn cap", "re-run the review"} {
		if !strings.Contains(hint, want) {
			t.Errorf("hint %q does not say %q", hint, want)
		}
	}
}

// The count comes from the transcript, not from the one cap measured so far:
// pi's limit is internal and can move between releases.
func TestTurnCapHintCountsTheTurnsThatRan(t *testing.T) {
	if hint := TurnCapHint("pi", piTranscript(t, 7, 0)); !strings.Contains(hint, "after 7 turns") {
		t.Errorf("hint %q does not count 7 turns", hint)
	}
}

// A reviewer that finished on its own ends on a message that said something.
// The same turn count must not be read as a cap.
func TestTurnCapHintIsSilentForARunThatFinished(t *testing.T) {
	if hint := TurnCapHint("pi", piTranscript(t, 25, 460)); hint != "" {
		t.Errorf("a finished run got a turn-cap hint: %q", hint)
	}
}

// Only pi's signature is measured. Another runner's transcript gets no line,
// even when it happens to carry the same events.
func TestTurnCapHintDoesNotGuessForOtherRunners(t *testing.T) {
	path := piTranscript(t, 25, 0)
	for _, runner := range []string{"opencode", "agy", "claude", ""} {
		if hint := TurnCapHint(runner, path); hint != "" {
			t.Errorf("runner %q got a turn-cap hint: %q", runner, hint)
		}
	}
}

// A transcript that is missing, or that stops before agent_end (the runner was
// killed rather than capped), is not the signature.
func TestTurnCapHintNeedsAgentEnd(t *testing.T) {
	if hint := TurnCapHint("pi", filepath.Join(t.TempDir(), "absent.jsonl")); hint != "" {
		t.Errorf("a missing transcript got a hint: %q", hint)
	}
	full, err := os.ReadFile(piTranscript(t, 25, 0))
	if err != nil {
		t.Fatal(err)
	}
	cut := strings.SplitAfter(string(full), `{"type":"agent_end"`)[0]
	cut = strings.TrimSuffix(cut, `{"type":"agent_end"`)
	path := filepath.Join(t.TempDir(), "killed.jsonl")
	if err := os.WriteFile(path, []byte(cut), 0o600); err != nil {
		t.Fatal(err)
	}
	if hint := TurnCapHint("pi", path); hint != "" {
		t.Errorf("a run with no agent_end got a hint: %q", hint)
	}
}
