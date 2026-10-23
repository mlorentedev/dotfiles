package prreview

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Ported from tests/pr-agent-route.bats: each case keeps the script's name and
// its assertions, with the stubbed curl replaced by a fake Anthropic API.

func TestRoute_EachMemberIsDrawnAtItsOwnPointsWithTheOtherProviderSecond(t *testing.T) {
	f := newFixture(t)
	r, _ := f.at(0).ok()
	want := Result{
		First: "nan", Second: "anthropic", Reviewer: "openai/mimo-v2.6-flash", Route: "draw",
		NanModel: "openai/mimo-v2.6-flash", NanFallbacks: []string{"openai/deepseek-v4-flash"},
		AnthropicModel: "anthropic/claude-haiku-5-5", AnthropicEffort: "high",
	}
	assertOutputs(t, r, want)

	r, _ = f.at(3).ok()
	// The drawn NaN model goes first; the other stays behind it.
	assertOutputs(t, r, Result{
		First: "nan", Second: "anthropic", Reviewer: "openai/deepseek-v4-flash", Route: "draw",
		NanModel: "openai/deepseek-v4-flash", NanFallbacks: []string{"openai/mimo-v2.6-flash"},
		AnthropicModel: "anthropic/claude-haiku-5-5", AnthropicEffort: "high",
	})

	r, _ = f.at(5).ok()
	// NaN is the second attempt, in declared order.
	assertOutputs(t, r, Result{
		First: "anthropic", Second: "nan", Reviewer: "anthropic/claude-haiku-5-5", Route: "draw",
		NanModel: "openai/mimo-v2.6-flash", NanFallbacks: []string{"openai/deepseek-v4-flash"},
		AnthropicModel: "anthropic/claude-haiku-5-5", AnthropicEffort: "high",
	})
}

// assertOutputs compares the route's outputs, not the context kept for the summary.
func assertOutputs(t *testing.T, got, want Result) {
	t.Helper()
	if want.NanFallbacks == nil {
		want.NanFallbacks = []string{}
	}
	if got.NanFallbacks == nil {
		got.NanFallbacks = []string{}
	}
	g, w := got.KeyValues(), want.KeyValues()
	if g != w {
		t.Errorf("outputs differ\n got: %q\nwant: %q", g, w)
	}
}

func TestRoute_AcrossEveryPointEachMemberIsDrawnExactlyItsWeight(t *testing.T) {
	got := newFixture(t).everyPoint(10)
	want := map[string]int{"anthropic/claude-haiku-5-5": 5, "openai/deepseek-v4-flash": 2, "openai/mimo-v2.6-flash": 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("drawn %v, want %v", got, want)
	}
}

func TestRoute_AMemberThatDidNotAnswerGivesUpItsPoints(t *testing.T) {
	f := newFixture(t)
	f.in.NanFallbacks = "[]"
	got := f.everyPoint(8)
	want := map[string]int{"anthropic/claude-haiku-5-5": 5, "openai/mimo-v2.6-flash": 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("drawn %v, want %v", got, want)
	}
	if _, _, err := f.at(8).run(); !isUsage(err) {
		t.Errorf("a point past the total weight must be a usage error, got %v", err)
	}
}

func TestRoute_ANaNModelWithNoWeightIsNeverDrawnFirstButStaysInTheChain(t *testing.T) {
	f := newFixture(t)
	f.in.NanFallbacks = `["openai/deepseek-v4-flash", "openai/glm5.3-flash"]`
	got := f.everyPoint(10)
	want := map[string]int{"anthropic/claude-haiku-5-5": 5, "openai/deepseek-v4-flash": 2, "openai/mimo-v2.6-flash": 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("drawn %v, want %v", got, want)
	}
	r, _ := f.at(0).ok()
	if !reflect.DeepEqual(r.NanFallbacks, []string{"openai/deepseek-v4-flash", "openai/glm5.3-flash"}) {
		t.Errorf("nan_fallbacks = %v", r.NanFallbacks)
	}
}

