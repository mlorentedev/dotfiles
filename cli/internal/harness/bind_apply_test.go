package harness

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestHookBinaryTokenMatchesWhatEachSetupScriptDeployed is the Windows
// duplicate-hook defect, caught statically rather than on the Windows box.
//
// Adoption of the pre-bind entry is by EXACT command equality, so the token this
// renders must equal, byte for byte, what the setup script of that OS wrote:
//
//	setup-linux.sh    $HOME/.local/bin/dotf mem session-start      (bare)
//	setup-windows.ps1 "…\.local\bin\dotf.exe" mem session-start    (quoted)
//
// A bare token on Windows matches neither, and bind would append a SECOND
// session-start hook on the first run there. Table-driven over goos because that
// is the only way the Windows leg is exercised from the machine that develops it.
func TestHookBinaryTokenMatchesWhatEachSetupScriptDeployed(t *testing.T) {
	for _, tc := range []struct {
		name, path, goos, want string
	}{
		{"windows is quoted, matching Merge-ClaudeSettings",
			`C:\Users\m\.local\bin\dotf.exe`, "windows", `"C:\Users\m\.local\bin\dotf.exe"`},
		{"linux is bare, matching merge_claude_settings",
			"/home/m/.local/bin/dotf", "linux", "/home/m/.local/bin/dotf"},
		{"a space forces quoting even where the old entry was bare",
			"/home/two words/.local/bin/dotf", "linux", `"/home/two words/.local/bin/dotf"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := HookBinaryToken(tc.path, tc.goos); got != tc.want {
				t.Errorf("HookBinaryToken(%q, %q) = %q, want %q", tc.path, tc.goos, got, tc.want)
			}
		})
	}
}

func TestHookBinaryTokenForTargetUsesAgyWindowsCommandSyntax(t *testing.T) {
	const windowsPath = `C:\Users\m\.local\bin\dotf.exe`

	for _, tc := range []struct {
		name, path, goos, format, want string
	}{
		{"agy on windows is bare", windowsPath, "windows", NamedHooksFormat, windowsPath},
		{"claude on windows stays quoted", windowsPath, "windows", "command-hook", `"` + windowsPath + `"`},
		{"agy path with spaces stays quoted until its runner supports escaping",
			`C:\Users\Two Words\.local\bin\dotf.exe`, "windows", NamedHooksFormat,
			`"C:\Users\Two Words\.local\bin\dotf.exe"`},
		{"agy on linux stays bare", "/home/m/.local/bin/dotf", "linux",
			NamedHooksFormat, "/home/m/.local/bin/dotf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := HookBinaryTokenForTarget(tc.path, tc.goos, tc.format); got != tc.want {
				t.Errorf("HookBinaryTokenForTarget(%q, %q, %q) = %q, want %q",
					tc.path, tc.goos, tc.format, got, tc.want)
			}
		})
	}
}

// TestResolveDotfPathCarriesTheWindowsSuffix pins the other half of the same
// defect: the path itself. `dotf` and `dotf.exe` are different commands to the
// equality check, so a suffix-less resolve on Windows duplicates just as surely
// as missing quotes.
func TestResolveDotfPathCarriesTheWindowsSuffix(t *testing.T) {
	home := t.TempDir()
	want := "dotf"
	if runtime.GOOS == "windows" {
		want = "dotf.exe"
	}
	if got := filepath.Base(ResolveDotfPath(home)); got != want {
		t.Errorf("ResolveDotfPath resolves to %q, want basename %q on %s", got, want, runtime.GOOS)
	}
}

// An unknown format is a refusal. The previous default arm handed it to claude's
// merge, which is how a format this code did not know would have been written into
// a file of another shape.
func TestBindOneRefusesAFormatItDoesNotKnow(t *testing.T) {
	home := t.TempDir()
	target := BindTarget{
		Agent: "future", File: ".future/hooks.json", Format: "hooks-yaml", Matcher: true,
		EmitHooks: []EmitHook{{ID: "gate", Event: "PreToolUse", Command: "harness gate", Timeout: 5}},
	}
	if _, _, err := bindOne(target, home, "/opt/dotf", false); err == nil || !strings.Contains(err.Error(), "unsupported bind format") {
		t.Fatalf("want an unsupported-format refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".future", "hooks.json")); !os.IsNotExist(err) {
		t.Errorf("a refused format must not create a file (err %v)", err)
	}
}

// TestBindKeepsTheOutcomeOfATargetThatFailedAfterWriting: the merge is written
// before the retirements run, so a retirement that fails leaves a rewritten
// file behind. Bind must return that target's outcome with the error, or its
// caller reports nothing for a file it changed.
func TestBindKeepsTheOutcomeOfATargetThatFailedAfterWriting(t *testing.T) {
	home := t.TempDir()
	broken := filepath.Join(home, ".old", "settings.json")
	if err := os.MkdirAll(filepath.Dir(broken), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(broken, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := BindTarget{
		Agent: "claude", File: ".claude/settings.json", Format: "command-hook",
		EmitHooks: []EmitHook{{ID: "start", Event: "SessionStart", Command: "mem session-start", Timeout: 5}},
		Retire:    []RetiredHook{{File: ".old/settings.json", Event: "SessionStart", ID: "start"}},
	}
	out, err := Bind([]BindTarget{target}, BindOptions{Home: home, Binary: "/opt/dotf", GOOS: "linux", Has: func(string) bool { return true }})
	if err == nil {
		t.Fatal("want the unreadable retirement to fail the bind")
	}
	if len(out) != 1 || out[0].Agent != "claude" || !out[0].Changed {
		t.Fatalf("want the failing target's outcome with Changed=true, got %+v", out)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".claude", "settings.json")); statErr != nil {
		t.Errorf("the merge was written before the retirement failed: %v", statErr)
	}
}

// TestBindReportsNothingForATargetThatFailedBeforeWriting: a settings file that
// cannot be read fails the bind before anything is written, and an outcome for
// it would render as "hooks already current".
func TestBindReportsNothingForATargetThatFailedBeforeWriting(t *testing.T) {
	home := t.TempDir()
	settings := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := BindTarget{
		Agent: "claude", File: ".claude/settings.json", Format: "command-hook",
		EmitHooks: []EmitHook{{ID: "start", Event: "SessionStart", Command: "mem session-start", Timeout: 5}},
	}
	out, err := Bind([]BindTarget{target}, BindOptions{Home: home, Binary: "/opt/dotf", GOOS: "linux", Has: func(string) bool { return true }})
	if err == nil || len(out) != 0 {
		t.Fatalf("want an error and no outcome, got %+v, %v", out, err)
	}
}
