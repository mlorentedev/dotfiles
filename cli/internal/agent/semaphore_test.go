package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSemaphore_BoundsConcurrencyAtCapacity(t *testing.T) {
	dir := t.TempDir()
	sem := NewSemaphore(dir)

	held := make([]*Slot, 0, 3)
	for i := 0; i < 3; i++ {
		s, err := sem.Acquire("nan", 3)
		if err != nil {
			t.Fatalf("acquire %d of 3: %v", i+1, err)
		}
		held = append(held, s)
	}

	// The fourth is refused, and refused as UNAVAILABLE rather than as an
	// error: a full pool is a reason to try the next chain entry, not a reason
	// to stop.
	if _, err := sem.Acquire("nan", 3); err == nil {
		t.Fatal("a fourth slot was granted against a capacity of 3")
	} else if !IsPoolBusy(err) {
		t.Errorf("a full pool reported %v; it must classify as busy so the chain advances", err)
	}

	held[1].Release()
	s, err := sem.Acquire("nan", 3)
	if err != nil {
		t.Fatalf("acquire after a release: %v", err)
	}
	s.Release()
	for _, h := range held {
		h.Release()
	}
}

// Pools are separate quotas, so one saturated pool must not block another. A
// single global counter would pass every other test in this file.
func TestSemaphore_PoolsAreIndependent(t *testing.T) {
	dir := t.TempDir()
	sem := NewSemaphore(dir)

	a, err := sem.Acquire("nan", 1)
	if err != nil {
		t.Fatalf("acquire nan: %v", err)
	}
	defer a.Release()

	if _, err := sem.Acquire("nan", 1); err == nil {
		t.Error("nan granted a second slot against a capacity of 1")
	}
	b, err := sem.Acquire("claude", 1)
	if err != nil {
		t.Fatalf("claude refused while only nan was saturated: %v", err)
	}
	b.Release()
}

// Release is idempotent because the timeout path releases early and the normal
// path releases again on the way out. A double release that freed someone
// else's slot would be worse than a leak.
func TestSemaphore_ReleaseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	sem := NewSemaphore(dir)

	s, err := sem.Acquire("nan", 1)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	s.Release()
	s.Release()

	other, err := sem.Acquire("nan", 1)
	if err != nil {
		t.Fatalf("the slot was not free after a double release: %v", err)
	}
	// The second Release must not have freed a slot it no longer owned.
	s.Release()
	if _, err := sem.Acquire("nan", 1); err == nil {
		t.Error("a stale Slot freed a slot owned by someone else")
	}
	other.Release()
}

// An unreadable semaphore is not an empty one (AC4's first half, and the half
// C1 can hold: the state directory is the counter). Reading a broken counter as
// "nothing in use" is the fail-OPEN direction — it would grant every slot.
func TestSemaphore_UnreadableStateIsAnErrorNotZeroInUse(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: mode bits do not deny access, so the case cannot be built")
	}
	dir := t.TempDir()
	blocked := filepath.Join(dir, "slots")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	sem := NewSemaphore(blocked)
	_, err := sem.Acquire("nan", 3)
	if err == nil {
		t.Fatal("an unusable state directory granted a slot; a counter that cannot be read reads as zero in use")
	}
	if IsPoolBusy(err) {
		t.Error("an unreadable counter classified as busy: that advances the chain and tries the next pool " +
			"against the same broken state, rather than stopping")
	}
}

// AC3's sharpest clause. The slot must be free BEFORE the worker is reaped —
// asserted by taking the freed slot while the worker is still running, which is
// only possible if the release does not wait on it.
func TestDispatch_TimeoutReleasesSlotBeforeReap(t *testing.T) {
	dir := t.TempDir()
	sem := NewSemaphore(dir)

	workerRunning := make(chan struct{})
	releaseWorker := make(chan struct{})
	var once sync.Once

	// A backend that ignores its context entirely: the dispatcher must not be
	// at the mercy of a backend that declines to notice its own deadline.
	be := backendFunc(func(_ context.Context, _ Request) Response {
		once.Do(func() { close(workerRunning) })
		<-releaseWorker
		return Response{Status: StatusOK, Output: "answered far too late"}
	})
	defer close(releaseWorker)

	done := make(chan Record, 1)
	go func() {
		done <- Dispatch(context.Background(), Options{
			Tier:      "mid",
			Role:      "reviewer",
			Task:      "hang",
			Chain:     []string{"nan:deepseek-v4-flash"},
			Timeout:   50 * time.Millisecond,
			Semaphore: sem,
			Capacity:  func(string) (int, bool) { return 1, true },
		}, be)
	}()

	<-workerRunning

	var rec Record
	select {
	case rec = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Dispatch never returned: it is waiting on a worker that ignores its deadline")
	}

	if rec.Status != StatusTimeout {
		t.Errorf("status = %q, want %q", rec.Status, StatusTimeout)
	}
	if ExitCode(rec.Status) == 0 {
		t.Error("a timed-out dispatch exited 0")
	}
	if strings.Contains(rec.Output, "answered far too late") {
		t.Error("the record carries the abandoned worker's answer")
	}

	// The load-bearing assertion: the worker is STILL RUNNING (releaseWorker is
	// not closed until this test returns), and its slot is already free.
	s, err := sem.Acquire("nan", 1)
	if err != nil {
		t.Fatalf("the slot is still held while the abandoned worker runs: %v — "+
			"the reserve then under-counts what is free for as long as the worker lives", err)
	}
	s.Release()
}

