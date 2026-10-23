package prreview

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// These cases are tests/pr-agent-model-preflight.bats, ported, plus what the
// bats suite could not say: a stub `curl` cannot tell a followed redirect from
// an unfollowed one, or a body read to its end from a status line alone.

// preflight runs the bats suite's chain (mimo primary, deepseek fallback)
// against api, and returns the result, the annotations printed and the error.
func preflight(t *testing.T, api *nanAPI, mutate ...func(*PreflightInputs)) (PreflightResult, string, error) {
	t.Helper()
	in := PreflightInputs{Model: "openai/mimo-v2.6-flash", Fallbacks: `["openai/deepseek-v4-flash"]`,
		Key: nanKey, APIBase: api.URL, Timeout: "1"}
	for _, m := range mutate {
		m(&in)
	}
	var out bytes.Buffer
	res, err := Preflight(t.Context(), in, PreflightEnv{Out: &out})
	return res, out.String(), err
}

func inOrder(t *testing.T, text string, parts ...string) {
	t.Helper()
	rest := text
	for _, p := range parts {
		i := strings.Index(rest, p)
		if i < 0 {
			t.Errorf("%q does not hold %q, in order after %q", text, p, parts[0])
			return
		}
		rest = rest[i+len(p):]
	}
}

func TestPreflight_ThePrimaryAnswersSoItReviewsAndTheFallbackStaysBehindIt(t *testing.T) {
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "200", "deepseek-v4-flash": "200"})
	res, out, err := preflight(t, api)
	if err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if want := "model=openai/mimo-v2.6-flash\nfallbacks=[\"openai/deepseek-v4-flash\"]\n"; res.KeyValues() != want {
		t.Errorf("outputs = %q, want %q", res.KeyValues(), want)
	}
	if strings.Contains(res.KeyValues(), "retry_model=") || strings.Contains(out, "::warning::") {
		t.Errorf("an answered primary warned or wrote retry_model: %q %q", out, res.KeyValues())
	}
}

func TestPreflight_ARetiredPrimaryHandsTheReviewToTheFallbackAndSaysSo(t *testing.T) {
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "401", "deepseek-v4-flash": "200"})
	res, out, err := preflight(t, api)
	if err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if want := "model=openai/deepseek-v4-flash\nfallbacks=[]\n"; res.KeyValues() != want {
		t.Errorf("outputs = %q, want %q", res.KeyValues(), want)
	}
	inOrder(t, out, "::warning::PR-Agent model openai/mimo-v2.6-flash answered HTTP 401",
		"::warning::PR-Agent reviews this PR with openai/deepseek-v4-flash, not the declared primary openai/mimo-v2.6-flash.\n")
}

func TestPreflight_APrimaryThatHangsIsSkippedLikeOneThatRefuses(t *testing.T) {
	for _, mode := range []string{"hang", "stall"} {
		t.Run(mode, func(t *testing.T) {
			api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": mode, "deepseek-v4-flash": "200"})
			start := time.Now()
			res, out, err := preflight(t, api, func(in *PreflightInputs) { in.Timeout = "0.3" })
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			if res.Model != "openai/deepseek-v4-flash" {
				t.Errorf("model = %q", res.Model)
			}
			inOrder(t, out, "::warning::PR-Agent model openai/mimo-v2.6-flash gave no answer within 0.3s and is skipped for this run: NaN did not serve it")
			if time.Since(start) > 3*time.Second {
				t.Errorf("the probe outlived its timeout: %s", time.Since(start))
			}
		})
	}
}

