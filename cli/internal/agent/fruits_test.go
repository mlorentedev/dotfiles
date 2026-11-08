package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestFruitDeck_64SequentialDrawsHaveZeroCollisions(t *testing.T) {
	deck := NewFruitDeck(42)
	seen := make(map[string]bool, 64)

	for i := 0; i < 64; i++ {
		id := deck.NextWorkerID("reviewer")
		if !strings.HasPrefix(id, "meeseek-reviewer-") {
			t.Fatalf("draw %d = %q, want prefix meeseek-reviewer-", i, id)
		}
		if seen[id] {
			t.Fatalf("collision on draw %d: %q already drawn", i, id)
		}
		seen[id] = true

		role, fruit, ok := ParseWorkerID(id)
		if !ok || role != "reviewer" || fruit == "" {
			t.Fatalf("ParseWorkerID(%q) = (%q, %q, %v), want reviewer, non-empty fruit, true", id, role, fruit, ok)
		}
	}
	if len(seen) != 64 {
		t.Fatalf("unique draws = %d, want 64", len(seen))
	}
}

func TestParseWorkerID_RejectsPluralAndMalformedNames(t *testing.T) {
	invalid := []string{
		"",
		"reviewer",
		"meeseeks-reviewer-avocado", // plural forbidden
		"meeseek-Reviewer-avocado",  // uppercase forbidden
		"meeseek-reviewer-notafruit",
		"meeseek-reviewer",
		"meeseek--avocado",
	}
	for _, in := range invalid {
		if role, fruit, ok := ParseWorkerID(in); ok {
			t.Errorf("ParseWorkerID(%q) = (%q, %q, true), want false", in, role, fruit)
		}
	}
}

func TestDispatch_PopulatesMeeseekWorkerIDAndPropagatesToRequest(t *testing.T) {
	var gotReq Request
	be := backendFunc(func(_ context.Context, req Request) Response {
		gotReq = req
		return Response{Status: StatusOK, Output: "done"}
	})

	rec := Dispatch(context.Background(), Options{
		Tier:    "mid",
		Role:    "builder",
		Task:    "implement feature",
		Chain:   []string{"claude:sonnet"},
		Timeout: time.Minute,
		Now:     fixedClock(time.Millisecond),
	}, be)

	if !strings.HasPrefix(rec.WorkerID, "meeseek-builder-") {
		t.Errorf("rec.WorkerID = %q, want meeseek-builder-<fruit>", rec.WorkerID)
	}
	if gotReq.WorkerID != rec.WorkerID {
		t.Errorf("req.WorkerID = %q, want %q", gotReq.WorkerID, rec.WorkerID)
	}
}

func TestPiBackend_IncludesProviderFlagAndClassifiesUnauthenticatedAsUnavailable(t *testing.T) {
	h, ok := harnessFor["nan"]
	if !ok {
		t.Fatal("nan missing from harnessFor")
	}
	args := strings.Join(h.args("deepseek-v4-flash"), " ")
	if !strings.Contains(args, "--provider nan") || !strings.Contains(args, "--model deepseek-v4-flash") {
		t.Errorf("nan args = %q, want --provider nan and --model deepseek-v4-flash", args)
	}

	resp := classifyProcess("", `Model "deepseek-v4-flash" is ambiguous across providers. No matching provider is authenticated.`, 1, nil)
	if resp.Status != StatusPoolUnavailable {
		t.Errorf("classifyProcess unauthenticated = %q, want %q", resp.Status, StatusPoolUnavailable)
	}
}
