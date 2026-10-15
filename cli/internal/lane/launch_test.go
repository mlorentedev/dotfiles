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
	} {
		got, err := LaunchArgv(tc.agent, prompt)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: LaunchArgv = %q, %v; want %q", tc.agent, got, err, tc.want)
		}
	}
}

// An agent not in the table is refused with the ones that are, never guessed: a
// wrong flag would start the successor in the wrong mode without a word.
func TestLaunchArgv_RefusesAnUnknownAgentNamingTheKnownOnes(t *testing.T) {
	_, err := LaunchArgv("codex", "go")
	if err == nil || !strings.Contains(err.Error(), "codex") || !strings.Contains(err.Error(), "agy, claude, pi") {
		t.Errorf("want an error naming codex and the known agents, got %v", err)
	}
}

// A prompt that starts like a flag or a file attachment (pi reads @path as one)
// would not reach the agent as a prompt.
func TestLaunchArgv_RefusesAPromptThatReadsAsAFlagOrAFile(t *testing.T) {
	for _, p := range []string{"-p do it", "@notes.md", "", "  "} {
		if _, err := LaunchArgv("claude", p); err == nil {
			t.Errorf("prompt %q: want a refusal", p)
		}
	}
}
