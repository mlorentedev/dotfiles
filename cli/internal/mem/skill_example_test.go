package mem

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// quoted matches one single-quoted shell argument.
var quoted = regexp.MustCompile(`'([^']*)'`)

// exampleBody returns the handoff body a skill pipes into `dotf mem
// handoff-write`: the single-quoted printf arguments, one per line, ahead of the
// pipe into handoff-write. printf rather than a heredoc, because a heredoc's
// terminator breaks when the example sits indented inside a list item.
func exampleBody(t *testing.T, skill string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("../../..", "harness", "skills", skill, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	for i, l := range lines {
		if !strings.Contains(l, "printf '%s\\n'") || !strings.HasSuffix(strings.TrimSpace(l), "\\") {
			continue
		}
		var body []string
		for _, next := range lines[i+1:] {
			for _, m := range quoted.FindAllStringSubmatch(next, -1) {
				body = append(body, m[1])
			}
			if strings.Contains(next, "dotf mem handoff-write") {
				return strings.Join(body, "\n") + "\n"
			}
		}
	}
	t.Fatalf("%s SKILL.md has no `printf '%%s\\n' ... | dotf mem handoff-write` example", skill)
	return ""
}

// A skill that tells an agent to pipe a body into handoff-write must show a body
// the writer accepts without warning: an example without `**Next action:**` is
// copied literally and writes a thread with no first step (#3110).
func TestAutopilotHandoffExampleCarriesNextAction(t *testing.T) {
	body := exampleBody(t, "autopilot")
	if w := ThreadWarnings(body); len(w) != 0 {
		t.Errorf("the autopilot example body draws handoff-write warnings: %q", w)
	}
}