func TestRoute_APointOutsideTheTotalWeightIsAUsageErrorNeverASilentPick(t *testing.T) {
	for _, draw := range []string{"10", "x", "-1", "99999999999999999999999"} {
		f := newFixture(t)
		f.in.Draw = draw
		_, _, err := f.run()
		if !isUsage(err) {
			t.Errorf("PR_AGENT_DRAW=%q: want a usage error, got %v", draw, err)
		}
	}
}

func isUsage(err error) bool {
	var u UsageError
	return errors.As(err, &u)
}

func TestRoute_TheRandomDrawReachesEveryMemberTheHeaviestMost(t *testing.T) {
	f := newFixture(t)
	counts := map[string]int{}
	for range 300 {
		r, _ := f.ok()
		counts[r.Reviewer]++
	}
	if len(counts) != 3 {
		t.Fatalf("reached %v", counts)
	}
	// Expected 90, 60 and 150; 30 is more than four standard deviations below
	// the smallest, and haiku leads deepseek by about seven.
	for m, n := range counts {
		if n < 30 {
			t.Errorf("%s drawn %d times of 300", m, n)
		}
	}
	if counts["anthropic/claude-haiku-5-5"] <= counts["openai/deepseek-v4-flash"] {
		t.Errorf("the heaviest member was not drawn most: %v", counts)
	}
}

// shippedFixture routes the pool dotf ships, with the NaN chain it declares.
func shippedFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	p, err := LoadPool(EmbeddedSource)
	if err != nil {
		t.Fatal(err)
	}
	f.pool, f.src = p, EmbeddedSource
	var nan []string
	for _, e := range p.Pool {
		if e.PRAgent != nil && strings.HasPrefix(e.PRAgent.Model, "openai/") {
			nan = append(nan, e.PRAgent.Model)
		}
	}
	f.in.NanModel = nan[0]
	f.in.NanFallbacks = `["` + strings.Join(nan[1:], `","`) + `"]`
	return f
}

// The shipped pool, not the fixture: every point of its total weight draws the
// member that owns it, so the shares in the pool are the shares CI runs.
func TestRoute_TheShippedPoolDrawsEachMemberExactlyItsDeclaredWeight(t *testing.T) {
	f := shippedFixture(t)
	want, total := map[string]int{}, 0
	for _, e := range f.pool.Pool {
		if e.PRAgent != nil && e.PRAgent.Weight > 0 {
			want[e.PRAgent.Model] = e.PRAgent.Weight
			total += e.PRAgent.Weight
		}
	}
	if got := f.everyPoint(total); !reflect.DeepEqual(got, want) {
		t.Errorf("drawn   %v\nweights %v", got, want)
	}
}

// The shares themselves are the budget (ADR-046 decision 3 names them), so the
// shipped weights must be the ADR's: a share changed in the pool without an
// amendment to the ADR fails here, and the test above only proves the draw
// honours whatever the pool says.
func TestRoute_TheShippedWeightsAreTheOnesTheADRAdopted(t *testing.T) {
	adrs, _ := filepath.Glob(repoFile("docs", "adr", "adr-046-*.md"))
	if len(adrs) != 1 {
		t.Skipf("docs/adr/adr-046-*.md is not here: %v", adrs)
	}
	raw, err := os.ReadFile(adrs[0])
	if err != nil {
		t.Fatal(err)
	}
	line := regexp.MustCompile(`(?m)^\s*Weights at adoption: (.*)$`).FindStringSubmatch(string(raw))
	if line == nil {
		t.Fatal("ADR-046 states no 'Weights at adoption:' line")
	}
	adopted := map[string]int{}
	for _, item := range strings.Split(strings.ReplaceAll(line[1], ".", ""), ",") {
		if f := strings.Fields(item); len(f) == 2 {
			adopted[strings.ToLower(f[0])], _ = strconv.Atoi(f[1])
		}
	}
	shipped := map[string]int{}
	for _, e := range shippedFixture(t).pool.Pool {
		if e.PRAgent == nil || e.PRAgent.Weight == 0 {
			continue
		}
		for _, short := range []string{"mimo", "glm", "deepseek", "haiku"} {
			if strings.Contains(e.PRAgent.Model, short) {
				shipped[short] = e.PRAgent.Weight
			}
		}
	}
	if len(adopted) != 4 || !maps.Equal(shipped, adopted) {
		t.Errorf("shipped %v, ADR-046 %v", shipped, adopted)
	}
}