// 401/403/404: the key cannot call it, so the chain is wrong. 402/429: quota or
// NaN's per-model concurrency (measured in #1107), where editing the chain is the
// wrong advice. Anything else, and a hang: NaN did not answer this time.
func TestPreflight_TheWarningNamesTheFixByStatusClassNotOneRemedyForAll(t *testing.T) {
	const fix, wait, outage = "this key cannot call it. Fix the chain", "quota or concurrency limit reached", "NaN did not serve it"
	for code, want := range map[string]string{"401": fix, "403": fix, "404": fix, "402": wait, "429": wait,
		"500": outage, "503": outage, "400": outage, "302": outage} {
		api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": code, "deepseek-v4-flash": "200"})
		_, out, err := preflight(t, api)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		inOrder(t, out, "openai/mimo-v2.6-flash answered HTTP "+code+" and is skipped for this run: "+want)
		for _, other := range []string{fix, wait, outage} {
			if other != want && strings.Contains(out, other) {
				t.Errorf("%s: the warning also says %q: %s", code, other, out)
			}
		}
	}
}

func TestPreflight_AnyTwoHundredSeriesStatusIsAnAnswer(t *testing.T) {
	for _, code := range []string{"200", "201", "204", "299"} {
		api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": code})
		res, _, err := preflight(t, api)
		if err != nil || res.Model != "openai/mimo-v2.6-flash" {
			t.Errorf("%s: model %q, err %v", code, res.Model, err)
		}
	}
}

func TestPreflight_WhenNoDeclaredModelAnswersNaNLeavesThePool(t *testing.T) {
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "401", "deepseek-v4-flash": "429"})
	res, out, err := preflight(t, api)
	if !errors.Is(err, ErrNoModelAnswered) {
		t.Fatalf("err = %v, want ErrNoModelAnswered", err)
	}
	const want = "::warning::No model in PR-Agent's declared NaN chain answered, so NaN is out of the review pool for this run." +
		" The warnings above give each model's status and what to do about it.\n"
	if !strings.HasSuffix(out, want) {
		t.Errorf("output = %q", out)
	}
	if res.Model != "" || len(res.Probes) != 2 {
		t.Errorf("result = %+v", res)
	}
	if s := res.Summary(); !strings.Contains(s, "No declared model answered, so NaN is out of the review pool for this run.") {
		t.Errorf("summary = %q", s)
	}
}

func TestPreflight_ItNeverPicksAModelOutsideTheDeclaredChain(t *testing.T) {
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "401", "deepseek-v4-flash": "401",
		"glm5.3-flash": "200", "qwen3.6": "200"})
	res, _, err := preflight(t, api)
	if !errors.Is(err, ErrNoModelAnswered) {
		t.Fatalf("err = %v", err)
	}
	if got := api.probed(); got != "mimo-v2.6-flash deepseek-v4-flash" {
		t.Errorf("probed %q", got)
	}
	if strings.Contains(res.KeyValues(), "glm5.3-flash") {
		t.Errorf("outputs = %q", res.KeyValues())
	}
}

func TestPreflight_ItProbesTheBareIdAndHandsPRAgentThePrefixedOne(t *testing.T) {
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "200", "deepseek-v4-flash": "200"})
	res, _, err := preflight(t, api)
	if err != nil {
		t.Fatal(err)
	}
	if res.Model != "openai/mimo-v2.6-flash" {
		t.Errorf("model = %q", res.Model)
	}
	reqs := api.requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests", len(reqs))
	}
	for _, r := range reqs {
		if strings.Contains(r.Raw, `"openai/`) {
			t.Errorf("a probe sent the prefixed id: %s", r.Raw)
		}
	}
	// The request, byte for byte: 512 tokens because a reasoning model thinks
	// before it answers.
	want := `{"model":"mimo-v2.6-flash","max_tokens":512,"messages":[{"role":"user","content":"Reply with OK."}]}`
	if r := reqs[0]; r.Raw != want || r.Path != "/chat/completions" || r.ContentType != "application/json" {
		t.Errorf("request %s %s %s", r.Path, r.ContentType, r.Raw)
	}
}

