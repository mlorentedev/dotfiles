package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CLI-094 (#1975): when the launcher stops a wedged reviewer it names the last
// tool call the transcript recorded, because that call is the wedge. The
// transcript holds pi's settled events, one JSON object per line.
func TestLastToolCallNamesTheLastCallTheTranscriptRecorded(t *testing.T) {
	p := filepath.Join(t.TempDir(), "review-transcript.jsonl")
	lines := []string{
		`{"type":"tool_execution_start","toolName":"read","args":{"path":"/repo/specs/X/proposal.md"}}`,
		`{"type":"message_end"}`,
		`{"type":"tool_execution_update","toolName":"bash","args":{"command":"find ~/Projects -name '*.go'"},"partialResult":{}}`,
		`{"type":"turn_end"}`,
	}
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, want := LastToolCall(p), "bash: find ~/Projects -name '*.go'"; got != want {
		t.Errorf("LastToolCall = %q, want %q", got, want)
	}
}

func TestLastToolCallFallsBackToThePathOfAFileTool(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.jsonl")
	if err := os.WriteFile(p, []byte(`{"type":"tool_execution_start","toolName":"read","args":{"path":"/repo/a.md"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, want := LastToolCall(p), "read: /repo/a.md"; got != want {
		t.Errorf("LastToolCall = %q, want %q", got, want)
	}
}

// No call recorded, no transcript, or a line that is not JSON: no guess. A
// guessed wedge is worse than none, the rule TurnCapHint already follows.
func TestLastToolCallSaysNothingWhenThereIsNothingToName(t *testing.T) {
	dir := t.TempDir()
	if got := LastToolCall(filepath.Join(dir, "absent.jsonl")); got != "" {
		t.Errorf("absent transcript: %q", got)
	}
	p := filepath.Join(dir, "t.jsonl")
	if err := os.WriteFile(p, []byte("not json\n"+`{"type":"message_end"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LastToolCall(p); got != "" {
		t.Errorf("no call recorded: %q", got)
	}
}

// A command can be a whole script: the line stays one line and bounded, so the
// stop message does not bury the one fact it carries.
func TestLastToolCallKeepsTheNameOnOneBoundedLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.jsonl")
	long := strings.Repeat("x", 600)
	body := `{"type":"tool_execution_update","toolName":"bash","args":{"command":"echo a\necho ` + long + `"}}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LastToolCall(p)
	if strings.Contains(got, "\n") || len(got) > 260 {
		t.Errorf("want one bounded line, got %d bytes: %q", len(got), got)
	}
}

// CLI-094 AC4: a stopped run says what it left behind. The launcher moves the
// previous round away before it starts, so review.md is either absent (nothing
// written), a provisional record (PENDING), or a verdict written before the stop.
func TestStoppedRunRecordSaysWhatTheRunLeftBehind(t *testing.T) {
	dir := t.TempDir()
	review := filepath.Join(dir, "review.md")

	if got := StoppedRunRecord(review); !strings.Contains(got, "review.md was not rewritten by this run") {
		t.Errorf("absent file: %q", got)
	}

	pending := "---\nverdict: PENDING\nreviewer: pi\n---\n\nfindings so far\n"
	if err := os.WriteFile(review, []byte(pending), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := StoppedRunRecord(review); !strings.Contains(got, "provisional review.md kept (verdict PENDING)") {
		t.Errorf("pending file: %q", got)
	}

	signed := "---\nverdict: PASS\nreviewer: pi\nreviewed_sha: abc1234\n---\n\nall verified\n"
	if err := os.WriteFile(review, []byte(signed), 0o600); err != nil {
		t.Fatal(err)
	}
	got := StoppedRunRecord(review)
	if !strings.Contains(got, "PASS") || strings.Contains(got, "was not rewritten") || strings.Contains(got, "PENDING") {
		t.Errorf("signed file must name its verdict and nothing else: %q", got)
	}
}

func TestStoppedRunRecordDoesNotClaimAnUnreadableFile(t *testing.T) {
	review := filepath.Join(t.TempDir(), "review.md")
	if err := os.WriteFile(review, []byte("no frontmatter at all\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := StoppedRunRecord(review)
	if strings.Contains(got, "provisional review.md kept") || strings.Contains(got, "was not rewritten") {
		t.Errorf("a file that cannot be read must not be described as either state: %q", got)
	}
	if !strings.Contains(got, "review.md") {
		t.Errorf("it must still name the file: %q", got)
	}
}

// CLI-094 round 1: the second signer writes review-second.md, and a stop of its
// run must not claim anything about review.md, the first signer's verdict.
func TestStoppedRunRecordNamesTheFileItWasAskedAbout(t *testing.T) {
	dir := t.TempDir()
	second := filepath.Join(dir, "review-second.md")
	if got := StoppedRunRecord(second); !strings.Contains(got, "review-second.md was not rewritten") || strings.Contains(got, " review.md") {
		t.Errorf("absent second file: %q", got)
	}
	if err := os.WriteFile(second, []byte("---\nverdict: PENDING\nreviewer: agy\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := StoppedRunRecord(second); !strings.Contains(got, "provisional review-second.md kept (verdict PENDING)") {
		t.Errorf("pending second file: %q", got)
	}
}
