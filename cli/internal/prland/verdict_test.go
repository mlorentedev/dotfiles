package prland

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// A review-attestation verdict reaches the PR 2 to 5 minutes after the review
// does: the workflow waits for a runner, so the status stays at an older and
// correct "fail" while the verdict is on its way. #2718: pr-agent finished at
// 19:59:49Z, the verdict was published at 20:04:37Z, and `pr land` read the
// status in between (#2724). These pin that land waits for it, for a bounded
// time and saying what for, and that it does not wait when nothing is coming.

const (
	attestRed   = `[{"name":"test","bucket":"pass"},{"name":"review-attestation","bucket":"fail"}]`
	attestGreen = `[{"name":"test","bucket":"pass"},{"name":"review-attestation","bucket":"pass"}]`
	lintRed     = `[{"name":"lint","bucket":"fail"},{"name":"review-attestation","bucket":"pass"}]`

	runsActive = `[{"status":"in_progress"}]`
	runsQueued = `[{"status":"queued"}]`
	runsDone   = `[{"status":"completed"}]`

	// The status was written at 19:50:30Z, newest first as the API lists it.
	statusAt = `[{"context":"review-attestation","state":"failure","created_at":"2026-10-17T19:50:30Z"},` +
		`{"context":"review-attestation","state":"failure","created_at":"2026-10-17T19:16:06Z"},` +
		`{"context":"CodeRabbit","state":"success","created_at":"2026-10-17T20:30:00Z"}]`

	commentsAfterStatus  = `{"comments":[{"createdAt":"2026-10-17T19:46:21Z"},{"createdAt":"2026-10-17T19:59:42Z"}],"reviews":[]}`
	commentsBeforeStatus = `{"comments":[{"createdAt":"2026-10-17T19:46:21Z"}],"reviews":[{"submittedAt":"2026-10-17T19:40:00Z"}]}`
	reviewAfterStatus    = `{"comments":[],"reviews":[{"submittedAt":"2026-10-17T20:01:00Z"}]}`
)

// verdictGH answers the calls the wait makes, on top of fakeGH's.
type verdictGH struct {
	*fakeGH
	checksSeq  []string // successive `pr checks --json` answers; the last repeats
	checkReads int
	runs       string
	runsErr    error
	statuses   string
	comments   string
	extra      []string // every call the wait added, in order
}

func (g *verdictGH) run(ctx context.Context, args ...string) ([]byte, error) {
	line := strings.Join(args, " ")
	switch {
	case strings.HasPrefix(line, "run list"):
		g.extra = append(g.extra, line)
		return []byte(g.runs), g.runsErr
	case strings.HasPrefix(line, "pr view") && strings.Contains(line, "comments"):
		g.extra = append(g.extra, line)
		return []byte(g.comments), nil
	case strings.HasPrefix(line, "api ") && strings.Contains(line, "/statuses"):
		g.extra = append(g.extra, line)
		return []byte(g.statuses), nil
	case strings.HasPrefix(line, "pr checks") && strings.Contains(line, "--json"):
		i := min(g.checkReads, len(g.checksSeq)-1)
		g.checkReads++
		return []byte(g.checksSeq[i]), errors.New("exit status 8")
	}
	return g.fakeGH.run(ctx, args...)
}

func newVerdictGH(checks ...string) *verdictGH {
	return &verdictGH{
		fakeGH:    &fakeGH{view: readyView, deps: `[]`},
		checksSeq: checks,
		runs:      runsDone,
		statuses:  statusAt,
		comments:  commentsBeforeStatus,
	}
}

// land runs Land with a recorded Sleep and Notify.
func (g *verdictGH) land(t *testing.T, o Options) (res Result, slept []time.Duration, notes []string) {
	t.Helper()
	o.Run, o.Untriaged = g.run, noneUntriaged
	o.Sleep = func(d time.Duration) { slept = append(slept, d) }
	o.Notify = func(m string) { notes = append(notes, m) }
	res, err := Land(context.Background(), o, 30)
	if err != nil {
		t.Fatalf("Land: %v", err)
	}
	return res, slept, notes
}

