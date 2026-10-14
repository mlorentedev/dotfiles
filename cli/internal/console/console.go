// Package console is how dotf's commands look on a terminal: when to colour,
// the palette and the duration format (CLI-114). converge and doctor both
// render through it, so they read the same on every OS. Each command keeps
// its own status vocabulary; what is shared is the look, not an enum.
package console

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// SGR codes of the palette.
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
)

// ColorEnabled says whether output to a writer that is (or is not) a terminal
// is coloured. A non-empty NO_COLOR (no-color.org) or TERM=dumb turns it off.
// getenv is a parameter, so a test sets nothing in the process environment.
func ColorEnabled(getenv func(string) string, terminal bool) bool {
	if getenv("NO_COLOR") != "" || getenv("TERM") == "dumb" {
		return false
	}
	return terminal
}

// IsTerminal reports whether w is a file open on a terminal. Decide it on the
// raw writer: a tee or a buffer around a terminal is never one.
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// Painter colours text when On, and returns it bare otherwise.
type Painter struct {
	On bool
}

// Paint wraps text in codes and a reset. With colour off, or no codes, the
// text comes back unchanged, so a pipe reads the same words.
func (p Painter) Paint(text string, codes ...string) string {
	if !p.On || len(codes) == 0 {
		return text
	}
	return strings.Join(codes, "") + text + Reset
}

// Duration formats a step's time at the precision a reader uses: 850ms,
// 12.3s, 2m05s, 2h03m. Each unit rounds before choosing the next, so 59.99s
// reads 1m00s, never 60.0s.
func Duration(d time.Duration) string {
	if d = d.Round(time.Millisecond); d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d = d.Round(100 * time.Millisecond); d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d = d.Round(time.Second); d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	d = d.Round(time.Minute)
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}
