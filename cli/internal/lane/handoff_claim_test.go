package lane

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeClaims records the claims a hand-off takes. held names the issues
// somebody else already holds; a claim of one is refused with *ClaimedError.
type fakeClaims struct {
	held  map[int]string
	calls []string
	log   *[]string // shared with the backend's calls, to assert the order
}

func (f *fakeClaims) record(s string) {
	f.calls = append(f.calls, s)
	if f.log != nil {
		*f.log = append(*f.log, s)
	}
}

func (f *fakeClaims) Acquire(n int) (Claim, error) {
	f.record("acquire " + itoa(n))
	if who, ok := f.held[n]; ok {
		return Claim{}, &ClaimedError{Issue: n, Holder: who, At: time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)}
	}
	return Claim{Issue: n}, nil
}

func (f *fakeClaims) Held(n int) (bool, error) {
	f.record("held " + itoa(n))
	_, ok := f.held[n]
	return ok, nil
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestHandoffClaimsBeforeItStarts(t *testing.T) {
	b := newFake(t, "claude")
	c := &fakeClaims{}
	o := opts(t)
	o.Claims = c

	out, err := Handoff(o, b, queue)
	if err != nil || !out.Started {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	if !reflect.DeepEqual(c.calls, []string{"acquire 7"}) {
		t.Fatalf("claims = %q, want one claim of #7", c.calls)
	}
	// The claim precedes the worktree: nothing is created for an issue the
	// session does not hold.
	var order []string
	b2 := newFake(t, "claude")
	c2 := &fakeClaims{log: &order}
	o2 := opts(t)
	o2.Claims = c2
	if _, err := Handoff(o2, &orderBackend{fakeBackend: b2, log: &order}, queue); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(order, ","); !strings.HasPrefix(got, "acquire 7,create") {
		t.Errorf("order = %s, want the claim before create", got)
	}
}

// orderBackend writes the backend's creates into the same log as the claims.
type orderBackend struct {
	*fakeBackend
	log *[]string
}

func (o *orderBackend) Create(repo, branch, base, label string) (Started, error) {
	*o.log = append(*o.log, "create")
	return o.fakeBackend.Create(repo, branch, base, label)
}

func TestHandoffSkipsToTheNextFreeItemOnALoss(t *testing.T) {
	b := newFake(t, "claude")
	c := &fakeClaims{held: map[int]string{7: "ana@box-a:/wt/a"}}
	o := opts(t)
	o.Claims = c

	out, err := Handoff(o, b, queue)
	if err != nil || !out.Started || out.Issue.Number != 8 {
		t.Fatalf("Handoff = %+v, %v; want #8 started", out, err)
	}
	if !reflect.DeepEqual(c.calls, []string{"acquire 7", "acquire 8"}) {
		t.Errorf("claims = %q", c.calls)
	}
	for _, call := range b.calls {
		if strings.HasPrefix(call, "create") && !strings.Contains(call, "fix/second-thing") {
			t.Errorf("the loser's issue got a worktree: %q", call)
		}
	}
	if !strings.Contains(out.Note, "#7") || !strings.Contains(out.Note, "ana@box-a") {
		t.Errorf("note %q must say #7 was skipped and who holds it", out.Note)
	}
}

func TestHandoffStartsNothingWhenEveryItemIsClaimed(t *testing.T) {
	b := newFake(t, "claude")
	c := &fakeClaims{held: map[int]string{7: "a", 8: "b", 9: "c"}}
	o := opts(t)
	o.Claims = c

	out, err := Handoff(o, b, queue)
	if err != nil || out.Started {
		t.Fatalf("Handoff = %+v, %v; want nothing started", out, err)
	}
	for _, call := range b.calls {
		if strings.HasPrefix(call, "create") || strings.HasPrefix(call, "type") {
			t.Errorf("a lane was started with every item claimed: %q", call)
		}
	}
}

// A claim that cannot be read or taken is "unknown", and unknown starts nothing.
func TestHandoffFailsClosedWhenTheClaimCannotBeTaken(t *testing.T) {
	b := newFake(t, "claude")
	o := opts(t)
	o.Claims = failingClaims{}

	out, err := Handoff(o, b, queue)
	if err == nil || out.Started || !strings.Contains(err.Error(), "#7") {
		t.Fatalf("Handoff = %+v, %v; want an error naming #7", out, err)
	}
	for _, call := range b.calls {
		if strings.HasPrefix(call, "create") || strings.HasPrefix(call, "type") {
			t.Errorf("started with the claim unknown: %q", call)
		}
	}
}

type failingClaims struct{}

func (failingClaims) Acquire(int) (Claim, error) { return Claim{}, errors.New("remote unreachable") }
func (failingClaims) Held(int) (bool, error)     { return false, errors.New("remote unreachable") }

// A branch a workspace already shows was claimed when it was started: a second
// run must not read its own earlier claim as a loss and move to the next item.
func TestHandoffDoesNotClaimAWorktreeThatExists(t *testing.T) {
	shown := t.TempDir()
	b := &fakeBackend{agent: "claude", existing: map[string]Worktree{"fix/first-thing": {Path: shown, Branch: "fix/first-thing", Open: true}}}
	c := &fakeClaims{}
	o := opts(t)
	o.Claims = c

	out, err := Handoff(o, b, queue)
	if err != nil || out.Started || out.Issue.Number != 7 {
		t.Fatalf("Handoff = %+v, %v; want #7 left to its workspace", out, err)
	}
	if len(c.calls) != 0 {
		t.Errorf("claims = %q, want none for an existing worktree", c.calls)
	}
}

func TestHandoffDryRunTakesNoClaimButSkipsAHeldItem(t *testing.T) {
	b := newFake(t, "claude")
	c := &fakeClaims{held: map[int]string{7: "ana@box-a:/wt/a"}}
	o := opts(t)
	o.Claims = c
	o.DryRun = true

	out, err := Handoff(o, b, queue)
	if err != nil || out.Started || out.Issue.Number != 8 {
		t.Fatalf("Handoff = %+v, %v; want a dry run naming #8", out, err)
	}
	for _, call := range c.calls {
		if strings.HasPrefix(call, "acquire") {
			t.Errorf("a dry run took a claim: %q", call)
		}
	}
}
