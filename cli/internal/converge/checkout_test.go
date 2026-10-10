package converge

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

// gitAt runs git in dir with a fixed identity and returns trimmed stdout.
func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "init.defaultBranch=main"}, args...)...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commitFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", rel)
	gitAt(t, dir, "commit", "-q", "-m", "write "+rel)
	gitAt(t, dir, "push", "-q", "origin", "HEAD")
}

// checkoutFixture is an upstream holding one dotfiles commit, a writer clone
// that can push more, and a checkout path that does not exist yet.
type checkoutFixture struct {
	upstream, writer string
	env              Env
	step             checkout
}

func newCheckoutFixture(t *testing.T) checkoutFixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	root := t.TempDir()
	f := checkoutFixture{upstream: filepath.Join(root, "upstream.git"), writer: filepath.Join(root, "writer")}
	gitAt(t, root, "init", "-q", "--bare", f.upstream)
	gitAt(t, root, "clone", "-q", f.upstream, f.writer)
	commitFile(t, f.writer, checkoutMarker, "DOTF_VERSION=1\n")
	f.env = Env{RepoRoot: filepath.Join(root, "Projects", "dotfiles"), GOOS: "linux"}
	f.step = checkout{run: gitconfig.ExecRunner, url: f.upstream, has: onPath}
	return f
}

// The criterion from zero: a plan clones nothing and holds the later steps
// back; an apply clones and passes the probe; a second run changes nothing.
func TestCheckout_ClonesFromZeroThenIsIdempotent(t *testing.T) {
	f := newCheckoutFixture(t)

	plan, err := f.step.Reconcile(f.env, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes != 1 || plan.Gate == "" {
		t.Fatalf("plan from zero = %+v, want one change and a gate", plan)
	}
	if _, err := os.Stat(f.env.RepoRoot); err == nil {
		t.Fatal("a plan must not clone")
	}

	if _, err := f.step.Reconcile(f.env, false); err != nil {
		t.Fatal(err)
	}
	if err := f.step.Probe(f.env); err != nil {
		t.Fatalf("probe after the clone: %v", err)
	}
	again, err := f.step.Reconcile(f.env, false)
	if err != nil || again.Changes != 0 || again.Skip != "" {
		t.Fatalf("second run = %+v, %v; want converged", again, err)
	}
}

// A checkout behind its upstream: the plan reports the fast-forward and leaves
// HEAD alone; the apply moves it, by a fast-forward only.
func TestCheckout_FastForwardsABehindCheckout(t *testing.T) {
	f := newCheckoutFixture(t)
	gitAt(t, f.writer, "clone", "-q", f.upstream, f.env.RepoRoot)
	commitFile(t, f.writer, "README.md", "new\n")
	before := gitAt(t, f.env.RepoRoot, "rev-parse", "HEAD")

	plan, err := f.step.Reconcile(f.env, true)
	if err != nil || plan.Changes != 1 || plan.Gate != "" {
		t.Fatalf("plan = %+v, %v; want one fast-forward and no gate", plan, err)
	}
	if got := gitAt(t, f.env.RepoRoot, "rev-parse", "HEAD"); got != before {
		t.Fatal("a plan must not move HEAD")
	}

	if _, err := f.step.Reconcile(f.env, false); err != nil {
		t.Fatal(err)
	}
	if got, want := gitAt(t, f.env.RepoRoot, "rev-parse", "HEAD"), gitAt(t, f.writer, "rev-parse", "HEAD"); got != want {
		t.Errorf("HEAD = %s, want the upstream's %s", got, want)
	}
}

// Local edits are never merged over (ADR-019 decision 2): the step reports
// why it left the checkout alone and the run goes on with it as it is.
func TestCheckout_LeavesADirtyCheckoutAlone(t *testing.T) {
	f := newCheckoutFixture(t)
	gitAt(t, f.writer, "clone", "-q", f.upstream, f.env.RepoRoot)
	commitFile(t, f.writer, "README.md", "new\n")
	if err := os.WriteFile(filepath.Join(f.env.RepoRoot, checkoutMarker), []byte("local edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := gitAt(t, f.env.RepoRoot, "rev-parse", "HEAD")

	res, err := f.step.Reconcile(f.env, false)
	if err != nil || !strings.Contains(res.Skip, "dirty") {
		t.Fatalf("got %+v, %v; want a skip naming the dirty worktree", res, err)
	}
	if got := gitAt(t, f.env.RepoRoot, "rev-parse", "HEAD"); got != before {
		t.Error("a dirty checkout must not move")
	}
}

// Another repository at the checkout path (a `dotf converge` run from inside
// some other project) is refused before git touches it.
func TestCheckout_RefusesARepositoryThatIsNotDotfiles(t *testing.T) {
	f := newCheckoutFixture(t)
	if err := os.MkdirAll(f.env.RepoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.env.RepoRoot, "go.mod"), []byte("module x\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := f.step.Reconcile(f.env, false); err == nil || !strings.Contains(err.Error(), "not a dotfiles checkout") {
		t.Fatalf("want a refusal naming the missing %s, got %v", checkoutMarker, err)
	}
}

// From zero without git there is nothing the run can do; the failure names
// the fix on the OS where git is most often missing.
func TestCheckout_NoGitFromZeroFailsWithTheFix(t *testing.T) {
	f := newCheckoutFixture(t)
	f.step.has = func(string) bool { return false }
	f.env.GOOS = "darwin"

	if _, err := f.step.Reconcile(f.env, true); err == nil || !strings.Contains(err.Error(), "xcode-select --install") {
		t.Fatalf("want the Command Line Tools hint, got %v", err)
	}
}

func TestCheckout_SkippedWithoutARunner(t *testing.T) {
	res, err := checkout{}.Reconcile(Env{RepoRoot: t.TempDir()}, false)
	if err != nil || res.Skip == "" {
		t.Fatalf("no runner must skip, got %+v, %v", res, err)
	}
}
