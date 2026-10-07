package prland

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func readyFacts() Facts {
	return Facts{
		State: "OPEN", HeadSHA: "abc123", HeadRef: "feat/x", BaseRef: "main", MergeState: "CLEAN",
		Checks: []Check{{Name: "test", Bucket: "pass"}, {Name: "goreleaser", Bucket: "skipping"}},
	}
}

func TestDecide_AReadyPRHasNoReasons(t *testing.T) {
	if r := Decide(readyFacts()); len(r) != 0 {
		t.Fatalf("want no reasons, got %v", r)
	}
}

func TestDecide_NamesEveryConditionThatFails(t *testing.T) {
	cases := map[string]struct {
		mutate func(*Facts)
		want   string
	}{
		"failing check":     {func(f *Facts) { f.Checks = append(f.Checks, Check{"lint", "fail"}) }, "lint: fail"},
		"pending check":     {func(f *Facts) { f.Checks = append(f.Checks, Check{"test-windows", "pending"}) }, "test-windows: pending"},
		"no checks at all":  {func(f *Facts) { f.Checks = nil }, "no checks reported"},
		"behind the base":   {func(f *Facts) { f.MergeState = "BEHIND" }, "merge state is BEHIND"},
		"conflicts":         {func(f *Facts) { f.MergeState = "DIRTY" }, "merge state is DIRTY"},
		"untriaged review":  {func(f *Facts) { f.Untriaged = true }, "reviewer output awaits triage"},
		"draft":             {func(f *Facts) { f.IsDraft = true }, "draft"},
		"closed":            {func(f *Facts) { f.State = "CLOSED" }, "not open"},
		"release-please PR": {func(f *Facts) { f.HeadRef = "release-please--branches--main" }, "release-please"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := readyFacts()
			tc.mutate(&f)
			got := strings.Join(Decide(f), "; ")
			if !strings.Contains(got, tc.want) {
				t.Errorf("reasons %q lack %q", got, tc.want)
			}
		})
	}
}

// fakeGH answers the gh calls Land makes and records the ones that change
// something.
type fakeGH struct {
	view      string   // JSON for `pr view --json ...`
	viewAfter string   // JSON for the second view; "" means the same as view
	checks    string   // JSON for `pr checks --json`
	deps      string   // JSON for `pr list --base`
	calls     []string // every mutating call, in order
	views     int
}

func (g *fakeGH) run(_ context.Context, args ...string) ([]byte, error) {
	line := strings.Join(args, " ")
	switch {
	case strings.HasPrefix(line, "pr view"):
		g.views++
		if g.views > 1 && g.viewAfter != "" {
			return []byte(g.viewAfter), nil
		}
		return []byte(g.view), nil
	case strings.HasPrefix(line, "pr checks") && strings.Contains(line, "--json"):
		return []byte(g.checks), errors.New("exit status 8") // gh exits non-zero while a check is pending
	case strings.HasPrefix(line, "pr checks"):
		return nil, nil
	case strings.HasPrefix(line, "pr list"):
		return []byte(g.deps), nil
	case strings.HasPrefix(line, "pr edit"), strings.HasPrefix(line, "pr merge"):
		g.calls = append(g.calls, line)
		return nil, nil
	}
	return nil, errors.New("unexpected gh call: " + line)
}

const readyView = `{"state":"OPEN","isDraft":false,"headRefOid":"abc123","headRefName":"feat/x","baseRefName":"main","mergeStateStatus":"CLEAN"}`
const greenChecks = `[{"name":"test","bucket":"pass"},{"name":"goreleaser","bucket":"skipping"}]`

func noneUntriaged(context.Context) ([]int, error) { return nil, nil }

func TestLand_RetargetsDependentsThenMergesTheCheckedHead(t *testing.T) {
	g := &fakeGH{view: readyView, checks: greenChecks, deps: `[{"number":31}]`}

	res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if err != nil || !res.Merged {
		t.Fatalf("want a merge, got %+v, %v", res, err)
	}
	want := []string{
		"pr edit 31 --base main",
		"pr merge 30 --squash --delete-branch --match-head-commit abc123",
	}
	if strings.Join(g.calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls:\n%s\nwant:\n%s", strings.Join(g.calls, "\n"), strings.Join(want, "\n"))
	}
}

func TestLand_RefusesWithoutMutatingWhenAConditionFails(t *testing.T) {
	g := &fakeGH{view: readyView, checks: `[{"name":"test","bucket":"pending"}]`, deps: `[]`}

	res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Merged || len(g.calls) != 0 || !strings.Contains(strings.Join(res.Reasons, ";"), "test: pending") {
		t.Errorf("want a refusal naming the pending check and no mutation, got %+v, calls %v", res, g.calls)
	}
}

func TestLand_RefusesAnUntriagedPR(t *testing.T) {
	g := &fakeGH{view: readyView, checks: greenChecks, deps: `[]`}
	untriaged := func(context.Context) ([]int, error) { return []int{12, 30}, nil }

	res, _ := Land(context.Background(), Options{Run: g.run, Untriaged: untriaged}, 30)
	if res.Merged || len(g.calls) != 0 {
		t.Errorf("an untriaged PR was merged or touched: %+v %v", res, g.calls)
	}
}

