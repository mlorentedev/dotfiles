//go:build windows

package memlink

import (
	"os/exec"
	"syscall"
)

// quoteArg wraps s in double quotes for cmd.exe's own tokenizer. NTFS forbids
// a literal '"' in a path component, so no embedded-quote escaping is needed.
func quoteArg(s string) string { return `"` + s + `"` }

// createLink makes a directory junction via the cmd.exe builtin `mklink /J`.
//
// mklink is a cmd.exe builtin, not a separate executable, so there is no
// second argv-parsing layer downstream of cmd — only cmd's own tokenizer ever
// sees the paths. That tokenizer treats a bare comma, semicolon, equals sign
// (and, unquoted, a space) as a word separator. Go's automatic argv-to-
// command-line escaping (exec.Command's ordinary argument passing) follows a
// different convention entirely — the C-runtime/CommandLineToArgvW one, which
// only quotes an argument containing a space, tab or embedded quote — so a
// path with a bare comma passed as a normal argv element reached cmd.exe
// unquoted, cmd split it into extra words, and mklink failed "the syntax of
// the command is incorrect" (HARNESS-050, #575). Spaces happened to round-
// trip only because the C-runtime convention quotes on space too, coincidence
// masking the real defect.
//
// The fix bypasses that automatic escaping entirely via SysProcAttr.CmdLine
// (Go uses this string verbatim as the process's full command line) and
// quotes both paths for cmd's tokenizer directly: `cmd /s /c "mklink /J
// "<link>" "<src>""`. The /S switch is what makes this work — it tells cmd to
// strip only the outermost quote pair from its command tail and hand the
// still-quoted `mklink /J "..." "..."` straight to the builtin's own argument
// scanner, rather than cmd's default behavior of treating a single quoted
// tail as one opaque token. Verified empirically against comma, semicolon,
// paren, equals and space path components.
func createLink(src, target string) error {
	cmdline := `cmd /s /c "mklink /J ` + quoteArg(target) + ` ` + quoteArg(src) + `"`
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cmdline}
	return cmd.Run()
}
