package prreview

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultProbeTimeout bounds one probe when ROUTE_PROBE_TIMEOUT names none.
const defaultProbeTimeout = 30 * time.Second

// ParseTimeout reads ROUTE_PROBE_TIMEOUT: seconds, fractions allowed. An empty,
// unreadable or non-positive value is the default.
func ParseTimeout(raw string) time.Duration {
	if s, err := strconv.ParseFloat(raw, 64); err == nil && s > 0 {
		return time.Duration(s * float64(time.Second))
	}
	return defaultProbeTimeout
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Probe asks the Messages API at base for 64 tokens from model and returns the
// HTTP status, or "000" when nothing answered in time. Any 2xx is an answer, and
// a truncated one costs the least.
//
// The body is read to its end: a server that sends a status line and never the
// rest has not answered. A redirect is reported as its status, not followed, so
// the key never travels to wherever it points. The key is one request header's
// value (x-api-key) and appears in nothing Probe returns. A nil client is a
// default one; any other keeps its transport and timeout.
func Probe(ctx context.Context, client *http.Client, base, key, model string, timeout time.Duration) string {
	body, _ := json.Marshal(struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		Messages  []message `json:"messages"`
	}{strings.TrimPrefix(model, "anthropic/"), 64, []message{{"user", "Reply with OK."}}})

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "000"
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
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
