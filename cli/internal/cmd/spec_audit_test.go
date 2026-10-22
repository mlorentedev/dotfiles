package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubSpecAuditGH replaces the gh runner for one test. states maps
// "owner/name/N" to the tab-separated output `gh api --jq` would print; a
// missing key behaves like a network failure.
func stubSpecAuditGH(t *testing.T, states map[string]string) {
	t.Helper()
	orig, origWait := specAuditRun, specAuditWait
	t.Cleanup(func() { specAuditRun, specAuditWait = orig, origWait })
	specAuditWait = func(int) {}
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
		{"a zombie the default branch already archived is a stale ref, not a failure",
			map[string]string{"mlorentedev/dotfiles/1": "open\tfalse", "mlorentedev/dotfiles/2": "closed\tfalse",
				"mlorentedev/dotfiles/contents/specs/archive/B-002-two": "proposal.md"},
			false, []string{"[WARN] B-002-two", "merge the base"}},
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

// The audit gates CI, so one transient failure is retried rather than failing
// the leg (#2450): the first answer arrives on the third attempt.
func TestSpecAuditRetriesATransientFailure(t *testing.T) {
	root := makeRepo(t)
	seedAuditSpec(t, root, "A-001-one", "#1")
	stubSpecAuditGH(t, nil)
	calls := 0
	specAuditRun = func(...string) (string, string, error) {
		if calls++; calls < specAuditAttempts {
			return "", "net/http: TLS handshake timeout\n", errors.New("exit status 1")
		}
		return "open\tfalse", "", nil
	}
	if stdout, _, err := execute(t, "spec", "audit", "--repo", "mlorentedev/dotfiles"); err != nil {
		t.Fatalf("err = %v after %d call(s)\n%s", err, calls, stdout)
	}
}

// TestBoundedRunHelper is the child process TestBoundedRun_ADeadlineIsReported
// starts: it outlives the parent's deadline. It does nothing as a plain test.
func TestBoundedRunHelper(t *testing.T) {
	if os.Getenv("DOTF_TEST_BOUNDED_RUN_CHILD") != "1" {
		return
	}
	time.Sleep(10 * time.Second)
}

// The audit's own deadline must read as context.DeadlineExceeded, which the
// lookup never retries: a retried deadline would turn the 20s bound into 60s.
func TestBoundedRun_ADeadlineIsReported(t *testing.T) {
	t.Setenv("DOTF_TEST_BOUNDED_RUN_CHILD", "1")
	_, _, err := boundedRun(200*time.Millisecond, os.Args[0], "-test.run=^TestBoundedRunHelper$")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want it to wrap context.DeadlineExceeded", err)
	}
}
