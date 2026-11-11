package doctor

import (
	"fmt"
	"strings"
	"unicode/utf16"
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
// a PATH that already held both), so the FAIL names them. The length is in
// UTF-16 code units, the unit cmd.exe's limit is in: a Go string's len counts
// UTF-8 bytes, which a non-ASCII directory (C:\Users\José) would inflate.
func checkCmdPathLength(sys *System, rep *Report) {
	if sys.GOOS != "windows" {
		return
	}
	raw := sys.Getenv("PATH")
	n := len(utf16.Encode([]rune(raw)))
	if n <= cmdPathLimit {
		rep.Pass(fmt.Sprintf("PATH is %d chars, under cmd.exe's %d", n, cmdPathLimit))
		return
	}
	msg := fmt.Sprintf("PATH is %d chars, past cmd.exe's %d: cmd.exe resolves nothing from it, so every mise shim reports 'mise' is not recognized.",
		n, cmdPathLimit)
	entries, repeated := windowsPathRepeats(raw)
	if len(repeated) > 0 {
		msg += fmt.Sprintf(" %d of its %d entries repeat an earlier one (%s); remove them from the Machine or User Path",
			countRepeats(repeated), entries, nameRepeats(repeated))
	} else {
		msg += " Shorten the Machine or User Path"
	}
	rep.Fail(msg)
}

// pathRepeat is a PATH entry that appears more than once: its first spelling
// and how many times it repeats after that.
type pathRepeat struct {
	entry string
	times int
}

// maxNamedRepeats caps the entries a FAIL names, so a PATH tripled by a rebuild
// does not print a hundred directories.
const maxNamedRepeats = 5

// windowsPathRepeats counts a Windows PATH's non-empty entries and returns the
// ones that repeat an earlier one, in first-seen order, compared as Windows
// compares directories: case-insensitively and ignoring a trailing backslash.
func windowsPathRepeats(raw string) (entries int, repeated []pathRepeat) {
	first := map[string]string{} // key -> first spelling
	index := map[string]int{}    // key -> position in repeated
	for _, e := range strings.Split(raw, ";") {
		e = strings.TrimSpace(e)
		key := strings.ToLower(strings.TrimRight(e, `\`))
		if key == "" {
			continue
		}
		entries++
		if _, seen := first[key]; !seen {
			first[key] = e
			continue
		}
		if pos, ok := index[key]; ok {
			repeated[pos].times++
			continue
		}
		index[key] = len(repeated)
		repeated = append(repeated, pathRepeat{entry: first[key], times: 1})
	}
	return entries, repeated
}

func countRepeats(rs []pathRepeat) int {
	n := 0
	for _, r := range rs {
		n += r.times
	}
	return n
}

// nameRepeats lists the first maxNamedRepeats repeated entries, then how many
// more there are.
func nameRepeats(rs []pathRepeat) string {
	var names []string
	for i, r := range rs {
		if i == maxNamedRepeats {
			names = append(names, fmt.Sprintf("and %d more", len(rs)-maxNamedRepeats))
			break
		}
		names = append(names, r.entry)
	}
	return strings.Join(names, ", ")
}
