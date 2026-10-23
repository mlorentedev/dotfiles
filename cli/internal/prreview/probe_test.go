package prreview

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const probeKey = "sk-ant-probe-not-a-real-key"

// probeServer answers every request by handle and records what it was sent.
type probeServer struct {
	URL     string
	mu      sync.Mutex
	seen    []seenRequest
	release chan struct{} // closed at cleanup, so a held request does not hold the server
}

type seenRequest struct {
	Method, URI, Body string
	Header            http.Header
}

func newProbeServer(t *testing.T, handle func(*probeServer, http.ResponseWriter, *http.Request)) *probeServer {
	t.Helper()
	ps := &probeServer{release: make(chan struct{})}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ps.mu.Lock()
		ps.seen = append(ps.seen, seenRequest{r.Method, r.RequestURI, string(body), r.Header.Clone()})
		ps.mu.Unlock()
		handle(ps, w, r)
	}))
	t.Cleanup(func() { close(ps.release); srv.Close() })
	ps.URL = srv.URL
	return ps
}

func (ps *probeServer) requests() []seenRequest {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return append([]seenRequest(nil), ps.seen...)
}

func status(code int) func(*probeServer, http.ResponseWriter, *http.Request) {
	return func(_ *probeServer, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte("{}"))
	}
}

func TestProbe_ReturnsTheStatusItGot(t *testing.T) {
	for _, code := range []int{200, 204, 401, 403, 404, 429, 529} {
		ps := newProbeServer(t, status(code))
		if got := Probe(t.Context(), nil, ps.URL, probeKey, "anthropic/claude-haiku-5-5", time.Second); got != strconv.Itoa(code) {
			t.Errorf("a %d answer probed as %q", code, got)
		}
	}
}

func TestProbe_AsksForSixtyFourTokensFromTheModelWithoutItsPrefix(t *testing.T) {
	ps := newProbeServer(t, status(200))
	Probe(t.Context(), nil, ps.URL, probeKey, "anthropic/claude-haiku-5-5", time.Second)
	reqs := ps.requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests", len(reqs))
	}
	r := reqs[0]
	want := `{"model":"claude-haiku-5-5","max_tokens":64,"messages":[{"role":"user","content":"Reply with OK."}]}`
	if r.Method != http.MethodPost || r.URI != "/v1/messages" || r.Body != want {
		t.Errorf("request %s %s %s", r.Method, r.URI, r.Body)
	}
	if r.Header.Get("anthropic-version") != "2023-06-01" || r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("headers %v", r.Header)
	}
}

