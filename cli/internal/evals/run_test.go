package evals

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeLauncher answers each attempt from a script keyed by "<case>/<epoch>",
// and records what the run directory held when each attempt started.
type fakeLauncher struct {
	script   map[string]func(a Attempt) (LaunchResult, error)
	started  []string
	seen     [][]string // records on disk when each attempt started
	onLaunch func()
}

func (f *fakeLauncher) Launch(_ context.Context, a Attempt) (LaunchResult, error) {
	key := a.Case.ID + "/" + itoa(a.Epoch)
	f.started = append(f.started, key)
	recs, _ := filepath.Glob(filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(a.OutDir))), "*", "*", "*", RecordFile))
	f.seen = append(f.seen, recs)
	if f.onLaunch != nil {
		f.onLaunch()
	}
	if s, ok := f.script[key]; ok {
		return s(a)
	}
	return answered(answer("FAIL", "| Major | REAL | x | the latch never clears | e | t | f |"))(a)
}

func itoa(n int) string { return string(rune('0' + n)) }

// answered writes body where the launcher keeps an answer, as a runner that
// finished would have.
func answered(body string) func(Attempt) (LaunchResult, error) {
	return func(a Attempt) (LaunchResult, error) {
		if err := os.MkdirAll(a.OutDir, 0o755); err != nil {
			return LaunchResult{}, err
		}
		if err := os.WriteFile(filepath.Join(a.OutDir, AnswerFile), []byte(body), 0o600); err != nil {
			return LaunchResult{}, err
		}
		return LaunchResult{Duration: 2 * time.Second}, nil
	}
}

func runCase() Case {
	return Case{
		ID: "a-r1", Kind: "replay", KeyAuthor: "nan/glm5.3-flash", ExpectedVerdict: "FAIL",
		Expected: []Finding{finding("latch", "Major", []string{"latch"})},
	}
}

func runPlan(epochs int, candidates ...string) Plan {
	var p Plan
	for _, c := range candidates {
		p.Cells = append(p.Cells, Cell{Case: "a-r1", Kind: "replay", Candidate: c, Epochs: epochs})
		p.Runs += epochs
	}
	return p
}

func readRecord(t *testing.T, path string) Record {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r Record
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

// Runs go one at a time, and each leaves its record before the next starts:
// an interrupted run keeps everything it already paid for.
func TestRunRecordsEachAttemptBeforeTheNext(t *testing.T) {
	dir := t.TempDir()
	l := &fakeLauncher{}
	recs, err := Run(context.Background(), runPlan(2, "agy/gemini-4-hydrogen-high"), []Case{runCase()}, dir, l)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 || strings.Join(l.started, ",") != "a-r1/1,a-r1/2" {
		t.Fatalf("records %d, started %v", len(recs), l.started)
	}
	if len(l.seen[0]) != 0 || len(l.seen[1]) != 1 {
		t.Fatalf("the first record must be on disk before the second attempt starts: %v", l.seen)
	}
	want := filepath.Join(dir, "runs", "agy-gemini-4-hydrogen-high", "a-r1", "1", RecordFile)
	r := readRecord(t, want)
	if r.Completion != CompletionAnswered || r.Grade == nil || r.Grade.Recall != 1 || !r.Grade.VerdictMatch {
		t.Fatalf("record = %+v", r)
	}
	if r.Tokens != nil || r.DurationSeconds != 2 {
		t.Fatalf("a runner that reports no tokens is recorded as not reported, never 0: %+v", r)
	}
	if tree := filepath.Join(dir, "trees", "agy-gemini-4-hydrogen-high", "a-r1-1"); recs[0].Tree != tree {
		t.Fatalf("tree = %s, want %s", recs[0].Tree, tree)
	}
}

// ADR-063 §3: an unparseable answer is a format failure, never zero findings,
// and an answer that never arrived is not graded at all.
func TestRunKeepsAFormatFailureApartFromZeroFindings(t *testing.T) {
	for _, tc := range []struct {
		name       string
		launch     func(Attempt) (LaunchResult, error)
		completion Completion
		format     Format // "" means no grade
	}{
		{"no table", answered("verdict: FAIL\n\nI found problems but wrote no table.\n"), CompletionAnswered, FormatNoTable},
		{"an empty table", answered(answer("PASS")), CompletionAnswered, FormatEmptyTable},
		{"a blank answer", answered(" \n"), CompletionEmptyAnswer, ""},
		{"no answer file", func(Attempt) (LaunchResult, error) { return LaunchResult{}, nil }, CompletionEmptyAnswer, ""},
		{"a timeout", func(Attempt) (LaunchResult, error) {
			return LaunchResult{TimedOut: true, ExitErr: errors.New("killed")}, nil
		}, CompletionTimeout, ""},
		{"a failed runner", func(Attempt) (LaunchResult, error) { return LaunchResult{ExitErr: errors.New("exit 1")}, nil }, CompletionProviderError, ""},
		{"our own failure", func(Attempt) (LaunchResult, error) { return LaunchResult{}, errors.New("checkout failed") }, CompletionInfra, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := &fakeLauncher{script: map[string]func(Attempt) (LaunchResult, error){"a-r1/1": tc.launch}}
			recs, err := Run(context.Background(), runPlan(1, "agy/x"), []Case{runCase()}, t.TempDir(), l)
			if err != nil {
				t.Fatal(err)
			}
			r := recs[0]
			if r.Completion != tc.completion {
				t.Fatalf("completion = %q, want %q (%s)", r.Completion, tc.completion, r.Detail)
			}
			if tc.format == "" {
				if r.Grade != nil {
					t.Fatalf("a run without an answer is not graded: %+v", r.Grade)
				}
				return
			}
			if r.Grade == nil || r.Grade.Format != tc.format {
				t.Fatalf("grade = %+v, want format %q", r.Grade, tc.format)
			}
		})
	}
}

