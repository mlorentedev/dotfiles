package evals

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

const pricedCandidates = `{
 "default_envelope_tokens": {"replay": 200000, "planted": 100000},
 "candidates":[
  {"id":"nan/glm5.3-flash","runner":"pi","provider":"nan","model":"glm5.3-flash","secret_id":"NAN_API_KEY","vendor":"zhipu",
   "credits_per_mtok":{"input":5.25,"cached":1.05,"output":17.5}},
  {"id":"agy/gemini-4-hydrogen-high","runner":"agy","model":"gemini-4-hydrogen-high","auth":"login","vendor":"google"}
]}`

func planCases() []Case {
	return []Case{{ID: "a-r1", Kind: "replay"}, {ID: "b-r2", Kind: "replay"}, {ID: "p1", Kind: "planted"}}
}

func loadPriced(t *testing.T, body string) Candidates {
	t.Helper()
	cs, err := LoadCandidates(writeCandidates(t, body))
	if err != nil {
		t.Fatal(err)
	}
	return cs
}

// The plan is the matrix a run would execute: every case for every named
// candidate, each epochs times, and nothing is launched to compute it.
func TestPlanCountsEveryCaseCandidateAndEpoch(t *testing.T) {
	cs := loadPriced(t, pricedCandidates)
	p, err := BuildPlan(planCases(), cs, []string{"nan/glm5.3-flash", "agy/gemini-4-hydrogen-high"}, 3, 30)
	if err != nil {
		t.Fatal(err)
	}
	if p.Runs != 18 || len(p.Cells) != 6 {
		t.Fatalf("runs %d over %d cells, want 18 over 6", p.Runs, len(p.Cells))
	}
	for _, c := range p.Cells {
		if c.Epochs != 3 {
			t.Errorf("cell %s × %s has %d epochs, want 3", c.Case, c.Candidate, c.Epochs)
		}
	}
}

// A NaN candidate is priced in credits from its kind's envelope, as an upper
// bound at the higher of its input and output rates, and labelled default
// until earlier runs measure the case. An agy candidate is a run count: its
// quota cannot be read.
func TestPlanEstimatesEachCandidateInItsRunnersUnit(t *testing.T) {
	cs := loadPriced(t, pricedCandidates)
	p, err := BuildPlan(planCases(), cs, []string{"nan/glm5.3-flash", "agy/gemini-4-hydrogen-high"}, 3, 30)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Estimate{}
	for _, e := range p.Estimates {
		got[e.Candidate] = e
	}
	nan := got["nan/glm5.3-flash"]
	// two replays at 200K and one planted at 100K, three epochs each, at 17.5 per 1M.
	want := 3 * (2*200000 + 100000) * 17.5 / 1e6
	if nan.Unit != UnitNaNCredits || nan.Amount != want || nan.Basis != BasisDefault {
		t.Errorf("NaN estimate = %+v, want %.4f %s (%s)", nan, want, UnitNaNCredits, BasisDefault)
	}
	agy := got["agy/gemini-4-hydrogen-high"]
	if agy.Unit != UnitAgyRuns || agy.Amount != 9 {
		t.Errorf("agy estimate = %+v, want 9 %s", agy, UnitAgyRuns)
	}
}

// The cap is what authorises spend, so a matrix above it is refused, and the
// plan that was refused is still returned to be printed.
func TestPlanRefusesAMatrixAboveTheCap(t *testing.T) {
	cs := loadPriced(t, pricedCandidates)
	p, err := BuildPlan(planCases(), cs, []string{"nan/glm5.3-flash", "agy/gemini-4-hydrogen-high"}, 3, 17)
	if !errors.Is(err, ErrOverCap) {
		t.Fatalf("18 runs under a cap of 17: err = %v, want ErrOverCap", err)
	}
	if p.Runs != 18 || !strings.Contains(err.Error(), "--max-runs") {
		t.Errorf("a refused plan must still count its runs and name the flag: %d, %v", p.Runs, err)
	}
	if _, err := BuildPlan(planCases(), cs, []string{"nan/glm5.3-flash", "agy/gemini-4-hydrogen-high"}, 3, 18); err != nil {
		t.Errorf("a matrix at the cap is allowed: %v", err)
	}
}

// An estimate is never zero or unknown: a candidate the plan cannot price, or a
// kind with no envelope, refuses the plan.
func TestPlanRefusesWhatItCannotPrice(t *testing.T) {
	unpriced := strings.Replace(pricedCandidates, `,
   "credits_per_mtok":{"input":5.25,"cached":1.05,"output":17.5}`, "", 1)
	noPlanted := strings.Replace(pricedCandidates, `, "planted": 100000`, "", 1)
	other := strings.Replace(pricedCandidates, `"agy/gemini-4-hydrogen-high","runner":"agy","model":"gemini-4-hydrogen-high","auth":"login"`,
		`"or/x","runner":"pi","provider":"openrouter","model":"x","secret_id":"OPENROUTER_API_KEY"`, 1)
	for _, tc := range []struct{ name, body, id, want string }{
		{"a NaN candidate without rates", unpriced, "nan/glm5.3-flash", "credits_per_mtok"},
		{"a kind without an envelope", noPlanted, "nan/glm5.3-flash", "planted"},
		{"a provider with no estimator", other, "or/x", "openrouter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildPlan(planCases(), loadPriced(t, tc.body), []string{tc.id}, 3, 30)
			if err == nil || errors.Is(err, ErrOverCap) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want a refusal naming %q, got %v", tc.want, err)
			}
		})
	}
}

