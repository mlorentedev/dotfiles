package spec

import (
	"strings"
	"testing"
)

// The PR-Agent route reads the pool through the parser the gate and the
// launcher use, so a block it needs must survive that parser.
func TestParseReviewerPool_ReadsThePRAgentBlocksAndRisk(t *testing.T) {
	raw := []byte(`{"pool":[
 {"id":"nan/a","pr_agent":{"model":"openai/a","weight":3}},
 {"id":"anthropic-review/h","signs":"fallback","pr_agent":{"model":"anthropic/h","weight":5,"reasoning_effort":"high","forge_rank":2}},
 {"id":"anthropic-review/s","signs":"second","pr_agent":{"model":"anthropic/s","route":"risk"}}
],"pr_agent_risk":{"min_changed_lines":900,"label":"deep-review"}}`)

	p, err := ParseReviewerPool(raw, "pool.json")
	if err != nil {
		t.Fatalf("ParseReviewerPool: %v", err)
	}
	if got := p.Pool[0].PRAgent; got == nil || got.Model != "openai/a" || got.Weight != 3 {
		t.Errorf("first block = %+v", got)
	}
	h := p.Pool[1].PRAgent
	if h == nil || h.ReasoningEffort != "high" || h.ForgeRank != 2 {
		t.Errorf("second block = %+v", h)
	}
	if s := p.Pool[2].PRAgent; s == nil || s.Route != "risk" {
		t.Errorf("third block = %+v", s)
	}
	if p.PRAgentRisk == nil || p.PRAgentRisk.MinChangedLines == nil || *p.PRAgentRisk.MinChangedLines != 900 || p.PRAgentRisk.Label != "deep-review" {
		t.Errorf("risk = %+v", p.PRAgentRisk)
	}
}

// One validator for every reader: the same refusals the gate applies, naming
// the source the caller passed.
func TestParseReviewerPool_RefusesWhatTheGateRefuses(t *testing.T) {
	cases := map[string]string{
		"not json":              `{"pool": [`,
		"empty pool":            `{"pool": []}`,
		"blank id":              `{"pool":[{"id":"  "}]}`,
		"anthropic signs first": `{"pool":[{"id":"anthropic-review/h"}]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseReviewerPool([]byte(raw), "my-pool.json")
			if err == nil || !strings.Contains(err.Error(), "my-pool.json") {
				t.Fatalf("want an error naming the source, got %v", err)
			}
		})
	}
}
