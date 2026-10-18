package converge

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
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
	commitFile(t, f.writer, env.CheckoutMarker, "DOTF_VERSION=1\n")
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

// A checkout behind its upstream: the plan reports the fast-forward, leaves
// HEAD alone and holds back the steps that would read the old tree; the apply
// moves it, by a fast-forward only.
func TestCheckout_FastForwardsABehindCheckout(t *testing.T) {
	f := newCheckoutFixture(t)
	gitAt(t, f.writer, "clone", "-q", f.upstream, f.env.RepoRoot)
	commitFile(t, f.writer, "README.md", "new\n")
	before := gitAt(t, f.env.RepoRoot, "rev-parse", "HEAD")

	plan, err := f.step.Reconcile(f.env, true)
	if err != nil || plan.Changes != 1 || plan.Gate == "" {
		t.Fatalf("plan = %+v, %v; want one fast-forward and a gate", plan, err)
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
	if err := os.WriteFile(filepath.Join(f.env.RepoRoot, env.CheckoutMarker), []byte("local edit\n"), 0o600); err != nil {
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
		t.Fatalf("want a refusal naming the missing %s, got %v", env.CheckoutMarker, err)
	}
}

// From zero without git, and no manager dotf can install it with, the
// failure names the command that installs it on each OS (#2734).
func TestCheckout_NoGitFromZeroFailsWithTheFix(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin":  "xcode-select --install",
		"linux":   "sudo apt-get install -y git",
		"windows": "winget install --id Git.Git -e",
	} {
		t.Run(goos, func(t *testing.T) {
			f := newCheckoutFixture(t)
			f.step.has = func(string) bool { return false }
			f.env.GOOS = goos

			if _, err := f.step.Reconcile(f.env, true); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want the hint %q, got %v", want, err)
			}
		})
	}
}

// gitInstaller is the OS manager as the checkout step sees it: it plans git
// as plan says and counts its installs.
type gitInstaller struct {
	plan     tools.PlanAction
	installs *int
	fail     error // returned by Install when set
}

func (g gitInstaller) Plan(t tools.Tool) tools.Plan {
	return tools.Plan{Name: t.Name, Action: g.plan}
}

func (g gitInstaller) Install(t tools.Tool) (tools.Result, error) {
	if t.Name != "git" || t.Source.Winget != "Git.Git" {
		return tools.Skipped, errors.New("unexpected install of " + t.Name)
	}
	*g.installs++
	if g.fail != nil {
		return tools.Skipped, g.fail
	}
	return tools.Installed, nil
}

// noGitUntilInstalled is a has that finds git only once it is installed and
// its directory is on this process's PATH.
func noGitUntilInstalled(dir string, installs *int) func(string) bool {
	return func(name string) bool {
		return name == "git" && *installs > 0 && slices.Contains(filepath.SplitList(os.Getenv("PATH")), dir)
	}
}

// A clean Windows has no git, so from zero the step installs it through the
// OS manager, reaches the directory it lands in, and clones in the same run.
// A plan names the install and runs nothing (#2734).
func TestCheckout_NoGitFromZeroInstallsItThroughTheOSManager(t *testing.T) {
	f := newCheckoutFixture(t)
	t.Setenv("PATH", os.Getenv("PATH"))
	installs, dir := 0, filepath.Join(t.TempDir(), "Git", "cmd")
	f.env.GOOS = "windows"
	f.step.has = noGitUntilInstalled(dir, &installs)
	f.step.installer = gitInstaller{plan: tools.PlanInstall, installs: &installs}
	f.step.gitDirs = []string{dir}

	plan, err := f.step.Reconcile(f.env, true)
	if err != nil || !strings.Contains(plan.Detail, "install git") || installs != 0 {
		t.Fatalf("plan = %+v, %v (%d installs); want the install named and none run", plan, err, installs)
	}
	res, err := f.step.Reconcile(f.env, false)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if installs != 1 || !strings.Contains(res.Detail, "installed git") || !strings.Contains(res.Detail, "cloned") {
		t.Errorf("apply = %+v after %d installs; want git installed once and the checkout cloned", res, installs)
	}
	if err := f.step.Probe(f.env); err != nil {
		t.Errorf("probe after the clone: %v", err)
	}
}

