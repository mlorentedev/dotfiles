package doctor

import (
	"bytes"
	"strings"
	"testing"
)

// TestCheckHerdrWorktreePath is the table for #2638 item 1: `dotf worktree add
// --agent` passes `herdr worktree open --path`, so a herdr whose API schema
// drops WorktreeOpenParams.path must be reported before an add finds out.
func TestCheckHerdrWorktreePath(t *testing.T) {
	const withPath = `{"schemas":{"request":{"$defs":{"WorktreeOpenParams":{"properties":{"cwd":{},"path":{}}}}}}}`
	const withoutPath = `{"schemas":{"request":{"$defs":{"WorktreeOpenParams":{"properties":{"cwd":{}}}}}}}`
	const noParams = `{"schemas":{"request":{"$defs":{}}}}`
	cases := []struct {
		name                    string
		onPath                  bool
		schema                  string // "" ⇒ the command fails
		wantFailures, wantWarns int
		wantSubstr              string
	}{
		{"herdr absent → skip", false, "", 0, 0, "herdr not installed"},
		{"path present → pass", true, withPath, 0, 0, "WorktreeOpenParams.path"},
		{"path dropped → fail", true, withoutPath, 1, 0, "no WorktreeOpenParams.path"},
		{"params gone → fail", true, noParams, 1, 0, "no WorktreeOpenParams.path"},
		{"schema command fails → warn", true, "", 0, 1, "cannot verify"},
		{"schema not JSON → warn", true, "Herdr API schema\nprotocol: 22", 0, 1, "cannot verify"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var onPath []string
			if tc.onPath {
				onPath = []string{"herdr"}
			}
			cmdOut := map[string]string{}
			if tc.schema != "" {
				cmdOut["herdr api schema --json"] = tc.schema
			}
			sys := newSys(nil, onPath, cmdOut)

			var buf bytes.Buffer
			rep := capture(&buf)
			checkHerdrWorktreePath(sys, rep)

			if rep.Failures() != tc.wantFailures || rep.Warnings() != tc.wantWarns {
				t.Fatalf("failures = %d, warnings = %d; want %d, %d\n%s", rep.Failures(), rep.Warnings(), tc.wantFailures, tc.wantWarns, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}
