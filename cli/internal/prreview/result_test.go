package prreview

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResult_KeyValuesAreTheScriptsLinesInTheScriptsOrder(t *testing.T) {
	r := Result{First: "nan", Second: "anthropic", Reviewer: "m", Route: "draw", NanModel: "m",
		NanFallbacks: []string{"a", "b"}, AnthropicModel: "x", AnthropicEffort: "high", Note: "n1 n2"}
	want := "first=nan\nsecond=anthropic\nreviewer=m\nroute=draw\nnan_model=m\nnan_fallbacks=[\"a\",\"b\"]\n" +
		"anthropic_model=x\nanthropic_effort=high\nnote=n1 n2\n"
	if got := r.KeyValues(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// An empty value keeps its line: the workflow reads every name, and a missing
// line would read as unset rather than empty.
func TestResult_KeyValuesKeepEveryLineWhenEmpty(t *testing.T) {
	want := "first=\nsecond=\nreviewer=\nroute=\nnan_model=\nnan_fallbacks=[]\nanthropic_model=\nanthropic_effort=\nnote=\n"
	if got := (Result{}).KeyValues(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// jq prints the chain compactly and does not escape <, > or &.
func TestResult_KeyValuesPrintTheChainAsJqDoes(t *testing.T) {
	r := Result{NanFallbacks: []string{"a&b", "<c>"}}
	if got := r.KeyValues(); !strings.Contains(got, "nan_fallbacks=[\"a&b\",\"<c>\"]\n") {
		t.Errorf("got %q", got)
	}
}

func TestResult_JSONNamesTheOutputsAndTheContext(t *testing.T) {
	r := Result{First: "nan", Second: "none", Reviewer: "m", Route: "draw", Note: "n", Override: "draw", Pool: "embedded",
		Version: "1.2.3", Members: []string{"m"}}
	js, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(js), &got); err != nil {
		t.Fatalf("%q: %v", js, err)
	}
	for k, want := range map[string]any{"first": "nan", "second": "none", "reviewer": "m", "route": "draw", "nan_model": "",
		"anthropic_model": "", "anthropic_effort": "", "note": "n", "override": "draw", "pool": "embedded", "dotf_version": "1.2.3"} {
		if got[k] != want {
			t.Errorf("%s = %v, want %v", k, got[k], want)
		}
	}
	if strings.Contains(js, "\n") {
		t.Errorf("JSON is not one line: %q", js)
	}
}

// Lists are lists, never null, and an unstamped version is absent.
func TestResult_JSONEmptyListsAndNoVersion(t *testing.T) {
	js, err := (Result{}).JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(js, `"nan_fallbacks":[]`) || !strings.Contains(js, `"members":[]`) || strings.Contains(js, "dotf_version") {
		t.Errorf("got %s", js)
	}
}

func TestResult_SummaryByRoute(t *testing.T) {
	base := Result{Pool: "embedded", Override: "draw", Second: "nan", Reviewer: "anthropic/h", Members: []string{"a", "anthropic/h"}}
	for name, tc := range map[string]struct {
		mod  func(*Result)
		want []string
	}{
		"drawn": {func(r *Result) { r.First, r.Route = "anthropic", "draw" }, []string{
			"### PR-Agent review pool\n\nPool: `embedded`.\n\nOverride: `draw`. Members that answered: `a` `anthropic/h`.\n\n",
			"Drawn to review first: `anthropic/h`. Second attempt, only if nothing is published: nan.\n"}},
		"risk": {func(r *Result) { r.First, r.Route, r.risk = "anthropic", "risk", "1500 changed lines (at least 1500)" }, []string{
			"Routed by risk (1500 changed lines (at least 1500)) to `anthropic/h`. Second attempt, only if nothing is published: nan.\n"}},
		"none": {func(r *Result) { r.First, r.Members = none, nil }, []string{
			"Members that answered: none.\n\nNo member answered, so no review attempt runs.\n"}},
		"notes":   {func(r *Result) { r.First, r.Route, r.Note = "nan", "draw", "why a member left" }, []string{"\n\nwhy a member left\n"}},
		"version": {func(r *Result) { r.First, r.Route, r.Version = "nan", "draw", "1.2.3" }, []string{"Pool: `embedded`. dotf: `1.2.3`.\n\nOverride"}},
	} {
		t.Run(name, func(t *testing.T) {
			r := base
			tc.mod(&r)
			got := r.Summary()
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("summary lacks %q:\n%s", w, got)
				}
			}
		})
	}
	if s := (Result{First: "nan", Route: "draw"}).Summary(); strings.Contains(s, "dotf:") {
		t.Errorf("an unstamped version was printed:\n%s", s)
	}
}

func TestResult_NoticeNamesTheReviewerAndThePool(t *testing.T) {
	r := Result{First: "nan", Second: "anthropic", Reviewer: "openai/m", Route: "draw", Override: "nan", Pool: "embedded"}
	want := "::notice::PR-Agent reviews first on openai/m (draw, override: nan, pool: embedded); second attempt: anthropic.\n"
	if got := r.Notice(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if got := (Result{First: none}).Notice(); got != "" {
		t.Errorf("a route with no reviewer announced %q", got)
	}
}