// An install that leaves git out of reach is a failure that says so, not a
// clone attempted without git.
func TestCheckout_AnInstalledGitItCannotReachFails(t *testing.T) {
	f := newCheckoutFixture(t)
	t.Setenv("PATH", os.Getenv("PATH"))
	installs := 0
	f.env.GOOS = "windows"
	f.step.has = noGitUntilInstalled(filepath.Join(t.TempDir(), "elsewhere"), &installs)
	f.step.installer = gitInstaller{plan: tools.PlanInstall, installs: &installs}
	f.step.gitDirs = []string{filepath.Join(t.TempDir(), "Git", "cmd")}

	if _, err := f.step.Reconcile(f.env, false); err == nil || !strings.Contains(err.Error(), "installed git, but it is not on PATH") {
		t.Fatalf("want the unreachable install named, got %v", err)
	}
}

// A failed install reports the manager's error and no hint: the hint would
// name the same install that just failed.
func TestCheckout_AFailedGitInstallNamesItsCauseNotTheHint(t *testing.T) {
	f := newCheckoutFixture(t)
	installs := 0
	f.env.GOOS = "windows"
	f.step.has = func(string) bool { return false }
	f.step.installer = gitInstaller{plan: tools.PlanInstall, installs: &installs, fail: errors.New("winget exited 1")}

	_, err := f.step.Reconcile(f.env, false)
	if err == nil || !strings.Contains(err.Error(), "winget exited 1") || strings.Contains(err.Error(), "winget install --id") {
		t.Fatalf("want the install's error without the hint, got %v", err)
	}
}

// Where the manager cannot install git (no winget; apt behind sudo), the step
// fails with the hint, as it did before it could install git.
func TestCheckout_NoGitAndNoUsableManagerFailsWithTheFix(t *testing.T) {
	f := newCheckoutFixture(t)
	installs := 0
	f.env.GOOS = "windows"
	f.step.has = func(string) bool { return false }
	f.step.installer = gitInstaller{plan: tools.PlanMissingManager, installs: &installs}

	if _, err := f.step.Reconcile(f.env, false); err == nil || !strings.Contains(err.Error(), "winget install --id Git.Git -e") || installs != 0 {
		t.Fatalf("want the hint and no install, got %v (%d installs)", err, installs)
	}
}

// winget's Git.Git lands in Program Files, or LocalAppData for a user-scope
// install; no other OS installs git through this step.
func TestGitBinDirs_NameWhereWingetPlacesGit(t *testing.T) {
	getenv := func(k string) string {
		return map[string]string{"ProgramFiles": "PF", "LOCALAPPDATA": "LA"}[k]
	}
	want := []string{filepath.Join("PF", "Git", "cmd"), filepath.Join("LA", "Programs", "Git", "cmd")}
	if got := GitBinDirs("windows", getenv); !slices.Equal(got, want) {
		t.Errorf("windows: %v, want %v", got, want)
	}
	if got := GitBinDirs("linux", getenv); got != nil {
		t.Errorf("linux: %v, want none", got)
	}
}

func TestCheckout_SkippedWithoutARunner(t *testing.T) {
	res, err := checkout{}.Reconcile(Env{RepoRoot: t.TempDir()}, false)
	if err != nil || res.Skip == "" {
		t.Fatalf("no runner must skip, got %+v, %v", res, err)
	}
}

// The catalog declares git too, so the tools step installs it where the
// checkout already exists and never needed a clone (a copied tree, the test
// guest). Both name the same package, or the two paths install different gits.
func TestShippedCatalog_DeclaresTheGitTheCheckoutInstalls(t *testing.T) {
	entries, err := loadCatalog(Env{RepoRoot: filepath.Join("..", "..", "..")})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name == "git" {
			if e.Source.Winget != gitTool.Source.Winget || !e.SupportsOS("windows") {
				t.Errorf("catalog git = %+v, want winget %q", e.Source, gitTool.Source.Winget)
			}
			return
		}
	}
	t.Fatal("packages.json declares no git")
}
