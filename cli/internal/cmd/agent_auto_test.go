package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/agent"
)

// recordingBackend keeps every Request it was handed.
//
// AC6 is "the persona's record reached the dispatched process", and stdout
// cannot answer that: DryRun echoes the role and the route and never the task
// (dryrun.go:18-24), so a preamble that was composed and then dropped would
// leave every assertion on the emitted record passing. The only witness that
// distinguishes the two is what the backend RECEIVED.
type recordingBackend struct {
	mu   sync.Mutex
	got  []agent.Request
	resp agent.Response
}

func (b *recordingBackend) Dispatch(_ context.Context, req agent.Request) agent.Response {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.got = append(b.got, req)
	if b.resp.Status == "" {
		return agent.Response{Status: agent.StatusOK, Output: "ok"}
	}
	return b.resp
}

func (b *recordingBackend) requests() []agent.Request {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]agent.Request(nil), b.got...)
}

// recordDispatches substitutes the probe order for the duration of one test, so
// `--backend recording` reaches the fake. It is installed through the seam and
// never added to DefaultBackends: a backend name that exists to make tests pass
// is a surface users reach by accident (dryrun.go:13-15).
func recordDispatches(t *testing.T) *recordingBackend {
	t.Helper()
	be := &recordingBackend{}
	prev := dispatchBackends
	dispatchBackends = func() []agent.NamedBackend {
		return []agent.NamedBackend{{
			Name: "recording", Backend: be,
			Serves: func(string) bool { return true }, ExplicitOnly: true,
		}}
	}
	t.Cleanup(func() { dispatchBackends = prev })
	return be
}

