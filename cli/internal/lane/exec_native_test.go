package lane

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nativeFile writes a regular file and returns its absolute path.
func nativeFile(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func lookAt(bin string) func(string) (string, error) {
	return func(string) (string, error) { return bin, nil }
}

// #3569: the first `pi` on PATH is an npm or mise .cmd shim while mise holds
// pi.exe. The launch starts the executable, and cmd.exe is never involved.
func TestNativeTarget_StartsTheExecutableBehindABatchShim(t *testing.T) {
	native := nativeFile(t, "pi.exe")
	asked := ""
	got, err := NativeTarget("pi", lookAt(`C:\Users\m\.pi\agent\bin\pi.cmd`), func(name string) string {
		asked = name
		return native
	})
	if err != nil {
		t.Fatalf("want the native executable, got %v", err)
	}
	if got != native || asked != "pi" {
		t.Errorf("target %q (asked for %q), want %q asked for pi", got, asked, native)
	}
}

// What PATH resolves is used as it is when it is no batch file: the tool
// manager is not asked, so a native install is never swapped for another copy.
func TestNativeTarget_LeavesANativeResolutionAlone(t *testing.T) {
	got, err := NativeTarget("claude", lookAt(`C:\bin\claude.exe`), func(string) string {
		t.Error("the tool manager was asked about a native executable")
		return ""
	})
	if err != nil || got != `C:\bin\claude.exe` {
		t.Errorf("got %q, %v", got, err)
	}
}

// The refusal stands whenever there is no native file to start instead. The
// answer must name an .exe: for an npm tool mise answers with the package's
// extensionless shell stub, which has a .cmd beside it that Windows would run
// through PATHEXT (measured for opencode and bats, 2026-11-16).
func TestNativeTarget_RefusesWithoutANativeExecutableBehindTheShim(t *testing.T) {
	dir := t.TempDir()
	batch := nativeFile(t, "pi.cmd")
	for name, behind := range map[string]string{
		"nothing known":         "",
		"another batch file":    batch,
		"a file that is absent": filepath.Join(dir, "gone.exe"),
		"a directory":           dir,
		"a relative path":       "pi.exe",
		"an npm shell stub":     nativeFile(t, "pi"),
		"a renamed batch file":  nativeFile(t, "pi.exe.cmd"),
		"two lines":             nativeFile(t, "pi.exe") + "\n" + batch,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := NativeTarget("pi", lookAt(`C:\npm\pi.cmd`), func(string) string { return behind })
			if err == nil {
				t.Fatalf("want the refusal, got target %q", got)
			}
			for _, want := range []string{"pi", `C:\npm\pi.cmd`, "cmd.exe", "native"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not name %q", err, want)
				}
			}
		})
	}
}

// A caller that names a file asked for that file. Only a bare command name is
// looked up behind its shim, and nothing that a command line reads as a flag
// reaches the tool manager.
func TestNativeTarget_NeverSubstitutesANamedFile(t *testing.T) {
	native := nativeFile(t, "pi.exe")
	for _, name := range []string{`C:\npm\pi.cmd`, `.\pi.cmd`, "bin/pi.cmd", "pi.cmd", "pi.bat", "--version", "c:pi"} {
		got, err := NativeTarget(name, lookAt(`C:\npm\pi.cmd`), func(string) string {
			t.Errorf("%q: the tool manager was asked", name)
			return native
		})
		if err == nil {
			t.Errorf("%q: want the refusal, got target %q", name, got)
		}
	}
}

// A command that is not on PATH is not this function's error to explain.
func TestNativeTarget_ReturnsTheLookupError(t *testing.T) {
	missing := errors.New("not found")
	_, err := NativeTarget("pi", func(string) (string, error) { return "", missing }, func(string) string {
		t.Error("the tool manager was asked about a command PATH does not hold")
		return ""
	})
	if !errors.Is(err, missing) {
		t.Errorf("got %v, want the lookup error", err)
	}
}
