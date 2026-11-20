package pushlease

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gitenv"
	"github.com/mlorentedev/dotfiles/cli/internal/gittest"
)

// TestMain keeps every git these tests start from forking a background gc that
// outlives the test (#2803).
func TestMain(m *testing.M) {
	gittest.Quiet()
	os.Exit(m.Run())
}

// fixture is a clone on feat/x whose origin is a local bare repository with
// main as its default branch. The clone's hooks directory is its own, so this
// machine's global hooks never decide a test.
type fixture struct {
	remote string
	clone  string
	hooks  string
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitenv.WithoutRepository(os.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	top := t.TempDir()
	f := fixture{
		remote: filepath.Join(top, "remote.git"),
		clone:  filepath.Join(top, "clone"),
		hooks:  filepath.Join(top, "hooks"),
	}
	if err := os.Mkdir(f.hooks, 0o750); err != nil {
		t.Fatal(err)
	}
	git(t, top, "init", "-q", "--bare", "-b", "main", f.remote)
	git(t, top, "init", "-q", "-b", "main", f.clone)
	for _, kv := range [][2]string{
		{"core.hooksPath", f.hooks},
		{"commit.gpgsign", "false"},
		{"user.email", "test@example.invalid"},
		{"user.name", "test"},
	} {
		git(t, f.clone, "config", kv[0], kv[1])
	}
	git(t, f.clone, "remote", "add", "origin", f.remote)
	git(t, f.clone, "commit", "-q", "--allow-empty", "-m", "base")
	git(t, f.clone, "push", "-q", "origin", "main")
	git(t, f.clone, "remote", "set-head", "origin", "main")
	git(t, f.clone, "switch", "-q", "-c", "feat/x")
	git(t, f.clone, "commit", "-q", "--allow-empty", "-m", "work")
	git(t, f.clone, "push", "-q", "-u", "origin", "feat/x")
	return f
}

// remoteSHA reads the branch from the bare repository itself, never from the
// clone's remote-tracking ref.
func (f fixture) remoteSHA(t *testing.T, branch string) string {
	t.Helper()
	return git(t, f.remote, "rev-parse", "refs/heads/"+branch)
}

// rebaseOntoNewMain lands a commit on the remote main and rebases feat/x onto
// it, so the next push of feat/x is not a fast-forward.
func (f fixture) rebaseOntoNewMain(t *testing.T) {
	t.Helper()
	git(t, f.clone, "switch", "-q", "main")
	git(t, f.clone, "commit", "-q", "--allow-empty", "-m", "upstream")
	git(t, f.clone, "push", "-q", "origin", "main")
	git(t, f.clone, "switch", "-q", "feat/x")
	git(t, f.clone, "rebase", "-q", "main")
}

// pushLease is the whole command: Prepare, then Push when nothing refused.
func pushLease(req Request) (string, string, error) {
	plan, err := Prepare(req)
	if err != nil {
		return "", "", err
	}
	var out, errOut bytes.Buffer
	err = Push(plan, &out, &errOut)
	return out.String(), errOut.String(), err
}

func TestPush_StaleExpectLeavesTheRemote(t *testing.T) {
	tests := []struct {
		name string
		// arrange returns the --expect to pass; the remote sha must not change.
		arrange func(t *testing.T, f fixture) string
	}{
		{
			name: "someone else pushed after the read",
			arrange: func(t *testing.T, f fixture) string {
				read := f.remoteSHA(t, "feat/x")
				git(t, f.clone, "switch", "-q", "-c", "someone-else")
				git(t, f.clone, "commit", "-q", "--allow-empty", "-m", "theirs")
				git(t, f.clone, "push", "-q", "origin", "someone-else:feat/x")
				git(t, f.clone, "switch", "-q", "feat/x")
				f.rebaseOntoNewMain(t)
				return read
			},
		},
		{
			// git reports a ref that already equals the pushed commit as up to
			// date before it consults the lease (see lane.Claimer.Acquire).
			name: "the remote already equals HEAD and the expected sha is another",
			arrange: func(t *testing.T, f fixture) string {
				return git(t, f.clone, "rev-parse", "main")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			expect := tt.arrange(t, f)
			before := f.remoteSHA(t, "feat/x")

			out, stderr, err := pushLease(Request{Dir: f.clone, Expect: expect})

			if err == nil {
				t.Fatalf("pushLease with a stale --expect succeeded; output %q, stderr:\n%s", out, stderr)
			}
			if out != "" {
				t.Errorf("output = %q, want nothing on a refusal", out)
			}
			if after := f.remoteSHA(t, "feat/x"); after != before {
				t.Errorf("remote feat/x moved from %s to %s", before, after)
			}
		})
	}
}

// remoteRefs is every ref the bare repository holds, to prove a refusal
// touched none of them.
func (f fixture) remoteRefs(t *testing.T) string {
	t.Helper()
	return git(t, f.remote, "for-each-ref", "--format=%(refname) %(objectname)")
}

// pushedBranch creates name from the current HEAD with one commit, pushes it
// and leaves it checked out.
func (f fixture) pushedBranch(t *testing.T, name string) string {
	t.Helper()
	git(t, f.clone, "switch", "-q", "-c", name)
	git(t, f.clone, "commit", "-q", "--allow-empty", "-m", "on "+name)
	git(t, f.clone, "push", "-q", "origin", name)
	return f.remoteSHA(t, name)
}

func TestPrepare_Refuses(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		arrange func(t *testing.T, f fixture) Request
	}{
		{"detached HEAD", CodeDetachedHead, func(t *testing.T, f fixture) Request {
			git(t, f.clone, "switch", "-q", "--detach")
			return Request{Dir: f.clone, Expect: f.remoteSHA(t, "feat/x")}
		}},
		{"main", CodeProtectedBranch, func(t *testing.T, f fixture) Request {
			git(t, f.clone, "switch", "-q", "main")
			return Request{Dir: f.clone, Expect: f.remoteSHA(t, "main")}
		}},
		{"master", CodeProtectedBranch, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Expect: f.pushedBranch(t, "master")}
		}},
		{"the remote default, read from origin/HEAD", CodeDefaultBranch, func(t *testing.T, f fixture) Request {
			expect := f.pushedBranch(t, "develop")
			git(t, f.clone, "remote", "set-head", "origin", "develop")
			return Request{Dir: f.clone, Expect: expect}
		}},
		{"the remote default, origin/HEAD unset, read from the remote", CodeDefaultBranch, func(t *testing.T, f fixture) Request {
			expect := f.pushedBranch(t, "develop")
			git(t, f.remote, "symbolic-ref", "HEAD", "refs/heads/develop")
			git(t, f.clone, "remote", "set-head", "origin", "--delete")
			return Request{Dir: f.clone, Expect: expect}
		}},
		{"the remote default changed since origin/HEAD was recorded", CodeDefaultBranch, func(t *testing.T, f fixture) Request {
			expect := f.pushedBranch(t, "develop")
			git(t, f.remote, "symbolic-ref", "HEAD", "refs/heads/develop")
			return Request{Dir: f.clone, Expect: expect}
		}},
		{"the remote reports no default and origin/HEAD is unset", CodeDefaultUnresolved, func(t *testing.T, f fixture) Request {
			git(t, f.remote, "symbolic-ref", "HEAD", "refs/heads/unborn")
			git(t, f.clone, "remote", "set-head", "origin", "--delete")
			return Request{Dir: f.clone, Expect: f.remoteSHA(t, "feat/x")}
		}},
		{"the remote cannot be read", CodeRemoteUnreadable, func(t *testing.T, f fixture) Request {
			git(t, f.clone, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
			return Request{Dir: f.clone, Expect: f.remoteSHA(t, "feat/x")}
		}},
		{"--remote looks like an option", CodeUnknownRemote, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Remote: "--mirror", Expect: f.remoteSHA(t, "feat/x")}
		}},
		{"--expect missing", CodeBadExpect, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone}
		}},
		{"--expect abbreviated", CodeBadExpect, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Expect: f.remoteSHA(t, "feat/x")[:7]}
		}},
		{"--expect one short of a full sha", CodeBadExpect, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Expect: f.remoteSHA(t, "feat/x")[:39]}
		}},
		{"--expect symbolic", CodeBadExpect, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Expect: "origin/feat/x"}
		}},
		{"--remote not configured", CodeUnknownRemote, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Remote: "nosuch", Expect: f.remoteSHA(t, "feat/x")}
		}},
		{"--remote given as a path", CodeUnknownRemote, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Remote: f.remote, Expect: f.remoteSHA(t, "feat/x")}
		}},
		{"--remote given as an https URL", CodeUnknownRemote, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Remote: "https://example.invalid/x.git", Expect: f.remoteSHA(t, "feat/x")}
		}},
		{"--remote given as an scp-style URL", CodeUnknownRemote, func(t *testing.T, f fixture) Request {
			return Request{Dir: f.clone, Remote: "git@example.invalid:x.git", Expect: f.remoteSHA(t, "feat/x")}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			req := tt.arrange(t, f)
			git(t, f.clone, "commit", "-q", "--allow-empty", "-m", "unpushed")
			before := f.remoteRefs(t)

			out, stderr, err := pushLease(req)

			var refusal *Refusal
			if !errors.As(err, &refusal) {
				t.Fatalf("err = %v, want a *Refusal %s; output %q, stderr:\n%s", err, tt.code, out, stderr)
			}
			if refusal.Code != tt.code {
				t.Errorf("code = %s (%s), want %s", refusal.Code, refusal.Reason, tt.code)
			}
			if after := f.remoteRefs(t); after != before {
				t.Errorf("remote refs changed:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestPrepare_RemoteResolutionOrder(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		config   map[string]string // "" unsets the key
		want     string
	}{
		{"--remote beats the configuration", "byflag", map[string]string{
			"branch.feat/x.pushRemote": "bypushremote",
		}, "byflag"},
		{"branch pushRemote beats remote.pushDefault", "", map[string]string{
			"branch.feat/x.pushRemote": "bypushremote", "remote.pushDefault": "bypushdefault", "branch.feat/x.remote": "byupstream",
		}, "bypushremote"},
		{"remote.pushDefault beats the upstream remote", "", map[string]string{
			"remote.pushDefault": "bypushdefault", "branch.feat/x.remote": "byupstream",
		}, "bypushdefault"},
		{"the upstream remote beats origin", "", map[string]string{
			"branch.feat/x.remote": "byupstream",
		}, "byupstream"},
		{"origin when nothing is configured", "", map[string]string{
			"branch.feat/x.remote": "",
		}, "origin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			// Every candidate names the same bare repository, so only the
			// choice differs and each would pass every other check.
			for _, name := range []string{"byflag", "bypushremote", "bypushdefault", "byupstream"} {
				git(t, f.clone, "remote", "add", name, f.remote)
			}
			for key, value := range tt.config {
				if value == "" {
					git(t, f.clone, "config", "--unset", key)
					continue
				}
				git(t, f.clone, "config", key, value)
			}

			plan, err := Prepare(Request{Dir: f.clone, Remote: tt.explicit, Expect: f.remoteSHA(t, "feat/x")})
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}
			if plan.Remote != tt.want {
				t.Errorf("remote = %s, want %s", plan.Remote, tt.want)
			}
		})
	}
}