// A failure on our side is recorded and the matrix goes on; it is never
// charged to the model.
func TestRunGoesOnAfterAnInfrastructureFailure(t *testing.T) {
	l := &fakeLauncher{script: map[string]func(Attempt) (LaunchResult, error){
		"a-r1/1": func(Attempt) (LaunchResult, error) { return LaunchResult{}, errors.New("worktree add failed") },
	}}
	recs, err := Run(context.Background(), runPlan(2, "agy/x"), []Case{runCase()}, t.TempDir(), l)
	if err != nil || len(recs) != 2 || recs[0].Completion != CompletionInfra || recs[1].Completion != CompletionAnswered {
		t.Fatalf("err %v, records %+v", err, recs)
	}
	if !strings.Contains(recs[0].Detail, "worktree add failed") {
		t.Fatalf("detail = %q", recs[0].Detail)
	}
}

// ADR-063 §3, key circularity: a candidate graded on its own key is flagged.
func TestRunFlagsACandidateGradedOnItsOwnKey(t *testing.T) {
	recs, err := Run(context.Background(), runPlan(1, "nan/glm5.3-flash", "agy/x"), []Case{runCase()}, t.TempDir(), &fakeLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	if !recs[0].OwnKey || recs[1].OwnKey {
		t.Fatalf("own key flags = %v, %v; want true, false", recs[0].OwnKey, recs[1].OwnKey)
	}
}

// An interrupt stops the matrix after the attempt in flight; what finished
// stays on disk.
func TestRunStopsOnInterrupt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	l := &fakeLauncher{onLaunch: cancel}
	recs, err := Run(ctx, runPlan(3, "agy/x"), []Case{runCase()}, t.TempDir(), l)
	if !errors.Is(err, context.Canceled) || len(l.started) != 1 || len(recs) != 1 {
		t.Fatalf("err %v, started %v, records %d", err, l.started, len(recs))
	}
}

func TestRunRefusesACellWithoutItsCase(t *testing.T) {
	if _, err := Run(context.Background(), runPlan(1, "agy/x"), nil, t.TempDir(), &fakeLauncher{}); err == nil || !strings.Contains(err.Error(), "a-r1") {
		t.Fatalf("want a refusal naming the case, got %v", err)
	}
}

// The interrupt reaches the runner in flight, and the attempt it cut short is
// not charged to the model as a provider error.
func TestRunPassesTheInterruptToTheRunnerInFlight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	l := &fakeLauncher{script: map[string]func(Attempt) (LaunchResult, error){}}
	l.script["a-r1/1"] = func(Attempt) (LaunchResult, error) { return LaunchResult{ExitErr: errors.New("signal: killed")}, nil }
	l.onLaunch = cancel
	var got context.Context
	recs, err := Run(ctx, runPlan(1, "agy/x"), []Case{runCase()}, t.TempDir(), launcherFunc(func(c context.Context, a Attempt) (LaunchResult, error) {
		got = c
		return l.Launch(c, a)
	}))
	if !errors.Is(err, context.Canceled) || got == nil || got.Err() == nil {
		t.Fatalf("the launcher must receive the run's context: err %v", err)
	}
	if recs[0].Completion != CompletionInfra || !strings.Contains(recs[0].Detail, "interrupted") {
		t.Fatalf("record = %+v", recs[0])
	}
}

type launcherFunc func(context.Context, Attempt) (LaunchResult, error)

func (f launcherFunc) Launch(ctx context.Context, a Attempt) (LaunchResult, error) { return f(ctx, a) }
