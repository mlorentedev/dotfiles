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
