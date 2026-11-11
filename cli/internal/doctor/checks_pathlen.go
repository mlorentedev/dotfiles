package doctor

import (
	"fmt"
	"strings"
)

// cmdPathLimit is the longest PATH cmd.exe still resolves commands from. Past
// it cmd.exe resolves nothing, wherever the entry sits: measured on the
// Windows guest, a .cmd calling a .cmd in the FIRST PATH entry ran at 8180
// chars and was "not recognized" at 8210 (#3285, lesson 574).
const cmdPathLimit = 8191

// checkCmdPathLength reports, on Windows, a process PATH too long for cmd.exe.
// Every mise shim is a .cmd that calls mise, so past the limit each pinned CLI
// fails with "'mise' is not recognized" while pwsh still resolves mise.exe:
// one cause that would otherwise read as many missing installs. Repeated
// entries are the usual padding (a PATH rebuilt from Machine + User on top of
// a PATH that already held both), so the FAIL counts them.
func checkCmdPathLength(sys *System, rep *Report) {
	if sys.GOOS != "windows" {
		return
	}
	raw := sys.Getenv("PATH")
	if len(raw) <= cmdPathLimit {
		rep.Pass(fmt.Sprintf("PATH is %d chars, under cmd.exe's %d", len(raw), cmdPathLimit))
		return
	}
	msg := fmt.Sprintf("PATH is %d chars, past cmd.exe's %d: cmd.exe resolves nothing from it, so every mise shim reports 'mise' is not recognized.",
		len(raw), cmdPathLimit)
	entries, repeats := windowsPathRepeats(raw)
	if repeats > 0 {
		msg += fmt.Sprintf(" %d of its %d entries repeat an earlier one; remove them from the Machine or User Path", repeats, entries)
	} else {
		msg += " Shorten the Machine or User Path"
	}
	rep.Fail(msg)
}

// windowsPathRepeats counts a Windows PATH's non-empty entries and how many of
// them repeat an earlier one, compared as Windows compares directories: case-
// insensitively and ignoring a trailing backslash.
func windowsPathRepeats(raw string) (entries, repeats int) {
	seen := map[string]bool{}
	for _, e := range strings.Split(raw, ";") {
		key := strings.ToLower(strings.TrimRight(strings.TrimSpace(e), `\`))
		if key == "" {
			continue
		}
		entries++
		if seen[key] {
			repeats++
			continue
		}
		seen[key] = true
	}
	return entries, repeats
}
