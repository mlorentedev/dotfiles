package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/console"
	"github.com/mlorentedev/dotfiles/cli/internal/converge"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
	"github.com/mlorentedev/dotfiles/cli/internal/sshtarget"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

func newConvergeCmd() *cobra.Command {
	var (
		plan bool
		repo string
		only []string
	)
	cmd := &cobra.Command{
		Use:   "converge",
		Short: "Bring this machine to the state the dotfiles checkout declares",
		Long: "converge runs an ordered list of reconcilers, each converging one part of\n" +
			"the machine from data in the checkout. The checkout comes first, then the\n" +
			"records, so nothing runs before the files it reads exist. On Linux and\n" +
			"Windows the setup script runs last, for what no native step covers yet; it\n" +
			"cannot plan, so it is reported opaque, never as converged.\n\n" +
			"--plan reports what each reconciler would change and writes nothing.\n" +
			"Without it, each reconciler applies and then proves its post-condition; the\n" +
			"first failure stops the run and names the reconciler. A reconciler that does\n" +
			"not apply to this OS is reported as skipped, never as passed.\n\n" +
			"Re-running on a converged machine reports 0 changed.\n\n" +
			"--only runs the named reconcilers, in registry order, and leaves the\n" +
			"persisted report of the last full run alone.\n\n" +
			"A full run that converged ends with the identity converge cannot\n" +
			"produce (age key, Bitwarden, GitHub login, vault): on a terminal it walks\n" +
			"the restore, as `dotf identity restore` does; elsewhere it prints what is\n" +
			"left. Neither changes the exit status.",
		Example:      "  dotf converge --plan\n  dotf converge\n  dotf converge --only git-config",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if repo == "" {
				repo = convergeCheckout(env.Home())
			}
			err := runConverge(cmd.OutOrStdout(), repo, plan, only)
			// The identity converge cannot produce: walked on a terminal,
			// printed elsewhere, after a full run that converged (#2013 D12).
			if err == nil && len(only) == 0 {
				identityEpilogue(cmd.OutOrStdout(), plan)
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&plan, "plan", false, "report what would change and write nothing")
	cmd.Flags().StringVar(&repo, "repo", "", "dotfiles checkout to converge from")
	cmd.Flags().StringSliceVar(&only, "only", nil, "run only these reconcilers (comma-separated names)")
	return cmd
}

// runConverge drives the registry over repo and prints the report. An applied
// run keeps a transcript (#1098); a full one also persists the report as
// last.json, which a plan or an --only run leaves alone. `dotf converge` and
// `dotf update` both run through here.
func runConverge(w io.Writer, repo string, plan bool, only []string) error {
	home := env.Home()
	e := converge.Env{RepoRoot: repo, Home: home, DeployDir: env.DotfilesDir(home), GOOS: runtime.GOOS}
	terminal, paint := convergeTerminal(w), convergePainter(w)
	out := w
	var rl *converge.RunLog
	if !plan {
		var done func()
		rl, out, done = beginRunLog(w, repo)
		defer done()
	}
	opts := convergeOptions()
	if rl != nil {
		opts.MiseRun = loggedRunner(opts.MiseRun, rl.Stream())
	}
	reg := converge.Registry(opts)
	if len(only) > 0 {
		var serr error
		if reg, serr = converge.Select(reg, only); serr != nil {
			return serr
		}
	}
	pr := newConvergePrinter(out, reg, paint, terminal)
	pr.header(plan, e.GOOS)
	rep, err := converge.RunObserved(reg, e, plan, pr)
	pr.finish(rep)
	if plan {
		return err
	}
	return saveConvergeReport(w, out, rl, rep, err, len(only) == 0)
}

// beginRunLog opens the run's transcript and captures this process's output
// into it. It returns the transcript (nil when it could not be opened), the
// writer the report goes to, and what to run when the run ends. The capture
// starts before the options are built: they bind os.Stderr (the installers'
// progress) when they are made.
func beginRunLog(w io.Writer, repo string) (*converge.RunLog, io.Writer, func()) {
	rl := openConvergeRunLog(w, repo, os.Args)
	if rl == nil {
		return nil, w, func() {}
	}
	c, err := startRunCapture(rl)
	if err != nil {
		_, _ = fmt.Fprintf(w, "warning: this run's transcript misses subprocess output: %v\n", err)
	}
	return rl, teeWriter{w, rl.Stream()}, func() {
		if c != nil {
			c.stop()
		}
		finishRunLog(w, rl)
	}
}

// saveConvergeReport writes the run's report beside its transcript and, for a
// full run, to last.json, which from-zero.yml reads with jq. Both carry the transcript's
// redaction: a step's detail or the run's error can echo a remote URL with a
// token in it. A failure to write either is joined to the run's own error.
func saveConvergeReport(w, out io.Writer, rl *converge.RunLog, rep converge.Report, err error, full bool) error {
	saved, savedErr := rep, err
	if rl != nil {
		saved, savedErr = rl.Redacted(rep, err)
		if werr := rl.WriteReport(saved, savedErr); werr != nil {
			_, _ = fmt.Fprintf(w, "warning: writing this run's report: %v\n", werr)
		}
		_, _ = fmt.Fprintf(out, "log: %s\n", rl.Path())
	}
	if !full {
		return err
	}
	state, serr := env.StateDir()
	if serr != nil {
		return errors.Join(err, fmt.Errorf("writing the converge report: %w", serr))
	}
	path := filepath.Join(state, "converge", "last.json")
	if werr := converge.WriteReport(path, saved, savedErr); werr != nil {
		return errors.Join(err, fmt.Errorf("writing the converge report: %w", werr))
	}
	_, _ = fmt.Fprintf(out, "report: %s\n", path)
	return err
}

// finishRunLog closes a run's transcript and prunes the oldest runs. Neither
// failing changes the run's result; both say so.
func finishRunLog(w io.Writer, rl *converge.RunLog) {
	if err := rl.Close(); err != nil {
		_, _ = fmt.Fprintf(w, "warning: closing this run's transcript: %v\n", err)
	}
	if err := converge.PruneRuns(rl.Dir(), converge.RunLogKeep); err != nil {
		_, _ = fmt.Fprintf(w, "warning: pruning old converge transcripts: %v\n", err)
	}
}

// convergeOptions wires the registry's side effects; tests replace it.
var convergeOptions = func() converge.Options {
	home := env.Home()
	run, stdout := tools.HomeRunners(home)
	dest := filepath.Join(home, ".local", "bin")
	return converge.Options{
		RunHarnessDeploy: converge.CompileHarnessDeploy,
		MiseRun:          run,
		MiseStdout:       stdout,
		// Install progress goes to stderr, apart from the report on stdout.
		ToolsCatalog:  &tools.Installer{Dest: dest, Out: os.Stderr},
		ToolsBinDirs:  []string{tools.MiseShimsDir(home, runtime.GOOS, os.Getenv), dest},
		GitRun:        gitconfig.ExecRunner,
		GitStoredAuth: gitconfig.StoredLoginRunner,
		CloneURL:      os.Getenv("DOTFILES_REPO"),
		RenderConfigs: strictDeployRenderer,
		ResolvePath:   env.ResolvePath,
		Launchctl:     env.ExecLaunchctl,
		UID:           os.Getuid(),
		SSHRoot:       sshtarget.Sudo{Terminal: identityTerminal},
		UserRun:       sshtarget.Exec,
		Claude:        claudeSyncer,
		AgyPaths:      agyPaths,
		HerdrRun:      tools.ExecRunner,
		RunSetup:      converge.ExecSetup,
		DotfPath:      harness.ResolveDotfPath,
	}
}

// convergeCheckout is the checkout a run converges from when --repo names
// none: the dotfiles checkout the working directory is in (another project's
// repository does not count), else the declared location
// (DOTFILES_REPO_DIR, machine.json, the contract), else the default. The last
// two may not exist yet: the checkout step clones them, which is how a machine
// from zero gets one.
func convergeCheckout(home string) string {
	if r := env.RepoDir(); r != "" && env.IsDotfilesCheckout(r) {
		return r
	}
	if r := env.ResolvePath("DOTFILES_REPO_DIR"); r != "" {
		return r
	}
	return env.DefaultCheckoutDir(home)
}

var convergeTag = map[converge.Status]string{
	converge.StatusOK:      "[ OK ]",
	converge.StatusChange:  "[CHANGE]",
	converge.StatusSkipped: "[SKIP]",
	converge.StatusFailed:  "[FAIL]",
	converge.StatusOpaque:  "[OPAQUE]",
}

// convergeColor is each tag's colour, doctor's palette read in converge's
// words: a change is what doctor's fix is, and opaque, a step that cannot
// say what it did, is a caution.
var convergeColor = map[converge.Status][]string{
	converge.StatusOK:      {console.Green},
	converge.StatusChange:  {console.Magenta},
	converge.StatusSkipped: {console.Cyan},
	converge.StatusFailed:  {console.Red, console.Bold},
	converge.StatusOpaque:  {console.Yellow},
}

// convergeTerminal says whether converge was handed a terminal. It is decided
// on that writer, before beginRunLog wraps it in the transcript tee, which is
// never a terminal. The transcript strips colour, so one stream serves both
// (CLI-114).
func convergeTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && isTerminal(f.Fd())
}

