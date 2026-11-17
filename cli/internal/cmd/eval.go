package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/evals"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// errEvalCheck is returned when at least one case fails its self-check.
var errEvalCheck = errors.New("eval check: cases failed")

func newEvalCmd() *cobra.Command {
	c := &cobra.Command{
		Use:          "eval",
		Short:        "Measure models on the harness's own tasks, graded without a judge model",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	c.AddCommand(newEvalCheckCmd(), newEvalRunCmd())
	return c
}

func newEvalCheckCmd() *cobra.Command {
	var root string
	c := &cobra.Command{
		Use:   "check",
		Short: "Prove every eval case's key against its own source, with no model involved",
		Long: "check reads the repository's eval cases (" + evals.ReplayCaseDir + "/*.json, and\n" +
			evals.PlantedCaseDir + "/*/case.json where it exists) and, for each one:\n" +
			"  - validates it against the case schema embedded in this binary;\n" +
			"  - holds its reviewed_sha, verdict and key_author to its source's frontmatter;\n" +
			"  - grades its key against its own source: every expected and optional entry\n" +
			"    must match a row, one to one, and a planted reference may match no decoy;\n" +
			"  - matches each decoy and noise entry against its recorded sample.\n" +
			"It exits non-zero when any case fails. A repository without cases passes.",
		Example:       "  dotf eval check\n  dotf eval check --repo-root ../kubelab",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if root == "" {
				root = env.RepoDir()
			}
			if root == "" {
				return fmt.Errorf("eval check: not inside a repository — pass --repo-root")
			}
			results, err := evals.Check(root)
			if err != nil {
				c.PrintErrln("eval check:", err)
				return err
			}
			report, failed := renderEvalCheck(results)
			if _, err := io.WriteString(c.OutOrStdout(), report); err != nil {
				return err
			}
			if failed > 0 {
				return errEvalCheck
			}
			return nil
		},
	}
	c.Flags().StringVar(&root, "repo-root", "", "repository to check (default: the current one)")
	return c
}

// evalRunOpts are eval run's flags.
type evalRunOpts struct {
	root       string
	candidates []string
	caseIDs    []string
	epochs     int
	maxRuns    int
	planOnly   bool
}

// evalRunDeps is every step of a run past the flags, so a test drives the
// command without a corpus on disk or a runner that spends anything.
type evalRunDeps struct {
	loadCases      func(root string) ([]evals.Case, error)
	loadCandidates func(root string) (evals.Candidates, error)
	check          func(root string) ([]evals.CheckResult, error)
	preflight      func(root string, p evals.Plan, cases []evals.Case, cands map[string]spec.ReviewerEntry) error
	newRunDir      func() (string, error)
	run            func(ctx context.Context, root string, p evals.Plan, cases []evals.Case, runDir string, cands map[string]spec.ReviewerEntry) ([]evals.Record, error)
}

func defaultEvalRunDeps() evalRunDeps {
	return evalRunDeps{
		loadCases:      evals.LoadCases,
		loadCandidates: evals.LoadCandidates,
		check:          evals.Check,
		preflight: func(root string, p evals.Plan, cases []evals.Case, cands map[string]spec.ReviewerEntry) error {
			return evals.Preflight(root, p, cases, cands, evals.PreflightDeps{LookPath: exec.LookPath, CheckSecret: verifySecret})
		},
		newRunDir: newEvalRunDir,
		run: func(ctx context.Context, root string, p evals.Plan, cases []evals.Case, runDir string, cands map[string]spec.ReviewerEntry) ([]evals.Record, error) {
			l := evals.ExecLauncher{Root: root, Candidates: cands, Deps: evals.DefaultLaunchDeps(root)}
			return evals.Run(ctx, p, cases, runDir, l)
		},
	}
}

