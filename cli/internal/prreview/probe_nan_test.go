package prreview

import (
	"net/http"
	"testing"
	"time"
)

func TestProbeNaN_ReturnsTheStatusItGotAndAsksFor512TokensFromTheBareId(t *testing.T) {
	for _, code := range []string{"200", "204", "401", "429", "503"} {
		api := newNanAPI(t, map[string]string{"mimo-v2.6-flash": code})
		if got := ProbeNaN(t.Context(), nil, api.URL, nanKey, "mimo-v2.6-flash", time.Second); got != code {
			t.Errorf("a %s answer probed as %q", code, got)
		}
		want := `{"model":"mimo-v2.6-flash","max_tokens":512,"messages":[{"role":"user","content":"Reply with OK."}]}`
		if r := api.requests()[0]; r.Path != "/chat/completions" || r.Raw != want || r.Auth != "Bearer "+nanKey || r.ContentType != "application/json" {
			t.Errorf("request %s %s %s %s", r.Path, r.Auth, r.ContentType, r.Raw)
		}
	}
}

// No answer is 000, whatever the reason: a server that holds the connection, one
// that sends a status line and never the body, one that is not there.
func TestProbeNaN_NoAnswerIs000(t *testing.T) {
	for _, mode := range []string{"hang", "stall"} {
		api := newNanAPI(t, map[string]string{"m": mode})
		start := time.Now()
		if got := ProbeNaN(t.Context(), nil, api.URL, nanKey, "m", 300*time.Millisecond); got != "000" {
			t.Errorf("%s probed as %q", mode, got)
		}
		if time.Since(start) > 2*time.Second {
			t.Errorf("%s: the probe outlived its timeout: %s", mode, time.Since(start))
		}
	}
	if got := ProbeNaN(t.Context(), nil, "http://127.0.0.1:1", nanKey, "m", time.Second); got != "000" {
		t.Errorf("a refused connection probed as %q", got)
	}
}

// A redirect is reported as its status. A client that follows redirects by
// default is not trusted to leave it alone.
func TestProbeNaN_DoesNotFollowARedirectWhateverClientItIsGiven(t *testing.T) {
	elsewhere := newNanAPI(t, map[string]string{"": "200"})
	api := newNanAPI(t, map[string]string{"m": "redirect"})
	api.redirect(elsewhere.URL + "/stolen")
	follower := &http.Client{}
	if got := ProbeNaN(t.Context(), follower, api.URL, nanKey, "m", time.Second); got != "307" {
		t.Errorf("a redirect probed as %q", got)
	}
	if n := len(elsewhere.requests()); n != 0 {
		t.Errorf("the redirect was followed: %d requests reached the target", n)
	}
	if follower.CheckRedirect != nil {
		t.Error("ProbeNaN changed the caller's client")
	}
}
