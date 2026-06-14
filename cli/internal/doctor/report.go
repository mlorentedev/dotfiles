package doctor

import (
	"fmt"
	"io"
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

	totals map[Status]int

	section     string
	secCounts   map[Status]int
	secPrinted  int  // lines actually emitted for the current section
	sectionOpen bool // a Section() header has been printed and not yet flushed
}

// NewReport builds a Report writing to w. When verbose is false, passing
// checks are summarised rather than listed line-by-line.
func NewReport(w io.Writer, verbose bool) *Report {
	return &Report{w: w, verbose: verbose, totals: map[Status]int{}, secCounts: map[Status]int{}}
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
	r.emit("\n[%s]\n", title)
}

func (r *Report) add(s Status, msg string) {
	r.totals[s]++
	r.secCounts[s]++
	if s == StatusPass && !r.verbose {
		return // suppressed; surfaced in the section summary instead
	}
	r.emit("  %s %s\n", statusTag[s], msg)
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
			r.emit("  (%d checks, all ok)\n", total)
		}
	}
	r.sectionOpen = false
}

// Summary flushes the last section and prints the global tally. Call once,
// after all checks have run.
func (r *Report) Summary() {
	r.flush()
	r.emit("\nResults: %d passed, %d failed, %d warned, %d skipped\n",
		r.totals[StatusPass], r.totals[StatusFail], r.totals[StatusWarn], r.totals[StatusSkip])
	if r.totals[StatusFix] > 0 {
		r.emit("Applied %d fix action(s)\n", r.totals[StatusFix])
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