func TestLand_RefusesWhenTheHeadMovesBetweenReads(t *testing.T) {
	moved := strings.Replace(readyView, "abc123", "def456", 1)
	g := &fakeGH{view: readyView, viewAfter: moved, checks: greenChecks, deps: `[]`}

	res, _ := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if res.Merged || len(g.calls) != 0 || !strings.Contains(strings.Join(res.Reasons, ";"), "head moved") {
		t.Errorf("want a refusal on a moved head, got %+v, calls %v", res, g.calls)
	}
}

func TestLand_AnUnanswerableTriageQueueIsARefusalNotAPass(t *testing.T) {
	g := &fakeGH{view: readyView, checks: greenChecks, deps: `[]`}
	broken := func(context.Context) ([]int, error) { return nil, errors.New("rate limited") }

	if _, err := Land(context.Background(), Options{Run: g.run, Untriaged: broken}, 30); err == nil || len(g.calls) != 0 {
		t.Errorf("want an error and no mutation when the triage queue cannot be read, got calls %v", g.calls)
	}
}

// Right after a push, `gh pr checks --watch` can return while required checks
// are not registered yet: every reported check is green but the merge state is
// BLOCKED. --wait reads again instead of refusing on that transient state.
func TestLand_WaitReadsAgainUntilTheStateSettles(t *testing.T) {
	blocked := strings.Replace(readyView, `"CLEAN"`, `"BLOCKED"`, 1)
	views := []string{blocked, readyView, readyView}
	g := &fakeGH{checks: greenChecks, deps: `[]`}
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		if args[0] == "pr" && args[1] == "view" {
			v := views[0]
			if len(views) > 1 {
				views = views[1:]
			}
			return []byte(v), nil
		}
		return g.run(ctx, args...)
	}

	pauses := 0
	res, err := Land(context.Background(), Options{Run: run, Untriaged: noneUntriaged, Wait: true, Sleep: func(time.Duration) { pauses++ }}, 30)
	if pauses == 0 {
		t.Error("--wait read again without pausing for checks to register")
	}
	if err != nil || !res.Merged {
		t.Fatalf("want a merge once the state settles, got %+v, %v", res, err)
	}
}

// Without --wait, the transient state is a refusal: nothing waits by default.
func TestLand_WithoutWaitABlockedStateIsARefusal(t *testing.T) {
	blocked := strings.Replace(readyView, `"CLEAN"`, `"BLOCKED"`, 1)
	g := &fakeGH{view: blocked, checks: greenChecks, deps: `[]`}

	res, _ := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if res.Merged || len(g.calls) != 0 {
		t.Errorf("merged a BLOCKED PR without --wait: %+v", res)
	}
}

func TestLand_UpdateBranchMergesTheBaseThenLandsTheNewHead(t *testing.T) {
	behind := strings.Replace(readyView, `"CLEAN"`, `"BEHIND"`, 1)
	updated := strings.Replace(readyView, "abc123", "def456", 1)
	view := behind
	var calls []string
	run := func(_ context.Context, args ...string) ([]byte, error) {
		line := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(line, "pr view"):
			return []byte(view), nil
		case strings.HasPrefix(line, "pr update-branch"):
			calls = append(calls, line)
			view = updated
			return nil, nil
		case strings.HasPrefix(line, "pr checks") && strings.Contains(line, "--json"):
			return []byte(greenChecks), nil
		case strings.HasPrefix(line, "pr list"):
			return []byte(`[]`), nil
		case strings.HasPrefix(line, "pr merge"):
			calls = append(calls, line)
		}
		return nil, nil
	}
	o := Options{Run: run, Untriaged: noneUntriaged, UpdateBranch: true, Sleep: func(time.Duration) {}}

	res, err := Land(context.Background(), o, 30)
	if err != nil || !res.Merged || !res.Updated {
		t.Fatalf("want an update then a merge, got %+v, %v", res, err)
	}
	want := "pr update-branch 30\npr merge 30 --squash --delete-branch --match-head-commit def456"
	if got := strings.Join(calls, "\n"); got != want {
		t.Errorf("calls:\n%s\nwant:\n%s", got, want)
	}
}

// Behind AND another failing condition: updating cannot make it landable, so
// nothing is pushed.
func TestLand_UpdateBranchDoesNotTouchAPRThatFailsForAnotherReason(t *testing.T) {
	behind := strings.Replace(readyView, `"CLEAN"`, `"BEHIND"`, 1)
	g := &fakeGH{view: behind, checks: `[{"name":"lint","bucket":"fail"}]`, deps: `[]`}

	res, _ := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged, UpdateBranch: true}, 30)
	if res.Merged || res.Updated || len(g.calls) != 0 {
		t.Errorf("updated or merged a PR with a failing check: %+v %v", res, g.calls)
	}
}
