package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubSpecAuditGH replaces the gh runner for one test. states maps
// "owner/name/N" to the tab-separated output `gh api --jq` would print; a
// missing key behaves like a network failure.
func stubSpecAuditGH(t *testing.T, states map[string]string) {
	t.Helper()
	orig := specAuditRun
	t.Cleanup(func() { specAuditRun = orig })
	specAuditRun = func(args ...string) (string, string, error) {
		key := strings.TrimPrefix(args[1], "repos/")
		key = strings.Replace(key, "/issues/", "/", 1)
		if out, ok := states[key]; ok {
			return out, "", nil
		}
		return "", "dial tcp: lookup api.github.com: no such host\n", errors.New("exit status 1")
	}
}

func seedAuditSpec(t *testing.T, root, id, issue string) {
	t.Helper()
	dir := filepath.Join(root, "specs", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: \"" + id + "\"\nissue: \"" + issue + "\"\n---\n# " + id + "\n"
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSpecAuditExitIssueState(t *testing.T) {
	cases := []struct {
		name     string
		states   map[string]string
		wantErr  bool
		wantText []string
	}{
		{"every issue open is a clean exit",
			map[string]string{"mlorentedev/dotfiles/1": "open\tfalse", "mlorentedev/dotfiles/2": "open\tfalse"},
			false, []string{"[OK] 2 active spec(s)"}},
		{"a closed issue fails and names the zombie",
			map[string]string{"mlorentedev/dotfiles/1": "open\tfalse", "mlorentedev/dotfiles/2": "closed\tfalse"},
			true, []string{"[FAIL] B-002-two", "zombie: mlorentedev/dotfiles#2 is CLOSED"}},
		{"an unanswerable lookup is never a clean exit",
			map[string]string{"mlorentedev/dotfiles/1": "open\tfalse"},
			true, []string{"[UNANSWERABLE] B-002-two", "no such host"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := makeRepo(t)
			seedAuditSpec(t, root, "A-001-one", "mlorentedev/dotfiles#1")
			seedAuditSpec(t, root, "B-002-two", "#2")
			stubSpecAuditGH(t, tc.states)

			stdout, _, err := execute(t, "spec", "audit", "--repo", "mlorentedev/dotfiles")
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v\n%s", err, tc.wantErr, stdout)
			}
			for _, w := range tc.wantText {
				if !strings.Contains(stdout, w) {
					t.Errorf("output missing %q:\n%s", w, stdout)
				}
			}
		})
	}
}

func TestSpecAuditRefusesWithoutAHomeRepo(t *testing.T) {
	root := makeRepo(t) // no origin remote
	seedAuditSpec(t, root, "A-001-one", "#1")
	stubSpecAuditGH(t, nil)
	if _, _, err := execute(t, "spec", "audit"); err == nil || !strings.Contains(err.Error(), "--repo") {
		t.Fatalf("want a refusal naming --repo, got %v", err)
	}
}
