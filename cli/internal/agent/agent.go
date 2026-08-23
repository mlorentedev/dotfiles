// Package agent is the execution layer ADR-032 §2 defines: "run role A on task
// T, isolated in W, return R". It owns the seam and the chain walk; it does not
// own the routing map. The caller resolves `chains` through
// harness.ResolveChain and hands the result in, so the registry keeps one
// reader and this package keeps no opinion about where a chain comes from.
package agent

import (
	"context"
	"time"
	"unicode/utf8"
)

// Status is a dispatch outcome carried as a VALUE, not as an error type.
//
// The reason is the cross-repo contract: this classification crosses a process
// boundary (a `hive delegate` exit code, a subprocess exit code) and a JSON
// boundary (the record on stdout). Exception types do not survive either, so
// the vocabulary has to be data on both sides. `mlorentedev/hive#395` fixed the
// same shape upstream for the same reason.
type Status string

const (
	// StatusOK — the backend ran the task and it succeeded.
	StatusOK Status = "ok"
	// StatusTaskFailed — the backend ran the task and it failed. This does NOT
	// advance the chain: retrying a genuine failure on a different model turns a
	// bad answer into a silent second opinion (ADR-032 §2).
	StatusTaskFailed Status = "task_failed"
	// StatusPoolUnavailable — the backend could not reach the pool at all, so
	// the task never ran. This advances the chain. It is never a final status:
	// the dispatcher resolves it to the next entry, or to chain_exhausted.
	StatusPoolUnavailable Status = "pool_unavailable"
	// StatusChainExhausted — every entry in the chain reported unavailable, so
	// nothing ran anywhere. Distinct from task_failed on purpose: one means "no
	// answer exists", the other means "the answer was wrong".
	StatusChainExhausted Status = "chain_exhausted"
	// StatusEscalated — the top tier's only entry was unavailable and the
	// dispatcher refused to degrade (ADR-032 §4). Distinct from chain_exhausted
	// because no chain was exhausted; a fallback was declined.
	StatusEscalated Status = "escalated"
	// StatusDryRun — the routing was resolved and deliberately not executed.
	StatusDryRun Status = "dry_run"
	// StatusTimeout — the dispatch outlived its deadline and was abandoned.
	// A dispatcher-level status: no backend returns it.
	//
	// It does NOT advance the chain, for the reason task_failed does not: the
	// task may well have been submitted and be running still, so spending a
	// second pool on it is a double-spend against work that may already have
	// been billed.
	StatusTimeout Status = "timeout"
)

// TierTop is the tier that must never degrade. Named rather than inlined
// because the no-fallback rule reads as arbitrary at its use site otherwise.
const TierTop = "top"

// OutputCap bounds the captured output the record carries, per ADR-032 §2's
// "captured output bounded by an output cap". A dispatcher composing many
// records must not have one verbose worker blow up the consumer, and stdout
// here is parsed by a machine, not scrolled by a person.
const OutputCap = 64 * 1024

// Request is what the dispatcher hands a backend for ONE chain entry. Pool and
// Model are already resolved: a backend chooses nothing about routing.
type Request struct {
	Pool    string
	Model   string
	Role    string
	Task    string
	Cwd     string
	Timeout time.Duration
}

// Response is what a backend reports for one attempt. Backends return the four
// dispatch-level statuses (ok, task_failed, pool_unavailable, dry_run); the two
// chain-level ones (chain_exhausted, escalated) are the dispatcher's to emit.
type Response struct {
	Status Status
	Exit   int
	Output string
}

// Backend is the seam. One method, the five semantics of ADR-032 §2, and no
// backend-specific type in either direction — that is what keeps subprocess,
// hive and (later) orca interchangeable rather than merely coexisting.
//
// It returns no error: a transport failure IS one of the statuses, and a
// backend that could report a failure two different ways would let the
// dispatcher's classification be bypassed.
type Backend interface {
	Dispatch(ctx context.Context, req Request) Response
}

// Classify maps a backend's reported status onto the dispatcher's vocabulary,
// failing CLOSED: anything unrecognised is a task failure, never an
// unavailability.
//
// The direction matters and is pinned on both sides of the seam. Reading an
// unknown code as "unavailable" would advance the chain and spend a second
// pool on a task that may already have been answered — and, where the pool
// bills, already been paid for. Reading it as "failed" merely stops.
func Classify(s Status) Status {
	switch s {
	case StatusOK, StatusTaskFailed, StatusPoolUnavailable, StatusDryRun:
		return s
	default:
		return StatusTaskFailed
	}
}

// ExitCode is the process exit status for a final record status. It is the
// coarse class; the record's `status` field carries the fine one.
//
// 0/1/3 mirror `hive delegate` deliberately (0 answered, 1 task failed, 3 pool
// unavailable). One vocabulary across the seam means a composer that already
// speaks to hive speaks to this without a translation table — and a translation
// table is where the two would drift.
// A timeout exits 1, with the rest of the "the task did not succeed" family: 3
// means *no pool could serve this, try elsewhere*, and a timed-out dispatch may
// have been served perfectly well and merely too slowly. Reporting it as 3
// would invite a composer to retry it against another pool.
func ExitCode(s Status) int {
	switch s {
	case StatusOK, StatusDryRun:
		return 0
	case StatusChainExhausted, StatusEscalated:
		return 3
	default:
		return 1
	}
}

// capOutput bounds output at OutputCap and reports the truncation to its
// caller, which records it in the record's `truncated` field. Truncating
// without that marker produces a record that reads as a complete short answer,
// which is worse than a long one.
//
// The cut moves back to a rune boundary. OutputCap counts bytes, and a model's
// output is not ASCII: cutting mid-rune leaves orphan bytes that decode as
// U+FFFD, so the record would report corruption where it meant to report a
// limit.
func capOutput(s string) (string, bool) {
	if len(s) <= OutputCap {
		return s, false
	}
	cut := OutputCap
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut], true
}
