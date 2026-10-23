package prreview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrNoModelAnswered is returned when no model of the declared chain answered.
// NaN is then out of the review pool for this run. The reason has already been
// written to PreflightEnv.Out as a ::warning:: line.
var ErrNoModelAnswered = errors.New("pr preflight: no declared model answered")

// PreflightSetupError is a caller mistake: a missing key, model or chain. It is
// not a verdict on NaN, and a caller must not read it as one.
type PreflightSetupError string

func (e PreflightSetupError) Error() string { return string(e) }

const (
	defaultNaNBase         = "https://api.nan.builders/v1"
	defaultPreflightSecond = 90
)

// PreflightInputs are the preflight's inputs. The environment variables of the
// script this replaces are the raw text of Key (NAN_API_KEY), APIBase
// (NAN_API_BASE, default NaN's) and Timeout (PREFLIGHT_TIMEOUT, seconds, default
// 90).
type PreflightInputs struct {
	Model     string // --model: the declared primary, `openai/<id>`
	Fallbacks string // --fallbacks: JSON array of the other declared models
	Key       string
	APIBase   string
	Timeout   string
}

// PreflightEnv is what the preflight reaches outside itself through.
type PreflightEnv struct {
	Out    io.Writer    // workflow annotations (::warning::)
	Client *http.Client // nil: a client that follows no redirect
}

// ModelStatus is one probe: the declared model and the HTTP status it got, or
// "000" when it gave no answer.
type ModelStatus struct {
	Model  string `json:"model"`
	Status string `json:"status"`
}

// PreflightResult is the models that answered, and every probe made.
type PreflightResult struct {
	Model     string        `json:"model"`
	Fallbacks []string      `json:"fallbacks"`
	Probes    []ModelStatus `json:"probes"`
	seconds   string
}

// Preflight finds the first model of the declared chain that NaN answers: the
// primary, then the fallbacks in declared order. It never chooses a model. It
// hands PR-Agent the first that answered, followed by the others that did, in
// declared order, and skips a dead entry loudly: a ::warning:: that names the fix
// for its status class.
//
// A status line is not an answer: only a completed transfer counts, since
// headers can arrive and the body never. A model outside the chain is never
// probed, whatever NaN serves: `GET /v1/models` lists what the cluster runs, not
// what the key may call, so only a real call counts.
//
// When no model answers it returns the probes it made with ErrNoModelAnswered. A
// setup problem comes back as PreflightSetupError before anything is probed.
func Preflight(ctx context.Context, in PreflightInputs, env PreflightEnv) (PreflightResult, error) {
	chain, err := in.chain()
	if err != nil {
		return PreflightResult{}, err
	}
	timeout, seconds := preflightTimeout(in.Timeout)
	res := PreflightResult{Fallbacks: []string{}, seconds: seconds}
	var answered []string
	for _, declared := range chain {
		bare := strings.TrimPrefix(declared, "openai/")
		code := ProbeNaN(ctx, env.Client, cmpOr(in.APIBase, defaultNaNBase), in.Key, bare, timeout)
		res.Probes = append(res.Probes, ModelStatus{declared, code})
		if n, _ := strconv.Atoi(code); n >= 200 && n < 300 {
			answered = append(answered, declared)
			continue
		}
		what := "answered HTTP " + code
		if code == "000" {
			what = "gave no answer within " + seconds + "s"
		}
		annotate(env.Out, "PR-Agent model %s %s and is skipped for this run: %s.", declared, what, remedy(code))
	}
	if len(answered) == 0 {
		annotate(env.Out, "No model in PR-Agent's declared NaN chain answered, so NaN is out of the review pool for this run."+
			" The warnings above give each model's status and what to do about it.")
		return res, ErrNoModelAnswered
	}
	res.Model, res.Fallbacks = answered[0], answered[1:]
	if res.Model != in.Model {
		annotate(env.Out, "PR-Agent reviews this PR with %s, not the declared primary %s.", res.Model, in.Model)
	}
	return res, nil
}

// chain is the primary followed by the fallbacks, without the blank entries.
func (in PreflightInputs) chain() ([]string, error) {
	if in.Model == "" || in.Fallbacks == "" {
		return nil, PreflightSetupError("usage: dotf pr preflight --model openai/<id> --fallbacks JSON_ARRAY [--output FILE]")
	}
	if in.Key == "" {
		return nil, PreflightSetupError("NAN_API_KEY is not set")
	}
	var fallbacks []string
	if err := json.Unmarshal([]byte(in.Fallbacks), &fallbacks); err != nil {
		return nil, PreflightSetupError(fmt.Sprintf("--fallbacks must be a JSON array of model ids, got: %s", in.Fallbacks))
	}
	var chain []string
	for _, m := range append([]string{in.Model}, fallbacks...) {
		if m != "" {
			chain = append(chain, m)
		}
	}
	return chain, nil
}

// preflightTimeout reads PREFLIGHT_TIMEOUT: seconds, fractions allowed. An empty,
// unreadable, non-positive or out-of-range value is the default. The second
// value is the number as the warnings and the summary print it.
func preflightTimeout(raw string) (time.Duration, string) {
	if d, ok := parseSeconds(raw); ok {
		return d, raw
	}
	return defaultPreflightSecond * time.Second, strconv.Itoa(defaultPreflightSecond)
}

// remedy is what the reader should do about a model that did not answer. Only a
// refusal means the chain is wrong. Quota and NaN's per-model concurrency limit
// (#1107) pass on their own, and so does an outage; telling the reader to edit
// the chain for those would be wrong advice.
func remedy(code string) string {
	switch code {
	case "401", "403", "404":
		return "this key cannot call it. Fix the chain in .pr_agent.toml and .github/workflows/pr-agent.yml"
	case "402", "429":
		return "quota or concurrency limit reached. Nothing to fix; it passes, or the quota resets"
	}
	return "NaN did not serve it this time. Nothing to fix unless it persists; check https://nan.builders/docs/models"
}

// annotate writes a ::warning:: line to w, when there is one.
func annotate(w io.Writer, format string, args ...any) {
	if w != nil {
		_, _ = fmt.Fprintf(w, "::warning::"+format+"\n", args...)
	}
}

// KeyValues is the outputs as `key=value` lines, in the order and with the names
// the workflow reads.
func (r PreflightResult) KeyValues() string {
	return fmt.Sprintf("model=%s\nfallbacks=%s\n", r.Model, compact(r.Fallbacks))
}

// JSON is the result as one JSON object.
func (r PreflightResult) JSON() (string, error) {
	if r.Fallbacks == nil {
		r.Fallbacks = []string{}
	}
	return marshal(r)
}

// Summary is the probes as Markdown for the job summary.
func (r PreflightResult) Summary() string {
	var b strings.Builder
	b.WriteString("### PR-Agent model preflight\n\n| Model | Result |\n|---|---|\n")
	for _, p := range r.Probes {
		result := "HTTP " + p.Status
		switch n, _ := strconv.Atoi(p.Status); {
		case n >= 200 && n < 300:
			result = "answered (HTTP " + p.Status + ")"
		case p.Status == "000":
			result = "no answer within " + r.seconds + "s"
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", p.Model, result)
	}
	if r.Model == "" {
		b.WriteString("\nNo declared model answered, so NaN is out of the review pool for this run.\n")
	} else {
		fmt.Fprintf(&b, "\nReviewing with `%s`.\n", r.Model)
	}
	return b.String()
}
