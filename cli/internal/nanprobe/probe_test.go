package nanprobe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// stub serves one status per model, and can hang or stall the body.
type stub struct {
	mu     sync.Mutex
	status map[string][]int // per model, consumed in order; the last one repeats
	hang   map[string]bool  // no status line at all
	stall  map[string]bool  // status line, then a body that never completes
	calls  map[string]int
	seen   []map[string]any
	paths  []string
	auth   []string
}

func (s *stub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	model, _ := req["model"].(string)

	s.mu.Lock()
	s.seen = append(s.seen, req)
	s.paths = append(s.paths, r.URL.Path)
	s.auth = append(s.auth, r.Header.Get("Authorization"))
	n := s.calls[model]
	s.calls[model]++
	codes := s.status[model]
	s.mu.Unlock()

	if s.hang[model] {
		<-r.Context().Done()
		return
	}
	code := http.StatusOK
	if len(codes) > 0 {
		code = codes[len(codes)-1]
		if n < len(codes) {
			code = codes[n]
		}
	}
	if s.stall[model] {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{"partial":`))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		return
	}
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
}

func newStub() *stub {
	return &stub{status: map[string][]int{}, hang: map[string]bool{}, stall: map[string]bool{}, calls: map[string]int{}}
}

func proberFor(t *testing.T, s *stub) Prober {
	t.Helper()
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	return Prober{Client: srv.Client(), BaseURL: srv.URL + "/v1", Key: "test-key", Timeout: 300 * time.Millisecond}
}

func TestProbeClassifiesEachStatus(t *testing.T) {
	s := newStub()
	s.status["ok"] = []int{200}
	s.status["gone"] = []int{401}
	s.status["forbidden"] = []int{403}
	s.status["unknown"] = []int{404}
	s.status["spent"] = []int{402}
	s.status["busy"] = []int{429}
	s.status["broken"] = []int{500}
	p := proberFor(t, s)

	want := map[string]Class{
		"ok": Answered, "gone": Refused, "forbidden": Refused, "unknown": Refused,
		"spent": Quota, "busy": Quota, "broken": Unavailable,
	}
	for model, class := range want {
		if got := p.Probe(context.Background(), Target{Model: model, API: ChatAPI}); got.Class != class {
			t.Errorf("%s: class %q, want %q (status %d)", model, got.Class, class, got.Status)
		}
	}
}

func TestProbeCountsOnlyACompletedReply(t *testing.T) {
	s := newStub()
	s.hang["held"] = true
	s.stall["headers-only"] = true
	p := proberFor(t, s)

	if got := p.Probe(context.Background(), Target{Model: "held"}); got.Class != Unavailable || got.Status != 0 {
		t.Errorf("a request NaN never answers: got %q status %d, want unavailable with no status", got.Class, got.Status)
	}
	// The case curl reported as 200 then timed out: a status line is not an answer.
	if got := p.Probe(context.Background(), Target{Model: "headers-only"}); got.Class != Unavailable || got.Status != 200 {
		t.Errorf("200 with a body that never completes: got %q status %d, want unavailable at 200", got.Class, got.Status)
	}
}

func TestProbeTellsATimeoutFromAnUnreachableEndpoint(t *testing.T) {
	s := newStub()
	s.hang["held"] = true
	p := proberFor(t, s)
	if got := p.Probe(context.Background(), Target{Model: "held"}); !strings.Contains(got.Detail, "no answer within") {
		t.Errorf("a request NaN holds: detail %q, want it to name the timeout", got.Detail)
	}

	// Refused in milliseconds: telling the reader to wait out 90s sends them
	// to NaN's status page when the runner never reached it.
	srv := httptest.NewServer(newStub())
	srv.Close()
	p.BaseURL = srv.URL + "/v1"
	got := p.Probe(context.Background(), Target{Model: "ok"})
	if got.Class != Unavailable || strings.Contains(got.Detail, "no answer within") || !strings.Contains(got.Detail, "could not reach") {
		t.Errorf("a closed endpoint: got %q detail %q, want unavailable naming the transport error", got.Class, got.Detail)
	}
	if strings.Contains(got.Detail, p.Key) {
		t.Errorf("the detail leaks the key: %q", got.Detail)
	}
}

func TestProbeSendsAReplyBudgetEveryModelAccepts(t *testing.T) {
	s := newStub()
	p := proberFor(t, s)
	p.Probe(context.Background(), Target{Model: "m", API: ChatAPI})

	if len(s.seen) != 1 {
		t.Fatalf("expected one request, got %d", len(s.seen))
	}
	if got, _ := s.seen[0]["max_tokens"].(float64); got < 512 {
		t.Errorf("max_tokens %v: a reasoning model needs at least 512 to answer at all", got)
	}
	if s.auth[0] != "Bearer test-key" {
		t.Errorf("Authorization header %q", s.auth[0])
	}
}

func TestProbeAllRetriesOnlyUnavailableAndRunsInOrder(t *testing.T) {
	s := newStub()
	s.status["blip"] = []int{503, 200}
	s.status["gone"] = []int{401}
	s.status["dead"] = []int{503}
	p := proberFor(t, s)

	got := p.ProbeAll(context.Background(), []Target{{Model: "blip"}, {Model: "gone"}, {Model: "dead"}})

	classes := map[string]Class{}
	for _, r := range got {
		classes[r.Model] = r.Class
	}
	if classes["blip"] != Answered {
		t.Errorf("one dropped reply then an answer must not open an issue: got %q", classes["blip"])
	}
	if classes["dead"] != Unavailable {
		t.Errorf("unavailable on both passes: got %q", classes["dead"])
	}
	if s.calls["gone"] != 1 {
		t.Errorf("a refusal is final and is not re-probed: %d calls", s.calls["gone"])
	}
	if got[0].Model != "blip" || got[1].Model != "gone" || got[2].Model != "dead" {
		t.Errorf("results out of declared order: %v", got)
	}
}

func TestProbeCallsEachModelThroughItsOwnAPI(t *testing.T) {
	s := newStub()
	p := proberFor(t, s)
	p.Probe(context.Background(), Target{Model: "chat", API: ChatAPI})
	p.Probe(context.Background(), Target{Model: "emb", API: EmbeddingsAPI})
	p.Probe(context.Background(), Target{Model: "rr", API: RerankAPI})

	want := []string{"/v1/chat/completions", "/v1/embeddings", "/v1/rerank"}
	for i, w := range want {
		if s.paths[i] != w {
			t.Errorf("request %d went to %s, want %s: a rerank model answers 404 on chat", i, s.paths[i], w)
		}
	}
	if _, ok := s.seen[1]["input"]; !ok {
		t.Errorf("embeddings request carries no input: %v", s.seen[1])
	}
	if docs, ok := s.seen[2]["documents"].([]any); !ok || len(docs) == 0 {
		t.Errorf("rerank request carries no documents: %v", s.seen[2])
	}
}