// No answer is 000, whatever the reason: a server that holds the connection, one
// that sends a status line and never the body, one that is not there.
func TestProbe_NoAnswerIs000(t *testing.T) {
	hold := func(ps *probeServer, _ http.ResponseWriter, _ *http.Request) { <-ps.release }
	stall := func(ps *probeServer, w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-ps.release
	}
	truncated := func(_ *probeServer, w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{"))
		// Returning with 99 bytes owed closes the connection mid-body.
	}
	for name, h := range map[string]func(*probeServer, http.ResponseWriter, *http.Request){"hang": hold, "stall": stall, "truncated body": truncated} {
		t.Run(name, func(t *testing.T) {
			ps := newProbeServer(t, h)
			start := time.Now()
			if got := Probe(t.Context(), nil, ps.URL, probeKey, "anthropic/claude-haiku-5-5", 300*time.Millisecond); got != "000" {
				t.Errorf("probed as %q", got)
			}
			if time.Since(start) > 2*time.Second {
				t.Errorf("the probe outlived its timeout: %s", time.Since(start))
			}
		})
	}
	if got := Probe(t.Context(), nil, "http://127.0.0.1:1", probeKey, "anthropic/x", time.Second); got != "000" {
		t.Errorf("a refused connection probed as %q", got)
	}
	if got := Probe(t.Context(), nil, "http://bad host", probeKey, "anthropic/x", time.Second); got != "000" {
		t.Errorf("an invalid base probed as %q", got)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if got := Probe(ctx, nil, "http://127.0.0.1:1", probeKey, "anthropic/x", time.Second); got != "000" {
		t.Errorf("a cancelled probe was %q", got)
	}
}

// A redirect is an answer to report, not a hop to follow: the key would travel
// to wherever it points.
func TestProbe_DoesNotFollowARedirect(t *testing.T) {
	elsewhere := newProbeServer(t, status(200))
	ps := newProbeServer(t, func(_ *probeServer, w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/stolen", http.StatusTemporaryRedirect)
	})
	if got := Probe(t.Context(), nil, ps.URL, probeKey, "anthropic/claude-haiku-5-5", time.Second); got != "307" {
		t.Errorf("a redirect probed as %q", got)
	}
	if n := len(elsewhere.requests()); n != 0 {
		t.Errorf("the redirect was followed: %d requests reached the target", n)
	}
}

// The key is one request header's value. It is in no URL, no body, no other
// header, and nothing the probe returns.
func TestProbe_TheKeyIsOnlyInTheKeyHeader(t *testing.T) {
	ps := newProbeServer(t, status(200))
	got := []string{Probe(t.Context(), nil, ps.URL, probeKey, "anthropic/claude-haiku-5-5", time.Second)}
	for _, base := range []string{"http://127.0.0.1:1", ps.URL} {
		got = append(got, Probe(t.Context(), nil, base, probeKey, "anthropic/claude-haiku-5-5", time.Second))
	}
	for _, g := range got {
		if strings.Contains(g, probeKey) {
			t.Errorf("a probe returned the key: %q", g)
		}
	}
	for _, r := range ps.requests() {
		if r.Header.Get("x-api-key") != probeKey {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if strings.Contains(r.URI, probeKey) || strings.Contains(r.Body, probeKey) {
			t.Errorf("the key is in the URL or body: %s %s", r.URI, r.Body)
		}
		for name, vals := range r.Header {
			if name != "X-Api-Key" && strings.Contains(strings.Join(vals, ","), probeKey) {
				t.Errorf("the key is in header %s", name)
			}
		}
	}
}

func TestVerdict_NamesTheFixByStatusClass(t *testing.T) {
	const m = "anthropic/claude-haiku-5-5"
	for _, tc := range []struct {
		code string
		ok   bool
		want []string
	}{
		{"200", true, nil},
		{"204", true, nil},
		{"401", false, []string{m + " answered HTTP 401: the key is refused", "dotf secrets rotate PR_AGENT_ANTHROPIC_API_KEY"}},
		{"403", false, []string{"answered HTTP 403: the key is refused"}},
		{"404", false, []string{"answered HTTP 404: the model id is wrong", "harness/reviewer-pool.json"}},
		{"000", false, []string{m + " gave no answer within 2.5s and was out of the review pool for this run."}},
		{"429", false, []string{"answered HTTP 429 (rate limit, overload or spend limit) and was out of the review pool"}},
		{"307", false, []string{"answered HTTP 307 (rate limit, overload or spend limit)"}},
	} {
		ok, note := Verdict(m, tc.code, "2.5")
		if ok != tc.ok {
			t.Errorf("%s: answered = %v", tc.code, ok)
		}
		for _, w := range tc.want {
			if !strings.Contains(note, w) {
				t.Errorf("%s: note %q lacks %q", tc.code, note, w)
			}
		}
		if tc.ok && note != "" {
			t.Errorf("%s: an answer carries the note %q", tc.code, note)
		}
	}
	if _, note := Verdict(m, "000", ""); !strings.Contains(note, "within 30s") {
		t.Errorf("the default timeout is not named: %q", note)
	}
}

func TestParseTimeout(t *testing.T) {
	for raw, want := range map[string]time.Duration{
		"": 30 * time.Second, "2": 2 * time.Second, "0.5": 500 * time.Millisecond,
		"0": 30 * time.Second, "-3": 30 * time.Second, "abc": 30 * time.Second,
		"1e10": 30 * time.Second, "Inf": 30 * time.Second, "NaN": 30 * time.Second,
	} {
		if got := ParseTimeout(raw); got != want {
			t.Errorf("ParseTimeout(%q) = %s, want %s", raw, got, want)
		}
	}
}

// The guarantee holds for any client the caller passes: Go strips only
// Authorization, WWW-Authenticate and Cookie when a redirect crosses hosts, so a
// client with the default policy would forward x-api-key to the target.
func TestProbe_DoesNotForwardTheKeyThroughAClientThatFollowsRedirects(t *testing.T) {
	elsewhere := newProbeServer(t, status(200))
	ps := newProbeServer(t, func(_ *probeServer, w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/stolen", http.StatusTemporaryRedirect)
	})
	follower := &http.Client{}
	if got := Probe(t.Context(), follower, ps.URL, probeKey, "anthropic/claude-haiku-5-5", time.Second); got != "307" {
		t.Errorf("a redirect probed as %q", got)
	}
	for _, r := range elsewhere.requests() {
		if r.Header.Get("x-api-key") != "" {
			t.Errorf("the key reached the redirect target: %v", r.Header)
		}
	}
	if n := len(elsewhere.requests()); n != 0 {
		t.Errorf("the redirect was followed: %d requests reached the target", n)
	}
	if follower.CheckRedirect != nil {
		t.Error("Probe changed the caller's client")
	}
}
