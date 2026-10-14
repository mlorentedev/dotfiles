package doctor

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

// golden compares got with testdata/<name>.golden, or rewrites the file under
// -update. A golden file is the console's contract: a change to it is a
// change to what the owner reads, reviewed as a diff.
func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	p := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p) //nolint:gosec // the package's own testdata
	if err != nil {
		t.Fatalf("%v (run the test with -update to create it)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output differs from %s\n--- got\n%q\n--- want\n%q", p, got, want)
	}
}

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
			golden(t, "report-"+c.name, buf.Bytes())
		})
	}
}
