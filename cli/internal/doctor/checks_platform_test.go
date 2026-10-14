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
	// DOTFILES_DIR set: the profile ran, so each missing variable is drift.
	checkToolHomeEnvVars(newSys(map[string]string{"DOTFILES_DIR": "/home/u/.dotfiles"}, nil, nil), rep)
	if rep.Failures() != len(toolHomeVars) {
		t.Errorf("failures = %d, want %d\n%s", rep.Failures(), len(toolHomeVars), b.String())
	}
}

// `ssh host dotf doctor` reads no profile: one WARN that says so, not a FAIL
// per variable (#2013 debt ledger, D12; measured on msi, 5 FAIL non-login and
// 0 under zsh -lic).
func TestToolHomes_UnloadedProfileWarnsOnce(t *testing.T) {
	var b bytes.Buffer
	rep := capture(&b)
	checkToolHomeEnvVars(newSys(map[string]string{}, nil, nil), rep)
	if rep.Failures() != 0 || rep.Warnings() != 1 || !strings.Contains(b.String(), "no shell profile reached this process") || !strings.Contains(b.String(), "'exec $SHELL -lic") {
		t.Fatalf("want one WARN naming the unloaded profile, got %d FAIL, %d WARN\n%s", rep.Failures(), rep.Warnings(), b.String())
	}

	// One tool home set means the profile ran: the others are real drift.
	b.Reset()
	rep = capture(&b)
	checkToolHomeEnvVars(newSys(map[string]string{"JAVA_HOME": "/opt/jdk"}, nil, nil), rep)
	if rep.Failures() != len(toolHomeVars)-1 || rep.Warnings() != 0 {
		t.Fatalf("a half-loaded profile must still fail each missing variable\n%s", b.String())
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
