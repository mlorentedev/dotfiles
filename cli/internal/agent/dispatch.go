package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Options is one dispatch request, with its route already resolved.
//
// Chain arrives resolved rather than being read here: harness.ResolveChain is
// the map's one reader, and a second parse in this package would be a second
// place the routing rules are true — the failure model-map.json exists to end.
type Options struct {
	Tier    string
	Role    string
	Task    string
	Cwd     string
	Chain   []string
	Timeout time.Duration
	// Semaphore bounds concurrent dispatches per pool. Nil means no bound —
	// only correct where the caller genuinely is not the launcher.
	Semaphore *Semaphore
	// Capacity answers how many slots a pool declares, and whether it declares
	// any at all. The second return distinguishes "declares zero" from "does
	// not state one": a seat-based pool's concurrency is a fleet property, and
	// reading its silence as zero would refuse every dispatch the map never
	// asked to refuse.
	Capacity func(pool string) (int, bool)
	// Now is injectable so duration is an exact expectation under test. A test
	// asserting `duration_ms >= 0` cannot fail, which is not a test.
	Now func() time.Time
}

// Attempt is one entry of the walk. Kept in the record because a chain that
// silently skipped two pools and answered on the third is indistinguishable,
// from the outside, from one that answered immediately.
type Attempt struct {
	Pool   string `json:"pool"`
	Model  string `json:"model"`
	Status Status `json:"status"`
}

// Record is the machine contract on stdout: one JSON object, always.
type Record struct {
	Status     Status `json:"status"`
	Tier       string `json:"tier"`
	Pool       string `json:"pool"`
	Model      string `json:"model"`
	Exit       int    `json:"exit"`
	DurationMS int64  `json:"duration_ms"`
	Output     string `json:"output"`
	Truncated  bool   `json:"truncated"`
	// Unbounded says the answering pool declares no concurrency, so no local
	// semaphore was held. Recorded rather than inferred: a consumer must not
	// have to guess whether a dispatch was counted.
	Unbounded bool      `json:"unbounded,omitempty"`
	Attempts  []Attempt `json:"attempts,omitempty"`
}

// Dispatch walks the chain and returns the record. It never returns an error:
// every outcome is a status, because the caller's job is to encode the record,
// not to decide what a failure means.
func Dispatch(ctx context.Context, opts Options, be Backend) Record {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	started := now()
	rec := Record{Tier: opts.Tier}

	// ADR-032 §4: the top tier queues or escalates, it never degrades. The rule
	// is applied to behaviour unconditionally rather than inferred from the map
	// happening to declare one entry today — a rule that holds only because of
	// the current data is not a rule.
	chain := opts.Chain
	noFallback := opts.Tier == TierTop
	if noFallback && len(chain) > 1 {
		chain = chain[:1]
	}

	for _, entry := range chain {
		pool, model, err := splitEntry(entry)
		if err != nil {
			// A malformed entry is a defect in the map, not an unavailable pool.
			// Treating it as unavailable would advance the chain and route around
			// a broken registry silently, which is how a map stays broken.
			rec.Status = StatusTaskFailed
			rec.Output = err.Error()
			rec.Exit = ExitCode(rec.Status)
			rec.DurationMS = elapsed(started, now())
			return rec
		}

		// The slot is taken BEFORE the backend is reached and released as soon
		// as the attempt is over, whether it answered or was abandoned.
		slot, unbounded, err := acquire(opts, pool)
		if err != nil {
			if IsPoolBusy(err) {
				// A full pool is an availability fact, indistinguishable to the
				// caller from the backend reporting the pool unavailable, so it
				// advances the chain — without ever reaching the backend.
				rec.Attempts = append(rec.Attempts, Attempt{Pool: pool, Model: model, Status: StatusPoolUnavailable})
				continue
			}
			// An unreadable counter is not an empty one. Stopping is the
			// fail-closed direction: advancing would try the next pool against
			// the same broken state.
			rec.Status = StatusTaskFailed
			rec.Output = err.Error()
			rec.Exit = ExitCode(rec.Status)
			rec.DurationMS = elapsed(started, now())
			return rec
		}

		resp, timedOut := attempt(ctx, opts, be, Request{
			Pool: pool, Model: model, Role: opts.Role, Task: opts.Task,
			Cwd: opts.Cwd, Timeout: opts.Timeout,
		}, slot)

		if timedOut {
			// Not an advance: the task may have been submitted and still be
			// running, so a second pool would be a double-spend on work that
			// may already have been billed.
			rec.Attempts = append(rec.Attempts, Attempt{Pool: pool, Model: model, Status: StatusTimeout})
			rec.Status, rec.Pool, rec.Model = StatusTimeout, pool, model
			rec.Exit = ExitCode(StatusTimeout)
			rec.Unbounded = unbounded
			rec.Output = fmt.Sprintf("dispatch to %s:%s exceeded its %s deadline and was abandoned; "+
				"its slot was released without waiting for the worker", pool, model, opts.Timeout)
			rec.DurationMS = elapsed(started, now())
			return rec
		}

		status := Classify(resp.Status)
		rec.Attempts = append(rec.Attempts, Attempt{Pool: pool, Model: model, Status: status})

		if status == StatusPoolUnavailable {
			continue
		}
		rec.Status, rec.Pool, rec.Model, rec.Exit = status, pool, model, resp.Exit
		rec.Output, rec.Truncated = capOutput(resp.Output)
		rec.Unbounded = unbounded
		rec.DurationMS = elapsed(started, now())
		return rec
	}

	// Nothing ran anywhere. The two ways to arrive here are different facts and
	// get different names: the top tier DECLINED a fallback, a lower tier ran
	// OUT of them.
	if noFallback {
		rec.Status = StatusEscalated
	} else {
		rec.Status = StatusChainExhausted
	}
	rec.Exit = ExitCode(rec.Status)
	rec.DurationMS = elapsed(started, now())
	return rec
}

