package evals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// AnswerFile is the answer a launcher keeps in an attempt's output directory,
// copied there from the tree the runner wrote it in.
var AnswerFile = spec.EvalSlot.Review

// RecordFile is an attempt's grading record, beside its answer.
const RecordFile = "record.json"

// Completion is how an attempt ended (ADR-063 §3).
type Completion string

const (
	// CompletionAnswered — an answer arrived and was graded, whatever its format.
	CompletionAnswered Completion = "answered"
	// CompletionEmptyAnswer — the runner finished without a usable answer.
	CompletionEmptyAnswer Completion = "empty-answer"
	// CompletionTimeout — the deadline killed the runner.
	CompletionTimeout Completion = "timeout"
	// CompletionStalled — the runner's transcript stopped changing and the
	// launcher stopped it.
	CompletionStalled Completion = "stalled"
	// CompletionProviderError — the runner failed and left no answer. Telling a
	// rate limit, a quota or a content filter apart needs its transcript.
	CompletionProviderError Completion = "provider-error"
	// CompletionInfra — the attempt failed on our side (checkout, tree, copy).
	// It is never charged to the model.
	CompletionInfra Completion = "infra-error"
)

// ErrInterrupted is returned when a launcher reports that an interrupt
// stopped the attempt in flight; the run ends there.
var ErrInterrupted = errors.New("eval run: interrupted")

// Attempt is one epoch of one cell, and where it runs and keeps its files.
type Attempt struct {
	Case      Case
	Candidate string
	Epoch     int
	Tree      string // <run>/trees/<candidate>/<case>-<epoch>, the launcher's to create and remove
	OutDir    string // <run>/runs/<candidate>/<case>/<epoch>
}

// LaunchResult is what a launcher saw of a runner that it did start.
type LaunchResult struct {
	ExitErr  error
	TimedOut bool
	// Stalled — the launcher stopped a runner whose transcript stopped changing.
	Stalled bool
	// Interrupted — the launcher killed the runner because the run was
	// interrupted. A runner that finished before the interrupt landed is not.
	Interrupted bool
	Duration    time.Duration
	Tokens      *Tokens // nil when the runner reports none
	// Leftover — the attempt ended but its tree could not be removed. It is
	// the operator's to clean up and says nothing about the model, so it is
	// recorded beside the result instead of replacing it.
	Leftover error
}

// Tokens is what a runner reported spending. Absent is "not reported", never 0.
// Input is the uncached prompt; Cached is read from or written to the
// provider's cache, which both pi and agy report apart from Input. Thinking is
// the reasoning count the runner reports, and pi counts it inside Output.
type Tokens struct {
	Input    int `json:"input"`
	Cached   int `json:"cached"`
	Output   int `json:"output"`
	Thinking int `json:"thinking"`
}

// Total is every token the run sent or received, the unit a plan's envelope
// is written in. Thinking is not added again.
func (t Tokens) Total() int { return t.Input + t.Cached + t.Output }

// Launcher runs one attempt. It returns an error only for a failure on our
// side; a runner that failed is a LaunchResult with ExitErr set. It copies the
// answer to OutDir only after the runner exited on its own, and the loop
// grades nothing from an attempt the deadline or the interrupt cut short.
type Launcher interface {
	Launch(ctx context.Context, a Attempt) (LaunchResult, error)
}

// Record is the grading of one attempt.
type Record struct {
	Case            string     `json:"case"`
	Kind            string     `json:"kind"`
	Candidate       string     `json:"candidate"`
	Epoch           int        `json:"epoch"`
	Completion      Completion `json:"completion"`
	Detail          string     `json:"detail,omitempty"`
	DurationSeconds float64    `json:"duration_seconds"`
	Tokens          *Tokens    `json:"tokens"`
	OwnKey          bool       `json:"own_key"`
	Grade           *Grade     `json:"grade"`
	Tree            string     `json:"-"`
	interrupted     bool
}

