package secrets

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"
)

// BUG-113: while bw serve runs a forced sync it answers the unfiltered item list
// with success and NO items. Every reader below must wait that window out, and
// one that never closes must surface as its own error — never as an empty vault
// and never as ErrBWItemNotFound, which `set` reads as "create it".

func syncWindowFake(emptyReads int) *fakeBWServe {
	return &fakeBWServe{
		status: "unlocked",
		names:  map[string]string{"id-1": "nan-api-key"},
		items: map[string]json.RawMessage{
			"id-1": json.RawMessage(`{"id":"id-1","name":"nan-api-key","login":{"password":"x"}}`),
		},
		emptyListReads: emptyReads,
	}
}

func recordingClient(url string) (BWServeClient, *[]time.Duration) {
	var slept []time.Duration
	return BWServeClient{BaseURL: url, Sleep: func(d time.Duration) { slept = append(slept, d) }}, &slept
}

func TestBWServeReader_ItemNames_WaitsOutASyncWindow(t *testing.T) {
	srv := httptest.NewServer(syncWindowFake(3).handler())
	defer srv.Close()
	c, slept := recordingClient(srv.URL)

	names, err := BWServeReader{Client: c}.ItemNames()
	if err != nil {
		t.Fatalf("ItemNames: %v", err)
	}
	if len(names) != 1 || names[0] != "nan-api-key" {
		t.Fatalf("names = %v, want [nan-api-key]", names)
	}
	if len(*slept) != 3 {
		t.Fatalf("waited %d times, want 3 (one per empty read)", len(*slept))
	}
}

func TestBWServeReader_EveryListReaderWaitsOutASyncWindow(t *testing.T) {
	readers := map[string]func(BWServeClient) error{
		"ItemRevisions": func(c BWServeClient) error { _, err := BWServeReader{Client: c}.ItemRevisions(); return err },
		"Field": func(c BWServeClient) error {
			_, err := BWServeReader{Client: c}.Field("nan-api-key", "password")
			return err
		},
		"ListItems": func(c BWServeClient) error { _, err := c.ListItems(); return err },
	}
	for name, read := range readers {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(syncWindowFake(2).handler())
			defer srv.Close()
			c, slept := recordingClient(srv.URL)
			if err := read(c); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if len(*slept) != 2 {
				t.Fatalf("waited %d times, want 2", len(*slept))
			}
		})
	}
}

func TestBWServeReader_PersistentEmptyListIsNotAbsence(t *testing.T) {
	srv := httptest.NewServer(syncWindowFake(1 << 20).handler())
	defer srv.Close()
	c, slept := recordingClient(srv.URL)

	_, err := BWServeReader{Client: c}.Field("nan-api-key", "password")
	if !errors.Is(err, ErrBWServeEmptyList) {
		t.Fatalf("err = %v, want ErrBWServeEmptyList", err)
	}
	if errors.Is(err, ErrBWItemNotFound) {
		t.Fatalf("an empty listing must not read as an absent item: %v", err)
	}
	var total time.Duration
	for _, d := range *slept {
		total += d
	}
	if total < 3*time.Second || total > 10*time.Second {
		t.Fatalf("budget %s: must outlast a multi-second sync yet stay bounded", total)
	}
}
