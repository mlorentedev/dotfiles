package doctor

import (
	"bytes"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/console/consoletest"
)

// reportScript drives every path of Report: a section of passes only (the
// all-ok line), one of every status, an empty section, and the summary.
func reportScript(r *Report, fail bool) {
	r.Section("Quiet")
	r.Pass("one")
	r.Pass("two")
	r.Section("Mixed")
	r.Pass("passes")
	if fail {
		r.Fail("fails")
	}
	r.Warn("warns")
	r.Skip("skips")
	r.Info("informs")
	r.Fix("fixes")
	r.Section("Empty")
	r.Summary()
}

// CLI-114 AC2: doctor's output stays byte-identical when its colour moves
// into the shared console package. Captured before the move.
func TestReport_Golden(t *testing.T) {
	for _, c := range []struct {
		name           string
		color, verbose bool
		fail           bool
	}{
		{"plain", false, false, true},
		{"plain-verbose-clean", false, true, false},
		{"color", true, false, true},
		{"color-verbose-clean", true, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			r := NewReport(&buf, c.verbose)
			r.SetColor(c.color)
			reportScript(r, c.fail)
			consoletest.Golden(t, "report-"+c.name, buf.Bytes())
		})
	}
}
