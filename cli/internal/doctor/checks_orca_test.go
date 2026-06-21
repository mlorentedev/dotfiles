package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckOrcaHook is the DX-006 table: one row per decision branch (skip when
// absent, timeoutSec pass/fail, transport pass/fail), driven by a real temp $HOME
// so the file probes are exercised end to end.
func TestCheckOrcaHook(t *testing.T) {
	cases := []struct {
		name         string
		orcaJSON     string // "" ⇒ orca.json absent
		hookPS       string // "" ⇒ copilot-hook.ps1 absent
		wantFailures int
		wantSubstr   string
	}{
		{"neither present → skip", "", "", 0, "Orca not installed"},
		{"timeoutSec >= 30 → pass", `{"hooks":{"x":{"timeoutSec":30}}}`, "", 0, "timeoutSec >= 30"},
		{"timeoutSec < 30 → fail", `{"hooks":{"x":{"timeoutSec":5}}}`, "", 1, "timeoutSec < 30"},
		{"any too-low among many → fail", `{"hooks":{"a":{"timeoutSec":30},"b":{"timeoutSec":5}}}`, "", 1, "timeoutSec < 30"},
		{"hook HttpWebRequest → pass", "", "$r = [System.Net.HttpWebRequest]::Create($u)", 0, "fast HttpWebRequest"},
		{"hook Invoke-WebRequest → fail", "", "Invoke-WebRequest -Uri $u -Method Post", 1, "slow Invoke-WebRequest"},
		{"both files healthy → two passes", `{"hooks":{"a":{"timeoutSec":60}}}`, "[System.Net.HttpWebRequest]", 0, "timeoutSec >= 30"},
		{"both files broken → two fails", `{"hooks":{"a":{"timeoutSec":5}}}`, "Invoke-WebRequest", 2, "Invoke-WebRequest"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.orcaJSON != "" {
				writeFile(t, filepath.Join(home, ".copilot", "hooks", "orca.json"), tc.orcaJSON)
			}
			if tc.hookPS != "" {
				writeFile(t, filepath.Join(home, ".orca", "agent-hooks", "copilot-hook.ps1"), tc.hookPS)
			}
			sys := newSys(map[string]string{"HOME": home}, nil, nil)

			var buf bytes.Buffer
			rep := capture(&buf)
			checkOrcaHook(sys, rep)

			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}