// The shipped risk line is the budget lever (ADR-046): 1,500 changed lines put
// about 3% of PRs on Sonnet. Moving it moves the month's spend, so the number is
// pinned here and the route is driven against the shipped pool on both sides.
func TestRoute_TheShippedPoolRoutesToSonnetAt1500ChangedLinesAndNotBelow(t *testing.T) {
	f := shippedFixture(t)
	if f.pool.PRAgentRisk == nil || *f.pool.PRAgentRisk.MinChangedLines != 1500 {
		t.Fatalf("the shipped risk line is %+v, want 1500", f.pool.PRAgentRisk)
	}
	f.in.ChangedLines = "1500"
	r, _ := f.at(0).ok()
	if r.Route != "risk" || r.Reviewer != "anthropic/claude-sonnet-5-5" {
		t.Errorf("at 1500: %s %s", r.Route, r.Reviewer)
	}
	f.in.ChangedLines = "1499"
	if r, _ = f.at(0).ok(); r.Route != "draw" {
		t.Errorf("at 1499: %s", r.Route)
	}
}

// Every fixture member declares its effort, so without this the default was
// never exercised (#2529): a member that declares none runs at medium.
func TestRoute_AnAnthropicMemberWithNoDeclaredEffortRunsAtMedium(t *testing.T) {
	f := newFixture(t)
	f.pool = mustPool(t, `{"pool": [
 {"id": "nan/mimo-v2.6-flash", "pr_agent": {"model": "openai/mimo-v2.6-flash", "weight": 5}},
 {"id": "anthropic-review/claude-haiku-5-5", "signs": "fallback", "pr_agent": {"model": "anthropic/claude-haiku-5-5", "weight": 5}}
]}`)
	r, _ := f.at(5).ok()
	if r.AnthropicModel != "anthropic/claude-haiku-5-5" || r.AnthropicEffort != "medium" {
		t.Errorf("model %q effort %q", r.AnthropicModel, r.AnthropicEffort)
	}
}

// --- the risk route ---

func TestRoute_APRAtTheLineThresholdReviewsFirstOnTheRiskMemberNaNSecond(t *testing.T) {
	f := newFixture(t)
	f.in.ChangedLines = "900"
	r, _ := f.at(0).ok()
	assertOutputs(t, r, Result{
		First: "anthropic", Second: "nan", Reviewer: "anthropic/claude-sonnet-5-5", Route: "risk",
		NanModel: "openai/mimo-v2.6-flash", NanFallbacks: []string{"openai/deepseek-v4-flash"},
		AnthropicModel: "anthropic/claude-sonnet-5-5", AnthropicEffort: "medium",
	})
	if !strings.Contains(f.api.probedModels(), "claude-sonnet-5-5") {
		t.Errorf("the risk member was not probed: %s", f.api.probedModels())
	}
	f.in.ChangedLines = "899"
	if r, _ = f.at(0).ok(); r.Route != "draw" || r.Reviewer != "openai/mimo-v2.6-flash" {
		t.Errorf("at 899: %s %s", r.Route, r.Reviewer)
	}
}

func TestRoute_TheLabelRoutesASmallPRToTheRiskMemberAndOnlyTheExactLabel(t *testing.T) {
	f := newFixture(t)
	f.in.ChangedLines, f.in.Labels = "3", `["docs", "deep-review"]`
	r, _ := f.at(0).ok()
	if r.Route != "risk" || r.Reviewer != "anthropic/claude-sonnet-5-5" {
		t.Errorf("label: %s %s", r.Route, r.Reviewer)
	}
	f.in.Labels = `["deep-review-later"]`
	if r, _ = f.at(0).ok(); r.Route != "draw" {
		t.Errorf("a longer label routed to %s", r.Route)
	}
}

