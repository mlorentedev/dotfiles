package evals

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// DefaultEpochs and DefaultMaxRuns are ADR-063's delegated budget: three epochs
// per (candidate, case), and at most 30 runs per invocation.
const (
	DefaultEpochs  = 3
	DefaultMaxRuns = 30
)

// ErrOverCap is returned by BuildPlan, with the plan, when the matrix holds
// more runs than the cap allows. The cap is what authorises spend.
var ErrOverCap = errors.New("plan exceeds the run cap")

// Unit is what an estimate counts, which differs per runner (ADR-063 §4, C38).
type Unit string

const (
	// UnitNaNCredits is NaN's one credit pool.
	UnitNaNCredits Unit = "NaN credits"
	// UnitAgyRuns counts agy launches: the subscription's quota cannot be read.
	UnitAgyRuns Unit = "agy runs"
)

// Basis says where an estimate's token envelope came from.
type Basis string

const (
	// BasisDefault is the kind's declared envelope, for a case that has never
	// run. It is priced as an upper bound.
	BasisDefault Basis = "default"
)

// Cell is one (case, candidate) pair of the matrix.
type Cell struct {
	Case      string
	Kind      string
	Candidate string
	Epochs    int
}

// Estimate is one candidate's projected spend over its cells.
type Estimate struct {
	Candidate string
	Unit      Unit
	Amount    float64
	Basis     Basis
}

// Plan is the matrix an evaluation run would execute, and what it would cost.
type Plan struct {
	Cells     []Cell
	Runs      int
	MaxRuns   int
	Estimates []Estimate
}

// BuildPlan builds the cases × candidates × epochs matrix and estimates each
// candidate's spend, launching nothing. A plan it cannot price is refused: an
// estimate is never zero or unknown. A plan above maxRuns is returned with
// ErrOverCap so it can still be printed.
func BuildPlan(cases []Case, cs Candidates, ids []string, epochs, maxRuns int) (Plan, error) {
	switch {
	case len(cases) == 0:
		return Plan{}, errors.New("no cases to run")
	case len(ids) == 0:
		return Plan{}, errors.New("no candidate named: pass --candidate")
	case epochs < 1:
		return Plan{}, fmt.Errorf("epochs must be at least 1, got %d", epochs)
	case maxRuns < 1:
		return Plan{}, fmt.Errorf("--max-runs must be at least 1, got %d", maxRuns)
	}
	p := Plan{MaxRuns: maxRuns}
	for _, id := range ids {
		entry, err := cs.Resolve(id)
		if err != nil {
			return Plan{}, err
		}
		est, err := estimate(entry, cs, cases, epochs)
		if err != nil {
			return Plan{}, err
		}
		p.Estimates = append(p.Estimates, est)
		for _, c := range cases {
			p.Cells = append(p.Cells, Cell{Case: c.ID, Kind: c.Kind, Candidate: id, Epochs: epochs})
			p.Runs += epochs
		}
	}
	if p.Runs > maxRuns {
		return p, fmt.Errorf("%w: %d runs, cap %d; name fewer candidates or cases, or raise --max-runs", ErrOverCap, p.Runs, maxRuns)
	}
	return p, nil
}

// estimate prices one candidate's runs in its runner's unit.
func estimate(e spec.ReviewerEntry, cs Candidates, cases []Case, epochs int) (Estimate, error) {
	switch {
	case e.Runner == "agy":
		return Estimate{Candidate: e.ID, Unit: UnitAgyRuns, Amount: float64(len(cases) * epochs), Basis: BasisDefault}, nil
	case e.Runner == "pi" && e.Provider == "nan":
		rates, ok := cs.Credits[e.ID]
		if !ok {
			return Estimate{}, fmt.Errorf("candidate %s declares no credits_per_mtok in %s, so its runs cannot be priced", e.ID, CandidatesFile)
		}
		// The envelopes are measured totals with no input/output split, so each
		// token is priced at the higher rate: an upper bound, never an
		// underestimate.
		rate := max(rates.Input, rates.Output)
		var tokens int
		for _, c := range cases {
			n, ok := cs.Envelopes[c.Kind]
			if !ok {
				return Estimate{}, fmt.Errorf("%s declares no default_envelope_tokens for kind %s (case %s)", CandidatesFile, c.Kind, c.ID)
			}
			tokens += n
		}
		return Estimate{Candidate: e.ID, Unit: UnitNaNCredits, Amount: float64(tokens*epochs) * rate / 1e6, Basis: BasisDefault}, nil
	default:
		return Estimate{}, fmt.Errorf("candidate %s: no estimator for runner %s with provider %q", e.ID, e.Runner, e.Provider)
	}
}

// Render prints the plan: the matrix, then each candidate's estimate.
func (p Plan) Render() string {
	var b strings.Builder
	candidates := len(p.Estimates)
	cases := 0
	if candidates > 0 {
		cases = len(p.Cells) / candidates
	}
	epochs := 0
	if len(p.Cells) > 0 {
		epochs = p.Cells[0].Epochs
	}
	fmt.Fprintf(&b, "%d cases × %d candidates × %d epochs = %d runs (cap %d)\n\n", cases, candidates, epochs, p.Runs, p.MaxRuns)
	fmt.Fprintf(&b, "%-32s %-8s %-32s %s\n", "CASE", "KIND", "CANDIDATE", "EPOCHS")
	for _, c := range p.Cells {
		fmt.Fprintf(&b, "%-32s %-8s %-32s %d\n", c.Case, c.Kind, c.Candidate, c.Epochs)
	}
	b.WriteString("\nEstimate:\n")
	for _, e := range p.Estimates {
		switch e.Unit {
		case UnitNaNCredits:
			fmt.Fprintf(&b, "  %-32s at most %.1f %s (%s envelope, priced at the higher of the input and output rates)\n", e.Candidate, e.Amount, e.Unit, e.Basis)
		default:
			fmt.Fprintf(&b, "  %-32s %.0f %s (its quota cannot be read)\n", e.Candidate, e.Amount, e.Unit)
		}
	}
	return b.String()
}
