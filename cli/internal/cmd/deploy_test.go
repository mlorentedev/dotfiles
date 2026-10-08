package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
)

// runDeploy runs `dotf deploy [args]` the way the setup scripts do: from inside
// the checkout. env.RepoDir prefers the .git walk-up from the cwd, so the fixture
// carries a .git marker and the test chdirs into it (same reason as
// runHarnessMirror); HOME is a temp dir so {HOME} destinations land in it.
func runDeploy(t *testing.T, repo, home string, args []string) (string, error) {
	t.Helper()
	return runDeployWithClaude(t, repo, home, args, nil)
}

// runDeployWithClaude is runDeploy with the claude CLI answered by run; nil
// means "claude not installed". runDeploy passes nil because a bare deploy
// installs plugins through the real claude when it is on PATH, and no test may
// drive the box's own install.
func runDeployWithClaude(t *testing.T, repo, home string, args []string, run claude.Runner) (string, error) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	t.Setenv("DOTFILES_REPO_DIR", repo)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	orig := deployClaudeRunner
	t.Cleanup(func() { deployClaudeRunner = orig })
	deployClaudeRunner = func(string) claude.Runner { return run }
	var out bytes.Buffer
	cmd := newDeployCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// twoConfigRepo writes a manifest declaring two render-free configs, so a test
// can tell "every declared config" from "the first one" by what lands in HOME.
func twoConfigRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "ai", "deploy.json"), `{
  "version": 3,
  "configs": [
    {"name": "one", "src": "ai/one.json", "dst": "{HOME}/.one/config.json", "render": false, "mode": "0644"},
    {"name": "two", "src": "ai/two.json", "dst": "{HOME}/.two/config.json", "render": false, "mode": "0644"}
  ]
}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "one.json"), `{"one":true}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "two.json"), `{"two":true}`)
	return repo
}

// The premise both setup scripts rely on since CLI-054 (#1301): a bare
// `dotf deploy` installs EVERY entry ai/deploy.json declares. Before, each
// setup named one config (`dotf deploy pi`), so a second manifest entry
// (orca-keybindings) was declared and installed by neither setup until two
// scripts were edited. The manifest is the SSOT of what gets deployed; the
// call site must not narrow it.
func TestDeployCmd_NoArgInstallsEveryDeclaredConfig(t *testing.T) {
	repo, home := twoConfigRepo(t), t.TempDir()

	out, err := runDeploy(t, repo, home, nil)
	if err != nil {
		t.Fatalf("dotf deploy: %v\n%s", err, out)
	}
	for _, name := range []string{"one", "two"} {
		dst := filepath.Join(home, "."+name, "config.json")
		if _, err := os.Stat(dst); err != nil {
			t.Errorf("config %q declared in the manifest was not installed at %s: %v", name, dst, err)
		}
		if !strings.Contains(out, row("deployed", name)) {
			t.Errorf("stdout must report config %q as deployed, got:\n%s", name, out)
		}
	}
}

// An entry declared for other OSes is skipped and said so, whether the run
// names it or not: Git Bash on Windows would read a POSIX rc file (#1843 B1).
func TestDeployCmd_SkipsAnEntryForAnotherOS(t *testing.T) {
	other := "windows"
	if runtime.GOOS == "windows" {
		other = "linux"
	}
	repo, home := t.TempDir(), t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "ai", "deploy.json"), `{
  "version": 4,
  "configs": [
    {"name": "one", "src": "ai/one.json", "dst": "{HOME}/.one/config.json"},
    {"name": "elsewhere", "src": "ai/two.json", "dst": "{HOME}/.two/config.json", "platforms": ["`+other+`"]}
  ]
}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "one.json"), `{"one":true}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "two.json"), `{"two":true}`)

	for _, args := range [][]string{nil, {"elsewhere"}} {
		out, err := runDeploy(t, repo, home, args)
		if err != nil {
			t.Fatalf("dotf deploy %v: %v\n%s", args, err, out)
		}
		if !strings.Contains(out, row("skipped", "elsewhere")) || !strings.Contains(out, "not for "+runtime.GOOS) {
			t.Errorf("dotf deploy %v must report the entry skipped for this OS:\n%s", args, out)
		}
		if _, err := os.Stat(filepath.Join(home, ".two")); err == nil {
			t.Errorf("dotf deploy %v installed an entry declared for %s", args, other)
		}
	}
}