func TestRoute_TheRiskMemberIsProbedOnlyForARiskyPR(t *testing.T) {
	f := newFixture(t)
	f.in.ChangedLines = "10"
	f.at(0).ok()
	if strings.Contains(f.api.probedModels(), "sonnet") {
		t.Errorf("probed %s", f.api.probedModels())
	}
}

func TestRoute_ARiskMemberThatDoesNotAnswerSendsThePRToTheDrawWithANote(t *testing.T) {
	f := newFixture(t)
	f.api.answer("claude-sonnet-5-5", "529")
	f.in.ChangedLines = "5000"
	r, _ := f.at(5).ok()
	if r.Route != "draw" || r.Reviewer != "anthropic/claude-haiku-5-5" || r.AnthropicModel != "anthropic/claude-haiku-5-5" {
		t.Errorf("route %s reviewer %s anthropic %s", r.Route, r.Reviewer, r.AnthropicModel)
	}
	if !regexp.MustCompile(`crossed the risk line \(5000 changed lines.*drawn by weight instead`).MatchString(r.Note) {
		t.Errorf("note = %q", r.Note)
	}
}

func TestRoute_AnUnreadSizeIsNamedAndTheLabelStillRoutes(t *testing.T) {
	f := newFixture(t)
	f.in.ChangedLines = ""
	r, _ := f.at(0).ok()
	if r.Route != "draw" || !strings.Contains(r.Note, "size could not be read") {
		t.Errorf("route %s note %q", r.Route, r.Note)
	}
	f.in.Labels = `["deep-review"]`
	if r, _ = f.at(0).ok(); r.Route != "risk" {
		t.Errorf("route %s", r.Route)
	}
}

func TestRoute_NaNAndNaNOnlyNeverTakeTheRiskRoute(t *testing.T) {
	f := newFixture(t)
	f.in.Provider, f.in.ChangedLines = "nan", "5000"
	if r, _ := f.at(0).ok(); r.First != "nan" || r.Route != "draw" {
		t.Errorf("nan: %s %s", r.First, r.Route)
	}
	before := len(f.api.probed())
	f.in.Provider = "nan-only"
	if r, _ := f.at(0).ok(); r.First != "nan" {
		t.Errorf("nan-only: %s", r.First)
	}
	if len(f.api.probed()) != before {
		t.Error("nan-only probed Anthropic")
	}
}

// --- the allowlist ---

func TestRoute_APoolNamingAnAnthropicModelOutsideTheAllowlistFailsBeforeAnyProbe(t *testing.T) {
	f := newFixture(t)
	f.pool.Pool[3].PRAgent.Model = "anthropic/claude-opus-5-5"
	_, out, err := f.at(0).run()
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("want ErrRefused, got %v", err)
	}
	if !regexp.MustCompile(`::error::.*anthropic/claude-opus-5-5.*allowlist`).MatchString(out) {
		t.Errorf("annotation = %q", out)
	}
	if n := len(f.api.probed()); n != 0 {
		t.Errorf("%d probes before the refusal", n)
	}
}

// --- members out of the draw ---

func TestRoute_AnAnthropicMemberThatDoesNotAnswerIsNeverDrawn(t *testing.T) {
	for _, mode := range []string{"401", "404", "429", "529", "hang"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			f.api.answer("claude-haiku-5-5", mode)
			r, out := f.at(3).ok()
			if r.First != "nan" || r.Second != "none" || r.AnthropicModel != "" {
				t.Errorf("first %s second %s anthropic %q", r.First, r.Second, r.AnthropicModel)
			}
			if !regexp.MustCompile(`::warning::.*claude-haiku-5-5`).MatchString(out) || r.Note == "" {
				t.Errorf("annotations %q note %q", out, r.Note)
			}
			// Only NaN's five points remain.
			if _, _, err := f.at(5).run(); !isUsage(err) {
				t.Errorf("point 5 must be outside the total weight, got %v", err)
			}
		})
	}
}