// convergePainter colours a terminal, unless NO_COLOR or TERM=dumb say not.
func convergePainter(w io.Writer) console.Painter {
	return console.Painter{On: console.ColorEnabled(os.Getenv, convergeTerminal(w))}
}

// convergePrinter renders a run as it happens (CLI-114). Each step's line is
// printed when the step is done; on a terminal a "→ name" line also says
// which step is running, ahead of whatever the step itself prints. finish
// prints the entries the run never ran, then the counts and the failures.
type convergePrinter struct {
	w        io.Writer
	p        console.Painter
	terminal bool
	width    int // the name column, which fits the longest name
	printed  int // how many of the report's entries Done has printed
}

func newConvergePrinter(w io.Writer, reg []converge.Reconciler, p console.Painter, terminal bool) *convergePrinter {
	c := &convergePrinter{w: w, p: p, terminal: terminal, width: 16}
	for _, r := range reg {
		c.width = max(c.width, len(r.Name()))
	}
	return c
}

// printConvergeReport prints a finished run whole.
func printConvergeReport(w io.Writer, rep converge.Report, p console.Painter) {
	c := &convergePrinter{w: w, p: p, width: 16}
	for _, e := range rep.Entries {
		c.width = max(c.width, len(e.Name))
	}
	c.header(rep.DryRun, rep.GOOS)
	c.finish(rep)
}