// verifySecret checks a secret the way the runner will receive it: through
// the dotf on PATH, by exit status. `secrets verify` prints a status per id,
// never a value.
func verifySecret(id string) error {
	out, err := exec.Command("dotf", "secrets", "verify", "--require-all", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// newEvalRunDir makes a fresh run directory under dotf's state directory,
// named for the UTC second it started. An existing one is never reused.
func newEvalRunDir() (string, error) {
	state, err := env.StateDir()
	if err != nil {
		return "", err
	}
	parent := filepath.Join(state, "evals")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	dir := filepath.Join(parent, time.Now().UTC().Format("20060102T150405Z"))
	if err := os.Mkdir(dir, 0o755); err != nil {
		return "", fmt.Errorf("eval run: a run directory: %w", err)
	}
	return dir, nil
}

func newEvalRunCmd() *cobra.Command {
	var o evalRunOpts
	c := &cobra.Command{
		Use:   "run",
		Short: "Run the eval cases for named candidates, or plan the run with --plan",
		Long: "run builds the matrix of every eval case × each --candidate × --epochs, and\n" +
			"refuses it above --max-runs: the cap is what authorises spend. --plan prints the\n" +
			"matrix and each candidate's estimate (NaN credits, agy runs) and launches nothing.\n" +
			"Without --plan the run checks the corpus as `dotf eval check` does, then runs a\n" +
			"preflight that names every runner, secret, commit or answer leak that would fail\n" +
			"an attempt, and launches nothing unless both pass. Attempts are recorded under\n" +
			"dotf's state directory (evals/<run-id>/). Candidates come from\n" +
			evals.CandidatesFile + ", never from the reviewer pool.",
		Example:       "  dotf eval run --plan --candidate nan/glm5.3-flash\n  dotf eval run --candidate agy/gemini-4-hydrogen-high --case harness-088-r2 --epochs 1",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if o.root == "" {
				o.root = env.RepoDir()
			}
			if o.root == "" {
				return fmt.Errorf("eval run: not inside a repository — pass --repo-root")
			}
			// The interrupt is caught here so the loop stops between attempts
			// too, not only while a runner is alive. The first one cancels the
			// run; releasing the handler then lets a second one kill it.
			ctx, stop := signal.NotifyContext(c.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			releaseOnCancel(ctx, stop)
			return runEvalRun(ctx, c.OutOrStdout(), defaultEvalRunDeps(), o)
		},
	}
	c.Flags().StringVar(&o.root, "repo-root", "", "repository holding the cases and candidates (default: the current one)")
	c.Flags().StringArrayVar(&o.candidates, "candidate", nil, "candidate id from "+evals.CandidatesFile+" (repeatable)")
	c.Flags().StringArrayVar(&o.caseIDs, "case", nil, "case id to run (repeatable; default: every case)")
	c.Flags().IntVar(&o.epochs, "epochs", evals.DefaultEpochs, "runs per (candidate, case)")
	c.Flags().IntVar(&o.maxRuns, "max-runs", evals.DefaultMaxRuns, "the most runs this invocation may launch")
	c.Flags().BoolVar(&o.planOnly, "plan", false, "print the matrix and the estimate, and launch nothing")
	return c
}

// releaseOnCancel calls stop once ctx is done, so the signal handler that
// cancelled the run stops swallowing the signals after it.
func releaseOnCancel(ctx context.Context, stop func()) {
	go func() {
		<-ctx.Done()
		stop()
	}()
}

// runEvalRun plans the run and prints it; without --plan it checks the corpus,
// runs the preflight, and launches the matrix.
func runEvalRun(ctx context.Context, w io.Writer, deps evalRunDeps, o evalRunOpts) error {
	cases, err := deps.loadCases(o.root)
	if err != nil {
		return fmt.Errorf("eval run: %w", err)
	}
	if cases, err = evals.SelectCases(cases, o.caseIDs); err != nil {
		return fmt.Errorf("eval run: %w", err)
	}
	cs, err := deps.loadCandidates(o.root)
	if err != nil {
		return fmt.Errorf("eval run: %w", err)
	}
	plan, err := evals.BuildPlan(cases, cs, o.candidates, o.epochs, o.maxRuns)
	// Only a plan refused for its size is printed with the error: any other
	// refusal has no plan worth showing.
	if (err == nil || errors.Is(err, evals.ErrOverCap)) && plan.Runs > 0 {
		if _, werr := io.WriteString(w, plan.Render()); werr != nil {
			return werr
		}
	}
	if err != nil {
		return fmt.Errorf("eval run: %w", err)
	}
	if o.planOnly {
		return nil
	}
	return launchEvalRun(ctx, w, deps, o.root, plan, cases, cs)
}

// launchEvalRun is everything that can spend, behind the corpus check and the
// preflight.
func launchEvalRun(ctx context.Context, w io.Writer, deps evalRunDeps, root string, plan evals.Plan, cases []evals.Case, cs evals.Candidates) error {
	results, err := deps.check(root)
	if err != nil {
		return fmt.Errorf("eval run: %w", err)
	}
	if report, failed := renderEvalCheck(results); failed > 0 {
		if _, werr := io.WriteString(w, report); werr != nil {
			return werr
		}
		return fmt.Errorf("eval run: the corpus fails `dotf eval check`; nothing was launched")
	}
	cands := make(map[string]spec.ReviewerEntry, len(cs.Entries))
	for _, e := range cs.Entries {
		cands[e.ID] = e
	}
	if err := deps.preflight(root, plan, cases, cands); err != nil {
		return fmt.Errorf("%w\nnothing was launched", err)
	}
	runDir, err := deps.newRunDir()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\nrun %s\n", runDir); err != nil {
		return err
	}
	records, runErr := deps.run(ctx, root, plan, cases, runDir, cands)
	if _, err := io.WriteString(w, renderEvalRecords(records)); err != nil {
		return err
	}
	if runErr != nil {
		return fmt.Errorf("eval run: %w", runErr)
	}
	return nil
}

// renderEvalRecords prints one line per attempt and a count per completion.
func renderEvalRecords(records []evals.Record) string {
	var b strings.Builder
	counts := map[evals.Completion]int{}
	for _, r := range records {
		counts[r.Completion]++
		fmt.Fprintf(&b, "  %-24s %-32s #%d  %-14s", r.Case, r.Candidate, r.Epoch, r.Completion)
		switch {
		case r.Grade != nil:
			match := "mismatch"
			if r.Grade.VerdictMatch {
				match = "match"
			}
			fmt.Fprintf(&b, " %s, verdict %s (%s), recall %.2f", r.Grade.Format, r.Grade.Verdict, match, r.Grade.Recall)
			if r.OwnKey {
				b.WriteString(", own key")
			}
		case r.Detail != "":
			b.WriteString(" " + r.Detail)
		}
		b.WriteString("\n")
	}
	kinds := make([]string, 0, len(counts))
	for k, n := range counts {
		kinds = append(kinds, fmt.Sprintf("%d %s", n, k))
	}
	sort.Strings(kinds)
	fmt.Fprintf(&b, "%d attempts: %s\n", len(records), strings.Join(kinds, ", "))
	return b.String()
}

// renderEvalCheck formats the check report and counts the failed cases.
func renderEvalCheck(results []evals.CheckResult) (string, int) {
	var b strings.Builder
	if len(results) == 0 {
		b.WriteString("[OK] no eval cases in this repository\n")
		return b.String(), 0
	}
	failed := 0
	for _, r := range results {
		if len(r.Problems) == 0 {
			fmt.Fprintf(&b, "[OK]   %s\n", r.Path)
			continue
		}
		failed++
		fmt.Fprintf(&b, "[FAIL] %s\n", r.Path)
		for _, p := range r.Problems {
			fmt.Fprintf(&b, "         %s\n", p)
		}
	}
	if failed > 0 {
		fmt.Fprintf(&b, "%d of %d cases failed\n", failed, len(results))
	} else {
		fmt.Fprintf(&b, "%d cases checked\n", len(results))
	}
	return b.String(), failed
}