func TestRoute_TheWarningNamesTheFixByStatusClass(t *testing.T) {
	for code, want := range map[string][]string{
		"401": {"key is refused", "dotf secrets rotate PR_AGENT_ANTHROPIC_API_KEY"},
		"403": {"key is refused"},
		"404": {"model id is wrong", "reviewer-pool.json"},
		"429": {"rate limit, overload or spend limit"},
		"301": {"answered HTTP 301 (rate limit, overload or spend limit)"},
	} {
		f := newFixture(t)
		f.api.answer("claude-haiku-5-5", code)
		r, _ := f.at(0).ok()
		for _, w := range want {
			if !strings.Contains(r.Note, w) {
				t.Errorf("HTTP %s: note %q lacks %q", code, r.Note, w)
			}
		}
	}
}

func TestRoute_NoKeyMeansNoProbeAndNoAnthropicMemberWithTheRemedy(t *testing.T) {
	f := newFixture(t)
	f.in.AnthropicKey, f.in.ChangedLines = "", "5000"
	r, _ := f.at(0).ok()
	if r.Second != "none" || r.Route != "draw" || len(f.api.probed()) != 0 {
		t.Errorf("second %s route %s probes %d", r.Second, r.Route, len(f.api.probed()))
	}
	if !strings.Contains(r.Note, "dotf secrets sync ci --repo o/r PR_AGENT_ANTHROPIC_API_KEY") {
		t.Errorf("note = %q", r.Note)
	}
}

func TestRoute_NaNDownAsAWholeLeavesAnthropicAloneWithNoSecondAttempt(t *testing.T) {
	f := newFixture(t)
	f.in.NanOutcome, f.in.NanModel, f.in.NanFallbacks = "failure", "", ""
	r, _ := f.at(0).ok()
	if r.First != "anthropic" || r.Second != "none" || r.NanModel != "" {
		t.Errorf("first %s second %s nan %q", r.First, r.Second, r.NanModel)
	}
}

func TestRoute_APreflightThatFailedOnItsSetupStopsTheDrawInsteadOfPayingForEveryReview(t *testing.T) {
	f := newFixture(t)
	f.in.NanOutcome, f.in.NanModel, f.in.NanFallbacks, f.in.PreflightExit = "failure", "", "", "2"
	_, out, err := f.run()
	if !errors.Is(err, ErrRefused) || !strings.Contains(out, "::error::The NaN preflight failed on its setup (exit 2)") {
		t.Errorf("err %v, annotations %q", err, out)
	}
	if len(f.api.probed()) != 0 {
		t.Error("probed before the refusal")
	}
}

func TestRoute_APreflightInWhichNoModelAnsweredStillHandsOverToAnthropic(t *testing.T) {
	f := newFixture(t)
	f.in.NanOutcome, f.in.NanModel, f.in.NanFallbacks, f.in.PreflightExit = "failure", "", "", "1"
	if r, _ := f.at(0).ok(); r.First != "anthropic" {
		t.Errorf("first = %s", r.First)
	}
}

func TestRoute_NoMemberAtAllAnswersFirstNoneAndTheJobSummarySaysSo(t *testing.T) {
	f := newFixture(t)
	f.in.NanOutcome, f.in.NanModel = "failure", ""
	f.api.answer("", "529")
	r, _ := f.ok()
	if r.First != "none" || r.Second != "none" || !strings.Contains(r.Summary(), "No member answered") {
		t.Errorf("first %s second %s\n%s", r.First, r.Second, r.Summary())
	}
	if r.Notice() != "" {
		t.Errorf("a route with no reviewer announced %q", r.Notice())
	}
}

// --- the override ---

