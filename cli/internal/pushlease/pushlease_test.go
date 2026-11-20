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
