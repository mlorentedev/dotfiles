package secrets

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

// TestLiveBWServeWriter_CanaryRoundTrip is AC1/AC2 of BUG-084 against a REAL unlocked
// `bw serve` daemon — the half a fake cannot prove.
//
// It is opt-in and skipped everywhere by default, including CI, because it needs an
// unlocked vault. Run it deliberately:
//
//	DOTF_LIVE_BW=1 go test ./internal/secrets/ -run TestLiveBWServeWriter -v
//
// Safety properties, each deliberate:
//
//   - It NEVER touches a registry secret. It creates its own throwaway item, named with
//     a timestamp so a leftover from an interrupted run can never be mistaken for a real
//     entry, and deletes it at the end.
//   - It NEVER prints a value. Every comparison is over `Fingerprint` (a truncated
//     SHA-256), which is enough to prove two values differ and is not reversible. The
//     values it writes are synthetic anyway, but the habit is the point:
//     docs/lessons.md, "Redact at the producer" — and a second credential was leaked in
//     this repo on the same day by curling this very endpoint to look at the reply.
//   - Cleanup runs via t.Cleanup, so a mid-test failure still removes the item.
func TestLiveBWServeWriter_CanaryRoundTrip(t *testing.T) {
	if os.Getenv("DOTF_LIVE_BW") != "1" {
		t.Skip("live Bitwarden smoke: set DOTF_LIVE_BW=1 with an unlocked `bw serve` daemon")
	}

	client := BWServeClient{}
	st, err := client.Status()
	if err != nil {
		t.Fatalf("no reachable bw serve daemon (`dotf secrets unlock` first): %v", err)
	}
	if st != "unlocked" {
		t.Fatalf("daemon is %q, need \"unlocked\" (`dotf secrets unlock`)", st)
	}

	// Prove the premise of the whole bug: no ambient session anywhere. If this is set,
	// a passing test would prove nothing about the daemon path.
	if os.Getenv("BW_SESSION") != "" {
		t.Fatal("BW_SESSION is set — unset it, or this test cannot show the daemon path works without one")
	}

	w := BWServeWriter{Client: client}
	item := fmt.Sprintf("dotf-canary-bug084-%d", time.Now().Unix())

	// --- AC2: create, including folder resolution (OPS-028 taxonomy) -------------
	folderID, err := w.ResolveFolder("apps")
	if err != nil {
		t.Fatalf("ResolveFolder(apps): %v", err)
	}
	t.Logf("resolved folder apps -> id present: %v", folderID != "")

	first := fmt.Sprintf("canary-%d-first", time.Now().UnixNano())
	if err := w.CreateItem(item, "password", first, folderID); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	t.Cleanup(func() { deleteCanaryItem(t, client, item) })

	r := BWServeReader{Client: client}
	got, err := r.Field(item, "password")
	if err != nil {
		t.Fatalf("read back created item: %v", err)
	}
	if Fingerprint(got) != Fingerprint(first) {
		t.Fatalf("created value fingerprint mismatch: got %s, want %s", Fingerprint(got), Fingerprint(first))
	}
	t.Logf("AC2 create: fingerprint %s", Fingerprint(first))

	// --- AC1: update an existing item -------------------------------------------
	second := fmt.Sprintf("canary-%d-second", time.Now().UnixNano())
	if err := w.SetField(item, "password", second); err != nil {
		t.Fatalf("SetField: %v", err)
	}
	// NO explicit sync here, deliberately: BWServeWriter.syncAfterWrite must make the
	// write visible on its own. This test failed exactly here before that existed.
	got2, err := r.Field(item, "password")
	if err != nil {
		t.Fatalf("read back updated item: %v", err)
	}
	if Fingerprint(got2) != Fingerprint(second) {
		t.Fatalf("updated value fingerprint mismatch: got %s, want %s", Fingerprint(got2), Fingerprint(second))
	}
	if Fingerprint(got2) == Fingerprint(first) {
		t.Fatal("value did not change — the write did not take effect on the read path")
	}
	t.Logf("AC1 update: %s -> %s", Fingerprint(first), Fingerprint(second))

	// --- the sibling-preservation property, live --------------------------------
	if err := w.SetField(item, "CANARY_FIELD", "sibling-value"); err != nil {
		t.Fatalf("SetField(custom field): %v", err)
	}
	still, err := r.Field(item, "password")
	if err != nil {
		t.Fatalf("read password after custom-field write: %v", err)
	}
	if Fingerprint(still) != Fingerprint(second) {
		t.Fatal("writing a custom field clobbered the login password — read-modify-write is broken")
	}
	t.Logf("sibling preserved: password still %s after writing CANARY_FIELD", Fingerprint(second))
}

// deleteCanaryItem removes the throwaway item. It calls the daemon directly rather than
// through a production seam on purpose: `dotf secrets` has no delete capability and this
// test must not be the reason one gets added (#938 designs the retire path deliberately).
func deleteCanaryItem(t *testing.T, c BWServeClient, item string) {
	t.Helper()
	raw, err := BWServeReader{Client: c}.getItemJSON(item)
	if err != nil {
		t.Errorf("CLEANUP FAILED - delete %q by hand: %v", item, err)
		return
	}
	id, err := itemID(raw)
	if err != nil {
		t.Errorf("CLEANUP FAILED - delete %q by hand: %v", item, err)
		return
	}
	if _, err := c.call(http.MethodDelete, "/object/item/"+url.PathEscape(id), nil); err != nil {
		t.Errorf("CLEANUP FAILED - delete %q by hand: %v", item, err)
		return
	}
	t.Logf("cleaned up canary item %s", item)
}
