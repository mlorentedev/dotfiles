package update

import (
	"errors"
	"strings"
	"testing"
)

const testExe = "/home/me/.local/bin/dotf"

// pinDeps wires Deps whose upstream pins upstream and whose checkout pins
// local ("" ref). A ref whose pin is "!" fails to read. Counters record what
// ran.
type pinDeps struct {
	upstream, local      string
	installErr, convErr  error
	installed            []string
	converges, convWiths int
	convergedWith        string
}

func (p *pinDeps) deps(g fakeGit) Deps {
	return Deps{
		Git: g.run,
		Pin: func(ref string) (string, error) {
			v := p.local
			if ref == "@{u}" {
				v = p.upstream
			}
			if v == "!" {
				return "", errors.New("cannot read versions.conf at " + ref)
			}
			return v, nil
		},
		Converge: func() error { p.converges++; return nil },
		Install: func(pin string) (string, error) {
			p.installed = append(p.installed, pin)
			return testExe, p.installErr
		},
		ConvergeWith: func(exe string) error { p.convWiths++; p.convergedWith = exe; return p.convErr },
	}
}

// TestRunPin covers BUG-144 (#2506) as CLI-133 reshapes it: the running dotf
// must match the pin after an update, and the converge that follows must be
// run by the binary that pin names, as a child process, not by the one it
// replaced.
func TestRunPin(t *testing.T) {
	const pin = "0.70.0"
	tests := []struct {
		name        string
		behind      bool   // true: the checkout fast-forwards; false: already current
		running     string // the running binary's version
		installErr  error
		convErr     error
		wantStatus  string
		wantErr     bool
		wantInstall bool
		wantConv    bool // the old binary's in-process converge
		wantConvNew bool // the installed binary's converge
		wantMsg     []string
	}{
		{name: "ff, running at the pin", behind: true, running: pin, wantStatus: "updated", wantConv: true},
		{name: "ff, running behind the pin", behind: true, running: "0.69.0", wantStatus: "reinstalled", wantInstall: true, wantConvNew: true, wantMsg: []string{"0.69.0 -> " + pin, "converged by the new binary"}},
		{name: "ff, install fails", behind: true, running: "0.69.0", installErr: errors.New("checksum mismatch"), wantStatus: "install-failed", wantErr: true, wantInstall: true, wantMsg: []string{"0.69.0", pin}},
		{name: "ff, the new binary's converge fails", behind: true, running: "0.69.0", convErr: errors.New("exit status 1"), wantStatus: "converge-failed", wantErr: true, wantInstall: true, wantConvNew: true, wantMsg: []string{"0.69.0 -> " + pin + " (the DOTF_VERSION pin) installed", "converge failed"}},
		{name: "ff, source build", behind: true, running: "dev", wantStatus: "updated", wantConv: true, wantMsg: []string{"source build"}},
		{name: "ff, source build, install would fail", behind: true, running: "dev", installErr: errors.New("offline"), wantStatus: "updated", wantConv: true},

		{name: "current, running at the pin", running: pin, wantStatus: StatusCurrent, wantMsg: []string{"no deploy needed"}},
		{name: "current, running behind the pin", running: "0.69.0", wantStatus: "reinstalled", wantInstall: true, wantConvNew: true},
		{name: "current, install fails", running: "0.69.0", installErr: errors.New("offline"), wantStatus: "install-failed", wantErr: true, wantInstall: true},
		{name: "current, a pin below the running dotf installs it", running: "0.71.0", wantStatus: "reinstalled", wantInstall: true, wantConvNew: true, wantMsg: []string{"0.71.0 -> " + pin}},
		{name: "current, source build", running: "dev", wantStatus: StatusCurrent, wantMsg: []string{"no deploy needed", "source build"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			if !tt.behind {
				g.out["rev-parse @{u}"] = "aaaa"
			}
			p := &pinDeps{upstream: pin, local: "0.1.0", installErr: tt.installErr, convErr: tt.convErr}

			out, err := Run(Config{Repo: "/repo", Version: tt.running}, p.deps(g))

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if out.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q (msg: %s)", out.Status, tt.wantStatus, out.Message)
			}
			if (len(p.installed) > 0) != tt.wantInstall {
				t.Errorf("Install calls = %v, want called = %v", p.installed, tt.wantInstall)
			}
			if tt.wantInstall && p.installed[0] != pin {
				t.Errorf("Install was handed %q, want the upstream pin %q", p.installed[0], pin)
			}
			if (p.converges > 0) != tt.wantConv {
				t.Errorf("in-process converges = %d, want called = %v: after an install only the new binary converges", p.converges, tt.wantConv)
			}
			if (p.convWiths > 0) != tt.wantConvNew {
				t.Errorf("converges by the installed binary = %d, want called = %v", p.convWiths, tt.wantConvNew)
			}
			if tt.wantConvNew && p.convergedWith != testExe {
				t.Errorf("converged with %q, want the installed binary %q", p.convergedWith, testExe)
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

// CLI-133 AC2: the pin is upstream's, read after the fetch. When the upstream
// cannot be read the checkout's pin is used, and the run says so.
func TestRun_PinSource(t *testing.T) {
	tests := []struct {
		name            string
		mutate          func(g fakeGit)
		upstream, local string
		wantPin         string
		wantMsg         string
	}{
		{name: "upstream's pin wins over the checkout's", upstream: "0.80.0", local: "0.70.0", wantPin: "0.80.0"},
		{name: "unreadable upstream falls back to the checkout", upstream: "!", local: "0.70.0", wantPin: "0.70.0", wantMsg: "read from the checkout"},
		{name: "no upstream reads the checkout", mutate: func(g fakeGit) { g.fail["rev-parse --abbrev-ref --symbolic-full-name @{u}"] = true }, upstream: "0.80.0", local: "0.70.0", wantPin: "0.70.0", wantMsg: "read from the checkout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			g.out["rev-parse @{u}"] = "aaaa"
			if tt.mutate != nil {
				tt.mutate(g)
			}
			p := &pinDeps{upstream: tt.upstream, local: tt.local}
			out, err := Run(Config{Repo: "/repo", Version: "0.69.0"}, p.deps(g))
			if err != nil {
				t.Fatalf("err = %v (msg: %s)", err, out.Message)
			}
			if len(p.installed) != 1 || p.installed[0] != tt.wantPin {
				t.Fatalf("installed %v, want [%s]", p.installed, tt.wantPin)
			}
			if tt.wantMsg != "" && !strings.Contains(out.Message, tt.wantMsg) {
				t.Errorf("message = %q, want it to contain %q", out.Message, tt.wantMsg)
			}
		})
	}
}

// A pin that cannot be read anywhere, or is not set, is not drift: the run
// behaves as it did before the pin check existed, and says so when it could not
// look.
func TestRunPinUnreadable(t *testing.T) {
	for _, tt := range []struct{ name, pin, wantMsg string }{
		{name: "versions.conf unreadable", pin: "!", wantMsg: "pin check skipped"},
		{name: "DOTF_VERSION not pinned", pin: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := &pinDeps{upstream: tt.pin, local: tt.pin}
			out, err := Run(Config{Repo: "/repo", Version: "0.69.0"}, p.deps(healthyGit()))
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if out.Status != "updated" || p.converges != 1 || len(p.installed) != 0 {
				t.Errorf("status = %q, converges = %d, installs = %v, want updated with one converge and no install", out.Status, p.converges, p.installed)
			}
			if tt.wantMsg != "" && !strings.Contains(out.Message, tt.wantMsg) {
				t.Errorf("message = %q, want it to contain %q", out.Message, tt.wantMsg)
			}
		})
	}
}

// CLI-133 AC1: a checkout that cannot be deployed still gets the pinned dotf.
// The converge is skipped and said so, the status stays the checkout's (doctor
// warns that it is stuck), and the run exits 0.
func TestRun_InstallsThePinOnADriftedCheckout(t *testing.T) {
	tests := []struct {
		name, wantStatus string
		mutate           func(g fakeGit)
	}{
		{name: "dirty", wantStatus: "dirty", mutate: func(g fakeGit) { g.out["status --porcelain"] = " M x" }},
		{name: "ahead", wantStatus: StatusAhead, mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "bbbb" }},
		{name: "diverged", wantStatus: "diverged", mutate: func(g fakeGit) { g.out["merge-base HEAD @{u}"] = "cccc" }},
		{name: "ff failed", wantStatus: "ff-failed", mutate: func(g fakeGit) { g.fail["merge --ff-only @{u}"] = true }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			tt.mutate(g)
			p := &pinDeps{upstream: "0.70.0", local: "0.1.0"}
			out, err := Run(Config{Repo: "/repo", Version: "0.69.0"}, p.deps(g))
			if err != nil {
				t.Fatalf("err = %v, want nil (msg: %s)", err, out.Message)
			}
			if out.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", out.Status, tt.wantStatus)
			}
			if len(p.installed) != 1 || p.installed[0] != "0.70.0" {
				t.Errorf("installed %v, want the upstream pin once", p.installed)
			}
			if p.converges+p.convWiths != 0 {
				t.Errorf("a checkout that cannot be deployed converged (%d in-process, %d new)", p.converges, p.convWiths)
			}
			for _, s := range []string{"0.69.0 -> 0.70.0", "converge skipped"} {
				if !strings.Contains(out.Message, s) {
					t.Errorf("message = %q, want it to contain %q", out.Message, s)
				}
			}
		})
	}
}

// A dirty checkout is not fetched by Assess, so Run fetches before reading the
// upstream's pin; when that fetch fails the machine is offline and nothing is
// downloaded, exactly as on an offline clean checkout.
func TestRun_OfflineInstallsNothing(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(g fakeGit)
	}{
		{name: "offline", mutate: func(g fakeGit) { g.fail["fetch --quiet"] = true }},
		{name: "dirty and offline", mutate: func(g fakeGit) { g.out["status --porcelain"] = " M x"; g.fail["fetch --quiet"] = true }},
		{name: "not a repo", mutate: func(g fakeGit) { g.fail["rev-parse --git-dir"] = true }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := healthyGit()
			tt.mutate(g)
			p := &pinDeps{upstream: "0.70.0", local: "0.70.0"}
			if _, err := Run(Config{Repo: "/repo", Version: "0.69.0"}, p.deps(g)); err != nil {
				t.Fatalf("err = %v, want a benign skip", err)
			}
			if len(p.installed)+p.converges+p.convWiths != 0 {
				t.Errorf("an offline run installed %v, converged %d+%d", p.installed, p.converges, p.convWiths)
			}
		})
	}
}