func TestPush_PrePushHookRuns(t *testing.T) {
	f := newFixture(t)
	old := f.remoteSHA(t, "feat/x")
	f.rebaseOntoNewMain(t)
	marker := filepath.Join(t.TempDir(), "pre-push-ran")
	hook := "#!/bin/sh\necho ran > '" + filepath.ToSlash(marker) + "'\nexit 1\n"
	if err := os.WriteFile(filepath.Join(f.hooks, "pre-push"), []byte(hook), 0o755); err != nil { //nolint:gosec // a hook must be executable
		t.Fatal(err)
	}

	out, stderr, err := pushLease(Request{Dir: f.clone, Expect: old})

	if err == nil {
		t.Fatalf("push succeeded past a refusing pre-push hook; output %q", out)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Errorf("pre-push hook did not run (%v); stderr:\n%s", statErr, stderr)
	}
	if got := f.remoteSHA(t, "feat/x"); got != old {
		t.Errorf("remote feat/x = %s, want it left at %s", got, old)
	}
	if out != "" {
		t.Errorf("output = %q, want nothing when the push failed", out)
	}
}

func TestPush_MovesTheRemoteAfterARebase(t *testing.T) {
	f := newFixture(t)
	old := f.remoteSHA(t, "feat/x")
	f.rebaseOntoNewMain(t)
	head := git(t, f.clone, "rev-parse", "HEAD")

	plan, err := Prepare(Request{Dir: f.clone, Expect: old})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	var out, errOut bytes.Buffer
	if err := Push(plan, &out, &errOut); err != nil {
		t.Fatalf("Push: %v\nstderr:\n%s", err, errOut.String())
	}

	if got := f.remoteSHA(t, "feat/x"); got != head {
		t.Errorf("remote feat/x = %s, want local HEAD %s", got, head)
	}
	if want := "feat/x " + old + ".." + head + "\n"; out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}