// AC1. The whole point of the command in one assertion: a task, no --role, no
// --tier, and a dispatch that names both.
//
// The task text is the one proposal.md names exactly rather than describes, so
// the criterion is reproducible from the spec file. It matches
// pattern-bitacora-tracking alone, whose skills intersect only planner, whose
// record declares model: mid.
func TestAgentAuto_DerivesBothRoleAndTierFromTheTask(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t)

	stdout, stderr, err := captureRealStreams(t,
		"agent", "auto",
		"--task", "open a ticket for the bitacora",
		"--backend", "dry-run",
		"--timeout", "90s",
		"--repo-root", root,
	)
	if err != nil {
		t.Fatalf("agent auto: %v (stderr: %s)", err, stderr)
	}

	var rec struct {
		Role       string `json:"role"`
		Tier       string `json:"tier"`
		Resolution struct {
			RoleFrom string `json:"role_from"`
			TierFrom string `json:"tier_from"`
			Pattern  string `json:"pattern"`
		} `json:"resolution"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
		t.Fatalf("no record on stdout: %v (%q)", jsonErr, stdout)
	}
	if rec.Role != "planner" {
		t.Errorf("role = %q, want planner", rec.Role)
	}
	if rec.Tier != "mid" {
		t.Errorf("tier = %q, want mid (planner's record declares it)", rec.Tier)
	}
	// Both halves derived, and SAID to be derived. A record that routed
	// correctly but reported nothing about how would leave a caller unable to
	// tell a match it should question from a route a human chose.
	if rec.Resolution.RoleFrom != inferred || rec.Resolution.TierFrom != inferred {
		t.Errorf("resolution = %+v, want both inferred", rec.Resolution)
	}
	if !strings.Contains(rec.Resolution.Pattern, "bitacora") {
		t.Errorf("pattern = %q, want the rule that matched: a route that cannot be "+
			"judged is obeyed on its worst day as readily as its best", rec.Resolution.Pattern)
	}
}

// AC6, at the seam that matters. Two personas, two dispatches, and the
// assertion is on what the BACKEND received.
//
// Before HARNESS-120, Request.Role was set at dispatch.go:136 and read only by
// the dry-run echo. So `--role reviewer` dispatched a generic agent that was
// merely logged as a reviewer, and the mandate, method and boundaries the six
// records carry stopped at the process boundary. This is the test that would
// have failed then and passes now.
func TestAgentAuto_SendsThePersonasOwnRecordToTheBackend(t *testing.T) {
	root := repoRootForTest(t)

	const (
		reviewerBoundary = "you do not edit"
		builderBoundary  = "you do not redecide the architecture"
	)

	for _, tc := range []struct {
		role, carries, omits string
	}{
		{"reviewer", reviewerBoundary, builderBoundary},
		{"builder", builderBoundary, reviewerBoundary},
	} {
		t.Run(tc.role, func(t *testing.T) {
			declareIdentity(t)
			be := recordDispatches(t)

			if _, stderr, err := captureRealStreams(t,
				"agent", "auto",
				"--task", "do the thing",
				"--role", tc.role,
				"--backend", "recording",
				"--timeout", "90s",
				"--repo-root", root,
			); err != nil {
				t.Fatalf("agent auto: %v (stderr: %s)", err, stderr)
			}

			reqs := be.requests()
			if len(reqs) != 1 {
				t.Fatalf("backend saw %d requests, want 1", len(reqs))
			}
			sent := reqs[0].Task

			if !strings.Contains(sent, tc.carries) {
				t.Errorf("the task sent to the backend omits %s's own boundary %q; "+
					"the persona did not travel", tc.role, tc.carries)
			}
			// Not merely "carries something": it must not carry the OTHER
			// persona's boundary. The two are near-opposites — the reviewer must
			// not edit, the builder must not redecide — so a preamble that mixed
			// them would be worse than one that carried neither.
			if strings.Contains(sent, tc.omits) {
				t.Errorf("the task sent as %s carries the other persona's boundary %q", tc.role, tc.omits)
			}
			// The task itself still arrives, and arrives LAST: a task read before
			// the record has had its operating instruction arrive too late to
			// shape how the work is read.
			if !strings.Contains(sent, "do the thing") {
				t.Error("the task was replaced by the preamble rather than prefixed with it")
			}
			if strings.Index(sent, "do the thing") < strings.Index(sent, tc.carries) {
				t.Error("the task precedes the record; the persona must be established first")
			}
		})
	}
}

// AC2 and AC3 together, because the criterion is that the two refusals DIFFER —
// asserted as inequality, never as two fixed strings, so a reworded message
// cannot fail this and a collapsed one cannot pass it.
//
// Both must dispatch NOTHING, and that is asserted against a backend that
// records its calls rather than against stdout. A refusal that printed an error
// while still spending a slot would look identical from the outside.
func TestAgentAuto_BothRefusalsDispatchNothingAndDiffer(t *testing.T) {
	root := repoRootForTest(t)

	// Ambiguous: spec-driven-development resolves to [planner, reviewer], and a
	// dispatcher runs one process. Unmatched: no trigger rule claims this.
	const (
		ambiguousTask = "run a spec-driven development review of this change"
		unmatchedTask = "xyzzy plugh frobnitz"
	)

	errFor := func(t *testing.T, task string) error {
		t.Helper()
		declareIdentity(t)
		be := recordDispatches(t)
		stdout, _, err := captureRealStreams(t,
			"agent", "auto", "--task", task,
			"--backend", "recording", "--timeout", "90s", "--repo-root", root,
		)
		if err == nil {
			t.Fatalf("dispatched %q without resolving one persona", task)
		}
		if n := len(be.requests()); n != 0 {
			t.Errorf("a refusal still reached the backend %d time(s)", n)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Errorf("stdout is not empty for a dispatch that never happened: %q", stdout)
		}
		return err
	}

	ambiguous := errFor(t, ambiguousTask)
	unmatched := errFor(t, unmatchedTask)

	if ambiguous.Error() == unmatched.Error() {
		t.Fatalf("both refusals read identically (%q); an ambiguous match is fixed by "+
			"choosing and an unmatched one cannot be, so an operator needs to tell them apart",
			ambiguous.Error())
	}
	// AC2 names BOTH candidates. Naming one would present a ranking the rules do
	// not express.
	for _, want := range []string{"planner", "reviewer"} {
		if !strings.Contains(ambiguous.Error(), want) {
			t.Errorf("the ambiguity refusal does not name %q: %s", want, ambiguous)
		}
	}
}

// AC5. Each override is asserted with the OTHER half left derived, so a command
// that ignored one flag and honoured the other cannot pass by accident.
func TestAgentAuto_OverridesAreHonouredAndReportedAsDictated(t *testing.T) {
	root := repoRootForTest(t)

	read := func(t *testing.T, extra ...string) struct {
		Role       string `json:"role"`
		Tier       string `json:"tier"`
		Resolution struct {
			RoleFrom string `json:"role_from"`
			TierFrom string `json:"tier_from"`
			Pattern  string `json:"pattern"`
		} `json:"resolution"`
	} {
		t.Helper()
		declareIdentity(t)
		args := append([]string{
			"agent", "auto", "--task", "open a ticket for the bitacora",
			"--backend", "dry-run", "--timeout", "90s", "--repo-root", root,
		}, extra...)
		stdout, stderr, err := captureRealStreams(t, args...)
		if err != nil {
			t.Fatalf("agent auto %v: %v (stderr: %s)", extra, err, stderr)
		}
		var rec struct {
			Role       string `json:"role"`
			Tier       string `json:"tier"`
			Resolution struct {
				RoleFrom string `json:"role_from"`
				TierFrom string `json:"tier_from"`
				Pattern  string `json:"pattern"`
			} `json:"resolution"`
		}
		if jsonErr := json.Unmarshal([]byte(stdout), &rec); jsonErr != nil {
			t.Fatalf("no record on stdout: %v (%q)", jsonErr, stdout)
		}
		return rec
	}

	// --role overrides a task that would otherwise resolve to planner, and the
	// tier still comes from the record it names — architect declares top.
	byRole := read(t, "--role", "architect")
	if byRole.Role != "architect" {
		t.Errorf("role = %q, want architect: --role must skip the join, not filter it", byRole.Role)
	}
	if byRole.Resolution.RoleFrom != dictated {
		t.Errorf("role_from = %q, want dictated", byRole.Resolution.RoleFrom)
	}
	if byRole.Resolution.TierFrom != inferred {
		t.Errorf("tier_from = %q, want inferred: --role must not make the tier dictated too",
			byRole.Resolution.TierFrom)
	}
	// A dictated role consulted no rule, so reporting one would be a claim about
	// a derivation that did not happen.
	if byRole.Resolution.Pattern != "" {
		t.Errorf("pattern = %q for a dictated role, want empty", byRole.Resolution.Pattern)
	}

	// --tier overrides planner's declared mid, and the role stays derived.
	byTier := read(t, "--tier", "low")
	if byTier.Tier != "low" {
		t.Errorf("tier = %q, want low: --tier must override the record", byTier.Tier)
	}
	if byTier.Resolution.TierFrom != dictated {
		t.Errorf("tier_from = %q, want dictated", byTier.Resolution.TierFrom)
	}
	if byTier.Role != "planner" || byTier.Resolution.RoleFrom != inferred {
		t.Errorf("role = %q/%q, want planner/inferred: --tier must not disturb the join",
			byTier.Role, byTier.Resolution.RoleFrom)
	}
}

// A --role the roster does not declare refuses rather than dispatching under
// that name. Without this, `auto --role potato` would send no record at all and
// report a dispatch as potato — exactly the generic-agent-wearing-a-name state
// the whole change exists to end.
func TestAgentAuto_RefusesAPersonaTheRosterDoesNotDeclare(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t)
	be := recordDispatches(t)

	_, _, err := captureRealStreams(t,
		"agent", "auto", "--task", "t", "--role", "potato",
		"--backend", "recording", "--timeout", "90s", "--repo-root", root,
	)
	if err == nil {
		t.Fatal("dispatched as a persona nobody declares")
	}
	if n := len(be.requests()); n != 0 {
		t.Errorf("an unknown --role still reached the backend %d time(s)", n)
	}
	// The refusal lists what IS available: a fail-closed refusal whose remedy the
	// operator has to go and look up is one people route around.
	if !strings.Contains(err.Error(), "planner") {
		t.Errorf("the refusal does not name the roster it checked against: %v", err)
	}
}

// `run` is deliberately NOT given the preamble, and that is a decision worth a
// test rather than a comment: it is the primitive that takes a route as given,
// its --role is a label rather than a lookup, and making it one would refuse a
// --role the roster does not name. This pins the boundary so the two commands
// cannot drift into each other unnoticed.
func TestAgentRun_DoesNotComposeAPreamble(t *testing.T) {
	root := repoRootForTest(t)
	declareIdentity(t)
	be := recordDispatches(t)

	if _, stderr, err := captureRealStreams(t,
		"agent", "run", "--role", "reviewer", "--task", "do the thing",
		"--tier", "mid", "--backend", "recording", "--timeout", "90s", "--repo-root", root,
	); err != nil {
		t.Fatalf("agent run: %v (stderr: %s)", err, stderr)
	}

	reqs := be.requests()
	if len(reqs) != 1 {
		t.Fatalf("backend saw %d requests, want 1", len(reqs))
	}
	if reqs[0].Task != "do the thing" {
		t.Errorf("run sent %q, want the task verbatim: composing here would make --role "+
			"a roster lookup and refuse names it has always accepted", reqs[0].Task)
	}
}
