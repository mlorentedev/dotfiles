package doctor

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

// TestCheckAntigravity_AbsolutePathAccepted guards the C20 fix (#691): the
// AGY_APP_DATA absolute-path check used strings.HasPrefix(_, "/"), which
// false-FAILed an absolute Windows path (C:\Users\...\antigravity-cli) whenever
// agy was on PATH on Windows. filepath.IsAbs recognizes each OS's absolute form,
// so an OS-native absolute path must PASS. On the windows-latest runner this
// exercises the exact regression; on ubuntu it confirms the POSIX form still
// passes.
func TestCheckAntigravity_AbsolutePathAccepted(t *testing.T) {
	abs := "/home/me/.gemini/antigravity-cli"
	if runtime.GOOS == "windows" {
		abs = `C:\Users\me\.gemini\antigravity-cli`
	}

	sys := newSys(map[string]string{"AGY_APP_DATA": abs}, []string{"agy"}, nil)
	var buf bytes.Buffer
	rep := capture(&buf)
	checkAntigravity(sys, rep)

	out := buf.String()
	if strings.Contains(out, "AGY_APP_DATA is relative or unset") {
		t.Errorf("absolute path %q was false-FAILed:\n%s", abs, out)
	}
	if !strings.Contains(out, "AGY_APP_DATA is absolute") {
		t.Errorf("absolute path %q should report absolute:\n%s", abs, out)
	}
}