func TestRoute_NaNDrawsAmongNaNOnlyWithAnthropicSecond(t *testing.T) {
	f := newFixture(t)
	f.in.Provider = "nan"
	if r, _ := f.at(3).ok(); r.Reviewer != "openai/deepseek-v4-flash" || r.Second != "anthropic" {
		t.Errorf("reviewer %s second %s", r.Reviewer, r.Second)
	}
	if _, _, err := f.at(5).run(); !isUsage(err) {
		t.Errorf("point 5 must be outside the NaN total, got %v", err)
	}
}

func TestRoute_AnthropicPutsAnthropicFirstNaNSecond(t *testing.T) {
	f := newFixture(t)
	f.in.Provider = "anthropic"
	if r, _ := f.at(0).ok(); r.First != "anthropic" || r.Second != "nan" {
		t.Errorf("first %s second %s", r.First, r.Second)
	}
}

func TestRoute_NaNOnlyNeverProbesOrSchedulesAnthropic(t *testing.T) {
	f := newFixture(t)
	f.in.Provider = "nan-only"
	r, _ := f.at(0).ok()
	if r.First != "nan" || r.Second != "none" || len(f.api.probed()) != 0 {
		t.Errorf("first %s second %s probes %d", r.First, r.Second, len(f.api.probed()))
	}
	// Even with NaN down, the switch holds: no review rather than spending.
	f.in.NanOutcome, f.in.NanModel = "failure", ""
	if r, _ = f.ok(); r.First != "none" || len(f.api.probed()) != 0 {
		t.Errorf("first %s probes %d", r.First, len(f.api.probed()))
	}
}

func TestRoute_AForcedProviderWithNoMemberThatAnsweredDrawsFromTheRestLoudly(t *testing.T) {
	f := newFixture(t)
	f.api.answer("", "529")
	f.in.Provider = "anthropic"
	r, out := f.at(0).ok()
	if r.First != "nan" || !strings.Contains(out, "::warning::PR_AGENT_PROVIDER=anthropic") {
		t.Errorf("first %s annotations %q", r.First, out)
	}
}

func TestRoute_AnUnknownOverrideFailsBeforeAnyProbe(t *testing.T) {
	f := newFixture(t)
	f.in.Provider = "nan_only"
	_, out, err := f.run()
	if !errors.Is(err, ErrRefused) || !strings.Contains(out, "::error::PR_AGENT_PROVIDER is nan_only") {
		t.Errorf("err %v, annotations %q", err, out)
	}
	if len(f.api.probed()) != 0 {
		t.Error("probed before the refusal")
	}
}

// --- the key, the request, the outputs ---

// The script gave the key to curl on stdin so it never reached an argv. There is
// no child process here: the key goes into one request header and into nothing
// else, and no output, error or JSON may carry it.
func TestRoute_TheKeyTravelsInTheHeaderAndAppearsInNoOutput(t *testing.T) {
	f := newFixture(t)
	f.in.ChangedLines = "5000"
	r, out := f.at(0).ok()
	for _, req := range f.api.probed() {
		if req.Key != testKey {
			t.Errorf("probe sent key %q", req.Key)
		}
	}
	js, _ := r.JSON()
	for name, text := range map[string]string{"annotations": out, "outputs": r.KeyValues(), "summary": r.Summary(), "notice": r.Notice(), "json": js} {
		if strings.Contains(text, "sk-ant") {
			t.Errorf("%s carries the key: %q", name, text)
		}
	}
	// A probe that fails must not put the key in its error path either.
	f.api.answer("", "401")
	_, out, _ = f.run()
	if strings.Contains(out, "sk-ant") {
		t.Errorf("a refused probe printed the key: %q", out)
	}
}

func TestRoute_TheProbeAsksForThePoolsModelBrieflyOnTheMessagesAPI(t *testing.T) {
	f := newFixture(t)
	f.at(0).ok()
	reqs := f.api.probed()
	if len(reqs) != 1 {
		t.Fatalf("%d probes", len(reqs))
	}
	r := reqs[0]
	if r.Path != "/v1/messages" || r.Version != "2023-06-01" || r.ContentType != "application/json" {
		t.Errorf("request %+v", r)
	}
	if r.Body["model"] != "claude-haiku-5-5" || r.Body["max_tokens"] != float64(64) {
		t.Errorf("body %v", r.Body)
	}
}

