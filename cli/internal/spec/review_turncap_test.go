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

// archiveCapFixture is a repo whose pool holds one pi reviewer and a spec that
// review was launched for: the sidecar names the reviewer, and the transcript
// of that run sits beside it. withReview leaves the previous round's review.md
// in place, unchanged since launch; without it the run left no review.md.
func archiveCapFixture(t *testing.T, lastOutput int, withReview bool) (repoRoot, specDir string) {
	t.Helper()
	repoRoot = t.TempDir()
	pool := `{"pool":[{"id":"nan/deepseek-v4-flash","runner":"pi","provider":"nan","model":"deepseek-v4-flash","role":"primary"}]}`
	if err := os.MkdirAll(filepath.Join(repoRoot, "harness"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ReviewerPoolFile), []byte(pool), 0o600); err != nil {
		t.Fatal(err)
	}
	// "X" is the spec provenanceReviewDoc declares.
	specDir = filepath.Join(repoRoot, "specs", "X")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if withReview {
		if err := os.WriteFile(filepath.Join(specDir, ReviewFile), []byte(provenanceReviewDoc), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := WriteReviewRequest(specDir, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "nan/deepseek-v4-flash", "basebasebase"); err != nil {
		t.Fatal(err)
	}
	transcript, err := os.ReadFile(piTranscript(t, 25, lastOutput))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, TranscriptFile), transcript, 0o600); err != nil {
		t.Fatal(err)
	}
	return repoRoot, specDir
}

// #2132: a detached review stopped by pi's turn cap is found at archive time,
// in both shapes the missing verdict takes there. The refusal must name the
// cap, as the foreground launcher does.
func TestArchiveGateNamesATurnCapOnADetachedRun(t *testing.T) {
	for _, tc := range []struct {
		name       string
		withReview bool
	}{
		{"no review.md", false},
		{"previous round's review.md unchanged", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot, specDir := archiveCapFixture(t, 0, tc.withReview)
			err := checkReviewGate(repoRoot, "X", specDir, nil)
			if err == nil {
				t.Fatal("a review that wrote no verdict must not archive")
			}
			if !strings.Contains(err.Error(), "after 25 turns") || !strings.Contains(err.Error(), "turn cap") {
				t.Errorf("the refusal does not name the turn cap: %v", err)
			}
		})
	}
}

// The same refusals over a run that finished on its own say nothing about a cap.
func TestArchiveGateIsSilentAboutACapForAFinishedRun(t *testing.T) {
	for _, withReview := range []bool{false, true} {
		repoRoot, specDir := archiveCapFixture(t, 460, withReview)
		err := checkReviewGate(repoRoot, "X", specDir, nil)
		if err == nil {
			t.Fatal("a review that wrote no verdict must not archive")
		}
		if strings.Contains(err.Error(), "turn cap") {
			t.Errorf("a finished run got a turn-cap hint: %v", err)
		}
	}
}

// A run that did write a verdict is refused for other reasons (here, a sha
// other than the launched one), and the cap is not the cause of those even
// when the transcript carries the signature.
func TestArchiveGateNamesTheCapOnlyForAMissingVerdict(t *testing.T) {
	repoRoot, specDir := archiveCapFixture(t, 0, false)
	fresh := strings.Replace(provenanceReviewDoc, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "cccccccccccccccccccccccccccccccccccccccc", 1)
	if err := os.WriteFile(filepath.Join(specDir, ReviewFile), []byte(fresh), 0o600); err != nil {
		t.Fatal(err)
	}
	err := checkReviewGate(repoRoot, "X", specDir, nil)
	if err == nil || !strings.Contains(err.Error(), "reviewed_sha") {
		t.Fatalf("want the reviewed_sha refusal, got %v", err)
	}
	if strings.Contains(err.Error(), "turn cap") {
		t.Errorf("a written verdict's refusal got a turn-cap hint: %v", err)
	}
}
