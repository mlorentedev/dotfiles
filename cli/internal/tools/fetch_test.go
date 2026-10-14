package tools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// serveSequence answers each request with the next status in codes (the last
// one repeats), and counts the requests it saw.
func serveSequence(t *testing.T, codes ...int) (url string, hits *int) {
	t.Helper()
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		code := codes[min(n, len(codes)-1)]
		n++
		w.WriteHeader(code)
		if code == http.StatusOK {
			_, _ = w.Write([]byte("payload"))
		}
	}))
	t.Cleanup(srv.Close)
	old := fetchWait
	fetchWait = 0
	t.Cleanup(func() { fetchWait = old })
	return srv.URL, &n
}

func TestHTTPFetch_RetriesATransientServerError(t *testing.T) {
	url, hits := serveSequence(t, http.StatusInternalServerError, http.StatusTooManyRequests, http.StatusOK)
	dest := filepath.Join(t.TempDir(), "asset")
	if err := HTTPFetch(url, dest); err != nil {
		t.Fatalf("HTTPFetch after a 500 and a 429 = %v, want success on the third attempt", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "payload" || *hits != 3 {
		t.Errorf("body %q after %d requests, want %q after 3", got, *hits, "payload")
	}
}

func TestHTTPFetch_GivesUpAfterTheLastAttempt(t *testing.T) {
	url, hits := serveSequence(t, http.StatusBadGateway)
	err := HTTPFetch(url, filepath.Join(t.TempDir(), "asset"))
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("HTTPFetch against a server that always answers 502 = %v, want the 502", err)
	}
	if *hits != fetchAttempts {
		t.Errorf("%d requests, want %d", *hits, fetchAttempts)
	}
}

func TestHTTPFetch_DoesNotRetryAMissingAsset(t *testing.T) {
	url, hits := serveSequence(t, http.StatusNotFound, http.StatusOK)
	err := HTTPFetch(url, filepath.Join(t.TempDir(), "asset"))
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("HTTPFetch on a 404 = %v, want the 404", err)
	}
	if *hits != 1 {
		t.Errorf("%d requests for a 404, want 1: a missing release asset does not appear on retry", *hits)
	}
}

func TestHTTPFetch_RetriesABodyCutShort(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		if n == 1 {
			// Promise more than is sent, then drop the connection.
			w.Header().Set("Content-Length", "100")
			_, _ = fmt.Fprint(w, "partial")
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close()
			}
			return
		}
		_, _ = w.Write([]byte("payload"))
	}))
	t.Cleanup(srv.Close)
	old := fetchWait
	fetchWait = 0
	t.Cleanup(func() { fetchWait = old })

	dest := filepath.Join(t.TempDir(), "asset")
	if err := HTTPFetch(srv.URL, dest); err != nil {
		t.Fatalf("HTTPFetch after a body cut short = %v, want success on the second attempt", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "payload" || n != 2 {
		t.Errorf("body %q after %d requests, want %q after 2", got, n, "payload")
	}
}

func TestHTTPFetch_DoesNotRetryALocalCreateError(t *testing.T) {
	url, hits := serveSequence(t, http.StatusOK)
	// A directory where the file should be: the create fails locally.
	dest := t.TempDir()
	if err := HTTPFetch(url, dest); err == nil {
		t.Fatal("HTTPFetch into a directory succeeded, want the local error")
	}
	if *hits != 1 {
		t.Errorf("%d requests for a local failure, want 1", *hits)
	}
}

// A full disk fails the write, not the download: repeating it cannot help, and
// the error must name the file. /dev/full answers every write with ENOSPC.
func TestHTTPFetch_DoesNotRetryAFullDisk(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("/dev/full exists on Linux only")
	}
	url, hits := serveSequence(t, http.StatusOK)
	err := HTTPFetch(url, "/dev/full")
	if err == nil || !strings.Contains(err.Error(), "write /dev/full") {
		t.Fatalf("HTTPFetch onto a full disk = %v, want the write error naming the file", err)
	}
	if *hits != 1 {
		t.Errorf("%d requests for a full disk, want 1", *hits)
	}
}