func (c *convergePrinter) header(dryRun bool, goos string) {
	heading := "converge"
	if dryRun {
		heading = "converge plan"
	}
	_, _ = fmt.Fprintf(c.w, "%s on %s:\n", heading, goos)
}

// Start announces a step on a terminal; a pipe gets only result lines.
func (c *convergePrinter) Start(name string) {
	if c.terminal {
		_, _ = fmt.Fprintf(c.w, "→ %s\n", name)
	}
}

// Done prints a step's line. converge.RunObserved calls it in the report's
// order, and the entries it never calls it for are the report's tail.
func (c *convergePrinter) Done(e converge.Entry) {
	c.line(e)
	c.printed++
}

func (c *convergePrinter) line(e converge.Entry) {
	// Pad the bare tag: escape codes would count toward a %-8s width.
	tag := convergeTag[e.Status]
	_, _ = fmt.Fprintf(c.w, "  %s%s %-*s %6s  %s\n", c.p.Paint(tag, convergeColor[e.Status]...), strings.Repeat(" ", max(0, 8-len(tag))), c.width, e.Name, stepDuration(e), e.Detail)
}

// finish prints the entries Done never saw, then the counts and the failures.
func (c *convergePrinter) finish(rep converge.Report) {
	for _, e := range rep.Entries[min(c.printed, len(rep.Entries)):] {
		c.line(e)
	}
	changed := "changed"
	if rep.DryRun {
		changed = "to change"
	}
	n := rep.Summary()
	failed := fmt.Sprintf("%d failed", n[converge.StatusFailed])
	if k := failedChanges(rep); k == 1 {
		failed += " (after 1 change)"
	} else if k > 1 {
		failed += fmt.Sprintf(" (after %d changes)", k)
	}
	if n[converge.StatusFailed] > 0 {
		failed = c.p.Paint(failed, console.Red, console.Bold)
	}
	_, _ = fmt.Fprintf(c.w, "%d %s, %d ok, %d opaque, %d skipped, %s\n",
		n[converge.StatusChange], changed, n[converge.StatusOK], n[converge.StatusOpaque], n[converge.StatusSkipped], failed)
	printFailureSummary(c.w, rep, c.p)
}

// failedChanges is how many changes the failed steps made before failing:
// the summary counts steps by status, and a failed step's changes would
// otherwise vanish from it (BUG-136).
func failedChanges(rep converge.Report) int {
	k := 0
	for _, e := range rep.Entries {
		if e.Status == converge.StatusFailed {
			k += e.Changes
		}
	}
	return k
}

// stepDuration is a step's duration column: blank for a step the run never
// ran, so "not reached" does not read as instant.
func stepDuration(e converge.Entry) string {
	if e.Duration <= 0 {
		return ""
	}
	return console.Duration(e.Duration)
}

// printFailureSummary repeats each failed step after the counts (CLI-114 AC6):
// in a long run the failure is one line among many, and the steps' own output
// can scroll it away. A failed entry's detail is its error (runOne).
func printFailureSummary(w io.Writer, rep converge.Report, p console.Painter) {
	var failed []converge.Entry
	for _, e := range rep.Entries {
		if e.Status == converge.StatusFailed {
			failed = append(failed, e)
		}
	}
	if len(failed) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "\n%s\n", p.Paint("failed:", console.Red, console.Bold))
	for _, e := range failed {
		_, _ = fmt.Fprintf(w, "  %s: %s\n", e.Name, e.Detail)
	}
}
