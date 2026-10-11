package secrets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// registrySize is the live registry's bw-backed entry count when CLI-024's AC2 was
// measured (2026-10-10: 41 entries, 1.361 s end to end). The exact number does not
// matter; a realistic one keeps the per-read cost visible as a total.
const registrySize = 41

// newRegistryFake returns a daemon holding registrySize items and a handler that
// counts every request and every POST /sync.
func newRegistryFake(requests, syncs *int) (*fakeBWServe, http.HandlerFunc) {
	f := &fakeBWServe{status: "unlocked", names: map[string]string{}, items: map[string]json.RawMessage{}}
	for i := 0; i < registrySize; i++ {
		id, name := fmt.Sprintf("id-%02d", i), fmt.Sprintf("item-%02d", i)
		f.names[id] = name
		f.items[id] = json.RawMessage(fmt.Sprintf(`{"id":%q,"name":%q,"notes":"n-%02d"}`, id, name, i))
	}
	inner := f.handler()
	return f, func(w http.ResponseWriter, r *http.Request) {
		*requests++
		if r.URL.Path == "/sync" {
			*syncs++
		}
		inner(w, r)
	}
}

// TestBWServeReader_ReadCostIsBoundedPerSecret is AC2's gate. The 14-50 s baseline
// was a `bw` process per secret; the daemon answers each read with local HTTP, and
// that only stays under 2 s while a read costs a fixed, small number of requests
// with no forced sync and no backoff. A forced sync is a server round trip (about a
// second live), so one per read would put a 41-entry verify back above the budget
// while every shape test stayed green. A wall-clock assertion against a local fake
// could not see that; counting the requests can.
func TestBWServeReader_ReadCostIsBoundedPerSecret(t *testing.T) {
	var requests, syncs int
	_, h := newRegistryFake(&requests, &syncs)
	srv := httptest.NewServer(h)
	defer srv.Close()
	r := BWServeReader{Client: BWServeClient{
		BaseURL: srv.URL,
		Sleep:   func(d time.Duration) { t.Fatalf("read path slept %v: a backoff on a populated vault", d) },
	}}

	for i := 0; i < registrySize; i++ {
		want := fmt.Sprintf("n-%02d", i)
		got, err := r.Field(fmt.Sprintf("item-%02d", i), "notes")
		if err != nil || got != want {
			t.Fatalf("Field(item-%02d) = %q, %v; want %q", i, got, err, want)
		}
	}

	// One list and one item get per read.
	if want := 2 * registrySize; requests != want {
		t.Fatalf("%d reads made %d requests, want %d (2 per read)", registrySize, requests, want)
	}
	if syncs != 0 {
		t.Fatalf("%d reads forced %d syncs, want 0: a sync is a server round trip per read", registrySize, syncs)
	}
}

// BenchmarkBWServeReader_Field measures one serve-backed read against a local fake:
// the HTTP and decode cost the daemon path adds, without the network or the vault.
// The gate is the test above; this is the number.
func BenchmarkBWServeReader_Field(b *testing.B) {
	var requests, syncs int
	_, h := newRegistryFake(&requests, &syncs)
	srv := httptest.NewServer(h)
	defer srv.Close()
	r := BWServeReader{Client: BWServeClient{BaseURL: srv.URL}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.Field(fmt.Sprintf("item-%02d", i%registrySize), "notes"); err != nil {
			b.Fatal(err)
		}
	}
}
