package update

import (
	"errors"
	"strings"
	"testing"
)

// TestRunPin covers BUG-144 (#2506): the running dotf must match the
// checkout's DOTF_VERSION after an update, and the converge that follows must
// be run by the binary that pin names, not by the one about to be replaced.
// Both statuses that leave the checkout level with its upstream are crossed
// with each relation of the running version to the pin.
func TestRunPin(t *testing.T) {
	const pin = "0.70.0"

	tests := []struct {
		name        string
		behind      bool   // true: the checkout fast-forwards; false: already current
		running     string // the running binary's version
		installErr  error
		wantStatus  string
		wantErr     bool
		wantInstall bool
		wantConv    bool
		wantMsg     []string // every substring the message must hold
	}{
		{name: "ff, running at the pin", behind: true, running: pin, wantStatus: "updated", wantConv: true},
		{name: "ff, running behind the pin", behind: true, running: "0.69.0", wantStatus: "reinstalled", wantInstall: true, wantMsg: []string{"0.69.0", pin}},
		{name: "ff, running behind, install fails", behind: true, running: "0.69.0", installErr: errors.New("offline"), wantStatus: "install-failed", wantErr: true, wantInstall: true, wantMsg: []string{"0.69.0", pin}},
		{name: "ff, source build", behind: true, running: "dev", wantStatus: "updated", wantConv: true, wantMsg: []string{"source build"}},
		{name: "ff, source build, install would fail", behind: true, running: "dev", installErr: errors.New("offline"), wantStatus: "updated", wantConv: true},
		{name: "ff, at the pin, install would fail", behind: true, running: pin, installErr: errors.New("offline"), wantStatus: "updated", wantConv: true},

		{name: "current, running at the pin", running: pin, wantStatus: StatusCurrent, wantMsg: []string{"no deploy needed"}},
		{name: "current, running behind the pin", running: "0.69.0", wantStatus: "reinstalled", wantInstall: true, wantMsg: []string{"0.69.0", pin}},
		{name: "current, running behind, install fails", running: "0.69.0", installErr: errors.New("offline"), wantStatus: "install-failed", wantErr: true, wantInstall: true, wantMsg: []string{"0.69.0", pin}},
		{name: "current, source build", running: "dev", wantStatus: StatusCurrent, wantMsg: []string{"no deploy needed", "source build"}},
		{name: "current, source build, install would fail", running: "dev", installErr: errors.New("offline"), wantStatus: StatusCurrent},
		{name: "current, at the pin, install would fail", running: pin, installErr: errors.New("offline"), wantStatus: StatusCurrent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			if !tt.behind {
				g.out["rev-parse @{u}"] = "aaaa"
			}
			var installedPin string
			installs, converges := 0, 0
			d := Deps{
				Git:      g.run,
				Pin:      func() (string, error) { return pin, nil },
				Converge: func() error { converges++; return nil },
				Install:  func(p string) error { installs++; installedPin = p; return tt.installErr },
			}

			out, err := Run(Config{Repo: "/repo", Version: tt.running}, d)

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if out.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q (msg: %s)", out.Status, tt.wantStatus, out.Message)
			}
			if (installs > 0) != tt.wantInstall {
				t.Errorf("Install calls = %d, want called = %v", installs, tt.wantInstall)
			}
			if tt.wantInstall && installedPin != pin {
				t.Errorf("Install was handed %q, want the checkout's pin %q", installedPin, pin)
			}
			if (converges > 0) != tt.wantConv {
				t.Errorf("Converge calls = %d, want called = %v — the installer converges with the new binary, so update must not converge a second time with the old one", converges, tt.wantConv)
			}
			for _, s := range tt.wantMsg {
				if !strings.Contains(out.Message, s) {
					t.Errorf("message = %q, want it to contain %q", out.Message, s)
				}
			}
			if tt.wantErr && !strings.Contains(err.Error(), pin) {
				t.Errorf("error = %q, want it to name the pinned version %q", err, pin)
			}
		})
	}
}

// A pin that cannot be read, or is not set, is not drift: the run behaves as
// it did before the pin check existed, and says so when it could not look.
func TestRunPinUnreadable(t *testing.T) {
	tests := []struct {
		name    string
		pin     func() (string, error)
		wantMsg string
	}{
		{name: "versions.conf unreadable", pin: func() (string, error) { return "", errors.New("no such file") }, wantMsg: "pin check skipped"},
		{name: "DOTF_VERSION not pinned", pin: func() (string, error) { return "", nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converges := 0
			d := Deps{
				Git:      healthyGit().run,
				Pin:      tt.pin,
				Converge: func() error { converges++; return nil },
				Install:  func(string) error { t.Fatal("Install must not run without a readable pin"); return nil },
			}
			out, err := Run(Config{Repo: "/repo", Version: "0.69.0"}, d)
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if out.Status != "updated" || converges != 1 {
				t.Errorf("status = %q, converges = %d, want updated with one converge", out.Status, converges)
			}
			if tt.wantMsg != "" && !strings.Contains(out.Message, tt.wantMsg) {
				t.Errorf("message = %q, want it to contain %q", out.Message, tt.wantMsg)
			}
		})
	}
}

// Every status that leaves the checkout where it was is a skip: it must not
// reach for the pin or the installer, whatever the running version is. An
// offline or dirty machine reinstalling dotf is not a self-update.
func TestRunPinSkippedBranchesNeverInstall(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(g fakeGit)
	}{
		{name: "dirty", mutate: func(g fakeGit) { g.out["status --porcelain"] = " M x" }},
		{name: "offline", mutate: func(g fakeGit) { g.fail["fetch --quiet"] = true }},
		{name: "ahead", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "bbbb" }},
		{name: "diverged", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "cccc" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			tt.mutate(g)
			d := Deps{
				Git:      g.run,
				Pin:      func() (string, error) { return "0.70.0", nil },
				Converge: func() error { t.Fatal("Converge ran on a skip branch"); return nil },
				Install:  func(string) error { t.Fatal("Install ran on a skip branch"); return nil },
			}
			out, err := Run(Config{Repo: "/repo", Version: "0.69.0"}, d)
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			// Nothing is installed, but the drift doctor FAILs is named.
			if !strings.Contains(out.Message, "dotf 0.69.0 is not the pinned 0.70.0") {
				t.Errorf("message does not name the drift: %q", out.Message)
			}
		})
	}
}
