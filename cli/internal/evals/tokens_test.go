package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// The event shapes are pi's, as a real run on nan/glm5.3-flash wrote them:
// usage rides on an assistant's message_end, turn_end repeats it, and
// agent_end carries every message again.
const (
	piUsage1 = `{"input":24249,"output":943,"cacheRead":0,"cacheWrite":0,"reasoning":733,"totalTokens":25192}`
	piUsage2 = `{"input":1113,"output":478,"cacheRead":24192,"cacheWrite":10,"reasoning":62,"totalTokens":25793}`
)

func piEnd(role, usage string) string {
	if usage == "" {
		return `{"type":"message_end","message":{"role":"` + role + `","content":""}}`
	}
	return `{"type":"message_end","message":{"role":"` + role + `","usage":` + usage + `}}`
}

func TestReadTokensSumsPiAssistantMessagesOnce(t *testing.T) {
	long := `{"type":"message_end","message":{"role":"system","content":"` + strings.Repeat("x", 200_000) + `"}}`
	p := writeTranscript(t,
		long,
		`{"type":"message_start","message":{"role":"assistant","usage":{"input":0,"output":0}}}`,
		piEnd("assistant", piUsage1),
		`{"type":"turn_end","message":{"role":"assistant","usage":`+piUsage1+`}}`,
		piEnd("toolResult", ""),
		piEnd("assistant", piUsage2),
		`{"type":"turn_end","message":{"role":"assistant","usage":`+piUsage2+`}}`,
		`{"type":"agent_end","messages":[{"role":"assistant","usage":`+piUsage1+`}]}`,
		`{"type":"message_end","message":{"role":"assistant","usa`, // cut by a kill
	)
	got := ReadTokens("pi", p)
	want := Tokens{Input: 24249 + 1113, Cached: 24192 + 10, Output: 943 + 478, Thinking: 733 + 62}
	if got == nil || *got != want {
		t.Fatalf("ReadTokens = %+v, want %+v", got, want)
	}
	if got.Total() != 25192+25793 {
		t.Errorf("Total = %d, want the runner's own totals summed, %d", got.Total(), 25192+25793)
	}
}

// agy reports usage once per finished step, on a step_update whose state is
// DONE (the shape of specs/archive/HARNESS-071-reviewer-pool/review-transcript.jsonl).
func TestReadTokensSumsAgyFinishedSteps(t *testing.T) {
	step := func(i, state, usage string) string {
		u := ""
		if usage != "" {
			u = `,"usage":` + usage
		}
		return `{"event":"step_update","step_update":{"step_index":` + i + `,"state":"` + state + `"` + u + `}}`
	}
	p := writeTranscript(t,
		`{"event":"init","init":{"model":"gemini"}}`,
		step("3", "DONE", `{"input_tokens":28535,"output_tokens":717,"thinking_tokens":533,"cache_read_tokens":0,"total_tokens":29252}`),
		// A running step's partial count is not its spend: only DONE counts,
		// or a step reported while it ran would be counted twice.
		step("4", "ACTIVE", `{"input_tokens":999,"output_tokens":9}`),
		step("4", "DONE", ""),
		step("9", "DONE", `{"input_tokens":11771,"output_tokens":737,"thinking_tokens":467,"cache_read_tokens":24437,"total_tokens":12508}`),
		`{"event":"result","result":{"status":"SUCCESS"}}`,
	)
	got := ReadTokens("agy", p)
	want := Tokens{Input: 28535 + 11771, Cached: 24437, Output: 717 + 737, Thinking: 533 + 467}
	if got == nil || *got != want {
		t.Fatalf("ReadTokens = %+v, want %+v", got, want)
	}
}

// Absent is "not reported", never zero: a record must not claim a run was free.
func TestReadTokensIsNilWhenNothingWasReported(t *testing.T) {
	cases := map[string]struct{ runner, path string }{
		"no usage":       {"pi", writeTranscript(t, piEnd("assistant", ""), `{"type":"agent_end"}`)},
		"all zero":       {"pi", writeTranscript(t, piEnd("assistant", `{"input":0,"output":0}`))},
		"empty file":     {"pi", writeTranscript(t)},
		"missing file":   {"pi", filepath.Join(t.TempDir(), "absent.jsonl")},
		"unknown runner": {"claude", writeTranscript(t, piEnd("assistant", piUsage1))},
		"agy, pi shape":  {"agy", writeTranscript(t, piEnd("assistant", piUsage1))},
	}
	for name, c := range cases {
		if got := ReadTokens(c.runner, c.path); got != nil {
			t.Errorf("%s: ReadTokens = %+v, want nil", name, got)
		}
	}
}
