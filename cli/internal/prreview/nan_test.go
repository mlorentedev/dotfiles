package prreview

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const nanKey = "sk-nan-test-not-a-real-key"

// nanReq is one probe the fake NaN API received.
type nanReq struct {
	Path, Auth, ContentType, Raw, Model string
	Header                              http.Header
}

// nanAPI answers each probe by the model it names: a status code, "hang" (no
// answer at all), "stall" (a 200 status line and no body) or "redirect" (a 307
// to redirectTo). A model nobody configured is answered 404, as the bats stub
// did.
type nanAPI struct {
	URL        string
	redirectTo string
	mu         sync.Mutex
	reqs       []nanReq
	modes      map[string]string
	def        string
	release    chan struct{}
}

func newNanAPI(t *testing.T, modes map[string]string) *nanAPI {
	t.Helper()
	a := &nanAPI{modes: modes, def: "404", release: make(chan struct{})}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct{ Model string }
		_ = json.Unmarshal(raw, &body)
		a.mu.Lock()
		a.reqs = append(a.reqs, nanReq{r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Content-Type"),
			string(raw), body.Model, r.Header.Clone()})
		mode, ok := a.modes[body.Model]
		if !ok {
			mode = a.def
		}
		to := a.redirectTo
		a.mu.Unlock()
		switch mode {
		case "hang":
			<-a.release
		case "stall":
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-a.release
		case "redirect":
			http.Redirect(w, r, to, http.StatusTemporaryRedirect)
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

// redirect sets where a "redirect" answer points.
func (a *nanAPI) redirect(to string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.redirectTo = to
}

func (a *nanAPI) requests() []nanReq {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]nanReq(nil), a.reqs...)
}

func (a *nanAPI) probed() string {
	var m []string
	for _, r := range a.requests() {
		m = append(m, r.Model)
	}
	return strings.Join(m, " ")
}
