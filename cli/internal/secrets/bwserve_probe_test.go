package secrets

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestProbe_ReportsTransportFactsWithoutInterpreting: a non-2xx is an observation, not
// an error. That inversion is the whole point — the caller is diagnosing a failure, so
// the failure must be returned as data.
func TestProbe_ReportsTransportFactsWithoutInterpreting(t *testing.T) {
	const body = "Internal Server Error"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	res, err := BWServeClient{BaseURL: srv.URL}.Probe(http.MethodGet, "/object/item/x")
	if err != nil {
		t.Fatalf("a non-2xx must NOT be an error — it is the observation: %v", err)
	}
	if res.Status != http.StatusInternalServerError {
		t.Fatalf("Status = %d, want 500", res.Status)
	}
	if res.Size != len(body) {
		t.Fatalf("Size = %d, want %d", res.Size, len(body))
	}
	if !strings.HasPrefix(res.ContentType, "text/plain") {
		t.Fatalf("ContentType = %q", res.ContentType)
	}
	if string(res.Body) != body {
		t.Fatalf("Body was not carried back verbatim")
	}
}

// TestProbe_UnreachableMatchesCallContract: Start()'s poll loop keys off
// ErrBWServeUnreachable, so Probe must not invent a different signal for the same state.
func TestProbe_UnreachableMatchesCallContract(t *testing.T) {
	srv := httptest.NewServer((&fakeBWServe{}).handler())
	url := srv.URL
	srv.Close() // nothing listening now

	res, err := BWServeClient{BaseURL: url}.Probe(http.MethodGet, "/status")
	if !errors.Is(err, ErrBWServeUnreachable) {
		t.Fatalf("an unreachable daemon must wrap ErrBWServeUnreachable, got %v", err)
	}
	if res.Status != 0 {
		t.Fatalf("Status = %d, want 0 when the request never completed", res.Status)
	}
}

// TestProbe_ErrorsNeverCarryBodyBytes is the security-critical assertion, and the reason
// this seam is separate from call() rather than a flag on it.
//
// Errors travel into logs, transcripts, and PR comments. A 200 from /object/item/<id> IS
// a credential, so no error path here may ever embed body bytes — the caller gets them
// in the struct, deliberately and visibly, or not at all. Three credential exposures in
// this repo in one day came from body bytes reaching a place nobody meant them to.
func TestProbe_ErrorsNeverCarryBodyBytes(t *testing.T) {
	const sentinel = "SUPER-SECRET-CANARY-VALUE"

	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "200 with a secret body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"data":{"password":"` + sentinel + `"}}`))
			},
		},
		{
			name: "500 whose body happens to contain one",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(sentinel))
			},
		},
		{
			name: "a body that claims a shorter length than it sends",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "5")
				_, _ = w.Write([]byte(sentinel))
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			_, err := BWServeClient{BaseURL: srv.URL}.Probe(http.MethodGet, "/object/item/x")
			if err != nil && strings.Contains(err.Error(), sentinel) {
				t.Fatalf("error leaked body bytes: %s", err.Error())
			}
		})
	}
}

// TestProbe_UsesTheSameTransportAsCall pins contract point 1: Probe must travel the same
// path as the calls it diagnoses. A probe on its own client would not reproduce the
// connection behaviour under investigation — which is exactly why an external curl could
// not characterise #988.
func TestProbe_UsesTheSameTransportAsCall(t *testing.T) {
	var probeUA, callUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/object") {
			probeUA = r.Header.Get("User-Agent")
			_, _ = w.Write([]byte(`{"success":true}`))
			return
		}
		callUA = r.Header.Get("User-Agent")
		writeEnvelope(w, true, "", map[string]any{"template": map[string]string{"status": "locked"}})
	}))
	defer srv.Close()

	c := BWServeClient{BaseURL: srv.URL}
	if _, err := c.Probe(http.MethodGet, "/object/item/x"); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if _, err := c.Status(); err != nil {
		t.Fatalf("Status: %v", err)
	}
	if probeUA != callUA {
		t.Fatalf("Probe and call must present as the same client: probe=%q call=%q", probeUA, callUA)
	}
}

// A failed read must not hand the caller the bytes it did manage to get. The
// error string was already covered; the STRUCT was not, and it was carrying the
// partial body while the comment above it said otherwise. A truncated read of a
// 200 is still credential material, so the bytes must not survive the failure.
func TestProbe_ReadFailureCarriesNoPartialBody(t *testing.T) {
	const sentinel = "SUPER-SECRET-CANARY-VALUE"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Promise more than is sent: io.ReadAll fails with ErrUnexpectedEOF after
		// the sentinel has already arrived.
		w.Header().Set("Content-Length", "4096")
		_, _ = w.Write([]byte(sentinel))
	}))
	defer srv.Close()

	res, err := (BWServeClient{BaseURL: srv.URL}).Probe(http.MethodGet, "/object/item/x")
	if err == nil {
		t.Fatal("want a read error from a truncated response")
	}
	if strings.Contains(string(res.Body), sentinel) {
		t.Error("the partial body survived a failed read; a truncated 200 is still a credential")
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Error("the error carried body bytes")
	}
}
