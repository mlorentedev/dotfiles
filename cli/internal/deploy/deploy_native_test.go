package deploy

import (
	"path/filepath"
	"testing"
)

// The manifest spells destinations with "/" on every OS. The result must be in
// the OS's own form: on Windows `{HOME}` resolves to `C:\Users\u`, and the
// literal tail used to stay `/.pi/agent/models.json` — a path the syscall
// accepts but that no filepath.Join'ed path ever string-equals, which is the
// comparison every doctor check makes (CLI-054/#1301).
func TestExpandDst_ReturnsAnOSNativePath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	got, err := ExpandDst("{HOME}/.pi/agent/models.json", home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".pi", "agent", "models.json"); got != want {
		t.Errorf("want the OS-native path %q, got %q", want, got)
	}
}
