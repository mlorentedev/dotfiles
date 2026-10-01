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
		unwant   string // must not appear in the output
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
			wantSub:  "PowerShell profile missing: ",
			// A file that never existed is not BUG-020 corruption, and the
			// heal has nothing to rebuild (CLI-066 review round 1, finding 1).
			unwant: "corrupted",
		},
		{
			name:     "pwsh prints noise before the path: the last line is the answer",
			onPath:   []string{"pwsh"},
			pwsh:     &answer{out: "WARNING: some module banner\r\n{REDIRECTED}\r\n"},
			where:    "Redirected/Docs/PowerShell/Microsoft.PowerShell_profile.ps1",
			wantFail: 0,
			wantSub:  "resolved by pwsh $PROFILE",
			unwant:   "WARNING",
		},
		{
			name:     "pwsh answers something that is not a profile path: the enumeration answers",
			onPath:   []string{"pwsh"},
			pwsh:     &answer{out: "WARNING: no profile today\r\n"},
			where:    "Documents/PowerShell/Microsoft.PowerShell_profile.ps1",
			wantFail: 0,
			wantSub:  "enumerated, pwsh did not answer $PROFILE",
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
			if tc.unwant != "" && strings.Contains(out, tc.unwant) {
				t.Fatalf("row must not mention %q\n%s", tc.unwant, out)
			}
			if (tc.pwsh != nil) != asked {
				t.Fatalf("pwsh asked = %v, want %v", asked, tc.pwsh != nil)
			}
			// Every case where pwsh answers the path, found or missing, must
			// name it: a missing-profile FAIL that drops the path would
			// otherwise pass on its prefix alone.
			if tc.pwsh != nil && strings.Contains(tc.pwsh.out, "{REDIRECTED}") && !strings.Contains(out, redirected) {
				t.Fatalf("the row must name the pwsh-resolved path\n%s", out)
			}
		})
	}
}

// CLI-066 review round 2: under --fix, a pwsh-named profile that was never
// written is still the missing FAIL, and the heal is never run for it — there
// is nothing for it to rebuild.
func TestCheckProfileFiles_FixDoesNotHealAMissingProfile(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "x")
	writeFile(t, filepath.Join(home, ".gemini", "AGY.md"), "x")
	missing := filepath.Join(home, "Redirected", "Docs", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	heal := filepath.Join(home, "scripts", profileHealScript)
	writeFile(t, heal, "# fake heal\r\n")
	sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home, "SCRIPTS_DIR": filepath.Dir(heal)}, []string{"pwsh"}, nil)
	sys.GOOS = "windows"
	sys.CommandOutput = func(name string, args ...string) (string, error) {
		t.Fatalf("unbounded CommandOutput reached: %s %v", name, args)
		return "", nil
	}
	sys.CommandOutputBounded = func(_ time.Duration, name string, args ...string) (string, string, error) {
		if name == "pwsh" && strings.Join(args, " ") == "-NoProfile -Command $PROFILE" {
			return missing + "\r\n", "", nil
		}
		t.Fatalf("nothing but the $PROFILE question may run for a missing profile, got %s %v", name, args)
		return "", "", nil
	}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkProfileFiles(sys, nil, rep, true)
	out := buf.String()
	if rep.Failures() != 1 || !strings.Contains(out, "PowerShell profile missing: "+missing) {
		t.Fatalf("want one missing FAIL naming %s\n%s", missing, out)
	}
	if strings.Contains(out, "corrupted") {
		t.Fatalf("a never-written profile is not BUG-020\n%s", out)
	}
}