func TestLand_WaitsForAPrAgentRunStillRunningOnTheHead(t *testing.T) {
	g := newVerdictGH(attestRed, attestGreen)
	g.runs = runsActive

	res, slept, notes := g.land(t, Options{})
	if !res.Merged {
		t.Fatalf("want a merge once the verdict turns green, got %+v", res)
	}
	if len(slept) != 1 || slept[0] != 30*time.Second {
		t.Errorf("slept %v, want one pause of 30s", slept)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "a pr-agent run is still running on abc123") {
		t.Errorf("notes %q must name what land waits for", notes)
	}
}

func TestLand_WaitsForAQueuedPrAgentRunToo(t *testing.T) {
	g := newVerdictGH(attestRed, attestGreen)
	g.runs = runsQueued

	_, _, notes := g.land(t, Options{})
	if len(notes) != 1 || !strings.Contains(notes[0], "a pr-agent run is still queued on abc123") {
		t.Errorf("notes %q", notes)
	}
}

// The #2718 gap: pr-agent had finished and its Guide was posted, so no run is
// active on the head. The only evidence that the verdict is stale is that the
// status is older than the newest comment.
func TestLand_WaitsWhenTheStatusIsOlderThanTheNewestComment(t *testing.T) {
	for name, comments := range map[string]string{"a comment": commentsAfterStatus, "a review": reviewAfterStatus} {
		t.Run(name, func(t *testing.T) {
			g := newVerdictGH(attestRed, attestRed, attestGreen)
			g.comments = comments

			res, slept, notes := g.land(t, Options{})
			if !res.Merged || len(slept) != 2 {
				t.Fatalf("want a merge after two pauses, got %+v, slept %v", res, slept)
			}
			if !strings.Contains(notes[0], "is newer than the review-attestation status") {
				t.Errorf("notes %q must say the status is older than the newest comment", notes)
			}
		})
	}
}

func TestLand_DoesNotWaitWhenNothingIsComing(t *testing.T) {
	g := newVerdictGH(attestRed)

	res, slept, notes := g.land(t, Options{})
	if res.Merged || len(slept) != 0 || len(notes) != 0 {
		t.Fatalf("want an immediate refusal, got %+v, slept %v, notes %q", res, slept, notes)
	}
	got := strings.Join(res.Reasons, "; ")
	if !strings.Contains(got, "review-attestation: fail") || strings.Contains(got, "waited") {
		t.Errorf("reasons %q", got)
	}
}

func TestLand_TheVerdictWaitIsBoundedAndTheRefusalSaysSo(t *testing.T) {
	g := newVerdictGH(attestRed)
	g.runs = runsActive

	res, slept, _ := g.land(t, Options{VerdictWait: 90 * time.Second})
	if res.Merged || len(slept) != 3 {
		t.Fatalf("want a refusal after three pauses, got %+v, slept %v", res, slept)
	}
	got := strings.Join(res.Reasons, "; ")
	for _, want := range []string{"review-attestation: fail", "waited 1m30s for the review-attestation verdict (a pr-agent run is still running on abc123)"} {
		if !strings.Contains(got, want) {
			t.Errorf("reasons %q lack %q", got, want)
		}
	}
}

func TestLand_TheLastVerdictPauseIsCutToTheBudget(t *testing.T) {
	g := newVerdictGH(attestRed)
	g.runs = runsActive

	_, slept, _ := g.land(t, Options{VerdictWait: 45 * time.Second})
	if len(slept) != 2 || slept[0] != 30*time.Second || slept[1] != 15*time.Second {
		t.Errorf("slept %v, want 30s then 15s", slept)
	}
}

