package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// specIssueStateRepo builds a checkout holding one active spec per entry of
// issues (spec id -> the `issue:` value) plus an archived one that must never
// be audited.
func specIssueStateRepo(t *testing.T, issues map[string]string) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".git/HEAD", "ref: refs/heads/main\n")
	for id, issue := range issues {
		write("specs/"+id+"/proposal.md", "---\nid: \""+id+"\"\nissue: \""+issue+"\"\n---\n")
	}
	write("specs/archive/Z-001-done/proposal.md", "---\nissue: \"#999\"\n---\n")
	return root
}

// specIssueStateSys answers `git remote get-url origin` and the gh REST
// lookups; gh maps "repos/<owner>/<name>/issues/<n>" to its tab-separated
// output, and a missing key fails the way an offline gh does.
func specIssueStateSys(repo string, ghOnPath bool, gh map[string]string) *System {
	onPath := []string{"git"}
	if ghOnPath {
		onPath = append(onPath, "gh")
	}
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, onPath, map[string]string{
		"git -C " + repo + " remote get-url origin": "git@github.com:mlorentedev/dotfiles.git\n",
	})
	sys.CommandOutputBounded = func(_ time.Duration, name string, args ...string) (string, string, error) {
		if name != "gh" || len(args) < 2 {
			return "", "unexpected command", errors.New("exit status 1")
		}
		if out, ok := gh[args[1]]; ok {
			return out, "", nil
		}
		if strings.HasSuffix(args[1], "/404") {
			return "", "gh: Not Found (HTTP 404)\n", errors.New("exit status 1")
		}
		return "", "error connecting to api.github.com\n", errors.New("exit status 1")
	}
	return sys
}

func TestCheckSpecIssueState(t *testing.T) {
	cases := []struct {
		name                    string
		issues                  map[string]string
		ghOnPath                bool
		gh                      map[string]string
		wantFail, wantWarn      int
		wantSubstr, forbidSubst string
	}{
		{"every tracked issue open passes",
			map[string]string{"A-001-a": "#1", "A-002-b": "mlorentedev/hive#2"}, true,
			map[string]string{"repos/mlorentedev/dotfiles/issues/1": "open\tfalse", "repos/mlorentedev/hive/issues/2": "open\tfalse"},
			0, 0, "2 active spec(s) track an open issue", "#999"},
		{"a closed issue is a FAIL naming the spec",
			map[string]string{"A-001-a": "#1", "A-002-zombie": "#2"}, true,
			map[string]string{"repos/mlorentedev/dotfiles/issues/1": "open\tfalse", "repos/mlorentedev/dotfiles/issues/2": "closed\tfalse"},
			1, 0, "A-002-zombie: zombie: mlorentedev/dotfiles#2 is CLOSED", ""},
		{"a ref to nothing is a FAIL",
			map[string]string{"A-001-gone": "mlorentedev/kubelab#404"}, true, nil,
			1, 0, "does not exist", ""},
		{"an unanswerable lookup is a WARN, never a PASS",
			map[string]string{"A-001-offline": "#7"}, true, nil,
			0, 1, "not verified", "track an open issue"},
		{"gh absent skips, never passes",
			map[string]string{"A-001-a": "#1"}, false, nil,
			0, 0, "gh not on PATH", "track an open issue"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := specIssueStateRepo(t, tc.issues)
			var buf bytes.Buffer
			rep := capture(&buf)
			checkSpecIssueState(specIssueStateSys(repo, tc.ghOnPath, tc.gh), rep)
			out := buf.String()
			if rep.Failures() != tc.wantFail || rep.Warnings() != tc.wantWarn {
				t.Fatalf("fail=%d warn=%d, want fail=%d warn=%d\n%s",
					rep.Failures(), rep.Warnings(), tc.wantFail, tc.wantWarn, out)
			}
			if !strings.Contains(out, "spec-issue-state") {
				t.Errorf("section title missing — AC-1.1 greps for it:\n%s", out)
			}
			if !strings.Contains(out, tc.wantSubstr) {
				t.Errorf("output missing %q:\n%s", tc.wantSubstr, out)
			}
			if tc.forbidSubst != "" && strings.Contains(out, tc.forbidSubst) {
				t.Errorf("output must not contain %q:\n%s", tc.forbidSubst, out)
			}
		})
	}
}
