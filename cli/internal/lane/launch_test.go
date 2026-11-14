package lane

import (
	"reflect"
	"strings"
	"testing"
)

// The prompt is one argv element for every agent: it holds newlines, quotes and
// a dollar sign, which is what a shell would have mangled and why dotf starts the
// agent itself instead of typing a command line into the pane.
func TestLaunchArgv_PinsEachAgentAndPassesThePromptAsOneArgument(t *testing.T) {
	prompt := "Carril \"fixes\"\nbug: $(rm -rf ~) it's `x`\n- item"
	for _, tc := range []struct {
		agent string
		want  []string
	}{
		{"claude", []string{"claude", prompt}},
		{"pi", []string{"dotf", "secrets", "run", "--only", "NAN_API_KEY,OPENROUTER_API_KEY", "--", "pi", prompt}},
		{"agy", []string{"agy", "-i", prompt}},
		{"copilot", []string{"copilot", "-i", prompt}},
		{"opencode", []string{"dotf", "secrets", "run", "--only", "NAN_API_KEY", "--", "opencode", "--prompt", prompt}},
	} {
		got, err := LaunchArgv(tc.agent, prompt, "")
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: LaunchArgv = %q, %v; want %q", tc.agent, got, err, tc.want)
		}
	}
}

// An agent not in the table is refused with the ones that are, never guessed: a
// wrong flag would start the successor in the wrong mode without a word.
func TestLaunchArgv_RefusesAnUnknownAgentNamingTheKnownOnes(t *testing.T) {
	_, err := LaunchArgv("codex", "go", "")
	if err == nil || !strings.Contains(err.Error(), "codex") || !strings.Contains(err.Error(), "agy, claude, copilot, opencode, pi") {
		t.Errorf("want an error naming codex and the known agents, got %v", err)
	}
}

// A prompt that starts like a flag or a file attachment (pi reads @path as one)
// would not reach the agent as a prompt.
func TestLaunchArgv_RefusesAPromptThatReadsAsAFlagOrAFile(t *testing.T) {
	for _, p := range []string{"-p do it", "@notes.md", "", "  "} {
		if _, err := LaunchArgv("claude", p, ""); err == nil {
			t.Errorf("prompt %q: want a refusal", p)
		}
	}
}

// --model goes right after the agent's own executable, before its mode flag and
// the prompt, so the agent reads it as its model and not as part of the prompt;
// for the agents started through `dotf secrets run` that is after the `--`.
// Every agent in the table was observed accepting `--model <id>` (#3485).
func TestLaunchArgv_InsertsTheModelAfterTheAgentExecutable(t *testing.T) {
	for _, tc := range []struct {
		agent string
		want  []string
	}{
		{"claude", []string{"claude", "--model", "m/x", "go"}},
		{"pi", []string{"dotf", "secrets", "run", "--only", "NAN_API_KEY,OPENROUTER_API_KEY", "--", "pi", "--model", "m/x", "go"}},
		{"agy", []string{"agy", "--model", "m/x", "-i", "go"}},
		{"copilot", []string{"copilot", "--model", "m/x", "-i", "go"}},
		{"opencode", []string{"dotf", "secrets", "run", "--only", "NAN_API_KEY", "--", "opencode", "--model", "m/x", "--prompt", "go"}},
	} {
		got, err := LaunchArgv(tc.agent, "go", "m/x")
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: LaunchArgv = %q, %v; want %q", tc.agent, got, err, tc.want)
		}
	}
	got, err := LaunchArgv("claude", "go", "")
	if err != nil || !reflect.DeepEqual(got, []string{"claude", "go"}) {
		t.Errorf("no model: LaunchArgv = %q, %v; want the bare launch", got, err)
	}
	// The id is inserted as validated: trimmed, so the agent never sees the
	// blank a shell left around it.
	got, err = LaunchArgv("claude", "go", " sonnet ")
	if err != nil || !reflect.DeepEqual(got, []string{"claude", "--model", "sonnet", "go"}) {
		t.Errorf("padded model: LaunchArgv = %q, %v; want it trimmed", got, err)
	}
}

// A model that is blank or starts like a flag would be read as something else
// by the agent, silently: refused before anything starts.
func TestLaunchArgv_RefusesAModelThatIsBlankOrReadsAsAFlag(t *testing.T) {
	for _, m := range []string{" ", "-m", "--model"} {
		if _, err := LaunchArgv("claude", "go", m); err == nil || !strings.Contains(err.Error(), "model") {
			t.Errorf("model %q: want a refusal naming the model, got %v", m, err)
		}
	}
}
