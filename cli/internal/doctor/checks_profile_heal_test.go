package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// healthyProfile is a deployed profile as setup-windows.ps1 writes it: one
// START/END pair around the SSOT content.
const healthyProfile = profileStartMarker + "\r\nfunction prompt { 'ok> ' }\r\n" + profileEndMarker + "\r\n"

// duplicatedProfile is the BUG-020 shape in miniature: the splice replaced the
// first pair and left a second one behind, so the section is sourced twice.
const duplicatedProfile = healthyProfile + "\r\n" + healthyProfile

// profileFixture lays down $PROFILE under a temp home and returns its path.
func profileFixture(t *testing.T, home, content string) string {
	t.Helper()
	p := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	writeFile(t, p, content)
	return p
}

// oversizedProfile is over profile-heal.ps1's 1MB signal, structurally healthy
// (one marker pair) so only size trips.
func oversizedProfile() string {
	pad := "# " + strings.Repeat("#", profileMaxBytes+1)
	return profileStartMarker + "\r\n" + pad + "\r\n" + profileEndMarker + "\r\n"
}

// #531 (BUG-020 class): a corrupted PowerShell profile was neither detected nor
// healed once doctor.ps1 -Fix retired — existence was the whole test. The two
// signals are profile-heal.ps1's own (size > 1MB, more than one marker pair), so
// what doctor flags is exactly what --fix can clear.
func TestCheckProfileFiles_DetectsBUG020Corruption(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		wantFail  int
		wantSub   string
		unwantSub string
	}{
		{
			name:     "one marker pair, small → healthy",
			content:  healthyProfile,
			wantFail: 0,
			wantSub:  "PowerShell profile exists",
		},
		{
			name:     "no markers at all → not corruption (setup has not spliced yet)",
			content:  "function prompt { 'user> ' }\r\n",
			wantFail: 0,
			wantSub:  "PowerShell profile exists",
		},
		{
			name:     "two marker pairs → FAIL naming the counts and the heal",
			content:  duplicatedProfile,
			wantFail: 1,
			wantSub:  "duplicate dotfiles markers (start=2, end=2",
		},
		{
			name:     "over 1 MiB → FAIL on size alone",
			content:  oversizedProfile(),
			wantFail: 1,
			wantSub:  "bytes > 1 MiB",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "x")
			writeFile(t, filepath.Join(home, ".gemini", "AGY.md"), "x")
			profileFixture(t, home, tc.content)
			sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home}, nil, nil)
			sys.GOOS = "windows"

			var buf bytes.Buffer
			rep := capture(&buf)
			checkProfileFiles(sys, nil, rep, false)

			if rep.Failures() != tc.wantFail {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFail, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSub) {
				t.Fatalf("output missing %q\n%s", tc.wantSub, buf.String())
			}
			if tc.wantFail > 0 && !strings.Contains(buf.String(), "profile-heal.ps1") {
				t.Fatalf("a corruption FAIL must name the heal script\n%s", buf.String())
			}
		})
	}
}

// The remedy path is resolved through the env contract, never a literal: the
// variable when set, else the contract's Windows default, else a visible token.
// Where scripts deploy is changing under WIN-013 (#1310); a literal here would
// point at the wrong directory the day it lands.
func TestCheckProfileFiles_HealPathFollowsTheContract(t *testing.T) {
	contract := &Contract{EnvVars: []ContractEnvVar{{
		Name:    "SCRIPTS_DIR",
		Default: map[string]string{"linux": "$HOME/.dotfiles/scripts", "windows": `$env:USERPROFILE\.dotfiles\scripts`},
	}}}
	cases := []struct {
		name    string
		env     map[string]string
		c       *Contract
		wantSub string
	}{
		{
			name:    "SCRIPTS_DIR set → its value",
			env:     map[string]string{"SCRIPTS_DIR": `D:\deployed\scripts`},
			c:       contract,
			wantSub: filepath.Join(`D:\deployed\scripts`, `profile-heal.ps1`),
		},
		{
			name: "SCRIPTS_DIR unset → the contract's windows default, home-expanded",
			c:    contract,
			// The contract default carries backslashes verbatim; only the last
			// separator is the host's, exactly as profileHealPath builds it.
			wantSub: filepath.Join(`.dotfiles\scripts`, `profile-heal.ps1`),
		},
		{
			name:    "no contract loaded → the token stays visible rather than a guessed literal",
			wantSub: `$env:SCRIPTS_DIR\profile-heal.ps1`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			env := map[string]string{"HOME": home, "USERPROFILE": home}
			for k, v := range tc.env {
				env[k] = v
			}
			sys := newSys(env, nil, nil)
			sys.GOOS = "windows"
			got := profileHealPath(sys, tc.c)
			if !strings.Contains(got, tc.wantSub) {
				t.Fatalf("heal path = %q, want it to contain %q", got, tc.wantSub)
			}
			if tc.c != nil && tc.env == nil && !strings.HasPrefix(got, home) {
				t.Fatalf("contract default must be home-expanded, got %q", got)
			}
		})
	}
}

