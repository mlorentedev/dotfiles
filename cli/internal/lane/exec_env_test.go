package lane

import (
	"reflect"
	"slices"
	"testing"
)

// herdr names a pane's agent by the environment its foreground process was
// started with, and a launch that goes through `dotf secrets run` makes that
// process dotf, so the pane reads agent: null (#2813, #3430, observed for
// opencode on 2026-11-14). The launch names the agent itself; a launch whose
// foreground process is the agent needs nothing.
func TestLaunchEnv_NamesTheAgentOnlyWhereTheWrapperHidesIt(t *testing.T) {
	for _, tc := range []struct {
		agent string
		want  []string
	}{
		{"pi", []string{"HERDR_AGENT=pi"}},
		{"opencode", []string{"HERDR_AGENT=opencode"}},
		{"claude", nil},
		{"agy", nil},
		{"copilot", nil},
	} {
		argv, err := LaunchArgv(tc.agent, "go")
		if err != nil {
			t.Fatal(err)
		}
		if got := LaunchEnv(argv); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: LaunchEnv(%q) = %q, want %q", tc.agent, argv, got, tc.want)
		}
	}
}

// A launch table entry added later through the wrapper is covered without a
// second list: the agent is what follows `--`.
func TestLaunchEnv_ReadsTheAgentFromTheWrappedCommand(t *testing.T) {
	for argv, want := range map[string][]string{
		"dotf secrets run -- codex hi":                     {"HERDR_AGENT=codex"},
		"dotf secrets run --only A,B -- /opt/bin/codex hi": {"HERDR_AGENT=codex"},
		"dotf secrets run --only A,B":                      nil,
		"dotf secrets run --only A,B --":                   nil,
		"dotf lane exec --agent pi --prompt-file p":        nil,
		"agy -i -- hi": nil,
		"":             nil,
		"dotf secrets run -- C:\\bin\\codex.EXE prompt": {"HERDR_AGENT=codex"},
	} {
		if got := LaunchEnv(splitWords(argv)); !reflect.DeepEqual(got, want) {
			t.Errorf("LaunchEnv(%q) = %q, want %q", argv, got, want)
		}
	}
}

// getenv returns the first match, so a HERDR_AGENT the parent already holds
// must be replaced, not shadowed by an appended copy.
func TestWithEnv_ReplacesAnExistingKeyInsteadOfAppendingADuplicate(t *testing.T) {
	got := withEnv([]string{"A=1", "HERDR_AGENT=stale", "B=2"}, []string{"HERDR_AGENT=pi"})
	want := []string{"A=1", "B=2", "HERDR_AGENT=pi"}
	if !slices.Equal(got, want) {
		t.Errorf("withEnv = %q, want %q", got, want)
	}
	if got := withEnv([]string{"A=1"}, nil); !slices.Equal(got, []string{"A=1"}) {
		t.Errorf("no additions must change nothing, got %q", got)
	}
}

func splitWords(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
