package agent

import (
	"context"
	"fmt"
	"math"
	"time"
)

// harnessFor maps a pool to the harness binary that reaches it, and how that
// binary is asked for one non-interactive answer.
//
// This table is Go rather than another block in model-map.json, deliberately.
// The map declares WHAT exists — pools, tiers, chains, budgets — and adding
// argv to it would make it declare how to invoke things too, which is the
// "map, never a director" drift ADR-035 §4 exists to stop. Two entries also do
// not justify a schema change; a third harness is the trigger to revisit.
var harnessFor = map[string]struct {
	bin  string
	args func(model string) []string
}{
	"claude": {bin: "claude", args: func(m string) []string { return []string{"-p", "--model", m} }},
	"nan":    {bin: "pi", args: func(m string) []string { return []string{"--print", "--model", m} }},
}

// Subprocess dispatches by running a harness binary in non-interactive mode.
// It is the floor bootstrap guarantees: no daemon, no protocol, no state.
type Subprocess struct{}

// Serves answers whether this machine has the binary that reaches the pool.
func (Subprocess) Serves(pool string) bool {
	h, ok := harnessFor[pool]
	return ok && binaryPresent(h.bin)
}

func (Subprocess) Dispatch(ctx context.Context, req Request) Response {
	h, ok := harnessFor[req.Pool]
	if !ok {
		return Response{Status: StatusPoolUnavailable,
			Output: fmt.Sprintf("no harness binary is mapped to pool %q", req.Pool)}
	}
	stdout, stderr, code, err := runProcess(ctx, h.bin, h.args(req.Model), req.Task)
	return classifyProcess(stdout, stderr, code, err)
}

// Hive dispatches through `hive delegate`, launched as a subprocess.
//
// Argv is its transport, not a second mechanism: the seam's hard semantics —
// kill on timeout, release the slot without waiting — are what the subprocess
// backend already implements, and the hive verb rides that code path verbatim.
// An MCP client in Go would add a dependency, a handshake that can drift, and a
// surface bats cannot smoke.
type Hive struct{}

// hivePool is the only pool hive's worker serves. Its worker became NaN-only
// upstream (mlorentedev/hive#384), which is what aligns it with this map.
const hivePool = "nan"

func (Hive) Serves(pool string) bool { return pool == hivePool && binaryPresent("hive") }

// Exit codes are the cross-repo contract, pinned on both sides. `hive delegate`
// documents them in its own help: 3 means the pool would not serve the request
// (try the next entry), 1 means the worker answered with a failure (do not).
const (
	hiveExitTaskFailed  = 1
	hiveExitUnavailable = 3
)

func (Hive) Dispatch(ctx context.Context, req Request) Response {
	args := []string{
		"delegate",
		"--model", req.Model,
		"--timeout", hiveTimeoutSeconds(req.Timeout),
		// --prompt is required on argv by the shipped verb: unlike claude and
		// pi it does not read stdin. Recorded as a limitation of that contract
		// rather than worked around here.
		"--prompt", req.Task,
	}
	stdout, stderr, code, err := runProcess(ctx, "hive", args, "")
	if err != nil {
		return Response{Status: StatusTaskFailed, Exit: 1,
			Output: fmt.Sprintf("could not run hive delegate: %v", err)}
	}
	switch code {
	case 0:
		return classifyProcess(stdout, stderr, 0, nil)
	case hiveExitUnavailable:
		return Response{Status: StatusPoolUnavailable, Exit: code, Output: combine(stdout, stderr)}
	case hiveExitTaskFailed:
		return Response{Status: StatusTaskFailed, Exit: code, Output: combine(stdout, stderr)}
	default:
		// An unrecognised code is a task failure, never an unavailability. The
		// fail-closed direction is the one that does not advance the chain and
		// spend a second pool on work that may already have been billed.
		return Response{Status: StatusTaskFailed, Exit: code, Output: combine(stdout, stderr)}
	}
}

// hiveTimeoutSeconds renders the deadline the way the verb accepts it: whole
// seconds. Rounded UP with a floor of one, because truncation would send `0`
// for any sub-second deadline and a zero timeout is not the deadline anyone
// asked for.
func hiveTimeoutSeconds(d time.Duration) string {
	secs := int(math.Ceil(d.Seconds()))
	if secs < 1 {
		secs = 1
	}
	return fmt.Sprintf("%d", secs)
}

// DefaultBackends is the probe order, and the order IS the tie-break.
//
// Settled here and in tasks.md rather than left to code review, per the open
// question in proposal.md: a `nan` entry is servable by both, and subprocess
// wins where `pi` is present because that is the transport that exists without
// a daemon. Where it is absent — headless, CI — hive takes it. `--backend`
// overrides both.
func DefaultBackends() []NamedBackend {
	sub, hive := Subprocess{}, Hive{}
	return []NamedBackend{
		{Name: "subprocess", Backend: sub, Serves: sub.Serves},
		{Name: "hive", Backend: hive, Serves: hive.Serves},
		{Name: "dry-run", Backend: DryRun{}, Serves: func(string) bool { return true }, ExplicitOnly: true},
	}
}
