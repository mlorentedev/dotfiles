package doctor

import (
	"bytes"
	"strings"
	"testing"
)

// BUG-069 (#912): git-for-windows below 2.55 could not execute hooks through a
// C:/-form core.hooksPath, so every commit aborted while doctor reported the
// GUARD active. The defect was fixed upstream (lesson 239); what the repo keeps
// is the FLOOR, declared in versions.conf and read here. One row per branch,
// driven through the fake System's version line.
func TestCheckGitWindowsFloor(t *testing.T) {
	cases := []struct {
		name     string
		goos     string
		pin      string
		onPath   []string
		version  string
		wantWarn int
		wantSub  string
	}{
		{
			name: "windows: 2.53.0 (the measured-broken release) → WARN naming #912 and the remedy",
			goos: "windows", pin: "2.55.0", onPath: []string{"git"},
			version:  "git version 2.53.0.windows.1",
			wantWarn: 1,
			wantSub:  "below the git-for-windows floor 2.55.0",
		},
		{
			name: "windows: 2.55.0.windows.5 (the measured-working release) → PASS",
			goos: "windows", pin: "2.55.0", onPath: []string{"git"},
			version:  "git version 2.55.0.windows.5",
			wantWarn: 0,
			wantSub:  "git 2.55.0 meets the git-for-windows floor 2.55.0",
		},
		{
			name: "windows: a newer release than the floor → PASS (a floor, not an exact pin)",
			goos: "windows", pin: "2.55.0", onPath: []string{"git"},
			version:  "git version 2.57.1.windows.2",
			wantWarn: 0,
			wantSub:  "meets the git-for-windows floor",
		},
		{
			name: "windows: unparseable version line → WARN, never a silent PASS",
			goos: "windows", pin: "2.55.0", onPath: []string{"git"},
			version:  "git: command not recognised",
			wantWarn: 1,
			wantSub:  "unparseable",
		},
		{
			name: "windows: git absent → SKIP (the env contract owns that FAIL)",
			goos: "windows", pin: "2.55.0",
			wantWarn: 0,
			wantSub:  "env contract owns that FAIL",
		},
		{
			name: "windows: no GIT_VERSION pin → SKIP saying the floor was not verified",
			goos: "windows", onPath: []string{"git"},
			version:  "git version 2.53.0.windows.1",
			wantWarn: 0,
			wantSub:  "GIT_VERSION not set in versions.conf",
		},
		{
			name: "linux: below the floor is not a finding — the defect was git-for-windows'",
			goos: "linux", pin: "2.55.0", onPath: []string{"git"},
			version:  "git version 2.34.1",
			wantWarn: 0,
			wantSub:  "Windows-only",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := map[string]string{}
			if tc.version != "" {
				cmd["git --version"] = tc.version
			}
			sys := newSys(nil, tc.onPath, cmd)
			sys.GOOS = tc.goos
			cfg := &Config{Versions: map[string]string{}}
			if tc.pin != "" {
				cfg.Versions[gitFloorKey] = tc.pin
			}

			var buf bytes.Buffer
			rep := capture(&buf)
			checkGitWindowsFloor(sys, cfg, rep)

			if rep.Failures() != 0 {
				t.Fatalf("the floor is advisory and must never FAIL; got %d\n%s", rep.Failures(), buf.String())
			}
			if rep.Warnings() != tc.wantWarn {
				t.Fatalf("warnings = %d, want %d\n%s", rep.Warnings(), tc.wantWarn, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSub) {
				t.Fatalf("output missing %q\n%s", tc.wantSub, buf.String())
			}
		})
	}
}

// The floor is reached through checkVersionMatch, the section that consumes
// versions.conf pins. Every other case calls checkGitWindowsFloor directly, so
// dropping the call from checkVersionMatch would leave the suite green.
func TestCheckVersionMatch_RunsTheGitWindowsFloor(t *testing.T) {
	sys := newSys(nil, []string{"git"}, map[string]string{"git --version": "git version 2.53.0.windows.1"})
	sys.GOOS = "windows"
	cfg := &Config{Versions: map[string]string{gitFloorKey: "2.55.0"}}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkVersionMatch(sys, cfg, rep)

	if !strings.Contains(buf.String(), "below the git-for-windows floor") {
		t.Fatalf("checkVersionMatch must run the git-for-windows floor; got:\n%s", buf.String())
	}
}
