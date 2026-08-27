package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMirrorFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runHarnessMirror runs the command the way setup does: from inside the
// checkout. env.RepoDir prefers the .git walk-up from the cwd over
// DOTFILES_REPO_DIR (BUG-072), so the fixture carries a .git marker and the
// test chdirs into it; otherwise the walk-up lands on the real checkout that
// contains this test file.
func runHarnessMirror(t *testing.T, repo, deploy string) (stdout, stderr string, err error) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	t.Setenv("DOTFILES_REPO_DIR", repo)
	t.Setenv("DOTFILES_DIR", deploy)
	var out, errb bytes.Buffer
	cmd := newHarnessMirrorCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(nil)
	err = cmd.Execute()
	return out.String(), errb.String(), err
}

// The command is what both setup scripts call, so its two lines of output are
// the contract: the counts a setup run reports, and the named gap it must not
// hide.
func TestHarnessMirrorCmd(t *testing.T) {
	repo, deploy := t.TempDir(), t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "harness", "manifest.json"), `{"targets":[{"file":"AGENTS.md"}]}`)
	writeMirrorFixture(t, filepath.Join(repo, "harness", "model-map.json"), `{}`)
	writeMirrorFixture(t, filepath.Join(repo, "AGENTS.md"), "# AGENTS\n")

	out, _, err := runHarnessMirror(t, repo, deploy)
	if err != nil {
		t.Fatalf("first run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "1 target(s)") || !strings.Contains(out, "3 updated, 0 unchanged") {
		t.Errorf("first run must report what it mirrored:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(deploy, "harness", "model-map.json")); err != nil {
		t.Errorf("model-map.json not mirrored: %v", err)
	}

	out, _, err = runHarnessMirror(t, repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "0 updated, 3 unchanged") {
		t.Errorf("re-run must report zero changes:\n%s", out)
	}

	// A declared target the checkout lacks: named on stderr, exit non-zero,
	// everything else still mirrored.
	writeMirrorFixture(t, filepath.Join(repo, "harness", "manifest.json"), `{"targets":[{"file":"AGENTS.md"},{"file":"ai/orca/ORCA.md"}]}`)
	_, stderr, err := runHarnessMirror(t, repo, deploy)
	if err == nil {
		t.Fatal("a missing declared target must exit non-zero")
	}
	if !strings.Contains(stderr, "ai/orca/ORCA.md") || !strings.Contains(stderr, "not mirrored") {
		t.Errorf("the gap must be named:\n%s", stderr)
	}
	got, _ := os.ReadFile(filepath.Join(deploy, "harness", "manifest.json"))
	if !strings.Contains(string(got), "ORCA.md") {
		t.Error("the rest of the harness must still have been mirrored")
	}
}

func TestHarnessMirrorCmd_SaysSoWhenTheCheckoutIsTheDeployDir(t *testing.T) {
	repo := t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "harness", "manifest.json"), `{"targets":[]}`)
	out, _, err := runHarnessMirror(t, repo, repo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "nothing to mirror") {
		t.Errorf("must state the outcome:\n%s", out)
	}
}
