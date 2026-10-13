package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCheckAntigravity_AbsolutePathAccepted guards the C20 fix (#691): the
// AGY_HOME absolute-path check used strings.HasPrefix(_, "/"), which
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

	sys := newSys(map[string]string{"AGY_HOME": abs}, []string{"agy"}, nil)
	var buf bytes.Buffer
	rep := capture(&buf)
	checkAntigravity(sys, rep)

	out := buf.String()
	if strings.Contains(out, "AGY_HOME is relative or unset") {
		t.Errorf("absolute path %q was false-FAILed:\n%s", abs, out)
	}
	if !strings.Contains(out, "AGY_HOME is absolute") {
		t.Errorf("absolute path %q should report absolute:\n%s", abs, out)
	}
}

// A machine still exporting the old name to another directory is told which
// one dotf uses (#2395); the same value, or the old name alone, says nothing.
func TestCheckAntigravity_ALegacyAppDataThatDisagreesIsWarned(t *testing.T) {
	for name, tc := range map[string]struct {
		env  map[string]string
		warn bool
	}{
		"differs":     {map[string]string{"AGY_HOME": "/a", "AGY_APP_DATA": "/l"}, true},
		"agrees":      {map[string]string{"AGY_HOME": "/a", "AGY_APP_DATA": "/a"}, false},
		"legacy only": {map[string]string{"AGY_APP_DATA": "/l"}, false},
	} {
		var buf bytes.Buffer
		checkAntigravity(newSys(tc.env, []string{"agy"}, nil), capture(&buf))
		if got := strings.Contains(buf.String(), "AGY_APP_DATA is set to /l but /a is used"); got != tc.warn {
			t.Errorf("%s: warned = %v, want %v:\n%s", name, got, tc.warn, buf.String())
		}
	}
}

// Every FAIL on the master config names its remedy. An empty master is what
// agy leaves on its first run before converge, and reporting it as "invalid
// JSON" with no remedy sent the Mac bring-up looking for a corrupt write.
func TestCheckAntigravity_TheMasterConfigFailuresNameTheRemedy(t *testing.T) {
	for name, content := range map[string]string{
		"empty":   "",
		"invalid": "{not json",
		"symlink": "{}",
		"missing": "",
	} {
		t.Run(name, func(t *testing.T) {
			gemini := t.TempDir()
			if err := os.MkdirAll(filepath.Join(gemini, "config"), 0o755); err != nil {
				t.Fatal(err)
			}
			master := filepath.Join(gemini, "config", "mcp_config.json")
			switch name {
			case "missing":
			case "symlink":
				target := filepath.Join(gemini, "target.json")
				if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, master); err != nil {
					t.Skipf("cannot create a symlink here: %v", err)
				}
			default:
				if err := os.WriteFile(master, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			sys := newSys(map[string]string{"GEMINI_HOME": gemini, "AGY_HOME": gemini}, []string{"agy"}, nil)
			var buf bytes.Buffer
			checkAntigravity(sys, capture(&buf))

			out := buf.String()
			var line string
			for _, l := range strings.Split(out, "\n") {
				if strings.Contains(l, "master mcp_config.json") {
					line = l
				}
			}
			if !strings.Contains(line, "FAIL") || !strings.HasSuffix(strings.TrimSpace(line), "(run `dotf converge --only agent-extensions`)") {
				t.Errorf("want a FAIL naming the remedy, got %q in:\n%s", line, out)
			}
			if name == "empty" && !strings.Contains(line, "is empty") {
				t.Errorf("an empty master was not reported as empty: %q", line)
			}
		})
	}
}
