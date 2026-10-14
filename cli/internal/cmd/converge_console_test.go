package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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

// renderConvergeReport prints a finished run through the printer runConverge
// uses: the column width comes from the registry, and each entry is a step the
// run reported done.
func renderConvergeReport(t *testing.T, w *bytes.Buffer, rep converge.Report, p console.Painter) {
	t.Helper()
	reg := make([]converge.Reconciler, 0, len(rep.Entries))
	for _, e := range rep.Entries {
		reg = append(reg, step{name: e.Name})
	}
	pr := newConvergePrinter(w, reg, p, false)
	pr.header(rep.DryRun, rep.GOOS)
	for _, e := range rep.Entries {
		pr.Done(e)
	}
	pr.finish(rep)
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
			renderConvergeReport(t, &plain, consoleReport(plan), console.Painter{})
			renderConvergeReport(t, &color, consoleReport(plan), console.Painter{On: true})
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
	if console.IsTerminalBy(&bytes.Buffer{}, func(uintptr) bool { return true }) {
		t.Error("a buffer passed for a terminal")
	}
}

func TestConvergeConsole_ColourOnATerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	if !convergePainter(true).On {
		t.Error("a terminal got no colour")
	}
	if convergePainter(false).On {
		t.Error("a pipe got colour")
	}
	t.Setenv("NO_COLOR", "1")
	if convergePainter(true).On {
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
	// The transcript records what the terminal showed, uncoloured: the step
	// announcements too, so a terminal run's log differs from a piped run's
	// by those lines (CLI-114 review).
	// A whole line: records-mirror's own detail holds a "→" too.
	if !bytes.Contains(raw, []byte("\n→ checkout\n")) {
		t.Errorf("the transcript misses the step announcements the terminal got:\n%q", raw)
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
	renderConvergeReport(t, &out, rep, console.Painter{})
	consoletest.Golden(t, "converge-failures-plain", out.Bytes())

	rep.Entries[1].Status, rep.Entries[2].Status = converge.StatusOK, converge.StatusOK
	var none bytes.Buffer
	renderConvergeReport(t, &none, rep, console.Painter{})
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
	renderConvergeReport(t, &out, rep, console.Painter{})
	lines := strings.Split(out.String(), "\n")
	if a, b := strings.Index(lines[1], "wired"), strings.Index(lines[2], "converged"); a < 0 || a != b {
		t.Errorf("detail columns at %d and %d, want one column:\n%s", a, b, out.String())
	}
}

// step is a reconciler for the streaming tests: it can write its own output
// to the console while it runs, as installers do, and fail.
type step struct {
	name, detail string
	out          *bytes.Buffer
	says         string
	err          error
}

func (s step) Name() string        { return s.name }
func (s step) Platforms() []string { return nil }
func (s step) Probe(converge.Env) error {
	return nil
}
func (s step) Reconcile(converge.Env, bool) (converge.Result, error) {
	if s.says != "" {
		s.out.WriteString(s.says + "\n")
	}
	return converge.Result{Detail: s.detail}, s.err
}

// fixedDurations holds every step's duration at one value, so the golden file
// holds no real time.
type fixedDurations struct{ *convergePrinter }

func (f fixedDurations) Done(e converge.Entry) {
	e.Duration = 1200 * time.Millisecond
	f.convergePrinter.Done(e)
}

// CLI-114 AC5: on a terminal each step's start line precedes its own output
// and its result line follows it; the step the run never reached is printed
// exactly once, at the end, before the counts and the failure summary.
func TestConvergeConsole_Streams(t *testing.T) {
	for _, terminal := range []bool{true, false} {
		name := "converge-stream-pipe"
		if terminal {
			name = "converge-stream-terminal"
		}
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			reg := []converge.Reconciler{
				step{name: "checkout", detail: "up to date"},
				step{name: "tools", out: &out, says: "installing lazygit", err: errors.New("mise install: exit status 1")},
				step{name: "workstation-target", detail: "converged"},
			}
			pr := newConvergePrinter(&out, reg, console.Painter{}, terminal)
			pr.header(false, "linux")
			rep, err := converge.RunObserved(reg, converge.Env{GOOS: "linux"}, false, fixedDurations{pr})
			if err == nil {
				t.Fatal("want the tools step's error")
			}
			pr.finish(rep)
			consoletest.Golden(t, name, out.Bytes())
			if n := strings.Count(out.String(), "workstation-target"); n != 1 {
				t.Errorf("the unreached step printed %d times, want once:\n%s", n, out.String())
			}
		})
	}
}

// CLI-114 AC6: a failed run's last lines are the failure summary, then the
// transcript's path: what failed, and where to read why.
func TestConvergeConsole_AFailedRunSummarisesBeforeTheLog(t *testing.T) {
	repo, home := convergeFixture(t)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	base := convergeOptions
	convergeOptions = func() converge.Options {
		o := base()
		fail := func(converge.Env) error { return errors.New("boom") }
		o.RunHarnessDeploy, o.RunSetup = fail, fail // one runs on every OS
		return o
	}
	out, _, err := execute(t, "converge", "--repo", repo)
	if err == nil {
		t.Fatal("the run should fail")
	}
	summary, log := strings.Index(out, "\nfailed:\n"), strings.Index(out, "\nlog: ")
	if summary < 0 || log < 0 || summary > log {
		t.Errorf("want the failure summary, then log:\n%s", out)
	}
}
