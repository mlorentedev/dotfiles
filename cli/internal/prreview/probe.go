package prreview

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultProbeTimeout bounds one probe when ROUTE_PROBE_TIMEOUT names none.
const defaultProbeTimeout = 30 * time.Second

// ParseTimeout reads ROUTE_PROBE_TIMEOUT: seconds, fractions allowed. An empty,
// unreadable, non-positive or out-of-range value is the default.
func ParseTimeout(raw string) time.Duration {
	if d, ok := parseSeconds(raw); ok {
		return d
	}
	return defaultProbeTimeout
}

// parseSeconds reads a positive number of seconds that fits a time.Duration.
// Converting a float beyond its range is implementation-defined in Go: arm64
// saturates, amd64 yields a negative duration, so every probe would time out at
// once on the runners. Such a value, Inf and NaN all report false.
func parseSeconds(raw string) (time.Duration, bool) {
	s, err := strconv.ParseFloat(raw, 64)
	if err != nil || !(s > 0) || s >= float64(math.MaxInt64)/float64(time.Second) {
		return 0, false
	}
	return time.Duration(s * float64(time.Second)), true
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Probe asks the Messages API at base for 64 tokens from model and returns the
// HTTP status, or "000" when nothing answered in time. Any 2xx is an answer, and
// a truncated one costs the least. The key is one request header's value
// (x-api-key) and appears in nothing Probe returns. See post for the rest.
func Probe(ctx context.Context, client *http.Client, base, key, model string, timeout time.Duration) string {
	return post(ctx, client, base+"/v1/messages", probeBody(strings.TrimPrefix(model, "anthropic/"), 64), timeout,
		"x-api-key", key, "anthropic-version", "2023-06-01")
}

// ProbeNaN asks NaN's chat completions API at base for 512 tokens from model
// (the bare id, without PR-Agent's `openai/` prefix) and returns the HTTP
// status, or "000" when nothing answered in time. A reasoning model spends
// tokens thinking before it answers, and 512 covers the shortest reply on every
// NaN chat model. The key is the Authorization header's value and appears in
// nothing ProbeNaN returns. See post for the rest.
func ProbeNaN(ctx context.Context, client *http.Client, base, key, model string, timeout time.Duration) string {
	return post(ctx, client, base+"/chat/completions", probeBody(model, 512), timeout, "Authorization", "Bearer "+key)
}

// probeBody is the request both providers take: a few tokens in answer to
// "Reply with OK.". The model id is written as it is, without HTML escaping.
func probeBody(model string, maxTokens int) []byte {
	body, _ := marshal(struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		Messages  []message `json:"messages"`
	}{model, maxTokens, []message{{"user", "Reply with OK."}}})
	return []byte(body)
}

// post sends body as JSON to url with the headers given as name, value pairs and
// returns the HTTP status, or "000" when nothing answered in time.
//
// The body is read to its end: a server that sends a status line and never the
// rest has not answered. A redirect is reported as its status, not followed, so
// the key never travels to wherever it points. A nil client is a default one;
// any other keeps its transport and timeout.
func post(ctx context.Context, client *http.Client, url string, body []byte, timeout time.Duration, headers ...string) string {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "000"
	}
	for i := 0; i < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	req.Header.Set("Content-Type", "application/json")
	// Whatever client the caller passes, a redirect is not followed: Go strips
	// only Authorization, WWW-Authenticate and Cookie when one crosses hosts, so
	// a following client would forward x-api-key to the target. The copy leaves
	// the caller's client as it was.
	c := http.Client{}
	if client != nil {
		c = *client
	}
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Do(req)
	if err != nil {
		return "000"
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return "000"
	}
	return strconv.Itoa(resp.StatusCode)
}

// Verdict reads a probe's status: whether the member answered, and when it did
// not, the note naming the fix by status class. timeout is ROUTE_PROBE_TIMEOUT
// as written, for the "no answer" note.
func Verdict(model, code, timeout string) (answered bool, note string) {
	switch {
	case strings.HasPrefix(code, "2"):
		return true, ""
	case code == "401" || code == "403":
		return false, model + " answered HTTP " + code + ": the key is refused. Rotate it: dotf secrets rotate PR_AGENT_ANTHROPIC_API_KEY"
	case code == "404":
		return false, model + " answered HTTP 404: the model id is wrong. Fix it in harness/reviewer-pool.json"
	case code == "000":
		return false, model + " gave no answer within " + cmpOr(timeout, "30") + "s and was out of the review pool for this run."
	default:
		return false, model + " answered HTTP " + code + " (rate limit, overload or spend limit) and was out of the review pool for this run."
	}
}

// cmpOr is v, or def when v is empty.
func cmpOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
