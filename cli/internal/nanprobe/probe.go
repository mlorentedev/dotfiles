// Package nanprobe answers "does NaN still serve every model this repository
// binds" with a real call per model, off the PR path (AI-045 AC8, #1860).
//
// NaN retired `mimo-v2.5` on 2026-09-30 without notice at the call site. For
// hours it accepted requests and never answered, then it answered 401. Seven
// files bound it, and it was found because PRs got green review jobs with no
// review. The PR-path preflight (scripts/pr-agent-model-preflight.sh) skips a
// dead model in one PR's chain; this finds the retirement before any PR does,
// for every model bound anywhere.
//
// Only a real call counts. GET /v1/models lists what the cluster runs, not what
// the key may call (https://nan.builders/docs/choose-a-model).
package nanprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is NaN's OpenAI-compatible endpoint.
const DefaultBaseURL = "https://api.nan.builders/v1"

// maxTokens is the smallest reply budget every NaN chat model accepts. A
// reasoning model spends tokens thinking before it answers, and
// mimo-v2.6-flash asks for at least 300 (measured for the PR-path preflight).
const maxTokens = 512

// Class is what one probe means for the reader, which is not the same as its
// status code: only a refusal says a file is wrong.
type Class string

const (
	// Answered — the model completed a reply.
	Answered Class = "answered"
	// Refused — 401, 403 or 404: this key cannot call the model. Retirement
	// looks like this once NaN stops holding the request, so the files that
	// bind it need a new model.
	Refused Class = "refused"
	// Quota — 402 or 429: the monthly quota is spent or NaN's per-model
	// concurrency limit was hit (#1107). It passes on its own.
	Quota Class = "quota"
	// Unavailable — no completed reply in time, or any other status. A
	// retirement in progress looked like this for hours; so does an outage.
	Unavailable Class = "unavailable"
)

// API is the endpoint a model is called through. A rerank or embeddings model
// answers 404 on /chat/completions (measured on NaN, 2026-10-01), which a chat
// probe would report as a refusal on every run.
type API string

const (
	ChatAPI       API = "chat"
	EmbeddingsAPI API = "embeddings"
	RerankAPI     API = "rerank"
)

// Target is one model and the API it serves.
type Target struct {
	Model string
	API   API
}

// requestFor builds the smallest request each API accepts.
func requestFor(t Target) (path string, body map[string]any) {
	switch t.API {
	case EmbeddingsAPI:
		return "/embeddings", map[string]any{"model": t.Model, "input": "OK"}
	case RerankAPI:
		return "/rerank", map[string]any{"model": t.Model, "query": "OK", "documents": []string{"OK"}}
	default:
		return "/chat/completions", map[string]any{
			"model":      t.Model,
			"max_tokens": maxTokens,
			"messages":   []map[string]string{{"role": "user", "content": "Reply with OK."}},
		}
	}
}

// Result is one model's probe.
type Result struct {
	Model  string
	Status int // 0 when no status line arrived
	Class  Class
	Detail string // why it is not Answered, for the report
}

// Prober sends one minimal chat completion per model.
type Prober struct {
	Client  *http.Client
	BaseURL string
	Key     string
	Timeout time.Duration
}

// Probe calls one model and classifies the outcome. The key travels in a
// header only, and no error message includes it.
func (p Prober) Probe(ctx context.Context, t Target) Result {
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	model := t.Model
	path, payload := requestFor(t)
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{Model: model, Class: Unavailable, Detail: err.Error()}
	}
	url := strings.TrimSuffix(p.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{Model: model, Class: Unavailable, Detail: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+p.Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "dotf-model-canary")

	resp, err := p.Client.Do(req)
	if err != nil {
		// A refused connection, a DNS or TLS failure returns in milliseconds;
		// calling it a timeout sends the reader to NaN's status page when the
		// runner never reached it.
		if ctx.Err() != nil {
			return Result{Model: model, Class: Unavailable, Detail: fmt.Sprintf("no answer within %s", p.Timeout)}
		}
		return Result{Model: model, Class: Unavailable, Detail: "could not reach the endpoint: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	// A status line is not an answer. Headers can arrive and the body never:
	// the PR-path preflight measured curl printing 200 and then timing out.
	// Only a body read to the end counts.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return Result{Model: model, Status: resp.StatusCode, Class: Unavailable,
			Detail: fmt.Sprintf("HTTP %d but the reply did not complete within %s", resp.StatusCode, p.Timeout)}
	}
	return classify(model, resp.StatusCode)
}

func classify(model string, status int) Result {
	r := Result{Model: model, Status: status}
	switch {
	case status >= 200 && status < 300:
		r.Class = Answered
	case status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusNotFound:
		r.Class, r.Detail = Refused, fmt.Sprintf("HTTP %d", status)
	case status == http.StatusPaymentRequired || status == http.StatusTooManyRequests:
		r.Class, r.Detail = Quota, fmt.Sprintf("HTTP %d", status)
	default:
		r.Class, r.Detail = Unavailable, fmt.Sprintf("HTTP %d", status)
	}
	return r
}

// ProbeAll probes models one at a time, then probes each Unavailable one a
// second time.
//
// SEQUENTIAL ON PURPOSE: NaN limits concurrency per model (#1107), and a
// parallel sweep would manufacture the 429s it then reports. The second pass
// keeps a single dropped connection at 3 a.m. from opening an issue; a model
// retiring stays unavailable on both.
func (p Prober) ProbeAll(ctx context.Context, targets []Target) []Result {
	out := make([]Result, 0, len(targets))
	for _, t := range targets {
		out = append(out, p.Probe(ctx, t))
	}
	for i, r := range out {
		if r.Class == Unavailable {
			out[i] = p.Probe(ctx, targets[i])
		}
	}
	return out
}