func TestPreflight_TheChainIsProbedInDeclaredOrderAndAnsweringModelsKeepIt(t *testing.T) {
	api := newNanAPI(t, map[string]string{"a": "200", "b": "503", "c": "200", "d": "200"})
	res, _, err := preflight(t, api, func(in *PreflightInputs) {
		in.Model, in.Fallbacks = "openai/a", `["openai/b","","openai/c", "openai/d"]`
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := api.probed(); got != "a b c d" {
		t.Errorf("probed %q", got)
	}
	if want := "model=openai/a\nfallbacks=[\"openai/c\",\"openai/d\"]\n"; res.KeyValues() != want {
		t.Errorf("outputs = %q", res.KeyValues())
	}
}

// The key is one request header's value. It is in no URL, no body, no other
// header, and in nothing Preflight prints, returns or marshals.
func TestPreflight_TheKeyIsOnlyInTheAuthorizationHeader(t *testing.T) {
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "401", "deepseek-v4-flash": "200"})
	res, out, err := preflight(t, api)
	if err != nil {
		t.Fatal(err)
	}
	js, _ := res.JSON()
	seen := res.KeyValues() + res.Summary() + js + out
	_, out2, err2 := preflight(t, newNanAPI(t, nil), func(in *PreflightInputs) { in.APIBase = "http://127.0.0.1:1" })
	seen += out2 + err2.Error()
	_, err3 := Preflight(t.Context(), PreflightInputs{Model: "m", Fallbacks: "not json", Key: nanKey}, PreflightEnv{})
	seen += err3.Error()
	if strings.Contains(seen, nanKey) {
		t.Errorf("the key is in what Preflight reports: %q", seen)
	}
	for _, r := range api.requests() {
		if r.Auth != "Bearer "+nanKey {
			t.Errorf("Authorization = %q", r.Auth)
		}
		if strings.Contains(r.Path, nanKey) || strings.Contains(r.Raw, nanKey) {
			t.Errorf("the key is in the URL or body: %s %s", r.Path, r.Raw)
		}
		for name, vals := range r.Header {
			if name != "Authorization" && strings.Contains(strings.Join(vals, ","), nanKey) {
				t.Errorf("the key is in header %s", name)
			}
		}
	}
}

// A redirect is a status to report, not a hop to follow: the key would travel to
// wherever it points. Holds for a client that follows redirects, as the default one does.
func TestPreflight_DoesNotFollowARedirectEvenWithAClientThatFollowsThem(t *testing.T) {
	elsewhere := newNanAPI(t, map[string]string{"": "200"})
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "redirect", "deepseek-v4-flash": "200"})
	api.redirect(elsewhere.URL + "/stolen")
	follower := &http.Client{}
	var out bytes.Buffer
	res, err := Preflight(t.Context(), PreflightInputs{Model: "openai/mimo-v2.6-flash", Fallbacks: `["openai/deepseek-v4-flash"]`,
		Key: nanKey, APIBase: api.URL, Timeout: "1"}, PreflightEnv{Out: &out, Client: follower})
	if err != nil {
		t.Fatal(err)
	}
	if res.Model != "openai/deepseek-v4-flash" || !strings.Contains(out.String(), "answered HTTP 307") {
		t.Errorf("model %q, output %q", res.Model, out.String())
	}
	if n := len(elsewhere.requests()); n != 0 {
		t.Errorf("the redirect was followed: %d requests reached the target, carrying %v", n, elsewhere.requests()[0].Header)
	}
	if follower.CheckRedirect != nil {
		t.Error("Preflight changed the caller's client")
	}
}

