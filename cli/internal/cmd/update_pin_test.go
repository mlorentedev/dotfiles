package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// pinnedCheckout is a clean clone of an upstream whose versions.conf pins
// DOTF_VERSION=0.70.0 and whose install.sh is a recorder: it writes the
// DOTF_VERSION it was run with and its arguments to $FAKE_INSTALL_OUT, and
// exits with $FAKE_INSTALL_RC. It stands for the real installer, which this
// test must not run (it downloads), while the command layer's wiring of it is
// what is under test.
func pinnedCheckout(t *testing.T) (checkout, recorded string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake installer is a shell script")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	files, home := convergeFixture(t)
	root := t.TempDir()
	upstream, writer := filepath.Join(root, "up.git"), filepath.Join(root, "w")
	checkout = filepath.Join(root, "dotfiles")
	gitIn(t, root, "init", "-q", "--bare", upstream)
	gitIn(t, root, "clone", "-q", upstream, writer)
	if err := os.CopyFS(writer, os.DirFS(files)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(writer, env.CheckoutMarker), []byte("# pins\nDOTF_VERSION=0.70.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n{ printf 'DOTF_VERSION=%s\\n' \"$DOTF_VERSION\"; printf 'arg=%s\\n' \"$@\"; } > \"$FAKE_INSTALL_OUT\"\nexit \"${FAKE_INSTALL_RC:-0}\"\n"
	if err := os.WriteFile(filepath.Join(writer, "install.sh"), []byte(script), 0o755); err != nil { //nolint:gosec // the fixture installer must be executable
		t.Fatal(err)
	}
	gitIn(t, writer, "add", "-A")
	gitIn(t, writer, "commit", "-q", "-m", "one")
	gitIn(t, writer, "push", "-q", "origin", "HEAD")
	gitIn(t, root, "clone", "-q", upstream, checkout)

	recorded = filepath.Join(t.TempDir(), "installer.out")
	t.Setenv("DOTFILES_REPO_DIR", checkout)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("FAKE_INSTALL_OUT", recorded)
	// A shell that sourced a stale versions.conf exports its own pin; the
	// installer must still be told the checkout's.
	t.Setenv("DOTF_VERSION", "0.0.1")
	return checkout, recorded
}

// runUpdateAs runs `dotf update` as a binary reporting version.
func runUpdateAs(t *testing.T, version string) (stdout string, err error) {
	t.Helper()
	t.Cleanup(func() { dotfVersion = "dev" })
	c := New(version, "")
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"update"})
	err = c.Execute()
	return out.String(), err
}

// BUG-144: a checkout that pins a dotf the running binary is not gets that
// dotf installed by the checkout's own installer, told the pin and the
// checkout explicitly, and the installer's converge replaces update's own.
func TestUpdate_InstallsThePinWhenTheRunningDotfDiffers(t *testing.T) {
	checkout, recorded := pinnedCheckout(t)
	setups := 0
	fixture := convergeOptions
	t.Cleanup(func() { convergeOptions = fixture })
	convergeOptions = func() converge.Options {
		o := fixture()
		o.RunSetup = func(converge.Env) error { setups++; return nil }
		return o
	}

	stdout, err := runUpdateAs(t, "0.69.0")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, stdout)
	}
	if !strings.Contains(stdout, "0.69.0 -> 0.70.0") {
		t.Errorf("output must name the move from the running dotf to the pin:\n%s", stdout)
	}
	if setups != 0 || strings.Contains(stdout, "converge on") {
		t.Errorf("update must not converge with the binary the installer replaces: setups=%d\n%s", setups, stdout)
	}
	got, rerr := os.ReadFile(recorded)
	if rerr != nil {
		t.Fatalf("the installer never ran: %v", rerr)
	}
	for _, want := range []string{"DOTF_VERSION=0.70.0\n", "arg=--repo\n", "arg=" + checkout + "\n"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("installer saw %q, want it to contain %q", got, want)
		}
	}
}

// An installer that fails is a real error: update exits non-zero naming the
// drift instead of reporting a deploy it did not make.
func TestUpdate_FailsWhenThePinCannotBeInstalled(t *testing.T) {
	_, _ = pinnedCheckout(t)
	t.Setenv("FAKE_INSTALL_RC", "7")

	stdout, err := runUpdateAs(t, "0.69.0")
	if err == nil {
		t.Fatalf("a failed install must fail update:\n%s", stdout)
	}
	if !strings.Contains(err.Error(), "0.70.0") || !strings.Contains(err.Error(), "0.69.0") {
		t.Errorf("error = %q, want both versions named", err)
	}
}

// A dotf already at the pin installs nothing: the installer never runs.
func TestUpdate_LeavesADotfAtThePinAlone(t *testing.T) {
	_, recorded := pinnedCheckout(t)

	stdout, err := runUpdateAs(t, "0.70.0")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, stdout)
	}
	if _, serr := os.Stat(recorded); serr == nil {
		t.Error("the installer ran although the running dotf is the pin")
	}
	if !strings.Contains(stdout, "no deploy needed") {
		t.Errorf("output = %q, want the current-checkout line", stdout)
	}
}

func TestInstallerInvocation(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		wantName string
		wantArgs []string
	}{
		{name: "unix runs install.sh itself", goos: "linux", wantName: filepath.Join("/repo", "install.sh"), wantArgs: []string{"--repo", "/repo"}},
		{name: "macOS runs install.sh itself", goos: "darwin", wantName: filepath.Join("/repo", "install.sh"), wantArgs: []string{"--repo", "/repo"}},
		{name: "windows runs install.ps1 under pwsh", goos: "windows", wantName: "pwsh", wantArgs: []string{"-NoProfile", "-File", filepath.Join("/repo", "install.ps1"), "--repo", "/repo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args := installerInvocation(tt.goos, "/repo")
			if name != tt.wantName || strings.Join(args, "|") != strings.Join(tt.wantArgs, "|") {
				t.Errorf("installerInvocation = %q %q, want %q %q", name, args, tt.wantName, tt.wantArgs)
			}
		})
	}
}

func TestPinnedDotfVersion(t *testing.T) {
	repo := t.TempDir()
	if _, err := pinnedDotfVersion(repo); err == nil {
		t.Error("a checkout without versions.conf must report an error, not an empty pin")
	}
	if err := os.WriteFile(filepath.Join(repo, "versions.conf"), []byte("GO_VERSION=1.26.0\nDOTF_VERSION=0.70.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := pinnedDotfVersion(repo); err != nil || got != "0.70.0" {
		t.Errorf("pinnedDotfVersion = %q, %v, want 0.70.0", got, err)
	}
}
