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
	Status     Status    `json:"status"`
	Tier       string    `json:"tier"`
	Pool       string    `json:"pool"`
	Model      string    `json:"model"`
	Exit       int       `json:"exit"`
	DurationMS int64     `json:"duration_ms"`
	Output     string    `json:"output"`
	Truncated  bool      `json:"truncated"`
	Attempts   []Attempt `json:"attempts,omitempty"`
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
			rec.DurationMS = elapsed(started, now())
			return rec
		}

		resp := attempt(ctx, opts, be, Request{
			Pool: pool, Model: model, Role: opts.Role, Task: opts.Task,
			Cwd: opts.Cwd, Timeout: opts.Timeout,
		})
		status := Classify(resp.Status)
		rec.Attempts = append(rec.Attempts, Attempt{Pool: pool, Model: model, Status: status})

		if status == StatusPoolUnavailable {
			continue
		}
		rec.Status, rec.Pool, rec.Model, rec.Exit = status, pool, model, resp.Exit
		rec.Output, rec.Truncated = capOutput(resp.Output)
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

// attempt gives the backend the per-dispatch deadline. The timeout is required
// by ADR-032 §2, and required means the backend receives it — enforcing the
// kill and the slot release is PR C's, but a deadline that never reaches the
// backend would make the flag a lie today.
func attempt(ctx context.Context, opts Options, be Backend, req Request) Response {
	if opts.Timeout <= 0 {
		return Response{Status: StatusTaskFailed, Exit: 1,
			Output: "no timeout was set for this dispatch; a backend that cannot be bounded is not eligible"}
	}
	dctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	return be.Dispatch(dctx, req)
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
