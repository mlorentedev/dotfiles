package lane

import (
	"fmt"
	"strings"
)

// refuseBatchShim refuses a launch target that Windows would run through
// cmd.exe. CreateProcess starts a .cmd or .bat file by way of cmd.exe, which
// re-parses % ^ & | < > " in the arguments, and Go quotes a command line for
// CommandLineToArgvW, not for cmd.exe (os/exec documents the exception). The
// lane prompt is free text, so it could be mangled or could run a command
// (BatBadBut, #3429). npm installs an agent as such a shim, which is why this
// is the check between a launch-table entry and the process.
//
// The target is refused whatever the arguments hold: cmd.exe reads more than
// the six characters above (newlines, `!`, `,;=`), and a list of the ones to
// look for is how this class keeps getting through.
//
// It has no build tag so a Linux or macOS run can test it; only the Windows
// Exec calls it, because only there does a shim get run by cmd.exe.
func refuseBatchShim(agent, bin string) error {
	// Windows drops trailing dots and spaces from a file name, so `pi.cmd.`
	// is `pi.cmd`.
	name := strings.ToLower(strings.TrimRight(bin, ". "))
	if !strings.HasSuffix(name, ".cmd") && !strings.HasSuffix(name, ".bat") {
		return nil
	}
	return fmt.Errorf("lane: %s resolves to %s, a batch file: cmd.exe would re-parse the prompt "+
		"(%% ^ & | < > \") and could run part of it as a command. Install a native %s executable "+
		"ahead of it on PATH, or launch it through a native wrapper that does not use cmd.exe", agent, bin, agent)
}
