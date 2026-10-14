package cmd

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/console"
	"github.com/mlorentedev/dotfiles/cli/internal/console/consoletest"
	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// consoleReport holds one entry of every status, the shape a real run prints:
// each step that ran has a duration, the one never reached has none.
func consoleReport(dryRun bool) converge.Report {
	return converge.Report{DryRun: dryRun, GOOS: "linux", Entries: []converge.Entry{
		{Name: "tools", Status: converge.StatusChange, Changes: 2, Detail: "installed herdr, lazygit", Duration: 125 * time.Second},
		{Name: "pi-settings", Status: converge.StatusOK, Detail: "settings.json present, enabledModels in sync", Duration: 850 * time.Millisecond},
		{Name: "legacy-setup", Status: converge.StatusOpaque, Detail: "setup-linux.sh exited 0", Duration: 12340 * time.Millisecond},
		{Name: "identity", Status: converge.StatusFailed, Detail: "the age key is locked", Duration: 40 * time.Millisecond},
		{Name: "records-harness", Status: converge.StatusSkipped, Detail: "not reached"},
	}}
}

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// CLI-114 AC3: a terminal gets each tag coloured by status; a pipe reads the
// same text it read before colour existed, and the coloured output is that
// text exactly once its escape codes are removed.
func TestConvergeConsole_Colour(t *testing.T) {
	for _, plan := range []bool{false, true} {
		name := "converge-apply"
		if plan {
			name = "converge-plan"
		}
		t.Run(name, func(t *testing.T) {
			var plain, color bytes.Buffer
			printConvergeReport(&plain, consoleReport(plan), console.Painter{})
			printConvergeReport(&color, consoleReport(plan), console.Painter{On: true})
			consoletest.Golden(t, name+"-plain", plain.Bytes())
			consoletest.Golden(t, name+"-color", color.Bytes())
			if bytes.IndexByte(plain.Bytes(), 0x1b) >= 0 {
				t.Error("the plain report carries an escape byte")
			}
			if !bytes.Contains(color.Bytes(), []byte(console.Red+console.Bold+"[FAIL]")) {
				t.Errorf("the failed tag is not red:\n%q", color.String())
			}
			if got := sgr.ReplaceAll(color.Bytes(), nil); !bytes.Equal(got, plain.Bytes()) {
				t.Errorf("colour changed the text:\n%q\nwant\n%q", got, plain.String())
			}
		})
	}
}

// The decision is made on the raw writer: a buffer, like the transcript tee,
// is never a terminal.
func TestConvergeConsole_ColourIsATerminals(t *testing.T) {
	if convergePainter(&bytes.Buffer{}).On {
		t.Error("a buffer got colour")
	}
}

func TestConvergeConsole_ColourOnATerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	isTerminal = func(uintptr) bool { return true }
	if !convergePainter(os.Stdout).On {
		t.Error("a terminal got no colour")
	}
	t.Setenv("NO_COLOR", "1")
	if convergePainter(os.Stdout).On {
		t.Error("NO_COLOR did not turn colour off")
	}
}

// runConverge decides colour on the writer it was handed, so a run on a
// terminal prints coloured tags. Only that file's fd is a terminal here, so
// nothing else in the run (the identity walk, sudo) sees one.
func TestConvergeConsole_ARunOnATerminalIsColoured(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	repo, _ := convergeFixture(t)
	f, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	isTerminal = func(fd uintptr) bool { return fd == f.Fd() }
	_ = runConverge(f, repo, true, nil)
	got, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("\x1b[")) || !bytes.Contains(got, []byte("converge plan on ")) {
		t.Errorf("a run on a terminal printed no colour:\n%q", got)
	}
}

// An applied run writes through the transcript tee: the terminal gets the
// colour, decided before the tee existed, and the transcript gets none.
func TestConvergeConsole_AnAppliedRunColoursTheTerminalNotTheTranscript(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	repo, state, _ := runLogFixture(t, "hello\n")
	f, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	isTerminal = func(fd uintptr) bool { return fd == f.Fd() }
	_ = runConverge(f, repo, false, nil)
	got, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("\x1b[")) {
		t.Errorf("the terminal got no colour:\n%q", got)
	}
	logs := runTranscripts(t, state)
	if len(logs) != 1 {
		t.Fatalf("want one transcript, got %v", logs)
	}
	raw, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.IndexByte(raw, 0x1b) >= 0 || !bytes.Contains(raw, []byte("converge on ")) {
		t.Errorf("the transcript holds escape codes or misses the report:\n%q", raw)
	}
}

// CLI-114 AC6: a run with failures ends with each failed step again, by name
// and error, so a failure is never only one line inside the table. A run
// without failures prints no summary.
func TestConvergeConsole_FailureSummary(t *testing.T) {
	rep := converge.Report{GOOS: "darwin", Entries: []converge.Entry{
		{Name: "checkout", Status: converge.StatusOK, Detail: "up to date", Duration: 300 * time.Millisecond},
		{Name: "tools", Status: converge.StatusFailed, Detail: "mise install: exit status 1", Duration: 61 * time.Second},
		{Name: "git-config", Status: converge.StatusFailed, Detail: "after tools: post-condition: user.email unset", Duration: 20 * time.Millisecond},
		{Name: "identity", Status: converge.StatusSkipped, Detail: "not reached"},
	}}
	var out bytes.Buffer
	printConvergeReport(&out, rep, console.Painter{})
	consoletest.Golden(t, "converge-failures-plain", out.Bytes())

	rep.Entries[1].Status, rep.Entries[2].Status = converge.StatusOK, converge.StatusOK
	var none bytes.Buffer
	printConvergeReport(&none, rep, console.Painter{})
	if strings.Contains(none.String(), "failed:") {
		t.Errorf("a run without failures printed a summary:\n%s", none.String())
	}
}

// A name longer than the default column widens it for every line, so the
// duration and detail columns stay aligned.
func TestConvergeConsole_ALongNameWidensTheColumn(t *testing.T) {
	rep := converge.Report{GOOS: "darwin", Entries: []converge.Entry{
		{Name: "git-hooks", Status: converge.StatusOK, Detail: "wired", Duration: 7 * time.Millisecond},
		{Name: "workstation-target", Status: converge.StatusOK, Detail: "converged", Duration: 3 * time.Millisecond},
	}}
	var out bytes.Buffer
	printConvergeReport(&out, rep, console.Painter{})
	lines := strings.Split(out.String(), "\n")
	if a, b := strings.Index(lines[1], "wired"), strings.Index(lines[2], "converged"); a != b {
		t.Errorf("detail columns at %d and %d, want one column:\n%s", a, b, out.String())
	}
}
