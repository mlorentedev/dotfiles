package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runDeploy runs `dotf deploy [args]` the way the setup scripts do: from inside
// the checkout. env.RepoDir prefers the .git walk-up from the cwd, so the fixture
// carries a .git marker and the test chdirs into it (same reason as
// runHarnessMirror); HOME is a temp dir so {HOME} destinations land in it.
func runDeploy(t *testing.T, repo, home string, args []string) (string, error) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	t.Setenv("DOTFILES_REPO_DIR", repo)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
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
		if !strings.Contains(out, "deployed  "+name) {
			t.Errorf("stdout must report config %q as deployed, got:\n%s", name, out)
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