func TestPlanRefusesAnUnknownCandidateAndAnEmptyMatrix(t *testing.T) {
	cs := loadPriced(t, pricedCandidates)
	if _, err := BuildPlan(planCases(), cs, []string{"nan/nope"}, 3, 30); err == nil || !strings.Contains(err.Error(), "nan/nope") {
		t.Errorf("an unknown candidate: %v", err)
	}
	if _, err := BuildPlan(planCases(), cs, nil, 3, 30); err == nil {
		t.Error("a plan naming no candidate must be refused")
	}
	if _, err := BuildPlan(nil, cs, []string{"nan/glm5.3-flash"}, 3, 30); err == nil {
		t.Error("a plan over no cases must be refused")
	}
	if _, err := BuildPlan(planCases(), cs, []string{"nan/glm5.3-flash"}, 0, 30); err == nil {
		t.Error("zero epochs must be refused")
	}
}

// Rates belong to NaN candidates only, and a negative rate or envelope is a
// typo the plan would turn into a reassuring number.
func TestLoadCandidatesRefusesMalformedPricing(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"rates on an agy candidate", strings.Replace(pricedCandidates, `"auth":"login","vendor":"google"}`,
			`"auth":"login","vendor":"google","credits_per_mtok":{"input":1,"cached":1,"output":1}}`, 1), "credits_per_mtok"},
		{"a negative rate", strings.Replace(pricedCandidates, `"output":17.5`, `"output":-17.5`, 1), "negative"},
		{"a zero envelope", strings.Replace(pricedCandidates, `"replay": 200000`, `"replay": 0`, 1), "replay"},
		{"an envelope for an unknown kind", strings.Replace(pricedCandidates, `"planted": 100000`, `"planted": 100000, "pr-replay": 1`, 1), "pr-replay"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadCandidates(writeCandidates(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want a refusal naming %q, got %v", tc.want, err)
			}
		})
	}
}

// The repository's candidates price its whole corpus: every NaN candidate
// carries rates and every case kind an envelope, so the first real --plan
// cannot refuse for a gap in the file.
func TestRepositoryCandidatesPriceTheWholeCorpus(t *testing.T) {
	root := repoFile(t)
	cases, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := LoadCandidates(root)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(cs.Entries))
	for _, e := range cs.Entries {
		ids = append(ids, e.ID)
	}
	p, err := BuildPlan(cases, cs, ids, DefaultEpochs, 1000)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range p.Estimates {
		if e.Amount <= 0 {
			t.Errorf("%s is estimated at %v %s", e.Candidate, e.Amount, e.Unit)
		}
	}
}

// A run grades against the keys, so a corpus whose keys do not check is not run.
func TestLoadCasesRefusesACorpusThatDoesNotCheck(t *testing.T) {
	root := writeTree(t, goodCase, "x-r1.json", sourceReview("FAIL", "nan/x"))
	if _, err := LoadCases(root); err == nil || !strings.Contains(err.Error(), "dotf eval check") {
		t.Fatalf("a corpus with a failing case loaded: %v", err)
	}
	cases, err := LoadCases(writeTree(t, goodCase, "x-r1.json", sourceReview("FAIL", "nan/x", goodRows...)))
	if err != nil || len(cases) != 1 || cases[0].ID != "x-r1" {
		t.Fatalf("LoadCases = %+v, %v", cases, err)
	}
}

// candidates.json copies ai/nan/README.md's credit rates so the plan reads no
// prose; this holds the copy to its source, so a rate change in one fails here.
func TestCandidateRatesMatchTheNaNReadme(t *testing.T) {
	raw, err := os.ReadFile(repoFile(t, "ai", "nan", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) < 5 {
			continue
		}
		model := strings.Trim(strings.TrimSpace(cells[1]), "`")
		readme[model] = strings.TrimSpace(cells[3])
	}
	cs, err := LoadCandidates(repoFile(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range cs.Entries {
		r, ok := cs.Credits[e.ID]
		if !ok {
			continue
		}
		want := fmt.Sprintf("%s / %s / %s", strconv.FormatFloat(r.Input, 'f', -1, 64),
			strconv.FormatFloat(r.Cached, 'f', -1, 64), strconv.FormatFloat(r.Output, 'f', -1, 64))
		if got := readme[e.Model]; got != want {
			t.Errorf("%s: candidates.json says %q, ai/nan/README.md says %q", e.ID, want, got)
		}
	}
}
