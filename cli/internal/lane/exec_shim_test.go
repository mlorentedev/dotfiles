package lane

import (
	"strings"
	"testing"
)

// A .cmd or .bat target is run by cmd.exe, which re-parses % ^ & | < > " in the
// arguments; Go quotes for CommandLineToArgvW, not for cmd.exe (BatBadBut). The
// decision looks at the target alone and never at the prompt: a metacharacter
// blocklist is the shape this class keeps slipping through (cmd.exe also reads
// newlines, `!` and `,;=`). exec_windows_test.go sends hostile prompts through
// the real Exec.
func TestRefuseBatchShim_RefusesABatchFileNamingTheAgentAndTheRemedy(t *testing.T) {
	for _, bin := range []string{
		`C:\Users\m\AppData\Roaming\npm\pi.cmd`,
		`C:\npm\pi.CMD`,
		`C:\npm\pi.Bat`,
		`pi.bat`,
		`C:\npm\pi.cmd.`, // Windows drops trailing dots and spaces
		`C:\npm\pi.cmd  `,
		`/tmp/pi.cmd`,
	} {
		err := refuseBatchShim("pi", bin)
		if err == nil {
			t.Errorf("bin %q: want a refusal", bin)
			continue
		}
		for _, want := range []string{"pi", bin, "cmd.exe", "native"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("bin %q: error %q does not name %q", bin, err, want)
			}
		}
	}
}

// A native executable is not this class: CreateProcess hands it the command
// line without cmd.exe.
func TestRefuseBatchShim_AllowsNativeExecutables(t *testing.T) {
	for _, bin := range []string{
		`C:\bin\claude.exe`,
		`C:\bin\agy.EXE`,
		`C:\bin\dotf.exe`,
		`/usr/local/bin/claude`,
		`C:\bin\cmd.exe.bak`,
		`C:\bin\pi.cmdx`,
		`C:\bin\batch`,
		`C:\bat\pi`,
		``,
	} {
		if err := refuseBatchShim("claude", bin); err != nil {
			t.Errorf("bin %q: want it allowed, got %v", bin, err)
		}
	}
}
