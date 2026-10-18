package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// Both installers run uv tools on the Python versions.conf pins (BUG-138):
// `dotf tools install` by hand and converge's catalog half.
func TestToolsInstallers_RunUvToolsOnThePinnedPython(t *testing.T) {
	want := pinnedPython(env.ResolveVersionsPath(), io.Discard)
	if want == "" {
		t.Fatal("this checkout's versions.conf pins no Python for mise")
	}
	if got := toolsInstaller(t.TempDir(), io.Discard).Python(); got != want {
		t.Errorf("tools install runs uv tools on %q, want %q", got, want)
	}
	in, ok := convergeOptions().ToolsCatalog.(*tools.Installer)
	if !ok || in.Python() != want {
		t.Errorf("converge's catalog installer = %#v, want Python %q", convergeOptions().ToolsCatalog, want)
	}
}

// With no Python pin to read, uv picks the interpreter, the BUG-138 failure,
// so the installer says so instead of falling back in silence.
func TestPinnedPython_SaysWhenItHasNoPin(t *testing.T) {
	dir := t.TempDir()
	unpinned := filepath.Join(dir, "versions.conf")
	if err := os.WriteFile(unpinned, []byte("NODE_VERSION=24.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dir, "missing.conf"), unpinned} {
		var out bytes.Buffer
		if got := pinnedPython(path, &out); got != "" {
			t.Errorf("pinnedPython(%s) = %q, want none", path, got)
		}
		if !strings.Contains(out.String(), "[WARN] no Python series read from "+path) {
			t.Errorf("a missing pin in %s went unsaid: %q", path, out.String())
		}
	}
}

// converge builds its installer before the checkout step clones and
// records-mirror writes the deploy dir, so the pin is read when a uv tool
// installs, not when the installer is built: on a clean Windows the eager
// read found no versions.conf, uv picked 3.15, and pre-commit's pyyaml had
// no wheel to install (#2769).
func TestToolsInstaller_ReadsThePinWhenAUvToolInstalls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DOTFILES_REPO_DIR", "")
	deploy := filepath.Join(home, "deploy")
	t.Setenv("DOTFILES_DIR", deploy)
	t.Chdir(home)
	var out bytes.Buffer
	in := toolsInstaller(t.TempDir(), &out)

	if err := os.MkdirAll(deploy, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "# mise: cli\nPYTHON_VERSION=3.13.16\n"
	if err := os.WriteFile(filepath.Join(deploy, "versions.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := in.Python(); got != "3.13" {
		t.Errorf("Python() = %q after versions.conf appeared, want 3.13 (out: %q)", got, out.String())
	}
}
