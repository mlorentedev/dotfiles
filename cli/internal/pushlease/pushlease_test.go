package pushlease

import (
	"bytes"
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
