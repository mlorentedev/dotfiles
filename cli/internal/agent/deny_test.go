package agent

import (
	"context"
	"testing"
	"time"
)

func denyOnly(pools ...string) func(string) bool {
	set := map[string]bool{}
	for _, p := range pools {
		set[p] = true
	}
	return func(pool string) bool { return set[pool] }
}

// A denied pool is skipped and the chain continues: denial is a fact about
// THIS machine, and another entry may name a pool it does allow. The point of
// per-pool denial would be lost if one denied entry failed the whole dispatch.
func TestDispatch_ADeniedPoolIsSkippedAndTheChainAdvances(t *testing.T) {
	be := &scriptedBackend{byEntry: map[string]Response{
		"claude:sonnet":         {Status: StatusOK, Output: "must never be reached"},
		"nan:deepseek-v4-flash": {Status: StatusOK, Output: "answered"},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier:    "mid",
		Chain:   []string{"claude:sonnet", "nan:deepseek-v4-flash"},
		Timeout: time.Minute,
		Denied:  denyOnly("claude"),
		Now:     fixedClock(time.Millisecond),
	}, be)

	if rec.Status != StatusOK || rec.Pool != "nan" {
		t.Errorf("record = %s on %s, want ok on nan", rec.Status, rec.Pool)
	}
	// The denied pool must never have been dispatched to. Asserting only the
	// final record would pass on a dispatcher that asked claude first and threw
	// the answer away — which would already have sent the task's content to a
	// forbidden pool, the exact thing denial exists to prevent.
	if len(be.seen) != 1 || be.seen[0] != "nan:deepseek-v4-flash" {
		t.Errorf("backend saw %v; a denied pool must not be reached at all", be.seen)
	}
	if len(rec.Attempts) != 2 || rec.Attempts[0].Status != StatusDenied {
		t.Errorf("attempts = %+v; the denial must leave a trace", rec.Attempts)
	}
}

// Every entry denied is its own outcome, not "no pool could serve this". The
// two send an operator to different places: one to quota and outages, the other
// to machine.json.
func TestDispatch_EveryEntryDeniedIsItsOwnStatus(t *testing.T) {
	be := &scriptedBackend{byEntry: map[string]Response{}}

	rec := Dispatch(context.Background(), Options{
		Tier:    "mid",
		Chain:   []string{"claude:sonnet", "nan:deepseek-v4-flash"},
		Timeout: time.Minute,
		Denied:  denyOnly("claude", "nan"),
		Now:     fixedClock(time.Millisecond),
	}, be)

	if rec.Status != StatusDenied {
		t.Errorf("status = %q, want %q", rec.Status, StatusDenied)
	}
	if rec.Exit == 0 {
		t.Error("exit = 0; a dispatch that ran nowhere must not read as success")
	}
	if len(be.seen) != 0 {
		t.Errorf("backend saw %v; nothing should have been dispatched", be.seen)
	}
}

// An unidentified machine denies everything, so the top tier's single entry is
// refused. The record must say DENIED rather than escalated: escalation means
// "the pool could not serve this", and here the pool was never asked.
func TestDispatch_AnUnidentifiedMachineDeniesEvenTheTopTier(t *testing.T) {
	be := &scriptedBackend{byEntry: map[string]Response{
		"claude:opus": {Status: StatusOK, Output: "must never be reached"},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier:    TierTop,
		Chain:   []string{"claude:opus"},
		Timeout: time.Minute,
		Denied:  func(string) bool { return true }, // the unidentified-machine policy
		Now:     fixedClock(time.Millisecond),
	}, be)

	if rec.Status != StatusDenied {
		t.Errorf("status = %q, want %q", rec.Status, StatusDenied)
	}
	if len(be.seen) != 0 {
		t.Errorf("backend saw %v on a machine that denies everything", be.seen)
	}
}

// Denial is evaluated before the semaphore, so a forbidden pool consumes
// nothing at all — not even a slot that some other dispatch could have used.
func TestDispatch_ADeniedPoolTakesNoSemaphoreSlot(t *testing.T) {
	dir := t.TempDir()
	sem := NewSemaphore(dir)
	be := &scriptedBackend{byEntry: map[string]Response{
		"nan:deepseek-v4-flash": {Status: StatusOK, Output: "answered"},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier:      "mid",
		Chain:     []string{"claude:sonnet", "nan:deepseek-v4-flash"},
		Timeout:   time.Minute,
		Denied:    denyOnly("claude"),
		Semaphore: sem,
		Capacity:  func(string) (int, bool) { return 1, true },
		Now:       fixedClock(time.Millisecond),
	}, be)

	if rec.Status != StatusOK {
		t.Fatalf("status = %q, want ok", rec.Status)
	}
	// claude's only slot must be free: it was denied, not dispatched to.
	s, err := sem.Acquire("claude", 1)
	if err != nil {
		t.Fatalf("the denied pool's slot was consumed: %v", err)
	}
	s.Release()
}
