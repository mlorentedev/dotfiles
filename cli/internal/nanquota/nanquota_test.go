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

// AC1: 83% of a metered quota WARNs, 10% PASSes, an unmetered model is never
// flagged however much it used.
func TestEvaluateClassifiesByShareOfQuota(t *testing.T) {
	served := map[string]bool{"qwen3.8-flash": true, "deepseek-v4-flash": true, "qwen3.6": true}
	got := byModel(Evaluate([]string{"qwen3.8-flash", "deepseek-v4-flash", "qwen3.6"}, served, mustUsage(t), mustTable(t), 80))

	if f := got["qwen3.8-flash"]; f.Level != Warn || !strings.Contains(f.Msg, "83%") {
		t.Errorf("qwen3.8-flash at 415M/500M: got %v %q, want WARN naming 83%%", f.Level, f.Msg)
	}
	if f := got["deepseek-v4-flash"]; f.Level != Pass || !strings.Contains(f.Msg, "10%") {
		t.Errorf("deepseek-v4-flash at 300M/3000M: got %v %q, want PASS naming 10%%", f.Level, f.Msg)
	}
	if f := got["qwen3.6"]; f.Level != Info {
		t.Errorf("qwen3.6 is unmetered: got %v %q, want INFO whatever it used", f.Level, f.Msg)
	}
}

func TestEvaluateFailsAtTheQuota(t *testing.T) {
	u := mustUsage(t)
	u.Totals.ByModel = []ModelTotal{{Model: "qwen3.8-flash", TotalTokens: 500000000}}
	f := Evaluate([]string{"qwen3.8-flash"}, map[string]bool{"qwen3.8-flash": true}, u, mustTable(t), 80)[0]
	if f.Level != Fail {
		t.Errorf("a spent quota answers 402: got %v %q, want FAIL", f.Level, f.Msg)
	}
}

// A metered model the member has not touched this period is absent from
// by_model. It is at 0%, not unknown.
func TestEvaluateTreatsAnAbsentModelAsUnused(t *testing.T) {
	f := Evaluate([]string{"glm5.3-flash"}, map[string]bool{"glm5.3-flash": true}, mustUsage(t), mustTable(t), 80)[0]
	if f.Level != Pass || !strings.Contains(f.Msg, "0%") {
		t.Errorf("got %v %q, want PASS at 0%%", f.Level, f.Msg)
	}
}

// The table is closed-world: a bound model NaN meters that nobody declared must
// not read as unmetered, or the first model NaN adds is never alarmed.
func TestEvaluateWarnsOnAnUndeclaredModel(t *testing.T) {
	f := Evaluate([]string{"mimo-v2.5"}, map[string]bool{"mimo-v2.5": true}, mustUsage(t), mustTable(t), 80)[0]
	if f.Level != Warn || !strings.Contains(f.Msg, "neither metered nor unmetered") {
		t.Errorf("got %v %q, want WARN naming the undeclared model", f.Level, f.Msg)
	}
}

// AC3: a bound id NaN does not serve is the class that bound qwen3-rerank for
// weeks while NaN answered 401.
func TestEvaluateFailsOnAnIDNaNDoesNotServe(t *testing.T) {
	f := Evaluate([]string{"qwen3-rerank"}, map[string]bool{"rerank": true}, mustUsage(t), mustTable(t), 80)[0]
	if f.Level != Fail || !strings.Contains(f.Msg, "qwen3-rerank") {
		t.Errorf("got %v %q, want FAIL naming qwen3-rerank", f.Level, f.Msg)
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
	_, err := ParseTable([]byte(`{"checked":"2026-09-26","metered":{"a":{"tokens":1,"period":"month"}},"unmetered":["a"]}`))
	if err == nil {
		t.Error("a model both metered and unmetered must be rejected")
	}
}
