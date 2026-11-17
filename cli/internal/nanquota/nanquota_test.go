package nanquota

import (
	"encoding/json"
	"strings"
	"testing"
)

// usageFixture is the shape GET /v1/usage returned on 2026-09-26, cut down to
// the fields the check reads plus one it must ignore (data rows are per day and
// already summed into totals.by_model).
const usageFixture = `{
  "object": "usage",
  "start_date": "2026-09-01",
  "end_date": "2026-09-27",
  "data": [{"date": "2026-09-26", "model": "qwen3.8-flash", "total_tokens": 1}],
  "totals": {
    "total_tokens": 1,
    "by_model": [
      {"model": "qwen3.8-flash", "total_tokens": 415000000},
      {"model": "deepseek-v4-flash", "total_tokens": 300000000},
      {"model": "qwen3.6", "total_tokens": 900000000}
    ]
  },
  "has_more": false
}`

func mustUsage(t *testing.T) Usage {
	t.Helper()
	u, err := ParseUsage([]byte(usageFixture))
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func byModel(fs []Finding) map[string]Finding {
	out := map[string]Finding{}
	for _, f := range fs {
		out[f.Model] = f
	}
	return out
}

// Usage is reported as tokens and nothing else: NaN bills credits from one pool
// (2026-11), so no share of a limit is computed, however much a model moved.
func TestEvaluateReportsTokensWithoutAShareOfAnything(t *testing.T) {
	served := map[string]bool{"qwen3.8-flash": true, "deepseek-v4-flash": true, "qwen3.6": true}
	got := byModel(Evaluate([]string{"qwen3.8-flash", "deepseek-v4-flash", "qwen3.6"}, served, mustUsage(t)))

	for id, want := range map[string]string{"qwen3.8-flash": "415.0M", "deepseek-v4-flash": "300.0M", "qwen3.6": "900.0M"} {
		f := got[id]
		if f.Level != Info || !strings.Contains(f.Msg, want) || !strings.Contains(f.Msg, "since 2026-09-01") || strings.Contains(f.Msg, "%") {
			t.Errorf("%s: got %v %q, want INFO naming %s since 2026-09-01 and no percentage", id, f.Level, f.Msg, want)
		}
	}
}

// A bound model with no traffic in the window is absent from by_model: it is
// reported at zero, not dropped, so every binding is visible.
func TestEvaluateReportsABoundModelWithNoUsage(t *testing.T) {
	f, ok := byModel(Evaluate([]string{"glm5.3-flash"}, map[string]bool{"glm5.3-flash": true}, mustUsage(t)))["glm5.3-flash"]
	if !ok || f.Level != Info || !strings.Contains(f.Msg, "0.0M") {
		t.Errorf("got %+v, want INFO at 0.0M", f)
	}
}

// AC3: a bound id NaN does not serve is the class that bound qwen3-rerank for
// weeks while NaN answered 401.
func TestEvaluateFailsOnAnIDNaNDoesNotServe(t *testing.T) {
	f := byModel(Evaluate([]string{"qwen3-rerank"}, map[string]bool{"rerank": true}, mustUsage(t)))["qwen3-rerank"]
	if f.Level != Fail || !strings.Contains(f.Msg, "qwen3-rerank") {
		t.Errorf("got %v %q, want FAIL naming qwen3-rerank", f.Level, f.Msg)
	}
}

// The account is spent by consumers model-map.json does not describe (pi,
// opencode, qq/qf), so a served model with traffic is reported unbound too.
func TestEvaluateReportsUsageOnAnUnboundModel(t *testing.T) {
	served := map[string]bool{"qwen3.8-flash": true, "deepseek-v4-flash": true, "qwen3.6": true}
	if f, ok := byModel(Evaluate(nil, served, mustUsage(t)))["qwen3.8-flash"]; !ok || f.Level != Info {
		t.Errorf("unbound qwen3.8-flash with traffic: got %+v, want INFO", f)
	}
}

// A retired model with leftover usage, which nothing binds and NaN no longer
// serves, can spend nothing more: no finding.
func TestEvaluateIgnoresARetiredModelNothingCanReach(t *testing.T) {
	u := mustUsage(t)
	u.Totals.ByModel = append(u.Totals.ByModel, ModelTotal{Model: "mimo-v2.5", TotalTokens: 5})
	served := map[string]bool{"qwen3.8-flash": true, "deepseek-v4-flash": true}
	if f, ok := byModel(Evaluate(nil, served, u))["mimo-v2.5"]; ok {
		t.Errorf("got %+v, want no finding", f)
	}
}

func TestBoundModelsReadsTiersChainsAndServices(t *testing.T) {
	var m map[string]any
	doc := `{
	  "tiers": {"mid": {"claude": "sonnet", "nan": "deepseek-v4-flash"}, "low": {"nan": "glm5.3-flash"}},
	  "chains": {"low": ["nan:qwen3.6", "claude:haiku", "nan:glm5.3-flash"]},
	  "services": {"rerank": {"pool": "nan", "model": "qwen3-rerank"}, "other": {"pool": "gemini", "model": "x"}}
	}`
	if err := json.Unmarshal([]byte(doc), &m); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(BoundModels(m, "nan"), ",")
	want := "deepseek-v4-flash,glm5.3-flash,qwen3-rerank,qwen3.6"
	if got != want {
		t.Errorf("BoundModels = %s, want %s (sorted, deduplicated, nan pool only)", got, want)
	}
}

func TestParseModelsReadsTheServedIDs(t *testing.T) {
	served, err := ParseModels([]byte(`{"object":"list","data":[{"id":"rerank"},{"id":"qwen3.6"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !served["rerank"] || !served["qwen3.6"] || len(served) != 2 {
		t.Errorf("ParseModels = %v", served)
	}
}

// A 200 that lacks the field the verdict comes from is not zero usage: read as
// zero it reports every model as idle.
func TestParseUsageRejectsAnAnswerWithoutByModel(t *testing.T) {
	for _, body := range []string{`{"totals":{}}`, `{}`, `{"totals":{"by_model":null}}`} {
		if _, err := ParseUsage([]byte(body)); err == nil {
			t.Errorf("ParseUsage(%s) accepted an answer with no totals.by_model", body)
		}
	}
	if _, err := ParseUsage([]byte(`{"totals":{"by_model":[]}}`)); err != nil {
		t.Errorf("an empty by_model is a month with no traffic, not a malformed answer: %v", err)
	}
}

// An empty served set would FAIL every binding at once, so it is inconclusive.
func TestParseModelsRejectsAnAnswerWithoutModels(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":null}`, `{"data":[]}`} {
		if _, err := ParseModels([]byte(body)); err == nil {
			t.Errorf("ParseModels(%s) accepted an answer that serves nothing", body)
		}
	}
}
