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
	// CompletionProviderError — the runner failed and left no answer. Telling a
	// rate limit, a quota or a content filter apart needs its transcript.
	CompletionProviderError Completion = "provider-error"
	// CompletionInfra — the attempt failed on our side (checkout, tree, copy).
	// It is never charged to the model.
	CompletionInfra Completion = "infra-error"
)

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
	Duration time.Duration
	Tokens   *Tokens // nil when the runner reports none
}

// Tokens is what a runner reported spending. Absent is "not reported", never 0.
type Tokens struct {
	Input    int `json:"input"`
	Output   int `json:"output"`
	Thinking int `json:"thinking"`
}

// Launcher runs one attempt. It returns an error only for a failure on our
// side; a runner that failed is a LaunchResult with ExitErr set.
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
}

// Run executes the plan's attempts one at a time and writes each record before
// the next attempt starts, so an interrupted run keeps what it paid for. An
// interrupt stops it after the attempt in flight.
func Run(ctx context.Context, p Plan, cases []Case, runDir string, l Launcher) ([]Record, error) {
	byID := make(map[string]Case, len(cases))
	for _, c := range cases {
		byID[c.ID] = c
	}
	for _, cell := range p.Cells {
		if _, ok := byID[cell.Case]; !ok {
			return nil, fmt.Errorf("eval run: the plan names case %s, which is not loaded", cell.Case)
		}
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

func grade(ctx context.Context, a Attempt, l Launcher) Record {
	rec := Record{
		Case: a.Case.ID, Kind: a.Case.Kind, Candidate: a.Candidate, Epoch: a.Epoch,
		OwnKey: a.Case.KeyAuthor == a.Candidate,
	}
	res, err := l.Launch(ctx, a)
	if err != nil {
		rec.Completion, rec.Detail = CompletionInfra, err.Error()
		return rec
	}
	rec.DurationSeconds, rec.Tokens = res.Duration.Seconds(), res.Tokens
	body, readErr := os.ReadFile(filepath.Join(a.OutDir, AnswerFile))
	switch {
	case res.TimedOut:
		rec.Completion = CompletionTimeout
	case readErr == nil && strings.TrimSpace(string(body)) != "":
		g := GradeAnswer(a.Case, string(body))
		rec.Completion, rec.Grade = CompletionAnswered, &g
		return rec
	case ctx.Err() != nil:
		rec.Completion = CompletionInfra
		res.ExitErr = errors.Join(errors.New("interrupted"), res.ExitErr)
	case res.ExitErr != nil:
		rec.Completion = CompletionProviderError
	default:
		rec.Completion = CompletionEmptyAnswer
	}
	if res.ExitErr != nil {
		rec.Detail = res.ExitErr.Error()
	} else if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		rec.Detail = readErr.Error()
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
