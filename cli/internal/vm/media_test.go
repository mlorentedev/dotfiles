package vm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func serve(t *testing.T, body []byte, hits *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++
		if r.URL.Path != "/media.iso" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEnsure_DownloadsAndVerifies(t *testing.T) {
	body := []byte("an iso")
	hits := 0
	srv := serve(t, body, &hits)
	dest := filepath.Join(t.TempDir(), "media", "x.iso")
	m := Medium{Name: "x", SHA256: strings.ToUpper(sum(body))}

	var out bytes.Buffer
	if err := Ensure(srv.Client(), dest, m, srv.URL+"/media.iso?sig=secret", &out); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); !bytes.Equal(got, body) {
		t.Fatalf("dest holds %q", got)
	}
	if strings.Contains(out.String(), "secret") {
		t.Errorf("progress printed the link's query: %s", out.String())
	}
	// A second call keeps the verified file and fetches nothing.
	if err := Ensure(srv.Client(), dest, m, srv.URL+"/media.iso", &out); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Errorf("fetched %d times, want 1", hits)
	}
}

// A file that does not hash to the pin never lands at dest: a wrong language,
// a truncated download or a tampered mirror all stop the build there.
func TestEnsure_RefusesAHashMismatch(t *testing.T) {
	hits := 0
	srv := serve(t, []byte("the wrong iso"), &hits)
	dir := t.TempDir()
	dest := filepath.Join(dir, "x.iso")
	m := Medium{Name: "x", SHA256: sum([]byte("the right iso"))}

	err := Ensure(srv.Client(), dest, m, srv.URL+"/media.iso", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), sum([]byte("the wrong iso"))) || !strings.Contains(err.Error(), m.SHA256) {
		t.Fatalf("want an error naming both hashes, got %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("left %d file(s) behind", len(entries))
	}
}

func TestEnsure_CopiesALocalFile(t *testing.T) {
	body := []byte("a local iso")
	src := filepath.Join(t.TempDir(), "Win11_Spanish_Arm64.iso")
	if err := os.WriteFile(src, body, 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "x.iso")
	if err := Ensure(http.DefaultClient, dest, Medium{Name: "x", SHA256: sum(body)}, src, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); !bytes.Equal(got, body) {
		t.Fatalf("dest holds %q", got)
	}
}

// A file already at dest that does not match is replaced when there is a
// source, and is an error naming the remedy when there is none.
func TestEnsure_AStaleFileAtDest(t *testing.T) {
	body := []byte("the right iso")
	m := Medium{Name: "Windows 11 Arm64 (Spanish)", SHA256: sum(body)}
	dest := filepath.Join(t.TempDir(), "x.iso")
	if err := os.WriteFile(dest, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Ensure(http.DefaultClient, dest, m, "", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), dest) {
		t.Fatalf("want an error naming %s, got %v", dest, err)
	}

	hits := 0
	srv := serve(t, body, &hits)
	if err := Ensure(srv.Client(), dest, m, srv.URL+"/media.iso", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); !bytes.Equal(got, body) {
		t.Fatalf("dest holds %q", got)
	}
}

func TestEnsure_NoFileAndNoSourceSaysWhatToFetch(t *testing.T) {
	m := Medium{Name: "Windows 11 Arm64 (Spanish)", SHA256: sum([]byte("x"))}
	dest := filepath.Join(t.TempDir(), "x.iso")
	err := Ensure(http.DefaultClient, dest, m, "", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), m.Name) || !strings.Contains(err.Error(), dest) {
		t.Fatalf("want an error naming the medium and %s, got %v", dest, err)
	}
}

func TestEnsure_AFailedDownloadIsAnError(t *testing.T) {
	hits := 0
	srv := serve(t, nil, &hits)
	dest := filepath.Join(t.TempDir(), "x.iso")
	err := Ensure(srv.Client(), dest, Medium{Name: "x", SHA256: sum(nil)}, srv.URL+"/gone.iso", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("want a 404 error, got %v", err)
	}
	if _, serr := os.Stat(dest); !os.IsNotExist(serr) {
		t.Error("a failed download left a file at dest")
	}
}

// The pins are what the build trusts: each is a full SHA256.
func TestMediaPins(t *testing.T) {
	for _, m := range []Medium{WindowsISO, VirtioWin} {
		if len(m.SHA256) != 64 || strings.Trim(strings.ToLower(m.SHA256), "0123456789abcdef") != "" {
			t.Errorf("%s: pin %q is not a SHA256", m.Name, m.SHA256)
		}
	}
	if !strings.HasPrefix(VirtioWin.URL, "https://") {
		t.Errorf("virtio-win URL %q is not https", VirtioWin.URL)
	}
}
