package prreview

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

const testKey = "sk-ant-test-not-a-real-key"

// fixturePool has weights 3, 2 and 5, so a point in [0, 10) names exactly one
// member: 0-2 mimo, 3-4 deepseek, 5-9 haiku. The `signs` fields are what the
// shared pool parser requires of an Anthropic member; the draw ignores them.
const fixturePool = `{"pool": [
 {"id": "nan/mimo-v2.6-flash", "pr_agent": {"model": "openai/mimo-v2.6-flash", "weight": 3}},
 {"id": "nan/deepseek-v4-flash", "pr_agent": {"model": "openai/deepseek-v4-flash", "weight": 2}},
 {"id": "nan/qwen3.8-flash"},
 {"id": "anthropic-review/claude-haiku-5-5", "signs": "fallback",
  "pr_agent": {"model": "anthropic/claude-haiku-5-5", "weight": 5, "reasoning_effort": "high"}},
 {"id": "anthropic-review/claude-sonnet-5-5", "signs": "second",
  "pr_agent": {"model": "anthropic/claude-sonnet-5-5", "route": "risk", "reasoning_effort": "medium"}}
], "pr_agent_risk": {"min_changed_lines": 900, "label": "deep-review"}}`

func mustPool(t *testing.T, raw string) spec.ReviewerPool {
	t.Helper()
	p, err := spec.ParseReviewerPool([]byte(raw), "fixture")
	if err != nil {
		t.Fatalf("fixture pool: %v", err)
	}
	return p
}

// apiReq is one probe the fake Anthropic API received.
type apiReq struct {
	Path, Key, Version, ContentType string
	Body                            map[string]any
}

// fakeAPI answers each probe by the model it names: a status code, "hang" (no
// answer at all) or "stall" (a 200 status line and no body).
type fakeAPI struct {
	URL     string
	mu      sync.Mutex
	reqs    []apiReq
	modes   map[string]string
	def     string
	release chan struct{}
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	a := &fakeAPI{modes: map[string]string{}, def: "200", release: make(chan struct{})}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		model, _ := body["model"].(string)
		a.mu.Lock()
		a.reqs = append(a.reqs, apiReq{r.URL.Path, r.Header.Get("x-api-key"), r.Header.Get("anthropic-version"),
			r.Header.Get("Content-Type"), body})
		mode, ok := a.modes[model]
		if !ok {
			mode = a.def
		}
		a.mu.Unlock()
		switch mode {
		case "hang":
			<-a.release
		case "stall":
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-a.release
		default:
			code, _ := strconv.Atoi(mode)
			w.WriteHeader(code)
			_, _ = w.Write([]byte("{}"))
		}
	}))
	t.Cleanup(func() { close(a.release); srv.Close() })
	a.URL = srv.URL
	return a
}

// answer sets how probes naming model are answered; the empty model sets the
// default for every other.
func (a *fakeAPI) answer(model, mode string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if model == "" {
		a.def = mode
		return
	}
	a.modes[model] = mode
}

func (a *fakeAPI) probed() []apiReq {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]apiReq(nil), a.reqs...)
}

func (a *fakeAPI) probedModels() string {
	var m []string
	for _, r := range a.probed() {
		s, _ := r.Body["model"].(string)
		m = append(m, s)
	}
	return strings.Join(m, " ")
}

// fixture is one route call's inputs, with the NaN preflight answered and the
// Anthropic key set, as a healthy run has them.
type fixture struct {
	t    *testing.T
	pool spec.ReviewerPool
	in   Inputs
	api  *fakeAPI
	src  string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	api := newFakeAPI(t)
	return &fixture{t: t, pool: mustPool(t, fixturePool), api: api, src: "fixture", in: Inputs{
		NanOutcome: "success", NanModel: "openai/mimo-v2.6-flash", NanFallbacks: `["openai/deepseek-v4-flash"]`,
		AnthropicKey: testKey, ChangedLines: "10", Repository: "o/r", APIBase: api.URL, ProbeTimeout: "1",
	}}
}

// run routes once and returns the result, the annotations printed, and the error.
func (f *fixture) run() (Result, string, error) {
	f.t.Helper()
	var out bytes.Buffer
	res, err := Route(f.t.Context(), f.pool, f.src, f.in, Env{Out: &out})
	return res, out.String(), err
}

// ok is run for a call that must succeed.
func (f *fixture) ok() (Result, string) {
	f.t.Helper()
	res, out, err := f.run()
	if err != nil {
		f.t.Fatalf("Route: %v\n%s", err, out)
	}
	return res, out
}

// at pins the draw to a point of the total weight.
func (f *fixture) at(point int) *fixture {
	f.in.Draw = strconv.Itoa(point)
	return f
}

// everyPoint draws at each point of [0, total) and counts the reviewers.
func (f *fixture) everyPoint(total int) map[string]int {
	f.t.Helper()
	got := map[string]int{}
	for p := 0; p < total; p++ {
		res, _ := f.at(p).ok()
		got[res.Reviewer]++
	}
	return got
}