// A name still narrows the run to that one entry — the shape a repair of a
// single config uses — and the other declared config is left alone.
func TestDeployCmd_NamedArgInstallsOnlyThatConfig(t *testing.T) {
	repo, home := twoConfigRepo(t), t.TempDir()

	out, err := runDeploy(t, repo, home, []string{"one"})
	if err != nil {
		t.Fatalf("dotf deploy one: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(home, ".one", "config.json")); err != nil {
		t.Errorf("the named config was not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".two", "config.json")); err == nil {
		t.Error("a named run must not install the other declared config")
	}
}

// An unknown name fails loudly and lists what IS declared, so a setup script
// that still names a retired entry cannot pass as a no-op.
func TestDeployCmd_UnknownNameFailsAndListsDeclared(t *testing.T) {
	repo, home := twoConfigRepo(t), t.TempDir()

	out, err := runDeploy(t, repo, home, []string{"nope"})
	if err == nil {
		t.Fatalf("an undeclared config must be an error, got success:\n%s", out)
	}
	if !strings.Contains(err.Error(), "one") || !strings.Contains(err.Error(), "two") {
		t.Errorf("the error must name the declared configs, got: %v", err)
	}
}

// #1664 (2): a mode-only fix has its own two lines, one per run kind. Swapping
// them told a dry run's reader the mode was fixed, and a real run's that it
// would be.
func TestDeployCmd_AModeOnlyFixIsReportedAsOne(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "ai", "deploy.json"), `{
  "version": 3,
  "configs": [
    {"name": "sec", "src": "ai/sec.json", "dst": "{HOME}/.sec/config.json", "render": false, "mode": "0600"}
  ]
}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "sec.json"), `{"sec":true}`)
	// Same bytes, looser than declared; on Windows, the DACL it inherits.
	writeMirrorFixture(t, filepath.Join(home, ".sec", "config.json"), `{"sec":true}`)
	if err := os.Chmod(filepath.Join(home, ".sec", "config.json"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		args      []string
		want, not string
	}{
		{[]string{"sec", "--dry-run"}, "would fix mode", "mode fixed"},
		{[]string{"sec"}, "mode fixed", "would fix mode"},
	} {
		out, err := runDeploy(t, repo, home, tc.args)
		if err != nil {
			t.Fatalf("dotf deploy %v: %v\n%s", tc.args, err, out)
		}
		if !strings.Contains(out, tc.want+" ") || strings.Contains(out, tc.not) {
			t.Errorf("dotf deploy %v must say %q, not %q:\n%s", tc.args, tc.want, tc.not, out)
		}
	}
}

// A directory that existed before the deploy keeps the mode it was created
// with, so a private file written into a 0755 ~/.pi/agent left it 0755 until
// `dotf doctor --fix` ran (#2161). privateDirRepo declares a 0600 entry and a
// 0644 one, each in a directory that already exists at 0755, and deploys the
// private one; then reopens its directory, the state the tests start from.
func privateDirRepo(t *testing.T) (repo, home, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory modes; deploy.TightenDir is a no-op on Windows")
	}
	repo, home = t.TempDir(), t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "ai", "deploy.json"), `{
  "version": 3,
  "configs": [
    {"name": "sec", "src": "ai/sec.json", "dst": "{HOME}/.sec/config.json", "render": false, "mode": "0600"},
    {"name": "pub", "src": "ai/pub.json", "dst": "{HOME}/.pub/config.json", "render": false, "mode": "0644"}
  ]
}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "sec.json"), `{"sec":true}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "pub.json"), `{"pub":true}`)
	dir = filepath.Join(home, ".sec")
	for _, d := range []string{dir, filepath.Join(home, ".pub")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	deployOK(t, repo, home, "sec")
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return repo, home, dir
}

func deployOK(t *testing.T, repo, home string, args ...string) string {
	t.Helper()
	out, err := runDeploy(t, repo, home, args)
	if err != nil {
		t.Fatalf("dotf deploy %v: %v\n%s", args, err, out)
	}
	return out
}

func permOf(t *testing.T, p string) os.FileMode {
	t.Helper()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

// A dry run reports the tightening and changes nothing; a named deploy of
// another entry never narrows a directory as a side effect.
func TestDeployCmd_ReportsButDoesNotTightenOnADryRunOrAnotherEntry(t *testing.T) {
	repo, home, dir := privateDirRepo(t)
	if out := deployOK(t, repo, home, "pub"); strings.Contains(out, "tighten") || permOf(t, dir) != 0o755 {
		t.Errorf("a named deploy of pub must not touch .sec (%04o):\n%s", permOf(t, dir), out)
	}
	out := deployOK(t, repo, home, "--dry-run")
	if !strings.Contains(out, "would tighten ") || !strings.Contains(out, dir+" from 0755 to 0700") || permOf(t, dir) != 0o755 {
		t.Errorf("a dry run must report the tightening and leave %04o:\n%s", permOf(t, dir), out)
	}
}

// The deploy narrows the directory, says so, leaves a public-only directory
// alone, and a second deploy has nothing left to do.
func TestDeployCmd_TightensAnExistingDirectoryThatHoldsAPrivateFile(t *testing.T) {
	repo, home, dir := privateDirRepo(t)
	out := deployOK(t, repo, home)
	if !strings.Contains(out, "tightened ") || !strings.Contains(out, dir+" from 0755 to 0700") {
		t.Errorf("the deploy must report the tightening:\n%s", out)
	}
	if got := permOf(t, dir); got != 0o700 {
		t.Errorf("%s is %04o after deploy, want 0700", dir, got)
	}
	if got := permOf(t, filepath.Join(home, ".pub")); got != 0o755 {
		t.Errorf("a directory holding only a public file must keep 0755, it is %04o", got)
	}
	if out := deployOK(t, repo, home); strings.Contains(out, "tighten") || strings.Count(out, "in sync ") != 2 {
		t.Errorf("a second deploy must have nothing left to do:\n%s", out)
	}
}

// The bootstrap state of #2161 before any deploy: the directory exists at 0755
// and the private file is not there yet. A dry run must predict the tightening
// the real run makes, not only report it once the file exists.
func TestDeployCmd_ADryRunPredictsTheTighteningBeforeTheFileExists(t *testing.T) {
	repo, home, dir := privateDirRepo(t)
	if err := os.Remove(filepath.Join(dir, "config.json")); err != nil {
		t.Fatal(err)
	}
	out := deployOK(t, repo, home, "--dry-run")
	if !strings.Contains(out, "would tighten ") || !strings.Contains(out, dir+" from 0755 to 0700") || permOf(t, dir) != 0o755 {
		t.Errorf("a first dry run must predict the tightening and leave %04o:\n%s", permOf(t, dir), out)
	}
}

// Never widened: a directory an operator made 0700 keeps it when the deploy
// writes only a public file into it.
func TestDeployCmd_AnOwnerOnlyDirectoryWithAPublicFileStaysOwnerOnly(t *testing.T) {
	repo, home, _ := privateDirRepo(t)
	pub := filepath.Join(home, ".pub")
	if err := os.Chmod(pub, 0o700); err != nil {
		t.Fatal(err)
	}
	if out := deployOK(t, repo, home, "pub"); strings.Contains(out, "tighten") {
		t.Errorf("a public entry must not report a tightening:\n%s", out)
	}
	if got := permOf(t, pub); got != 0o700 {
		t.Errorf("%s is %04o after deploying a public file, want 0700 kept", pub, got)
	}
}
