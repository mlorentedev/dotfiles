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
	if got := toolsInstaller(t.TempDir(), io.Discard).Python; got != want {
		t.Errorf("tools install runs uv tools on %q, want %q", got, want)
	}
	in, ok := convergeOptions().ToolsCatalog.(*tools.Installer)
	if !ok || in.Python != want {
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