// acquire takes the pool's slot, or reports that it could not. It answers
// `unbounded` when the pool declares no concurrency at all — which is not the
// same as declaring zero. A seat-based pool's concurrency is a fleet property,
// and refusing on its silence would enforce a limit the map never stated.
func acquire(opts Options, pool string) (slot *Slot, unbounded bool, err error) {
	if opts.Semaphore == nil || opts.Capacity == nil {
		return nil, true, nil
	}
	capacity, declared := opts.Capacity(pool)
	if !declared {
		return nil, true, nil
	}
	s, err := opts.Semaphore.Acquire(pool, capacity)
	if err != nil {
		return nil, false, err
	}
	return s, false, nil
}

// attempt runs one backend call under the per-dispatch deadline and reports
// whether that deadline expired first.
//
// The backend runs on its own goroutine and the result is selected against the
// deadline, rather than the call simply being handed a context. The difference
// is the whole of AC3: a backend that ignores its context — a subprocess that
// blocks on a read, a remote that never answers — would otherwise hold this
// call open forever, and no timeout the dispatcher declared would be real.
//
// On expiry the slot is released HERE, before returning and without waiting for
// the abandoned worker. That ordering is the criterion: while a worker outlives
// its deadline, the reserve must not go on counting its slot as taken.
func attempt(ctx context.Context, opts Options, be Backend, req Request, slot *Slot) (Response, bool) {
	if opts.Timeout <= 0 {
		slot.Release()
		return Response{Status: StatusTaskFailed, Exit: 1,
			Output: "no timeout was set for this dispatch; a backend that cannot be bounded is not eligible"}, false
	}
	dctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Buffered so the goroutine can finish and be collected even after the
	// deadline won the race and nobody is left reading.
	done := make(chan Response, 1)
	go func() { done <- be.Dispatch(dctx, req) }()

	select {
	case resp := <-done:
		slot.Release()
		return resp, false
	case <-dctx.Done():
		slot.Release() // before the worker is reaped, deliberately
		return Response{}, true
	}
}

func splitEntry(entry string) (pool, model string, err error) {
	pool, model, ok := strings.Cut(entry, ":")
	if !ok || strings.TrimSpace(pool) == "" || strings.TrimSpace(model) == "" {
		return "", "", fmt.Errorf(
			"chain entry %q is not `pool:model` — the chain cannot be walked past a route that names nothing",
			entry)
	}
	return pool, model, nil
}

func elapsed(from, to time.Time) int64 { return to.Sub(from).Milliseconds() }
