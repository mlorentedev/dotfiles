package doctor

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// CLI-066 (#1364): doctor measures the file pwsh names as $PROFILE, not the
// first of four guessed roots. The redirected-Documents box is the case the
// CLI-064 review called out as a THEORETICAL Major: with a profile outside
// every enumerated root, the old code measured nothing while --fix healed
// the real file.
func TestCheckProfileFiles_MeasuresThePwshResolvedProfile(t *testing.T) {
	type answer struct {
		out string
		err error
	}
	cases := []struct {
		name     string
		onPath   []string
		pwsh     *answer // nil: pwsh must not be asked
		where    string  // profile location relative to home; "" for none on disk
		wantFail int
		wantSub  string
	}{
		{
			name:     "redirected Documents: pwsh names a file outside the four roots, doctor measures it",
			onPath:   []string{"pwsh"},
			pwsh:     &answer{out: "{REDIRECTED}\r\n"},
			where:    "Redirected/Docs/PowerShell/Microsoft.PowerShell_profile.ps1",
			wantFail: 0,
			wantSub:  "resolved by pwsh $PROFILE",
		},
		{
			name:     "pwsh answers a path that does not exist yet: FAIL names that path, not four guesses",
			onPath:   []string{"pwsh"},
			pwsh:     &answer{out: "{REDIRECTED}\r\n"},
			where:    "",
			wantFail: 1,
			wantSub:  "resolved by pwsh $PROFILE",
		},
		{
			name:     "no pwsh on PATH: the enumeration answers and the row says so",
			onPath:   nil,
			pwsh:     nil,
			where:    "Documents/PowerShell/Microsoft.PowerShell_profile.ps1",
			wantFail: 0,
			wantSub:  "enumerated, pwsh not on PATH",
		},
		{
			name:     "pwsh present but fails to answer: the enumeration answers, with the reason",
			onPath:   []string{"pwsh"},
			pwsh:     &answer{out: "", err: errors.New("exit status 1")},
			where:    "Documents/PowerShell/Microsoft.PowerShell_profile.ps1",
			wantFail: 0,
			wantSub:  "enumerated, pwsh did not answer $PROFILE (exit status 1)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "x")
			writeFile(t, filepath.Join(home, ".gemini", "AGY.md"), "x")
			redirected := filepath.Join(home, "Redirected", "Docs", "PowerShell", "Microsoft.PowerShell_profile.ps1")
			if tc.where != "" {
				writeFile(t, filepath.Join(home, filepath.FromSlash(tc.where)), healthyProfile)
			}
			sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home}, tc.onPath, nil)
			sys.GOOS = "windows"
			asked := false
			sys.CommandOutputBounded = func(d time.Duration, name string, args ...string) (string, string, error) {
				asked = true
				if tc.pwsh == nil {
					t.Fatalf("pwsh must not be asked here, got %s %v", name, args)
				}
				if name != "pwsh" || strings.Join(args, " ") != "-NoProfile -Command $PROFILE" {
					t.Fatalf("unexpected question: %s %v", name, args)
				}
				if d != profileQueryTimeout {
					t.Fatalf("the question must be bounded by profileQueryTimeout, got %v", d)
				}
				return strings.ReplaceAll(tc.pwsh.out, "{REDIRECTED}", redirected), "", tc.pwsh.err
			}
			var buf bytes.Buffer
			rep := capture(&buf)
			checkProfileFiles(sys, nil, rep, false)
			out := buf.String()
			if rep.Failures() != tc.wantFail {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFail, out)
			}
			if !strings.Contains(out, tc.wantSub) {
				t.Fatalf("row must say how the target was found (%q)\n%s", tc.wantSub, out)
			}
			if (tc.pwsh != nil) != asked {
				t.Fatalf("pwsh asked = %v, want %v", asked, tc.pwsh != nil)
			}
			if tc.pwsh != nil && tc.pwsh.err == nil && !strings.Contains(out, redirected) {
				t.Fatalf("the row must name the pwsh-resolved path\n%s", out)
			}
		})
	}
}
