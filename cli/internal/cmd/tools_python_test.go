package cmd

import (
	"io"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// Both installers run uv tools on the Python versions.conf pins (BUG-138):
// `dotf tools install` by hand and converge's catalog half.
func TestToolsInstallers_RunUvToolsOnThePinnedPython(t *testing.T) {
	want := pinnedPython()
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
