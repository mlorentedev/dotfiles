package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// NamedBackend is one implementation behind the seam, plus the question only it
// can answer: which pools it can actually serve on this machine.
//
// Serves is separate from Dispatch because the answer decides ROUTING, and
// routing must be decided without spending anything. Asking a backend to try
// and reporting the failure would already have sent the task's content
// somewhere it does not belong.
type NamedBackend struct {
	Name    string
	Backend Backend
	Serves  func(pool string) bool
	// ExplicitOnly keeps a member out of the probe: it is reachable only when
	// --backend names it.
	//
	// dry-run is the reason this field exists. It serves every pool by
	// construction, so leaving it in probe order would make it the answer to
	// every dispatch that named no backend — a dispatch that silently ran
	// nothing and exited 0, which is the worst option available and the one
	// PR B's decision table already rejected. It was reintroduced here by
	// accident and caught by the test below.
	ExplicitOnly bool
}

// Router is a Backend that picks a member per REQUEST rather than per dispatch.
//
// Per-request is forced by the data: `chains.mid` mixes `claude:sonnet` with
// `nan:deepseek-v4-flash`, and hive serves only nan. A router chosen once for a
// whole dispatch could not walk that chain. Being a Backend itself is what
// keeps `Dispatch` untouched — the seam absorbs the fan-out instead of the
// dispatcher learning about backends.
type Router struct {
	members []NamedBackend
	// forced is the --backend override. Empty means probe order decides.
	forced string
}

// NewRouter builds a router over members, in tie-break order: the FIRST member
// that serves a pool wins it.
//
// For a `nan` entry both subprocess and hive can serve, and the order encodes
// the answer `proposal.md` proposed and `tasks.md` settles: prefer subprocess
// where the harness binary is present, fall back to hive where it is not.
func NewRouter(members []NamedBackend, forced string) *Router {
	return &Router{members: members, forced: forced}
}

// ResolveRouter builds a router and validates the --backend override against
// the members that exist, so a typo is a refusal rather than a dispatch that
// silently uses something else.
func ResolveRouter(members []NamedBackend, forced string) (*Router, error) {
	if forced != "" {
		known := make([]string, 0, len(members))
		found := false
		for _, m := range members {
			known = append(known, m.Name)
			if m.Name == forced {
				found = true
			}
		}
		if !found {
			sort.Strings(known)
			return nil, fmt.Errorf("unknown backend %q: this machine has %s",
				forced, strings.Join(known, ", "))
		}
	}
	return NewRouter(members, forced), nil
}

// Dispatch routes one request to the first member that serves its pool.
//
// A pool nothing can serve is POOL UNAVAILABLE, not a task failure. Nothing
// ran, so the next chain entry deserves its turn — and that is also what makes
// `--backend hive --tier mid` work as AC6's smoke expects: the claude entry is
// skipped rather than failing the dispatch, and nan answers.
func (r *Router) Dispatch(ctx context.Context, req Request) Response {
	for _, m := range r.members {
		if r.forced == "" && m.ExplicitOnly {
			continue
		}
		if r.forced != "" && m.Name != r.forced {
			continue
		}
		if m.Serves == nil || !m.Serves(req.Pool) {
			continue
		}
		return m.Backend.Dispatch(ctx, req)
	}
	return Response{
		Status: StatusPoolUnavailable,
		Output: fmt.Sprintf("no backend on this machine serves pool %q%s", req.Pool, r.forcedNote()),
	}
}

func (r *Router) forcedNote() string {
	if r.forced == "" {
		return ""
	}
	return fmt.Sprintf(" through the forced backend %q", r.forced)
}
