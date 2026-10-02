package nanquota

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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

const tableFixture = `{
  "checked": "2026-09-26",
  "source": "https://nan.builders/docs/models",
  "metered": {
    "qwen3.8-flash": {"tokens": 500000000, "period": "month"},
    "deepseek-v4-flash": {"tokens": 3000000000, "period": "month"},
    "glm5.3-flash": {"tokens": 2000000000, "period": "month"}
  },
  "unmetered": ["qwen3.6", "rerank"]
}`

func mustUsage(t *testing.T) Usage {
	t.Helper()
	u, err := ParseUsage([]byte(usageFixture))
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func mustTable(t *testing.T) Table {
	t.Helper()
	tb, err := ParseTable([]byte(tableFixture))
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

func byModel(fs []Finding) map[string]Finding {
	out := map[string]Finding{}
	for _, f := range fs {
		out[f.Model] = f
	}
	return out
}

// AC1: 83% of a metered quota WARNs, 10% is reported (INFO, visible without
// --verbose, AC2), an unmetered model is never flagged however much it used.
func TestEvaluateClassifiesByShareOfQuota(t *testing.T) {
	served := map[string]bool{"qwen3.8-flash": true, "deepseek-v4-flash": true, "qwen3.6": true}
	got := byModel(Evaluate([]string{"qwen3.8-flash", "deepseek-v4-flash", "qwen3.6"}, served, mustUsage(t), mustTable(t), 80))

	if f := got["qwen3.8-flash"]; f.Level != Warn || !strings.Contains(f.Msg, "83%") {
		t.Errorf("qwen3.8-flash at 415M/500M: got %v %q, want WARN naming 83%%", f.Level, f.Msg)
	}
	if f := got["deepseek-v4-flash"]; f.Level != Info || !strings.Contains(f.Msg, "10%") {
		t.Errorf("deepseek-v4-flash at 300M/3000M: got %v %q, want INFO naming 10%%", f.Level, f.Msg)
	}
	if f := got["qwen3.6"]; f.Level != Pass {
		t.Errorf("qwen3.6 is unmetered: got %v %q, want PASS whatever it used", f.Level, f.Msg)
	}
}

func TestEvaluateFailsAtTheQuota(t *testing.T) {
	u := mustUsage(t)
	u.Totals.ByModel = []ModelTotal{{Model: "qwen3.8-flash", TotalTokens: 500000000}}
	f := byModel(Evaluate([]string{"qwen3.8-flash"}, map[string]bool{"qwen3.8-flash": true}, u, mustTable(t), 80))["qwen3.8-flash"]
	if f.Level != Fail {
		t.Errorf("a spent quota answers 402: got %v %q, want FAIL", f.Level, f.Msg)
	}
}

// A metered model the member has not touched this period is absent from
// by_model. It is at 0%, not unknown.
func TestEvaluateTreatsAnAbsentModelAsUnused(t *testing.T) {
	f := byModel(Evaluate([]string{"glm5.3-flash"}, map[string]bool{"glm5.3-flash": true}, mustUsage(t), mustTable(t), 80))["glm5.3-flash"]
	if f.Level != Info || !strings.Contains(f.Msg, "0%") {
		t.Errorf("got %v %q, want INFO at 0%%", f.Level, f.Msg)
	}
}

// The table is closed-world: a bound model NaN meters that nobody declared must
// not read as unmetered, or the first model NaN adds is never alarmed.
func TestEvaluateWarnsOnAnUndeclaredModel(t *testing.T) {
	f := byModel(Evaluate([]string{"mimo-v2.5"}, map[string]bool{"mimo-v2.5": true}, mustUsage(t), mustTable(t), 80))["mimo-v2.5"]
	if f.Level != Warn || !strings.Contains(f.Msg, "neither metered nor unmetered") {
		t.Errorf("got %v %q, want WARN naming the undeclared model", f.Level, f.Msg)
	}
}

// AC3: a bound id NaN does not serve is the class that bound qwen3-rerank for
// weeks while NaN answered 401.
func TestEvaluateFailsOnAnIDNaNDoesNotServe(t *testing.T) {
	f := byModel(Evaluate([]string{"qwen3-rerank"}, map[string]bool{"rerank": true}, mustUsage(t), mustTable(t), 80))["qwen3-rerank"]
	if f.Level != Fail || !strings.Contains(f.Msg, "qwen3-rerank") {
		t.Errorf("got %v %q, want FAIL naming qwen3-rerank", f.Level, f.Msg)
	}
}

// AC7: the quota is the account's, not a binding's. qwen3.8-flash reached 83%
// unnoticed on 2026-09-26 while pi, not model-map.json, was spending it; a
// watch over bindings alone still missed it after AI-047 shipped.
func TestEvaluateWatchesEveryMeteredModelNotOnlyBindings(t *testing.T) {
	served := map[string]bool{"qwen3.8-flash": true, "deepseek-v4-flash": true, "glm5.3-flash": true, "qwen3.6": true}
	got := byModel(Evaluate([]string{"qwen3.6"}, served, mustUsage(t), mustTable(t), 80))

	if f, ok := got["qwen3.8-flash"]; !ok || f.Level != Warn || !strings.Contains(f.Msg, "83%") {
		t.Errorf("unbound qwen3.8-flash at 83%%: got %+v, want WARN", f)
	}
	if f, ok := got["glm5.3-flash"]; !ok || f.Level != Info {
		t.Errorf("unbound, unused metered glm5.3-flash: got %+v, want INFO at 0%%", f)
	}
}

// A model with usage this period that the table does not declare is spending
// the account whoever routes to it.
func TestEvaluateWatchesAnUndeclaredModelWithUsage(t *testing.T) {
	u := mustUsage(t)
	u.Totals.ByModel = append(u.Totals.ByModel, ModelTotal{Model: "brand-new", TotalTokens: 10000000})
	served := map[string]bool{"brand-new": true, "qwen3.8-flash": true, "deepseek-v4-flash": true, "glm5.3-flash": true}
	f, ok := byModel(Evaluate(nil, served, u, mustTable(t), 80))["brand-new"]
	if !ok || f.Level != Warn || !strings.Contains(f.Msg, "neither metered nor unmetered") {
		t.Errorf("got %+v, want WARN naming the undeclared model", f)
	}
}

// /v1/models hides premium models from a key without the tier (glm5.3, AI-045):
// a metered model nothing binds and the key cannot see spends nothing.
func TestEvaluateIgnoresAMeteredModelTheKeyCannotSee(t *testing.T) {
	served := map[string]bool{"qwen3.8-flash": true, "deepseek-v4-flash": true}
	if f, ok := byModel(Evaluate(nil, served, mustUsage(t), mustTable(t), 80))["glm5.3-flash"]; ok {
		t.Errorf("got %+v, want no finding", f)
	}
}

// A retired model with leftover usage, which nothing binds, the table does not
// declare and NaN no longer serves, can spend nothing more: no finding.
func TestEvaluateIgnoresARetiredModelNothingCanReach(t *testing.T) {
	u := mustUsage(t)
	u.Totals.ByModel = append(u.Totals.ByModel, ModelTotal{Model: "mimo-v2.5", TotalTokens: 5})
	served := map[string]bool{"qwen3.8-flash": true, "deepseek-v4-flash": true, "glm5.3-flash": true}
	if f, ok := byModel(Evaluate(nil, served, u, mustTable(t), 80))["mimo-v2.5"]; ok {
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

func TestPeriodStartIsTheUTCCalendarMonth(t *testing.T) {
	// 01:30 on the 1st in UTC+2 is still the previous month in UTC.
	loc := time.FixedZone("UTC+2", 2*3600)
	if got := PeriodStart(time.Date(2026, 10, 1, 1, 30, 0, 0, loc)); got != "2026-09-01" {
		t.Errorf("PeriodStart = %s, want 2026-09-01", got)
	}
}

func TestTableAge(t *testing.T) {
	tb := mustTable(t)
	now := time.Date(2026, 12, 26, 0, 0, 0, 0, time.UTC)
	if d, err := tb.Age(now); err != nil || d != 91*24*time.Hour {
		t.Errorf("Age = %v, %v; want 91 days", d, err)
	}
}

func TestParseTableRejectsAModelDeclaredTwice(t *testing.T) {
	_, err := ParseTable([]byte(`{"checked":"2026-09-26","source":"s","metered":{"a":{"tokens":1,"period":"month"}},"unmetered":["a"]}`))
	if err == nil {
		t.Error("a model both metered and unmetered must be rejected")
	}
}

// A 200 that lacks the field the verdict comes from is not zero usage: read as
// zero it PASSes a spent model.
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

// Omitting a list is not a decision that it is empty (closed-world table).
func TestParseTableRequiresItsDecisionFields(t *testing.T) {
	cases := map[string]string{
		"no source":    `{"checked":"2026-09-26","metered":{},"unmetered":[]}`,
		"no metered":   `{"checked":"2026-09-26","source":"s","unmetered":[]}`,
		"no unmetered": `{"checked":"2026-09-26","source":"s","metered":{}}`,
	}
	for name, doc := range cases {
		if _, err := ParseTable([]byte(doc)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := ParseTable([]byte(`{"checked":"2026-09-26","source":"s","metered":{},"unmetered":[]}`)); err != nil {
		t.Errorf("explicitly empty lists are a decision and must parse: %v", err)
	}
}
