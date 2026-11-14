package evals

import (
	"errors"
	"fmt"
	"sort"
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
	// run. Only its rate is an upper bound: the envelope itself was measured on
	// another model and case.
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
	// Envelopes are the token envelopes the NaN estimates priced, by case kind.
	Envelopes map[string]int
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
	p := Plan{MaxRuns: maxRuns, Envelopes: map[string]int{}}
	named := map[string]bool{}
	for _, id := range ids {
		if named[id] {
			return Plan{}, fmt.Errorf("candidate %s is named twice, which would run its matrix twice", id)
		}
		named[id] = true
		entry, err := cs.Resolve(id)
		if err != nil {
			return Plan{}, err
		}
		est, err := estimate(entry, cs, cases, epochs)
		if err != nil {
			return Plan{}, err
		}
		p.Estimates = append(p.Estimates, est)
		if est.Unit == UnitNaNCredits {
			for _, c := range cases {
				p.Envelopes[c.Kind] = cs.Envelopes[c.Kind]
			}
		}
		for _, c := range cases {
			p.Cells = append(p.Cells, Cell{Case: c.ID, Kind: c.Kind, Candidate: id, Epochs: epochs})
			p.Runs += epochs
		}
	}
	if p.Runs > maxRuns {
		return p, fmt.Errorf("%w: %d runs, cap %d; name fewer candidates or cases (--case), lower --epochs, or raise --max-runs", ErrOverCap, p.Runs, maxRuns)
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
		// The envelopes are measured totals with no split by token type, so each
		// token is priced at the highest of the three rates rather than at an
		// invented split.
		rate := max(rates.Input, rates.Cached, rates.Output)
		if rate == 0 {
			return Estimate{}, fmt.Errorf("candidate %s has zero credits_per_mtok, and an estimate is never zero; a model that draws no credits needs no evaluation budget, so it is not a candidate here", e.ID)
		}
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

// SelectCases narrows cases to the named ids, in corpus order. No ids keeps
// every case; an unknown or repeated id is refused.
func SelectCases(cases []Case, ids []string) ([]Case, error) {
	if len(ids) == 0 {
		return cases, nil
	}
	want := map[string]bool{}
	for _, id := range ids {
		if want[id] {
			return nil, fmt.Errorf("case %s is named twice", id)
		}
		want[id] = true
	}
	known := make([]string, 0, len(cases))
	var out []Case
	for _, c := range cases {
		known = append(known, c.ID)
		if want[c.ID] {
			out = append(out, c)
			delete(want, c.ID)
		}
	}
	for _, id := range ids {
		if want[id] {
			return nil, fmt.Errorf("case %s is not in the corpus; the cases are: %s", id, strings.Join(known, ", "))
		}
	}
	return out, nil
}

// Render prints the plan: the matrix, each candidate's estimate, a total per
// unit (NaN bills every model from one pool), and the envelopes it priced.
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
	totals := map[Unit]float64{}
	var units []Unit
	for _, e := range p.Estimates {
		if _, seen := totals[e.Unit]; !seen {
			units = append(units, e.Unit)
		}
		totals[e.Unit] += e.Amount
		switch e.Unit {
		case UnitNaNCredits:
			fmt.Fprintf(&b, "  %-32s %.1f %s (%s envelope)\n", e.Candidate, e.Amount, e.Unit, e.Basis)
		default:
			fmt.Fprintf(&b, "  %-32s %.0f %s (its quota cannot be read)\n", e.Candidate, e.Amount, e.Unit)
		}
	}
	for _, u := range units {
		if u == UnitNaNCredits {
			fmt.Fprintf(&b, "Total: %.1f %s\n", totals[u], u)
		} else {
			fmt.Fprintf(&b, "Total: %.0f %s\n", totals[u], u)
		}
	}
	if len(p.Envelopes) > 0 {
		kinds := make([]string, 0, len(p.Envelopes))
		for k := range p.Envelopes {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		parts := make([]string, 0, len(kinds))
		for _, k := range kinds {
			parts = append(parts, fmt.Sprintf("%d tokens per %s run", p.Envelopes[k], k))
		}
		fmt.Fprintf(&b, "\nNaN envelopes (%s): %s.\n"+
			"Measured on another model and case, so a run may spend more. Each token is\n"+
			"priced at the highest of the model's input, cached and output rates.\n", CandidatesFile, strings.Join(parts, ", "))
	}
	return b.String()
}
