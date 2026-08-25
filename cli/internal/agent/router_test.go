package agent

import (
	"context"
	"testing"
	"time"
)

// stubBackend records what it was asked and answers a fixed response.
type stubBackend struct {
	name  string
	serve map[string]bool // pools it can serve
	seen  []string
	resp  Response
}

func (b *stubBackend) Serves(pool string) bool { return b.serve[pool] }
func (b *stubBackend) Dispatch(_ context.Context, req Request) Response {
	b.seen = append(b.seen, b.name+"->"+req.Pool)
	return b.resp
}

func TestRouter_PicksTheBackendThatServesThePool(t *testing.T) {
	sub := &stubBackend{name: "subprocess", serve: map[string]bool{"claude": true, "nan": true},
		resp: Response{Status: StatusOK, Output: "sub"}}
	hive := &stubBackend{name: "hive", serve: map[string]bool{"nan": true},
		resp: Response{Status: StatusOK, Output: "hive"}}

	r := NewRouter([]NamedBackend{
		{Name: "subprocess", Backend: sub, Serves: sub.Serves},
		{Name: "hive", Backend: hive, Serves: hive.Serves},
	}, "")

	// claude is served only by subprocess.
	if got := r.Dispatch(context.Background(), Request{Pool: "claude", Model: "opus"}); got.Output != "sub" {
		t.Errorf("claude routed to %q, want subprocess", got.Output)
	}
	// nan is served by both; declaration order is the tie-break, and subprocess
	// is declared first when pi is present.
	if got := r.Dispatch(context.Background(), Request{Pool: "nan", Model: "x"}); got.Output != "sub" {
		t.Errorf("nan routed to %q, want subprocess (first that serves it)", got.Output)
	}
}

// A pool no backend can serve is UNAVAILABLE, not a task failure: nothing ran,
// and the next chain entry may name a pool that something can serve.
func TestRouter_APoolNoBackendServesIsUnavailable(t *testing.T) {
	hive := &stubBackend{name: "hive", serve: map[string]bool{"nan": true}, resp: Response{Status: StatusOK}}
	r := NewRouter([]NamedBackend{{Name: "hive", Backend: hive, Serves: hive.Serves}}, "")

	got := r.Dispatch(context.Background(), Request{Pool: "claude", Model: "opus"})
	if got.Status != StatusPoolUnavailable {
		t.Errorf("status = %q, want %q", got.Status, StatusPoolUnavailable)
	}
	if len(hive.seen) != 0 {
		t.Errorf("hive was asked to serve a pool it does not: %v", hive.seen)
	}
}

// --backend restricts the router to one member. The semantics that matter for
// AC6's smoke: a forced backend that cannot serve an entry reports the entry
// unavailable so the CHAIN advances, rather than failing the whole dispatch.
// `--backend hive --tier mid` must skip claude:sonnet and answer on nan.
func TestRouter_AForcedBackendSkipsEntriesItCannotServe(t *testing.T) {
	sub := &stubBackend{name: "subprocess", serve: map[string]bool{"claude": true, "nan": true},
		resp: Response{Status: StatusOK, Output: "sub"}}
	hive := &stubBackend{name: "hive", serve: map[string]bool{"nan": true},
		resp: Response{Status: StatusOK, Output: "hive"}}

	r := NewRouter([]NamedBackend{
		{Name: "subprocess", Backend: sub, Serves: sub.Serves},
		{Name: "hive", Backend: hive, Serves: hive.Serves},
	}, "hive")

	rec := Dispatch(context.Background(), Options{
		Tier:    "mid",
		Chain:   []string{"claude:sonnet", "nan:deepseek-v4-flash"},
		Timeout: time.Minute,
		Now:     fixedClock(time.Millisecond),
	}, r)

	if rec.Status != StatusOK || rec.Pool != "nan" {
		t.Errorf("record = %s on %s, want ok on nan", rec.Status, rec.Pool)
	}
	if rec.Output != "hive" {
		t.Errorf("output = %q; the forced backend was not the one that answered", rec.Output)
	}
	if len(sub.seen) != 0 {
		t.Errorf("subprocess was used despite --backend hive: %v", sub.seen)
	}
	if len(rec.Attempts) != 2 || rec.Attempts[0].Status != StatusPoolUnavailable {
		t.Errorf("attempts = %+v; the unservable entry must be skipped, not fatal", rec.Attempts)
	}
}

func TestRouter_AnUnknownForcedBackendIsRefused(t *testing.T) {
	sub := &stubBackend{name: "subprocess", serve: map[string]bool{"claude": true}}
	_, err := ResolveRouter([]NamedBackend{{Name: "subprocess", Backend: sub, Serves: sub.Serves}}, "orca")
	if err == nil {
		t.Fatal("an unknown --backend was accepted")
	}
	if got := err.Error(); got == "" {
		t.Error("refusal carries no message")
	}
}

// With no backend able to serve anything, the router must still be usable and
// report every entry unavailable — the chain-exhausted path, not a crash.
func TestRouter_NoBackendsAtAll(t *testing.T) {
	r := NewRouter(nil, "")
	rec := Dispatch(context.Background(), Options{
		Tier:    "mid",
		Chain:   []string{"claude:sonnet", "nan:x"},
		Timeout: time.Minute,
		Now:     fixedClock(time.Millisecond),
	}, r)

	if rec.Status != StatusChainExhausted {
		t.Errorf("status = %q, want %q", rec.Status, StatusChainExhausted)
	}
}

// dry-run serves every pool, so leaving it in probe order would make it the
// answer to every dispatch that named no backend: a dispatch that ran nothing
// and exited 0. That is the failure PR B's decision table rejected, and it was
// reintroduced here by accident when dry-run joined DefaultBackends.
func TestRouter_DryRunIsNeverChosenByTheProbe(t *testing.T) {
	dry := &stubBackend{name: "dry-run", serve: map[string]bool{}, resp: Response{Status: StatusDryRun}}
	r := NewRouter([]NamedBackend{
		{Name: "dry-run", Backend: dry, Serves: func(string) bool { return true }, ExplicitOnly: true},
	}, "")

	got := r.Dispatch(context.Background(), Request{Pool: "nan", Model: "m"})
	if got.Status != StatusPoolUnavailable {
		t.Errorf("status = %q, want %q: an unforced dispatch must not silently resolve to dry-run",
			got.Status, StatusPoolUnavailable)
	}

	// Named explicitly, it is reachable.
	forced := NewRouter([]NamedBackend{
		{Name: "dry-run", Backend: dry, Serves: func(string) bool { return true }, ExplicitOnly: true},
	}, "dry-run")
	if got := forced.Dispatch(context.Background(), Request{Pool: "nan", Model: "m"}); got.Status != StatusDryRun {
		t.Errorf("status = %q, want dry_run when named explicitly", got.Status)
	}
}