func TestLand_ANegativeVerdictWaitDoesNotWaitOrLook(t *testing.T) {
	g := newVerdictGH(attestRed)
	g.runs = runsActive

	res, slept, _ := g.land(t, Options{VerdictWait: -1})
	if res.Merged || len(slept) != 0 || len(g.extra) != 0 {
		t.Errorf("want a plain refusal with no extra reads, got %+v, slept %v, extra %v", res, slept, g.extra)
	}
}

// An unanswered question is named in the refusal; it is never read as "nothing
// in flight" and never turned into a merge.
func TestLand_AnUnreadableRunListIsNamedInTheRefusal(t *testing.T) {
	g := newVerdictGH(attestRed)
	g.runsErr = errors.New("gh: HTTP 502")

	res, slept, _ := g.land(t, Options{})
	got := strings.Join(res.Reasons, "; ")
	if res.Merged || len(slept) != 0 || !strings.Contains(got, "could not check whether a review-attestation verdict is in flight") || !strings.Contains(got, "HTTP 502") {
		t.Errorf("want a refusal naming the unreadable run list, got %+v", res)
	}
}

// Only the attestation check being red is a stale-verdict suspect: a lint
// failure with a green attestation is a real failure, and waiting would only
// delay it.
func TestLand_OnlyARedAttestationIsWaitedFor(t *testing.T) {
	g := newVerdictGH(lintRed)
	g.runs = runsActive

	res, slept, _ := g.land(t, Options{})
	if res.Merged || len(slept) != 0 || len(g.extra) != 0 {
		t.Errorf("want a plain refusal, got %+v, slept %v, extra %v", res, slept, g.extra)
	}
}

func TestLand_AGreenAttestationCostsNoExtraReads(t *testing.T) {
	g := newVerdictGH(attestGreen)
	g.runs = runsActive

	res, _, _ := g.land(t, Options{})
	if !res.Merged || len(g.extra) != 0 {
		t.Errorf("want a merge with no extra reads, got %+v, extra %v", res, g.extra)
	}
}

func TestLand_AVerdictThatTurnsRedForGoodStopsTheWaitWhenNothingIsComing(t *testing.T) {
	g := newVerdictGH(attestRed)
	g.runs = runsActive
	// pr-agent finishes during the first pause and nothing newer than the status exists.
	runs := []string{runsActive, runsDone}
	i := 0
	wrapped := func(ctx context.Context, args ...string) ([]byte, error) {
		if strings.HasPrefix(strings.Join(args, " "), "run list") {
			g.runs = runs[min(i, len(runs)-1)]
			i++
		}
		return g.run(ctx, args...)
	}
	var slept []time.Duration
	res, err := Land(context.Background(), Options{
		Run: wrapped, Untriaged: noneUntriaged, Sleep: func(d time.Duration) { slept = append(slept, d) },
	}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Merged || len(slept) != 1 {
		t.Errorf("want one pause and then a refusal, got %+v, slept %v", res, slept)
	}
}

func TestBlockers_DoesNotWaitForAVerdict(t *testing.T) {
	g := newVerdictGH(attestRed)
	g.runs = runsActive

	got, err := Blockers(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged, Sleep: func(time.Duration) { t.Error("Blockers slept") }}, 30)
	if err != nil || !strings.Contains(strings.Join(got, "; "), "review-attestation: fail") || len(g.extra) != 0 {
		t.Errorf("want the plain reason and no extra reads, got %v, %v, extra %v", got, err, g.extra)
	}
}

// 457 s is the longest a workflow_run verdict waited for a runner in the
// 2026-10-17 window measured on #2724; the budget must outlast it.
func TestDefaultVerdictWaitOutlastsTheLongestMeasuredVerdict(t *testing.T) {
	if DefaultVerdictWait <= 457*time.Second {
		t.Errorf("DefaultVerdictWait %s does not outlast the measured 457s", DefaultVerdictWait)
	}
}