func TestPreflight_TheSummaryNamesTheModelThatReviewsAndEveryOneThatDidNotAnswer(t *testing.T) {
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "401", "deepseek-v4-flash": "200"})
	res, _, err := preflight(t, api)
	if err != nil {
		t.Fatal(err)
	}
	want := "### PR-Agent model preflight\n\n| Model | Result |\n|---|---|\n" +
		"| `openai/mimo-v2.6-flash` | HTTP 401 |\n| `openai/deepseek-v4-flash` | answered (HTTP 200) |\n\n" +
		"Reviewing with `openai/deepseek-v4-flash`.\n"
	if res.Summary() != want {
		t.Errorf("summary = %q\nwant      %q", res.Summary(), want)
	}
	api = newNanAPI(t, map[string]string{"mimo-v2.6-flash": "hang", "deepseek-v4-flash": "200"})
	res, _, _ = preflight(t, api, func(in *PreflightInputs) { in.Timeout = "0.2" })
	if !strings.Contains(res.Summary(), "| `openai/mimo-v2.6-flash` | no answer within 0.2s |") {
		t.Errorf("summary = %q", res.Summary())
	}
}

func TestPreflight_JSONCarriesTheOutputsAndEachProbe(t *testing.T) {
	api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": "429", "deepseek-v4-flash": "200"})
	res, _, err := preflight(t, api)
	if err != nil {
		t.Fatal(err)
	}
	js, err := res.JSON()
	if err != nil {
		t.Fatal(err)
	}
	want := `{"model":"openai/deepseek-v4-flash","fallbacks":[],"probes":[{"model":"openai/mimo-v2.6-flash","status":"429"},{"model":"openai/deepseek-v4-flash","status":"200"}]}`
	if js != want {
		t.Errorf("json = %s\nwant   %s", js, want)
	}
}

// A missing key, model or chain is a usage error, not a model failure; nothing
// is probed.
func TestPreflight_ASetupErrorIsNotAModelFailure(t *testing.T) {
	good := PreflightInputs{Model: "openai/x", Fallbacks: "[]", Key: nanKey}
	for name, tc := range map[string]struct {
		mutate func(*PreflightInputs)
		want   string
	}{
		"no key":         {func(in *PreflightInputs) { in.Key = "" }, "NAN_API_KEY is not set"},
		"no model":       {func(in *PreflightInputs) { in.Model = "" }, "usage:"},
		"no fallbacks":   {func(in *PreflightInputs) { in.Fallbacks = "" }, "usage:"},
		"not json":       {func(in *PreflightInputs) { in.Fallbacks = "not json" }, "--fallbacks must be a JSON array of model ids, got: not json"},
		"an object":      {func(in *PreflightInputs) { in.Fallbacks = `{"a":1}` }, "--fallbacks must be a JSON array"},
		"a string":       {func(in *PreflightInputs) { in.Fallbacks = `"x"` }, "--fallbacks must be a JSON array"},
		"a number":       {func(in *PreflightInputs) { in.Fallbacks = `5` }, "--fallbacks must be a JSON array"},
		"non-string ids": {func(in *PreflightInputs) { in.Fallbacks = `[1]` }, "--fallbacks must be a JSON array"},
	} {
		t.Run(name, func(t *testing.T) {
			api := newNanAPI(t, nil)
			in := good
			in.APIBase = api.URL
			tc.mutate(&in)
			var setup PreflightSetupError
			_, err := Preflight(t.Context(), in, PreflightEnv{})
			if !errors.As(err, &setup) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want a setup error mentioning %q", err, tc.want)
			}
			if n := len(api.requests()); n != 0 {
				t.Errorf("%d probes were sent", n)
			}
		})
	}
}

func TestPreflight_ABlankTimeoutIsNinetySecondsAndAnUnreadableOneIsToo(t *testing.T) {
	for raw, want := range map[string]time.Duration{"": 90 * time.Second, "2": 2 * time.Second, "0.5": 500 * time.Millisecond,
		"0": 90 * time.Second, "-3": 90 * time.Second, "abc": 90 * time.Second,
		// Out of time.Duration's range: the conversion would wrap or be undefined.
		"1e10": 90 * time.Second, "Inf": 90 * time.Second, "NaN": 90 * time.Second} {
		if got, _ := preflightTimeout(raw); got != want {
			t.Errorf("preflightTimeout(%q) = %s, want %s", raw, got, want)
		}
	}
}
