package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

// pushLeaseRepo is a clone on feat/x, already pushed, whose origin is a local
// bare repository with main as its default branch. Its hooks directory is its
// own and empty, so the push dotf runs meets no machine-wide hook.
func pushLeaseRepo(t *testing.T) (clone, remote string) {
	t.Helper()
	top := t.TempDir()
	remote = filepath.Join(top, "remote.git")
	clone = filepath.Join(top, "clone")
	gitIn(t, top, "init", "-q", "--bare", "-b", "main", remote)
	gitIn(t, top, "init", "-q", "-b", "main", clone)
	gitIn(t, clone, "config", "core.hooksPath", t.TempDir())
	gitIn(t, clone, "config", "commit.gpgsign", "false")
	gitIn(t, clone, "config", "user.name", "t")
	gitIn(t, clone, "config", "user.email", "t@example.invalid")
	gitIn(t, clone, "remote", "add", "origin", remote)
	gitIn(t, clone, "commit", "-q", "--allow-empty", "-m", "base")
	gitIn(t, clone, "push", "-q", "origin", "main")
	gitIn(t, clone, "remote", "set-head", "origin", "main")
	gitIn(t, clone, "switch", "-q", "-c", "feat/x")
	gitIn(t, clone, "commit", "-q", "--allow-empty", "-m", "work")
	gitIn(t, clone, "push", "-q", "-u", "origin", "feat/x")
	return clone, remote
}

func TestPrPushLease_PushesAndPrintsTheMove(t *testing.T) {
	clone, remote := pushLeaseRepo(t)
	old := gitIn(t, remote, "rev-parse", "refs/heads/feat/x")
	gitIn(t, clone, "commit", "-q", "--amend", "--allow-empty", "-m", "work, rewritten")
	head := gitIn(t, clone, "rev-parse", "HEAD")
	inDir(t, clone)

	stdout, stderr, err := execute(t, "pr", "push-lease", "--expect", old)

	if err != nil {
		t.Fatalf("push-lease: %v\nstderr:\n%s", err, stderr)
	}
	if want := "feat/x " + old + ".." + head + "\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if got := gitIn(t, remote, "rev-parse", "refs/heads/feat/x"); got != head {
		t.Errorf("remote feat/x = %s, want %s", got, head)
	}
}

func TestPrPushLease_Refusals(t *testing.T) {
	tests := []struct {
		name     string
		args     func(t *testing.T, remote string) []string
		prepare  func(t *testing.T, clone string)
		wantCode int
		wantErr  string
	}{
		{
			name: "on main",
			args: func(t *testing.T, remote string) []string {
				return []string{"--expect", gitIn(t, remote, "rev-parse", "refs/heads/main")}
			},
			prepare:  func(t *testing.T, clone string) { gitIn(t, clone, "switch", "-q", "main") },
			wantCode: 1,
			wantErr:  "main is never force-updated",
		},
		{
			name:     "without --expect",
			args:     func(*testing.T, string) []string { return nil },
			wantCode: 1,
			wantErr:  "--expect must be the full sha",
		},
		{
			name: "with a positional argument",
			args: func(t *testing.T, remote string) []string {
				return []string{"origin", "--expect", gitIn(t, remote, "rev-parse", "refs/heads/feat/x")}
			},
			wantCode: 2,
			wantErr:  "takes no arguments",
		},
		{
			name:     "with an unknown flag",
			args:     func(*testing.T, string) []string { return []string{"--force"} },
			wantCode: 2,
			wantErr:  "unknown flag",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone, remote := pushLeaseRepo(t)
			if tt.prepare != nil {
				tt.prepare(t, clone)
			}
			before := gitIn(t, remote, "for-each-ref", "--format=%(refname) %(objectname)")
			inDir(t, clone)

			stdout, stderr, err := execute(t, append([]string{"pr", "push-lease"}, tt.args(t, remote)...)...)

			if got := ExitCode(err); got != tt.wantCode {
				t.Errorf("exit = %d (%v), want %d", got, err, tt.wantCode)
			}
			if !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.wantErr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing on a refusal", stdout)
			}
			if after := gitIn(t, remote, "for-each-ref", "--format=%(refname) %(objectname)"); after != before {
				t.Errorf("remote refs changed:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}
