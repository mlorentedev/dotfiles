package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestIsDeployDirPath(t *testing.T) {
	managed := []string{
		"versions.conf", ".zshrc", ".bashrc", ".profile", "tmux.conf", "packages.json",
		".zsh/aliases.zsh", "ssh/config", "scripts/utils.sh", "sensitive/chatgpt.api-key.secret.age",
		"secrets/registry.yaml",
	}
	unmanaged := []string{
		"README.md", "go.mod", "cli/main.go", "docs/lessons.md", ".github/workflows/ci.yml",
		"scripts", "versions.conf.bak", ".zshrc.d/x", "harness/manifest.json",
	}
	for _, p := range managed {
		if !IsDeployDirPath(p) {
			t.Errorf("IsDeployDirPath(%q) = false, want true", p)
		}
	}
	for _, p := range unmanaged {
		if IsDeployDirPath(p) {
			t.Errorf("IsDeployDirPath(%q) = true, want false", p)
		}
	}
}

// mirrorDeployDir skips an entry the checkout lacks, so a rename would drop it
// without a sound. This test is the sound.
func TestDeployDirSetExistsInTheCheckout(t *testing.T) {
	root := repoRootForTest(t)
	for _, f := range DeployDirFiles {
		if !isRegular(filepath.Join(root, f)) {
			t.Errorf("DeployDirFiles names %q, which the checkout does not have", f)
		}
	}
	for _, d := range DeployDirTrees {
		if !isDir(filepath.Join(root, d)) {
			t.Errorf("DeployDirTrees names %q, which the checkout does not have", d)
		}
	}
}

// The mirror filters what git ignores only where it walks a tree. A file it
// copies by name is a declaration, so it must be tracked: an ignored local file
// under one of these names would deploy, the #2268 defect by another door.
func TestMirrorNamedFilesAreTracked(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := repoRootForTest(t)
	if err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		t.Skip("the checkout is not a git repository")
	}
	targets, err := manifestTargets(filepath.Join(root, filepath.FromSlash(ManifestFile)))
	if err != nil {
		t.Fatal(err)
	}
	named := append(append([]string{}, DeployDirFiles...), targets...)
	args := append([]string{"-C", root, "ls-files", "--error-unmatch", "--"}, named...)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil { //nolint:gosec // fixed binary, args built here
		t.Errorf("a file the mirror copies by name is not tracked:\n%s", out)
	}
}

// setupCopy matches setup-linux.sh's copy lines into the deploy dir:
// `safe_copy` or `cp -f "$CURRENT_DIR/<file>" …` and `cp -rf "$CURRENT_DIR/<dir>/…" …`.
var setupCopy = regexp.MustCompile(`(?m)^\s*(?:safe_copy|cp -rf|cp -f) "\$CURRENT_DIR/([^"/]+)(/[^"]*)?"`)

// Until setup-linux.sh's early copy block is deleted, it and the mirror are two
// writers of one set. They must name the same paths, or doctor checks a path
// converge does not refresh, which is #2224 again.
func TestDeployDirSetMatchesSetupCopyBlock(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRootForTest(t), "setup-linux.sh"))
	if err != nil {
		t.Fatal(err)
	}
	var files, trees []string
	for _, m := range setupCopy.FindAllStringSubmatch(string(b), -1) {
		if m[2] == "" {
			files = append(files, m[1])
		} else {
			trees = append(trees, m[1])
		}
	}
	if len(files) == 0 || len(trees) == 0 {
		t.Fatal("no copy lines found in setup-linux.sh: the parser or the script changed shape")
	}
	// A copy written another way (cp -a, install, a loop) would escape the
	// parser and let the writers diverge with this test green, so every line
	// that copies from the checkout into the deploy dir must be one it read.
	for _, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, `"$CURRENT_DIR/`) && strings.Contains(line, `"$DOTFILES_DIR/`) &&
			!setupCopy.MatchString(line) {
			t.Errorf("setup-linux.sh copies into the deploy dir in a form this test cannot read: %s", strings.TrimSpace(line))
		}
	}
	assertSameSet(t, "files", files, DeployDirFiles)
	assertSameSet(t, "trees", trees, DeployDirTrees)
}

func assertSameSet(t *testing.T, what string, setup, owned []string) {
	t.Helper()
	norm := func(in []string) []string {
		seen := map[string]bool{}
		var out []string
		for _, s := range in {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
		sort.Strings(out)
		return out
	}
	a, b := norm(setup), norm(owned)
	if len(a) != len(b) {
		t.Fatalf("%s: setup-linux.sh copies %v, the deploy-dir set is %v", what, a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("%s: setup-linux.sh copies %v, the deploy-dir set is %v", what, a, b)
		}
	}
}

// The #2224 case: the checkout moved, the deploy dir kept the old copy, and
// only setup could refresh it. Mirror now does, and leaves everything outside
// the set alone.
func TestMirror_RefreshesTheDeployDirSet(t *testing.T) {
	repo, deploy := mirrorRepo(t), t.TempDir()
	writeFile(t, filepath.Join(repo, "versions.conf"), "PYTHON_VERSION=3.13.16\n")
	writeFile(t, filepath.Join(repo, "scripts", "test.sh"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(repo, "README.md"), "# not deployed\n")
	writeFile(t, filepath.Join(deploy, "versions.conf"), "PYTHON_VERSION=3.12.6\n")

	res, err := Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(deploy, "versions.conf")); string(got) != "PYTHON_VERSION=3.13.16\n" {
		t.Errorf("versions.conf not refreshed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(deploy, "scripts", "test.sh")); err != nil {
		t.Errorf("scripts/test.sh not mirrored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(deploy, "README.md")); !os.IsNotExist(err) {
		t.Errorf("README.md is outside the set and must not be mirrored (stat err %v)", err)
	}
	// harness/ (3) + targets (2) + versions.conf + scripts/test.sh
	if res.Updated != 7 {
		t.Errorf("first run: want 7 updated, got %d", res.Updated)
	}

	plan, err := PlanMirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Updated != 0 {
		t.Errorf("after an apply the plan must be empty, got %d to write", plan.Updated)
	}
}
