package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func darwinSys(env map[string]string, onPath []string, cmdOut map[string]string) *System {
	s := newSys(env, onPath, cmdOut)
	s.GOOS = "darwin"
	return s
}

// The APPS_HOME/<tool>-<version> layout is Linux's. On the Mac it reported
// eleven FAILs for directories and variables darwin never lays down.
func TestAppsLayout_DarwinSkipsNamingMise(t *testing.T) {
	home := t.TempDir()
	cfg := &Config{Versions: map[string]string{"JAVA_VERSION": "21.0.4", "GO_VERSION": "1.26.0"}}

	var b bytes.Buffer
	rep := capture(&b)
	checkVersionMatch(darwinSys(map[string]string{"HOME": home}, nil, nil), cfg, rep)
	checkToolHomeEnvVars(darwinSys(map[string]string{"HOME": home}, nil, nil), rep)

	if rep.Failures() != 0 {
		t.Errorf("darwin must not fail the APPS_HOME layout\n%s", b.String())
	}
	if !strings.Contains(b.String(), "toolchains through mise") {
		t.Errorf("the skip must name how darwin gets its toolchains\n%s", b.String())
	}
}

func TestAppsLayout_LinuxStillRequiresIt(t *testing.T) {
	var b bytes.Buffer
	rep := capture(&b)
	checkToolHomeEnvVars(newSys(map[string]string{}, nil, nil), rep)
	if rep.Failures() != len(toolHomeVars) {
		t.Errorf("failures = %d, want %d\n%s", rep.Failures(), len(toolHomeVars), b.String())
	}
}

// A *_HOME that is set must point at a real toolchain on every OS (F-041: a
// JAVA_HOME pointing at nothing breaks macOS's /usr/bin/java).
func TestVersionedPaths_ASetHomeIsCheckedOnDarwinToo(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "jdk-21")
	var b bytes.Buffer
	rep := capture(&b)
	checkVersionedPaths(darwinSys(map[string]string{"JAVA_HOME": missing}, nil, nil), rep)
	if rep.Failures() != 1 {
		t.Errorf("a JAVA_HOME pointing at nothing must FAIL on darwin\n%s", b.String())
	}
}

// tmux is class 3 (ADR-044): the remedy is the OS's package manager.
func TestCheckTmux_TheRemedyNamesThisOSPackageManager(t *testing.T) {
	for goos, want := range map[string]string{"linux": "apt install", "darwin": "brew install tmux"} {
		var b bytes.Buffer
		rep := capture(&b)
		s := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
		s.GOOS = goos
		checkTmux(s, rep)
		if rep.Failures() != 1 || !strings.Contains(b.String(), want) {
			t.Errorf("%s: want one FAIL naming %q\n%s", goos, want, b.String())
		}
		if goos == "darwin" && strings.Contains(b.String(), "apt") {
			t.Errorf("darwin remedy names apt\n%s", b.String())
		}
	}
}