// A timeout must not advance the chain. The task may well have been submitted
// and be running somewhere; spending a second pool on it is the double-spend
// that task_failed avoids for the same reason.
func TestDispatch_TimeoutDoesNotAdvanceTheChain(t *testing.T) {
	dir := t.TempDir()
	sem := NewSemaphore(dir)

	var attempted []string
	var mu sync.Mutex
	be := backendFunc(func(ctx context.Context, req Request) Response {
		mu.Lock()
		attempted = append(attempted, req.Pool+":"+req.Model)
		mu.Unlock()
		<-ctx.Done()
		return Response{Status: StatusOK}
	})

	rec := Dispatch(context.Background(), Options{
		Tier:      "mid",
		Chain:     []string{"claude:sonnet", "nan:deepseek-v4-flash"},
		Timeout:   50 * time.Millisecond,
		Semaphore: sem,
		Capacity:  func(string) (int, bool) { return 2, true },
	}, be)

	if rec.Status != StatusTimeout {
		t.Errorf("status = %q, want %q", rec.Status, StatusTimeout)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(attempted) != 1 {
		t.Errorf("attempted %v; a timeout must not advance to a second pool", attempted)
	}
}

// A pool the semaphore reports full is an availability fact, so the chain
// advances — the same class as the backend reporting the pool unavailable.
func TestDispatch_AFullPoolAdvancesTheChain(t *testing.T) {
	dir := t.TempDir()
	sem := NewSemaphore(dir)

	// Saturate claude at capacity 1 before dispatching.
	held, err := sem.Acquire("claude", 1)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer held.Release()

	be := &scriptedBackend{byEntry: map[string]Response{
		"nan:deepseek-v4-flash": {Status: StatusOK, Output: "answered"},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier:      "mid",
		Chain:     []string{"claude:sonnet", "nan:deepseek-v4-flash"},
		Timeout:   time.Minute,
		Semaphore: sem,
		Capacity:  func(string) (int, bool) { return 1, true },
		Now:       fixedClock(time.Millisecond),
	}, be)

	if rec.Status != StatusOK || rec.Pool != "nan" {
		t.Errorf("record = %s on %s, want ok on nan", rec.Status, rec.Pool)
	}
	if len(be.seen) != 1 || be.seen[0] != "nan:deepseek-v4-flash" {
		t.Errorf("backend saw %v; the saturated pool must be skipped without being dispatched to", be.seen)
	}
	if len(rec.Attempts) != 2 || rec.Attempts[0].Status != StatusPoolUnavailable {
		t.Errorf("attempts = %+v; the skipped pool must leave a trace", rec.Attempts)
	}
}

// A pool that declares no concurrency is not a pool with zero capacity. Copilot
// is seat-based and its concurrency is a fleet property, so there is no honest
// number — the honest behaviour is no local semaphore, said out loud, rather
// than a refusal the map never asked for.
func TestDispatch_AnUndeclaredBudgetIsNotZeroCapacity(t *testing.T) {
	dir := t.TempDir()
	sem := NewSemaphore(dir)
	be := &scriptedBackend{byEntry: map[string]Response{
		"copilot:gpt": {Status: StatusOK, Output: "answered"},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier:      "mid",
		Chain:     []string{"copilot:gpt"},
		Timeout:   time.Minute,
		Semaphore: sem,
		Capacity:  func(string) (int, bool) { return 0, false }, // not declared
		Now:       fixedClock(time.Millisecond),
	}, be)

	if rec.Status != StatusOK {
		t.Errorf("status = %q, want ok: an undeclared budget must not read as zero capacity", rec.Status)
	}
	if !rec.Unbounded {
		t.Error("the record does not say the dispatch was unbounded; " +
			"a pool with no local semaphore must say so rather than imply one held")
	}
}
