package cmd

import (
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// The regression this file exists for, measured 2026-08-26.
//
// `normaliseToolCall` used to return a zero ToolCall on a payload it could not
// parse, and the doc comment claimed Decide would resolve that to Allow. Decide
// does no such thing: given a valid persona and nothing consumed it BLOCKS, so a
// malformed payload blocked every tool call. The unit tests over Decide all
// passed, because the defect lived in the seam between them — prose asserting a
// behaviour the code did not have, visible only by running the binary.
//
// A gate that blocks on input it cannot read blocks on every harness upgrade,
// which is the "normal state is red" failure this design refuses.
func TestNormaliseReportsWhetherThePayloadWasUnderstood(t *testing.T) {
	for _, tc := range []struct {
		name, harnessName, payload string
		wantOK                     bool
		wantTool, wantSkill, wantS string
	}{
		{name: "claude tool call", harnessName: "claude",
			payload:  `{"session_id":"s1","tool_name":"Bash","tool_input":{"command":"ls"}}`,
			wantOK:   true,
			wantTool: "Bash", wantS: "s1"},
		{name: "claude skill invocation", harnessName: "claude",
			payload:   `{"session_id":"s1","tool_name":"Skill","tool_input":{"skill":"audit"}}`,
			wantOK:    true,
			wantTool:  "Skill",
			wantSkill: "audit", wantS: "s1"},
		// pi and opencode run in-process TypeScript, not command hooks, so the
		// wrapper this repository generates for them emits the canonical shape.
		{name: "pi via the generated wrapper", harnessName: "pi",
			payload:  `{"session":"s2","tool":"bash"}`,
			wantOK:   true,
			wantTool: "bash", wantS: "s2"},
		{name: "opencode via the generated wrapper", harnessName: "opencode",
			payload:  `{"session":"s3","tool":"bash"}`,
			wantOK:   true,
			wantTool: "bash", wantS: "s3"},
		{name: "canonical skill invocation", harnessName: "pi",
			payload:   `{"session":"s2","tool":"skill","skill":"audit"}`,
			wantOK:    true,
			wantTool:  "skill",
			wantSkill: "audit", wantS: "s2"},
		// agy uses claude's command-hook shape: ~/.gemini/settings.json declares
		// BeforeTool in that exact format, so it takes the default branch.
		{name: "agy command hook", harnessName: "agy",
			payload:  `{"session_id":"s4","tool_name":"Bash","tool_input":{}}`,
			wantOK:   true,
			wantTool: "Bash", wantS: "s4"},

		// Every one of these must report NOT understood, so the caller allows.
		{name: "not json", harnessName: "claude", payload: `not json at all`},
		{name: "empty", harnessName: "claude", payload: ``},
		{name: "json without a tool", harnessName: "claude", payload: `{"session_id":"s1"}`},
		{name: "command shape sent to a wrapper harness", harnessName: "pi", payload: `{"session_id":"s1","tool_name":"Bash"}`},
		{name: "canonical shape sent to a command harness", harnessName: "claude", payload: `{"session":"s1","tool":"bash"}`},
		{name: "json array", harnessName: "claude", payload: `[1,2,3]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call, ok := normaliseToolCall(tc.harnessName, []byte(tc.payload))
			if ok != tc.wantOK {
				t.Fatalf("understood = %v, want %v (payload %q)", ok, tc.wantOK, tc.payload)
			}
			if !tc.wantOK {
				return
			}
			if call.Tool != tc.wantTool || call.Skill != tc.wantSkill || call.SessionID != tc.wantS {
				t.Errorf("got {tool:%q skill:%q session:%q}, want {tool:%q skill:%q session:%q}",
					call.Tool, call.Skill, call.SessionID, tc.wantTool, tc.wantSkill, tc.wantS)
			}
		})
	}
}

// And the seam itself: an unparseable payload must not reach Decide as "a real
// call with nothing consumed". Pinned end to end so the two halves cannot drift
// back apart.
func TestUnreadablePayloadNeverReachesDecideAsARealCall(t *testing.T) {
	persona := &harness.Persona{
		Name:   "reviewer",
		Skills: []harness.SkillBinding{{ID: "adversarial-review", Enforce: harness.EnforceBlock}},
	}
	call, ok := normaliseToolCall("claude", []byte("not json"))
	if ok {
		t.Fatal("garbage must not be reported as understood")
	}
	// The proof that the `ok` flag is load-bearing: were the caller to ignore it
	// and hand this straight to Decide, the result would be Block.
	if r := harness.Decide(harness.GateInput{Persona: persona, Call: call}); r.Decision != harness.Block {
		t.Fatal("Decide no longer blocks on an empty call — this test's premise is stale, re-check the caller")
	}
}

// The skill name is read from whichever key a harness uses, and only when the
// tool IS the skill primitive. A tool merely named in the arguments is not an
// invocation.
func TestSkillArgOnlyReadsTheSkillPrimitive(t *testing.T) {
	if got := skillArg("Bash", map[string]any{"skill": "audit"}); got != "" {
		t.Errorf("a non-skill tool must yield no skill, got %q", got)
	}
	for _, key := range []string{"skill", "name", "command"} {
		if got := skillArg("skill", map[string]any{key: "audit"}); got != "audit" {
			t.Errorf("key %q: got %q, want audit", key, got)
		}
	}
	if got := skillArg("Skill", map[string]any{"skill": "   "}); got != "" {
		t.Errorf("a blank skill name must yield no skill, got %q", got)
	}
}
