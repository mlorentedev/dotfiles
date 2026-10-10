package prland

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// queueGH answers gh for several PRs at once. A PR with no entry in states is
// CLEAN; the PRs in unreadable fail on `pr view`. Every call that changes
// something, and every update-branch, lands in calls in the order it happened.
type queueGH struct {
	states     map[string]string // PR number -> merge state
	unreadable map[string]bool
	calls      []string
}

func (g *queueGH) run(_ context.Context, args ...string) ([]byte, error) {
	line := strings.Join(args, " ")
	f := strings.Fields(line)
	switch {
	case strings.HasPrefix(line, "pr view"):
		if g.unreadable[f[2]] {
			return nil, errors.New("HTTP 502")
		}
		state := "CLEAN"
		if s, ok := g.states[f[2]]; ok {
			state = s
		}
		return fmt.Appendf(nil, `{"state":"OPEN","isDraft":false,"headRefOid":"sha%s","headRefName":"feat/%s","baseRefName":"main","mergeStateStatus":%q}`, f[2], f[2], state), nil
	case strings.HasPrefix(line, "pr checks") && strings.Contains(line, "--json"):
		return []byte(greenChecks), nil
	case strings.HasPrefix(line, "pr checks"):
		return nil, nil
	case strings.HasPrefix(line, "pr list"):
		return []byte(`[]`), nil
	case strings.HasPrefix(line, "pr update-branch"):
		g.calls = append(g.calls, line)
		// the base is merged in: the PR is up to date from here on
		g.states[f[2]] = "CLEAN"
		return nil, nil
	case strings.HasPrefix(line, "pr merge"):
		g.calls = append(g.calls, line)
		return nil, nil
	case strings.HasPrefix(line, "api "):
		return grantAnswer(false)
	}
	return nil, errors.New("unexpected gh call: " + line)
}

func numbersOf(rs []Result) []int {
	var out []int
	for _, r := range rs {
		out = append(out, r.Number)
	}
	return out
}

func TestLandQueue_LandsInTheOrderGiven(t *testing.T) {
	g := &queueGH{}
	got := LandQueue(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, []int{30, 12, 31}, nil)

	if fmt.Sprint(numbersOf(got)) != "[30 12 31]" {
		t.Fatalf("results are not in the order given: %v", numbersOf(got))
	}
	want := []string{
		"pr merge 30 --squash --delete-branch --match-head-commit sha30",
		"pr merge 12 --squash --delete-branch --match-head-commit sha12",
		"pr merge 31 --squash --delete-branch --match-head-commit sha31",
	}
	if strings.Join(g.calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("merges:\n%s\nwant:\n%s", strings.Join(g.calls, "\n"), strings.Join(want, "\n"))
	}
	for _, r := range got {
		if !r.Merged {
			t.Errorf("#%d was not merged: %v", r.Number, r.Reasons)
		}
	}
}

// The queue's reason to exist: a PR that is BEHIND while another one lands is
// not touched until it is its turn, so it pays one update, not one per merge.
func TestLandQueue_UpdatesAPROnlyDuringItsOwnTurn(t *testing.T) {
	g := &queueGH{states: map[string]string{"12": "BEHIND", "31": "BEHIND"}}
	o := Options{Run: g.run, Untriaged: noneUntriaged, UpdateBranch: true, Sleep: func(_ time.Duration) {}}

	got := LandQueue(context.Background(), o, []int{30, 12, 31}, nil)

	want := []string{
		"pr merge 30 --squash --delete-branch --match-head-commit sha30",
		"pr update-branch 12",
		"pr merge 12 --squash --delete-branch --match-head-commit sha12",
		"pr update-branch 31",
		"pr merge 31 --squash --delete-branch --match-head-commit sha31",
	}
	if strings.Join(g.calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls:\n%s\nwant:\n%s", strings.Join(g.calls, "\n"), strings.Join(want, "\n"))
	}
	if !got[1].Updated || !got[2].Updated || got[0].Updated {
		t.Errorf("Updated is %v %v %v, want false true true", got[0].Updated, got[1].Updated, got[2].Updated)
	}
}

func TestLandQueue_ContinuesPastAStoppedPR(t *testing.T) {
	g := &queueGH{states: map[string]string{"12": "DIRTY"}}
	var seen []int
	got := LandQueue(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, []int{30, 12, 31},
		func(r Result) { seen = append(seen, r.Number) })

	if len(got) != 3 || !got[0].Merged || got[1].Merged || !got[2].Merged {
		t.Fatalf("want merged, stopped, merged; got %+v", got)
	}
	if !strings.Contains(strings.Join(got[1].Reasons, ";"), "merge state is DIRTY") {
		t.Errorf("the stopped PR lost its reason: %v", got[1].Reasons)
	}
	if fmt.Sprint(seen) != "[30 12 31]" {
		t.Errorf("done was called for %v, want each PR as it finished", seen)
	}
}

func TestLandQueue_AnErrorStopsThatPRNotTheQueue(t *testing.T) {
	g := &queueGH{unreadable: map[string]bool{"12": true}}
	got := LandQueue(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, []int{12, 31}, nil)

	if got[0].Merged || len(got[0].Reasons) != 1 || !strings.Contains(got[0].Reasons[0], "HTTP 502") {
		t.Errorf("the error is not the PR's reason: %+v", got[0])
	}
	if !got[1].Merged {
		t.Errorf("the PR after an error was not landed: %+v", got[1])
	}
}

func TestLandQueue_ACancelledContextLeavesTheRestNotAttempted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	g := &queueGH{}
	got := LandQueue(ctx, Options{Run: g.run, Untriaged: noneUntriaged}, []int{30, 12, 31},
		func(r Result) {
			if r.Number == 30 {
				cancel()
			}
		})

	if len(got) != 3 || !got[0].Merged {
		t.Fatalf("want the first PR merged and a result per PR, got %+v", got)
	}
	for _, r := range got[1:] {
		if r.Merged || len(r.Reasons) != 1 || !strings.Contains(r.Reasons[0], "not attempted") {
			t.Errorf("#%d: %+v, want one 'not attempted' reason", r.Number, r)
		}
	}
	if len(g.calls) != 1 {
		t.Errorf("a cancelled queue kept landing: %v", g.calls)
	}
}
