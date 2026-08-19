package doctor

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Status is the outcome of a single check. Only StatusFail drives a non-zero
// exit; the others are advisory, exactly as in the shell twins (a WARN was a
// drift worth surfacing but not a setup failure; SKIP/INFO were informational).
type Status int

const (
	StatusPass Status = iota
	StatusFail
	StatusWarn
	StatusSkip
	StatusInfo
	StatusFix
)

var statusTag = map[Status]string{
	StatusPass: "[ OK ]",
	StatusFail: "[FAIL]",
	StatusWarn: "[WARN]",
	StatusSkip: "[SKIP]",
	StatusInfo: "[INFO]",
	StatusFix:  "[FIX ]",
}

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
)

var coloredTag = map[Status]string{
	StatusPass: ansiGreen + "[ OK ]" + ansiReset,
	StatusFail: ansiRed + "[FAIL]" + ansiReset,
	StatusWarn: ansiYellow + "[WARN]" + ansiReset,
	StatusSkip: ansiCyan + "[SKIP]" + ansiReset,
	StatusInfo: ansiBlue + "[INFO]" + ansiReset,
	StatusFix:  ansiMagenta + "[FIX ]" + ansiReset,
}

// Report accumulates check outcomes, prints them grouped by section, and
// computes the process exit code. It folds healthcheck's PASS/FAIL/WARN/SKIP
// and doctor's [ok]/[warn]/[fail]/[info]/[fix] into one tag vocabulary.
//
// In non-verbose mode passing checks are suppressed (noise reduction, matching
// doctor.sh); a section that produced only passes still prints a one-line
// "(N checks, all ok)" so a bare header never looks like a stalled run.
type Report struct {
	w       io.Writer
	verbose bool
	color   bool

	totals map[Status]int

	section     string
	secCounts   map[Status]int
	secPrinted  int  // lines actually emitted for the current section
	sectionOpen bool // a Section() header has been printed and not yet flushed
}

// isColorEnabled checks whether the output writer is an interactive terminal
// and that color emission is not disabled via NO_COLOR or TERM=dumb.
func isColorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// NewReport builds a Report writing to w. When verbose is false, passing
// checks are summarised rather than listed line-by-line. Color is enabled
// automatically when w is an interactive terminal.
func NewReport(w io.Writer, verbose bool) *Report {
	return &Report{
		w:         w,
		verbose:   verbose,
		color:     isColorEnabled(w),
		totals:    map[Status]int{},
		secCounts: map[Status]int{},
	}
}

// SetColor forces color output on or off (useful in tests or explicit flags).
func (r *Report) SetColor(c bool) {
	r.color = c
}

// emit writes to the report's sink. A write error to stdout/a buffer is not
// actionable, so it is deliberately ignored — centralised here so the rest of
// the file stays free of `_, _ =` noise (and errcheck-clean).
func (r *Report) emit(format string, a ...any) {
	_, _ = fmt.Fprintf(r.w, format, a...)
}

// Section starts a new named group; the previous one is flushed first.
func (r *Report) Section(title string) {
	r.flush()
	r.section = title
	r.secCounts = map[Status]int{}
	r.secPrinted = 0
	r.sectionOpen = true
	if r.color {
		r.emit("\n%s[%s]%s\n", ansiBold, title, ansiReset)
	} else {
		r.emit("\n[%s]\n", title)
	}
}

func (r *Report) add(s Status, msg string) {
	r.totals[s]++
	r.secCounts[s]++
	if s == StatusPass && !r.verbose {
		return // suppressed; surfaced in the section summary instead
	}
	tag := statusTag[s]
	if r.color {
		tag = coloredTag[s]
	}
	r.emit("  %s %s\n", tag, msg)
	r.secPrinted++
}

// Pass/Fail/Warn/Skip/Info/Fix record one check outcome.
func (r *Report) Pass(msg string) { r.add(StatusPass, msg) }
func (r *Report) Fail(msg string) { r.add(StatusFail, msg) }
func (r *Report) Warn(msg string) { r.add(StatusWarn, msg) }
func (r *Report) Skip(msg string) { r.add(StatusSkip, msg) }
func (r *Report) Info(msg string) { r.add(StatusInfo, msg) }
func (r *Report) Fix(msg string)  { r.add(StatusFix, msg) }

// flush closes the current section, printing an all-ok summary when nothing
// else was emitted for it.
func (r *Report) flush() {
	if !r.sectionOpen {
		return
	}
	if r.secPrinted == 0 {
		total := 0
		for _, c := range r.secCounts {
			total += c
		}
		if total > 0 {
			if r.color {
				r.emit("  %s(%d checks, all ok)%s\n", ansiGreen, total, ansiReset)
			} else {
				r.emit("  (%d checks, all ok)\n", total)
			}
		}
	}
	r.sectionOpen = false
}

// Summary flushes the last section and prints the global tally. Call once,
// after all checks have run.
func (r *Report) Summary() {
	r.flush()
	if r.color {
		if r.totals[StatusFail] > 0 {
			r.emit("\nResults: %s%d passed%s, %s%d failed%s, %s%d warned%s, %s%d skipped%s\n",
				ansiGreen, r.totals[StatusPass], ansiReset,
				ansiRed+ansiBold, r.totals[StatusFail], ansiReset,
				ansiYellow, r.totals[StatusWarn], ansiReset,
				ansiCyan, r.totals[StatusSkip], ansiReset)
		} else {
			r.emit("\n%sResults: %d passed, 0 failed, %d warned, %d skipped%s\n",
				ansiGreen+ansiBold, r.totals[StatusPass], r.totals[StatusWarn], r.totals[StatusSkip], ansiReset)
		}
		if r.totals[StatusFix] > 0 {
			r.emit("%sApplied %d fix action(s)%s\n", ansiMagenta, r.totals[StatusFix], ansiReset)
		}
	} else {
		r.emit("\nResults: %d passed, %d failed, %d warned, %d skipped\n",
			r.totals[StatusPass], r.totals[StatusFail], r.totals[StatusWarn], r.totals[StatusSkip])
		if r.totals[StatusFix] > 0 {
			r.emit("Applied %d fix action(s)\n", r.totals[StatusFix])
		}
	}
}

// ExitCode is 1 when any check failed, else 0 — the healthcheck.sh /
// doctor.sh exit contract (advisory WARN/SKIP/INFO never fail the run).
func (r *Report) ExitCode() int {
	if r.totals[StatusFail] > 0 {
		return 1
	}
	return 0
}

// Failures returns how many checks failed (handy for assertions in tests).
func (r *Report) Failures() int { return r.totals[StatusFail] }
