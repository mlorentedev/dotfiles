package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCheckGHScopes(t *testing.T) {
	reply := func(out, errOut string, err error) func(time.Duration, string, ...string) (string, string, error) {
		return func(time.Duration, string, ...string) (string, string, error) { return out, errOut, err }
	}
	cases := []struct {
		name     string
		onPath   bool
		gh       func(time.Duration, string, ...string) (string, string, error)
		wantWarn int
		want     string
		forbid   string
	}{
		{"every scope granted passes", true, reply("HTTP/2.0 200 OK\nX-Oauth-Scopes: gist, read:org, read:project, repo, workflow\n\n", "", nil), 0, "carries every scope", ""},
		{"a missing scope names the refresh", true, reply("HTTP/2.0 200 OK\nX-Oauth-Scopes: gist, read:org, repo, workflow\n\n", "", nil), 1, "gh auth refresh -h github.com -s read:project", "carries every scope"},
		{"logged out names the login", true, reply("", "To get started with GitHub CLI, please run:  gh auth login\n", errors.New("exit status 4")), 1, "gh auth login -s gist,read:org,read:project,repo,workflow", "carries every scope"},
		{"an unreachable API warns, never passes", true, reply("", "error connecting to api.github.com\n", errors.New("exit status 1")), 1, "could not read the token's scopes", "carries every scope"},
		{"a token that reports no scopes skips", true, reply("HTTP/2.0 200 OK\nContent-Type: application/json\n\n{}", "", nil), 0, "does not report its scopes", "carries every scope"},
		{"gh absent skips", false, reply("", "", nil), 0, "gh not on PATH", "carries every scope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			onPath := []string{}
			if tc.onPath {
				onPath = append(onPath, "gh")
			}
			sys := newSys(nil, onPath, nil)
			sys.CommandOutputBounded = tc.gh
			var buf bytes.Buffer
			rep := capture(&buf)
			checkGHScopes(sys, rep)
			out := buf.String()
			if rep.Failures() != 0 || rep.Warnings() != tc.wantWarn {
				t.Fatalf("fail=%d warn=%d, want 0/%d\n%s", rep.Failures(), rep.Warnings(), tc.wantWarn, out)
			}
			if !strings.Contains(out, "GitHub token scopes") || !strings.Contains(out, tc.want) {
				t.Errorf("output missing the section or %q:\n%s", tc.want, out)
			}
			if tc.forbid != "" && strings.Contains(out, tc.forbid) {
				t.Errorf("output must not contain %q:\n%s", tc.forbid, out)
			}
		})
	}
}