// Run executes the plan's attempts one at a time and writes each record before
// the next attempt starts, so an interrupted run keeps what it paid for. An
// interrupt stops it after the attempt in flight.
func Run(ctx context.Context, p Plan, cases []Case, runDir string, l Launcher) ([]Record, error) {
	byID := make(map[string]Case, len(cases))
	for _, c := range cases {
		byID[c.ID] = c
	}
	slugs := map[string]string{}
	for _, cell := range p.Cells {
		if _, ok := byID[cell.Case]; !ok {
			return nil, fmt.Errorf("eval run: the plan names case %s, which is not loaded", cell.Case)
		}
		slug := Slug(cell.Candidate)
		if other, ok := slugs[slug]; ok && other != cell.Candidate {
			return nil, fmt.Errorf("eval run: candidates %s and %s would share the directory %s", other, cell.Candidate, slug)
		}
		slugs[slug] = cell.Candidate
	}
	var recs []Record
	for _, cell := range p.Cells {
		for epoch := 1; epoch <= cell.Epochs; epoch++ {
			if err := ctx.Err(); err != nil {
				return recs, err
			}
			a := attemptFor(runDir, byID[cell.Case], cell.Candidate, epoch)
			rec := grade(ctx, a, l)
			rec.Tree = a.Tree
			if err := writeRecord(a.OutDir, rec); err != nil {
				return recs, err
			}
			recs = append(recs, rec)
			if rec.interrupted {
				return recs, errors.Join(ErrInterrupted, ctx.Err())
			}
		}
	}
	return recs, ctx.Err()
}

func attemptFor(runDir string, c Case, candidate string, epoch int) Attempt {
	slug, n := Slug(candidate), strconv.Itoa(epoch)
	return Attempt{
		Case: c, Candidate: candidate, Epoch: epoch,
		Tree:   filepath.Join(runDir, "trees", slug, c.ID+"-"+n),
		OutDir: filepath.Join(runDir, "runs", slug, c.ID, n),
	}
}

func grade(ctx context.Context, a Attempt, l Launcher) (rec Record) {
	rec = Record{
		Case: a.Case.ID, Kind: a.Case.Kind, Candidate: a.Candidate, Epoch: a.Epoch,
		OwnKey: a.Case.KeyAuthor == a.Candidate,
	}
	res, err := l.Launch(ctx, a)
	if err != nil {
		rec.Completion, rec.Detail = CompletionInfra, err.Error()
		return rec
	}
	if res.Leftover != nil {
		defer func() { rec.Detail = strings.TrimPrefix(rec.Detail+"; "+res.Leftover.Error(), "; ") }()
	}
	rec.DurationSeconds, rec.Tokens = res.Duration.Seconds(), res.Tokens
	if res.ExitErr != nil {
		rec.Detail = res.ExitErr.Error()
	}
	// Only a runner that exited on its own leaves an answer worth grading: one
	// the deadline or the interrupt killed may have left half a file.
	switch {
	case res.TimedOut:
		rec.Completion = CompletionTimeout
		return rec
	case res.Stalled:
		rec.Completion = CompletionStalled
		return rec
	case res.Interrupted:
		rec.interrupted = true
		rec.Completion, rec.Detail = CompletionInfra, strings.TrimSuffix("interrupted: "+rec.Detail, ": ")
		return rec
	}
	body, err := os.ReadFile(filepath.Join(a.OutDir, AnswerFile))
	switch {
	case err != nil && !errors.Is(err, os.ErrNotExist):
		rec.Completion, rec.Detail = CompletionInfra, err.Error()
	case err == nil && strings.TrimSpace(string(body)) != "":
		g := GradeAnswer(a.Case, string(body))
		rec.Completion, rec.Grade = CompletionAnswered, &g
	case res.ExitErr != nil:
		rec.Completion = CompletionProviderError
	default:
		rec.Completion = CompletionEmptyAnswer
	}
	return rec
}

func writeRecord(dir string, r Record) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, RecordFile), append(raw, '\n'), 0o600)
}

// Slug turns a candidate id into one path segment: "nan/glm5.3-flash" becomes
// "nan-glm5.3-flash".
func Slug(id string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, id)
}
