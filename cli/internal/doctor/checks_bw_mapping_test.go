package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The exact live state that took the archive gate down on 2026-08-15: the
// registry names `dockerhub`, the vault holds `DockerHub`. Nothing resolved it
// because item lookup is an exact-name match, and nothing reported it because no
// check compared the two sets — the only symptom was `dotf spec review`
// producing an empty transcript.
func TestCheckBWMapping_MissingItemFails(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT}, expose: {env: DOCKERHUB_TOKEN}}\n" +
		"  - {id: NAN_API_KEY, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key}, expose: {env: NAN_API_KEY}}\n"

	sys := newSys(nil, nil, nil)
	sys.BWItemNames = func() ([]string, error) {
		return []string{"DockerHub", "nan-api-key", "Stripe"}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	if rep.Failures() != 1 {
		t.Fatalf("one declared item is absent from the vault; want 1 failure, got %d\n%s", rep.Failures(), buf.String())
	}
	out := buf.String()
	// The message has to be actionable: which item, and which secret named it.
	for _, want := range []string{"dockerhub", "DOCKERHUB_TOKEN"} {
		if !strings.Contains(out, want) {
			t.Errorf("output must name %q\n%s", want, out)
		}
	}
	// And it must state the consequence, because the symptom never points here.
	if !strings.Contains(out, "without --only") {
		t.Errorf("output must say what breaks, not just what mismatches\n%s", out)
	}
	// The item that DOES exist must not be reported.
	if strings.Contains(out, "nan-api-key: no such item") {
		t.Errorf("a present item must not be reported missing\n%s", out)
	}
}

// Every declared item present is the healthy case: one PASS, no failures, and no
// per-item noise in a section that is clean.
func TestCheckBWMapping_AllPresentPasses(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: NAN_API_KEY, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key}, expose: {env: NAN_API_KEY}}\n"

	sys := newSys(nil, nil, nil)
	sys.BWItemNames = func() ([]string, error) { return []string{"nan-api-key", "unrelated"}, nil }

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	if rep.Failures() != 0 {
		t.Fatalf("all items present; want 0 failures, got %d\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "exist in the vault") {
		t.Errorf("expected the clean-state PASS\n%s", buf.String())
	}
}

// A locked or unreachable vault is not this section's finding — checkBitwardenReach
// owns that severity. Reporting it twice inflates the failure count and trains the
// reader to ignore the section that is actually specific.
func TestCheckBWMapping_UnavailableVaultSkips(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: NAN_API_KEY, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key}, expose: {env: NAN_API_KEY}}\n"

	sys := newSys(nil, nil, nil)
	sys.BWItemNames = func() ([]string, error) { return nil, errors.New("vault is locked") }

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	if rep.Failures() != 0 {
		t.Fatalf("an unavailable vault is not a mapping failure; got %d\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "mapping unverifiable") {
		t.Errorf("expected the unverifiable SKIP\n%s", buf.String())
	}
}

func TestCheckBWMapping_UnreachableDaemonCleansError(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: NAN_API_KEY, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key}, expose: {env: NAN_API_KEY}}\n"

	sys := newSys(nil, nil, nil)
	sys.BWItemNames = func() ([]string, error) {
		return nil, errors.New("bw serve list items: bw serve daemon unreachable: GET /list/object/items: dial tcp 127.0.0.1:8087: connect: connection refused")
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	out := buf.String()
	if !strings.Contains(out, "vault item list unavailable (bw serve daemon not running) — mapping unverifiable") {
		t.Errorf("expected clean daemon unreachable error, got:\n%s", out)
	}
	if strings.Contains(out, "dial tcp 127.0.0.1:8087") {
		t.Errorf("raw dial error should not leak into SKIP message:\n%s", out)
	}
}