func TestRoute_TheJobSummaryNamesTheMemberDrawnTheRiskRouteTheSecondAttemptAndThePool(t *testing.T) {
	f := newFixture(t)
	r, _ := f.at(5).ok()
	r.Version = "1.2.3"
	s := r.Summary()
	for _, want := range []string{"Drawn to review first: `anthropic/claude-haiku-5-5`", "nothing is published: nan.",
		"Pool: `fixture`. dotf: `1.2.3`.", "\n\nOverride: `draw`. Members that answered: `openai/mimo-v2.6-flash` `openai/deepseek-v4-flash` `anthropic/claude-haiku-5-5`.\n\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("summary lacks %q:\n%s", want, s)
		}
	}
	f.in.Labels = `["deep-review"]`
	r, _ = f.ok()
	if !strings.Contains(r.Summary(), "Routed by risk (the deep-review label) to `anthropic/claude-sonnet-5-5`") {
		t.Errorf("summary:\n%s", r.Summary())
	}
	if n := r.Notice(); !strings.Contains(n, "::notice::") || !strings.Contains(n, "pool: fixture") {
		t.Errorf("notice = %q", n)
	}
}

// --- ported from tests/pr-agent-route-real.bats: a real HTTP exchange ---

func TestRoute_ARefusalComesBackAsItsStatus(t *testing.T) {
	f := newFixture(t)
	f.api.answer("", "401")
	r, _ := f.at(0).ok()
	if r.Second != "none" || !strings.Contains(r.Note, "answered HTTP 401") {
		t.Errorf("second %s note %q", r.Second, r.Note)
	}
}

func TestRoute_AServerThatNeverAnswersEndsAtTheTimeoutAsNoAnswer(t *testing.T) {
	for _, mode := range []string{"hang", "stall"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			f.api.answer("", mode)
			r, _ := f.at(0).ok()
			if r.Second != "none" || !strings.Contains(r.Note, "no answer within 1s") {
				t.Errorf("second %s note %q", r.Second, r.Note)
			}
		})
	}
}

func TestRoute_ARefusedConnectionIsNoAnswer(t *testing.T) {
	f := newFixture(t)
	f.in.APIBase = "http://127.0.0.1:1"
	r, _ := f.at(0).ok()
	if r.Second != "none" || !strings.Contains(r.Note, "no answer within 1s") {
		t.Errorf("second %s note %q", r.Second, r.Note)
	}
}

// The workflow hands its values over by these names; a rename here would route
// every review on defaults and nothing would fail.
func TestInputsFromEnvReadsTheScriptsVariables(t *testing.T) {
	env := map[string]string{
		"NAN_OUTCOME": "success", "NAN_MODEL": "m", "NAN_FALLBACKS": "[]", "NAN_PREFLIGHT_EXIT": "1",
		"PR_AGENT_ANTHROPIC_API_KEY": "k", "PR_AGENT_PROVIDER": "nan", "PR_CHANGED_LINES": "7", "PR_LABELS": `["x"]`,
		"PR_AGENT_DRAW": "3", "ANTHROPIC_API_BASE": "http://b", "ROUTE_PROBE_TIMEOUT": "5", "GITHUB_REPOSITORY": "o/r",
	}
	got := InputsFromEnv(func(k string) string { return env[k] })
	want := Inputs{NanOutcome: "success", NanModel: "m", NanFallbacks: "[]", PreflightExit: "1", AnthropicKey: "k",
		Provider: "nan", ChangedLines: "7", Labels: `["x"]`, Draw: "3", APIBase: "http://b", ProbeTimeout: "5", Repository: "o/r"}
	if got != want {
		t.Errorf("got %+v", got)
	}
}
