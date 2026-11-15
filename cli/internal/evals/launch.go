package evals

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// Tree is the directory one attempt's runner works in, and how to remove it.
type Tree struct {
	Dir    string
	Remove func() error
}

// LaunchDeps are the launcher's effects: checking out a replay, and running a
// command under limits. Tests replace them; production uses DefaultLaunchDeps.
type LaunchDeps struct {
	Checkout func(c Case, runDir string, epoch int) (Tree, error)
	Run      func(l spec.Limits, argv []string, stdin io.Reader, stdout, stderr io.Writer) (spec.RunResult, error)
}

// DefaultLaunchDeps checks replays out of repo and runs the real runner.
func DefaultLaunchDeps(repo string) LaunchDeps {
	return LaunchDeps{
		Checkout: func(c Case, runDir string, epoch int) (Tree, error) {
			r, err := CheckoutReplay(repo, c, runDir, epoch)
			if err != nil {
				return Tree{}, err
			}
			return Tree{Dir: r.Dir, Remove: r.Remove}, nil
		},
		Run: spec.RunWithLimits,
	}
}

// ExecLauncher runs one attempt with a real runner: it prepares the tree,
// launches the candidate under a deadline and a stall detector, keeps the
// transcript and the answer in the attempt's output directory, and removes the
// tree however the run ended.
type ExecLauncher struct {
	Root       string // the repository the cases and planted prompts are read from
	Candidates map[string]spec.ReviewerEntry
	SkillPath  string
	Timeout    time.Duration
	Stall      time.Duration
	Deps       LaunchDeps
}

// Launch implements Launcher.
func (l ExecLauncher) Launch(_ context.Context, a Attempt) (res LaunchResult, err error) {
	entry, ok := l.Candidates[a.Candidate]
	if !ok {
		return LaunchResult{}, fmt.Errorf("eval launch: candidate %s is not in candidates.json", a.Candidate)
	}
	tree, err := l.prepareTree(a)
	if err != nil {
		return LaunchResult{}, err
	}
	defer func() {
		if rmErr := tree.Remove(); rmErr != nil && err == nil {
			err = fmt.Errorf("eval launch: remove the tree %s: %w", tree.Dir, rmErr)
		}
	}()
	if err := os.MkdirAll(filepath.Join(tree.Dir, spec.EvalAnswerDir), 0o755); err != nil {
		return LaunchResult{}, err
	}
	if err := os.MkdirAll(a.OutDir, 0o755); err != nil {
		return LaunchResult{}, err
	}
	plan, err := l.plan(a, entry, tree.Dir)
	if err != nil {
		return LaunchResult{}, err
	}
	began := time.Now()
	ran, err := l.run(plan, tree.Dir)
	if err != nil {
		return LaunchResult{}, fmt.Errorf("eval launch: %s could not run: %w", a.Candidate, err)
	}
	res = resultOf(ran)
	res.Duration = time.Since(began)
	switch keepErr := keepAnswer(plan.Answer, plan.Result); {
	case errors.Is(keepErr, errNotRegular):
		res.ExitErr = errors.Join(res.ExitErr, keepErr)
	case keepErr != nil:
		return res, fmt.Errorf("eval launch: keep the answer of %s: %w", a.Candidate, keepErr)
	}
	return res, nil
}

func (l ExecLauncher) prepareTree(a Attempt) (Tree, error) {
	if a.Case.Kind == "planted" {
		if err := os.MkdirAll(a.Tree, 0o755); err != nil {
			return Tree{}, err
		}
		return Tree{Dir: a.Tree, Remove: func() error { return os.RemoveAll(a.Tree) }}, nil
	}
	t, err := l.Deps.Checkout(a.Case, filepath.Dir(a.Tree), a.Epoch)
	if err != nil {
		return Tree{}, err
	}
	if t.Dir != a.Tree {
		_ = t.Remove()
		return Tree{}, fmt.Errorf("eval launch: case %s was checked out at %s, not %s", a.Case.ID, t.Dir, a.Tree)
	}
	return t, nil
}

func (l ExecLauncher) plan(a Attempt, entry spec.ReviewerEntry, tree string) (spec.EvalPlan, error) {
	if a.Case.Kind == "planted" {
		full, err := insideRoot(l.Root, a.Case.Prompt)
		if err != nil {
			return spec.EvalPlan{}, fmt.Errorf("case %s: prompt: %w", a.Case.ID, err)
		}
		prompt, err := os.ReadFile(full)
		if err != nil {
			return spec.EvalPlan{}, fmt.Errorf("case %s: %w", a.Case.ID, err)
		}
		return spec.PlanPlantedLaunch(spec.PlantedLaunch{Candidate: entry, Prompt: string(prompt), Tree: tree, OutDir: a.OutDir, Timeout: l.Timeout})
	}
	launch := spec.EvalLaunch{
		Candidate: entry, SpecID: a.Case.Spec, Tree: tree, ReviewedSHA: a.Case.ReviewedSHA, BaseSHA: a.Case.BaseSHA,
		OutDir: a.OutDir, SkillPath: l.SkillPath, Timeout: l.Timeout,
	}
	p, err := spec.PlanEvalLaunch(launch)
	if err != nil {
		return spec.EvalPlan{}, err
	}
	return p, spec.WriteEvalRequest(p, launch)
}

// run starts the runner in the tree, with its stdout filtered into the
// transcript the stall detector watches, as `spec review` stores it.
func (l ExecLauncher) run(p spec.EvalPlan, tree string) (spec.RunResult, error) {
	pr, pw := io.Pipe()
	sunk := make(chan error, 1)
	go func() { sunk <- spec.SinkTranscript(pr, io.Discard, p.Transcript) }()
	stderr, err := os.Create(filepath.Join(filepath.Dir(p.Transcript), "stderr.log"))
	if err != nil {
		_ = pw.Close()
		<-sunk
		return spec.RunResult{}, err
	}
	defer func() { _ = stderr.Close() }()
	limits := spec.Limits{Deadline: l.Timeout, StallTimeout: l.Stall, Transcript: p.Transcript, Dir: tree}
	ran, runErr := l.Deps.Run(limits, p.Argv, nil, pw, stderr)
	_ = pw.Close()
	if sinkErr := <-sunk; runErr == nil && sinkErr != nil {
		runErr = sinkErr
	}
	return ran, runErr
}

func resultOf(r spec.RunResult) LaunchResult {
	var res LaunchResult
	switch {
	case r.Stop == spec.StopDeadline:
		res.TimedOut = true
		res.ExitErr = fmt.Errorf("stopped at the deadline")
	case r.Stop == spec.StopStall:
		res.Stalled = true
		res.ExitErr = fmt.Errorf("stopped: the transcript stopped changing; last tool call %q", r.LastCall)
	case r.Signaled:
		res.Interrupted = true
		res.ExitErr = fmt.Errorf("stopped by a signal (exit %d)", r.Code)
	case r.Code != 0:
		res.ExitErr = fmt.Errorf("the runner exited %d", r.Code)
	}
	return res
}

var errNotRegular = errors.New("the answer is not a regular file")

// keepAnswer copies the runner's answer out of the tree. It reads only a
// regular file: a symlink the runner left there is not followed.
func keepAnswer(from, to string) error {
	fi, err := os.Lstat(from)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%w: %s (%s); it was not read", errNotRegular, from, fi.Mode().Type())
	}
	body, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, body, 0o600)
}