// An age-only registry has nothing to compare: SKIP, never a PASS implying the
// mapping was checked. "Nothing to check" and "checked, all good" are different
// statements and only one of them is evidence.
func TestCheckBWMapping_NoBwSecretsSkips(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: SSH_KEY, plane: floor, backend: age-offline, age: ssh.key, expose: {env: SSH_KEY}}\n"

	sys := newSys(nil, nil, nil)
	var called bool
	sys.BWItemNames = func() ([]string, error) { called = true; return nil, nil }

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	if called {
		t.Error("must not touch the vault when no bw-backed secret is declared")
	}
	if rep.Failures() != 0 || !strings.Contains(buf.String(), "no bw-backed secrets") {
		t.Errorf("expected the no-bw-secrets SKIP\n%s", buf.String())
	}
}

// #1820's second reader: the item list comes from bw serve's own cache. An item
// created after the daemon last synced is absent from that cache, so it was
// reported as missing from the vault.
// The listing below answers stale until something syncs.
func TestCheckBWMapping_SyncsBeforeListing(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: NAN_API_KEY, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key}, expose: {env: NAN_API_KEY}}\n"

	sys := newSys(nil, nil, nil)
	synced := false
	sys.BWSync = func() error { synced = true; return nil }
	sys.BWItemNames = func() ([]string, error) {
		if synced {
			return []string{"nan-api-key"}, nil
		}
		return []string{"unrelated"}, nil
	}

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	if rep.Failures() != 0 || rep.Warnings() != 0 {
		t.Fatalf("the item exists once the cache is synced; it was judged on the stale listing:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "exist in the vault") {
		t.Errorf("want the pass line, got:\n%s", buf.String())
	}
}

// Once the daemon has synced, its listing IS the vault: an absent item is absent.
func TestCheckBWMapping_SyncedListingIsAuthoritative(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT}, expose: {env: DOCKERHUB_TOKEN}}\n"

	sys := newSys(nil, nil, nil)
	sys.BWSync = func() error { return nil }
	sys.BWItemNames = func() ([]string, error) { return []string{"nan-api-key"}, nil }

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	if rep.Failures() != 1 {
		t.Fatalf("a synced listing without the item proves it absent; want 1 failure, got %d\n%s", rep.Failures(), buf.String())
	}
}

// A failed sync leaves the daemon's cache at an unknown age. The check refuses to
// answer rather than report a stale absence as a missing item.
func TestCheckBWMapping_FailedSyncSkips(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT}, expose: {env: DOCKERHUB_TOKEN}}\n"

	sys := newSys(nil, nil, nil)
	sys.BWSync = func() error { return errors.New("bw serve: sync timed out") }
	listed := false
	sys.BWItemNames = func() ([]string, error) { listed = true; return []string{"nan-api-key"}, nil }

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	out := buf.String()
	if rep.Failures() != 0 || rep.Warnings() != 0 {
		t.Fatalf("a listing whose sync failed must not be judged:\n%s", out)
	}
	if listed {
		t.Errorf("the cache was read after its sync failed")
	}
	if !strings.Contains(out, "mapping unverifiable") || !strings.Contains(out, "timed out") {
		t.Errorf("the skip must say it did not check, and why, got:\n%s", out)
	}
}

// With no way to sync, the listing is a cache of unknown age, the same as after a
// failed sync. The check does not guess from `bw status`'s lastSync, which dates a
// different cache: it skips, like the escrow check.
func TestCheckBWMapping_NoSyncSkips(t *testing.T) {
	registry := "version: 1\nsecrets:\n" +
		"  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT}, expose: {env: DOCKERHUB_TOKEN}}\n"

	sys := newSys(nil, nil, nil)
	sys.BWSync = nil
	listed := false
	sys.BWItemNames = func() ([]string, error) { listed = true; return []string{"nan-api-key"}, nil }

	var buf bytes.Buffer
	rep := capture(&buf)
	checkBWMapping(sys, patCfg(t, registry), rep)

	out := buf.String()
	if rep.Failures() != 0 || rep.Warnings() != 0 {
		t.Fatalf("a listing nobody synced must not be judged:\n%s", out)
	}
	if listed {
		t.Error("the listing must not be read when there is no way to sync it")
	}
	if !strings.Contains(out, "no way to sync") {
		t.Errorf("the skip must say why, got:\n%s", out)
	}
}