// --fix runs profile-heal.ps1 through the System seam and reports the OUTCOME:
// the heal exits 0 by design even when it changes nothing, so only a profile
// that re-measures healthy is a Fix. Never run against the real $PROFILE here —
// the fake pwsh is the only thing that touches the fixture.
func TestCheckProfileFiles_FixRunsTheHealAndVerifiesByConsequence(t *testing.T) {
	type fixture struct {
		home, profile, heal string
	}
	setup := func(t *testing.T, deployHeal bool) fixture {
		t.Helper()
		home := t.TempDir()
		writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "x")
		writeFile(t, filepath.Join(home, ".gemini", "AGY.md"), "x")
		profile := profileFixture(t, home, duplicatedProfile)
		heal := filepath.Join(home, "scripts", profileHealScript)
		if deployHeal {
			writeFile(t, heal, "# fake heal\r\n")
		}
		return fixture{home: home, profile: profile, heal: heal}
	}
	run := func(t *testing.T, fx fixture, onPath []string, pwsh func(args []string) (string, error)) (string, *Report) {
		t.Helper()
		sys := newSys(map[string]string{"HOME": fx.home, "USERPROFILE": fx.home, "SCRIPTS_DIR": filepath.Dir(fx.heal)}, onPath, nil)
		sys.GOOS = "windows"
		sys.CommandOutput = func(name string, args ...string) (string, error) {
			if name != "pwsh" {
				t.Fatalf("unexpected command %s %v", name, args)
			}
			return pwsh(args)
		}
		var buf bytes.Buffer
		rep := capture(&buf)
		checkProfileFiles(sys, nil, rep, true)
		return buf.String(), rep
	}

	t.Run("heal rewrites the profile → FIX, invoked as pwsh -NoProfile -File <SCRIPTS_DIR>\\profile-heal.ps1", func(t *testing.T) {
		fx := setup(t, true)
		var gotArgs []string
		out, rep := run(t, fx, []string{"pwsh"}, func(args []string) (string, error) {
			gotArgs = args
			if err := os.WriteFile(fx.profile, []byte(healthyProfile), 0o644); err != nil {
				t.Fatal(err)
			}
			return "[profile-heal] rewrote profile from SSOT\n", nil
		})
		want := []string{"-NoProfile", "-File", fx.heal}
		if strings.Join(gotArgs, " ") != strings.Join(want, " ") {
			t.Fatalf("heal invoked as %v, want %v", gotArgs, want)
		}
		if rep.Failures() != 0 || rep.totals[StatusFix] != 1 {
			t.Fatalf("a verified heal is exactly one FIX and no FAIL; got fail=%d fix=%d\n%s", rep.Failures(), rep.totals[StatusFix], out)
		}
		if !strings.Contains(out, "rebuilt from powershell/profile.ps1") {
			t.Fatalf("the FIX must say what happened\n%s", out)
		}
	})

	t.Run("heal exits 0 but changes nothing → FAIL, never a FIX", func(t *testing.T) {
		fx := setup(t, true)
		out, rep := run(t, fx, []string{"pwsh"}, func([]string) (string, error) {
			return "[profile-heal] ERROR: cannot find powershell/profile.ps1 SSOT\n", nil
		})
		if rep.totals[StatusFix] != 0 || rep.Failures() != 1 {
			t.Fatalf("an unrepaired profile must FAIL, not FIX; got fail=%d fix=%d\n%s", rep.Failures(), rep.totals[StatusFix], out)
		}
		if !strings.Contains(out, "still corrupted") || !strings.Contains(out, "cannot find powershell/profile.ps1 SSOT") {
			t.Fatalf("the FAIL must carry the re-measurement and the heal's own line\n%s", out)
		}
	})

	t.Run("heal script not deployed → FAIL naming setup, pwsh never invoked", func(t *testing.T) {
		fx := setup(t, false)
		out, rep := run(t, fx, []string{"pwsh"}, func([]string) (string, error) {
			t.Fatal("pwsh must not run when the heal script is absent")
			return "", nil
		})
		if rep.Failures() != 1 || !strings.Contains(out, "not deployed at "+fx.heal) || !strings.Contains(out, "setup-windows.ps1") {
			t.Fatalf("expected one FAIL naming the missing script and setup\n%s", out)
		}
	})

	t.Run("pwsh absent → FAIL, nothing invoked", func(t *testing.T) {
		fx := setup(t, true)
		out, rep := run(t, fx, nil, func([]string) (string, error) {
			t.Fatal("nothing must run without pwsh")
			return "", nil
		})
		if rep.Failures() != 1 || !strings.Contains(out, "pwsh is not in PATH") {
			t.Fatalf("expected one FAIL naming pwsh\n%s", out)
		}
	})

	t.Run("healthy profile under --fix → PASS, heal not invoked", func(t *testing.T) {
		fx := setup(t, true)
		if err := os.WriteFile(fx.profile, []byte(healthyProfile), 0o644); err != nil {
			t.Fatal(err)
		}
		out, rep := run(t, fx, []string{"pwsh"}, func([]string) (string, error) {
			t.Fatal("a healthy profile must not be healed")
			return "", nil
		})
		if rep.Failures() != 0 || rep.totals[StatusFix] != 0 || !strings.Contains(out, "PowerShell profile exists") {
			t.Fatalf("expected a plain PASS\n%s", out)
		}
	})
}
