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
		Checks:        []Check{{Name: "test", Bucket: "pass"}, {Name: "goreleaser", Bucket: "skipping"}},
		GrantDeclared: true,
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
		"no merge grant":    {func(f *Facts) { f.GrantDeclared = false }, GrantFile},
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
	noGrant   bool   // the base branch has no grant file
	grantRef  string // the contents path the grant was read from
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
		if g.checks == "" {
			return nil, errors.New("exit status 1: no checks reported on the 'feat/x' branch")
		}
		return []byte(g.checks), errors.New("exit status 8") // gh exits non-zero while a check is pending
	case strings.HasPrefix(line, "pr checks"):
		return nil, nil
	case strings.HasPrefix(line, "pr list"):
		return []byte(g.deps), nil
	case strings.HasPrefix(line, "pr edit"), strings.HasPrefix(line, "pr merge"):
		g.calls = append(g.calls, line)
		return nil, nil
	case strings.HasPrefix(line, "api "):
		g.grantRef = line
		return grantAnswer(g.noGrant)
	}
	return nil, errors.New("unexpected gh call: " + line)
}

// grantAnswer is what `gh api .../contents/<GrantFile>` says.
func grantAnswer(absent bool) ([]byte, error) {
	if absent {
		return []byte(`{"message":"Not Found","status":"404"}`), errors.New("gh: Not Found (HTTP 404)")
	}
	return []byte(`{"type":"file"}`), nil
}

// The grant is read from the base branch, never the head: a PR cannot opt its
// own repository in by adding the file.
func TestLand_RefusesInARepositoryWithoutTheGrant(t *testing.T) {
	g := &fakeGH{view: readyView, checks: greenChecks, deps: `[]`, noGrant: true}

	res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Merged || len(g.calls) != 0 || !strings.Contains(strings.Join(res.Reasons, ";"), GrantFile) {
		t.Errorf("want a refusal naming %s and no mutation, got %+v, calls %v", GrantFile, res, g.calls)
	}
	want := "api repos/{owner}/{repo}/contents/" + GrantFile + "?ref=main"
	if g.grantRef != want {
		t.Errorf("grant read as %q, want %q", g.grantRef, want)
	}
}

func TestLand_ReadsTheGrantOfTheNamedRepository(t *testing.T) {
	g := &fakeGH{view: readyView, checks: greenChecks, deps: `[]`}

	if _, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged, Repo: "o/r"}, 30); err != nil {
		t.Fatal(err)
	}
	if want := "api repos/o/r/contents/" + GrantFile + "?ref=main"; g.grantRef != want {
		t.Errorf("grant read as %q, want %q", g.grantRef, want)
	}
}

// Only a 404 means "not declared"; any other failure is an unanswered
// question, and an unanswered question never lands a PR.
func TestLand_AnUnreadableGrantIsAnError(t *testing.T) {
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		if args[0] == "api" {
			return nil, errors.New("gh: HTTP 502")
		}
		return (&fakeGH{view: readyView, checks: greenChecks, deps: `[]`}).run(ctx, args...)
	}
	res, err := Land(context.Background(), Options{Run: run, Untriaged: noneUntriaged}, 30)
	if err == nil || res.Merged {
		t.Errorf("want an error and no merge, got %+v, %v", res, err)
	}
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

// Another PR lands on the base while the updated head's CI runs, so the PR is
// behind again when its checks go green: it is updated again, not given up on.
func TestLand_UpdateBranchUpdatesAgainWhenTheBaseMovesDuringTheWait(t *testing.T) {
	behind := strings.Replace(readyView, `"CLEAN"`, `"BEHIND"`, 1)
	views := []string{
		strings.Replace(behind, "abc123", "def456", 1),
		strings.Replace(readyView, "abc123", "fed789", 1),
	}
	view := behind
	var calls []string
	run := func(_ context.Context, args ...string) ([]byte, error) {
		line := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(line, "pr view"):
			return []byte(view), nil
		case strings.HasPrefix(line, "pr update-branch"):
			calls = append(calls, line)
			view, views = views[0], views[1:]
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
	if err != nil || !res.Merged {
		t.Fatalf("want two updates then a merge, got %+v, %v", res, err)
	}
	want := "pr update-branch 30\npr update-branch 30\npr merge 30 --squash --delete-branch --match-head-commit fed789"
	if got := strings.Join(calls, "\n"); got != want {
		t.Errorf("calls:\n%s\nwant:\n%s", got, want)
	}
}

// A base that never stops moving is a refusal after maxUpdates, not a loop.
func TestLand_UpdateBranchGivesUpAfterMaxUpdates(t *testing.T) {
	behind := strings.Replace(readyView, `"CLEAN"`, `"BEHIND"`, 1)
	updates := 0
	run := func(_ context.Context, args ...string) ([]byte, error) {
		line := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(line, "pr view"):
			return []byte(behind), nil
		case strings.HasPrefix(line, "pr update-branch"):
			updates++
		case strings.HasPrefix(line, "pr checks") && strings.Contains(line, "--json"):
			return []byte(greenChecks), nil
		case strings.HasPrefix(line, "pr merge"):
			t.Error("merged a PR that is still behind")
		}
		return nil, nil
	}
	o := Options{Run: run, Untriaged: noneUntriaged, UpdateBranch: true, Sleep: func(time.Duration) {}}

	res, err := Land(context.Background(), o, 30)
	want := "merge state is BEHIND after merging the base in 3 times"
	if err != nil || res.Merged || len(res.Reasons) != 1 || !strings.HasPrefix(res.Reasons[0], want) || updates != maxUpdates {
		t.Errorf("want one reason starting %q after %d updates, got %+v, %v, %d updates", want, maxUpdates, res, err, updates)
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

// failingMergeGH serves a landable PR with one dependent, fails the merge, and
// optionally fails restoring the dependent's base.
func failingMergeGH(restoreFails bool) (Runner, *[]string) {
	var calls []string
	run := func(_ context.Context, args ...string) ([]byte, error) {
		line := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(line, "pr view"):
			return []byte(readyView), nil
		case strings.HasPrefix(line, "pr checks"):
			return []byte(greenChecks), nil
		case strings.HasPrefix(line, "pr list"):
			return []byte(`[{"number":31}]`), nil
		case strings.HasPrefix(line, "pr merge"):
			calls = append(calls, line)
			return nil, errors.New("head branch was modified")
		case strings.HasPrefix(line, "api "):
			return grantAnswer(false)
		case strings.HasPrefix(line, "pr edit"):
			calls = append(calls, line)
			if restoreFails && strings.HasSuffix(line, "--base feat/x") {
				return nil, errors.New("permission denied")
			}
			return nil, nil
		}
		return nil, errors.New("unexpected gh call: " + line)
	}
	return run, &calls
}

// A merge that fails after the dependents were retargeted puts them back on
// the head branch, so the stack is as it was before land ran.
func TestLand_AFailedMergeRestoresTheRetargetedDependents(t *testing.T) {
	run, calls := failingMergeGH(false)

	res, err := Land(context.Background(), Options{Run: run, Untriaged: noneUntriaged}, 30)
	if err == nil || !strings.Contains(err.Error(), "head branch was modified") {
		t.Fatalf("want the merge error, got %v", err)
	}
	want := []string{
		"pr edit 31 --base main",
		"pr merge 30 --squash --delete-branch --match-head-commit abc123",
		"pr edit 31 --base feat/x",
	}
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls:\n%s\nwant:\n%s", strings.Join(*calls, "\n"), strings.Join(want, "\n"))
	}
	if res.Merged || len(res.Retargeted) != 0 {
		t.Errorf("a failed merge reported %+v", res)
	}
}

// When the restore fails too, both errors surface and the dependent left on
// the wrong base is named.
func TestLand_AFailedRestoreNamesTheDependentLeftRetargeted(t *testing.T) {
	run, _ := failingMergeGH(true)

	_, err := Land(context.Background(), Options{Run: run, Untriaged: noneUntriaged}, 30)
	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{"head branch was modified", "#31", "feat/x", "permission denied"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q: %v", want, err)
		}
	}
}

// Right after a push gh reports no checks at all: it exits non-zero with empty
// output. That is a refusal ("no checks reported"), not an error, and --wait
// keeps waiting for the checks to appear.
func TestLand_NoChecksYetIsARefusalNotAnError(t *testing.T) {
	g := &fakeGH{view: readyView, checks: "", deps: `[]`}

	res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if err != nil {
		t.Fatalf("empty checks output was an error: %v", err)
	}
	if res.Merged || !strings.Contains(strings.Join(res.Reasons, ";"), "no checks reported") {
		t.Errorf("want a refusal naming the missing checks, got %+v", res)
	}
	for _, state := range []string{"BLOCKED", "CLEAN"} {
		if !unsettled(Facts{MergeState: state}) {
			t.Errorf("no checks yet (state %s) must keep --wait waiting", state)
		}
	}
}

// unknownViews answers UNKNOWN for n reads, then CLEAN.
func unknownViews(g *fakeGH, n int) Runner {
	unknown := strings.Replace(readyView, `"CLEAN"`, `"UNKNOWN"`, 1)
	reads := 0
	return func(ctx context.Context, args ...string) ([]byte, error) {
		if args[0] == "pr" && args[1] == "view" {
			reads++
			if reads <= n {
				return []byte(unknown), nil
			}
			return []byte(readyView), nil
		}
		return g.run(ctx, args...)
	}
}

// #2118: GitHub left a green PR's merge state UNKNOWN for 27 minutes, and the
// six rounds that bound the checks wait gave up after about three.
func TestLand_WaitOutlastsTheChecksBudgetForAnUncomputedMergeState(t *testing.T) {
	g := &fakeGH{checks: greenChecks, deps: `[]`}
	run := unknownViews(g, 3*maxWaitRounds)

	res, err := Land(context.Background(), Options{Run: run, Untriaged: noneUntriaged, Wait: true, Sleep: func(time.Duration) {}}, 30)
	if err != nil || !res.Merged {
		t.Fatalf("want a merge once GitHub computes the state, got %+v, %v", res, err)
	}
}

func TestLand_AnUncomputedMergeStateGivesUpAtItsBudgetAndSaysHowLong(t *testing.T) {
	g := &fakeGH{checks: greenChecks, deps: `[]`}
	run := unknownViews(g, 1000)

	var slept time.Duration
	o := Options{Run: run, Untriaged: noneUntriaged, Wait: true, UnknownWait: 5 * time.Minute, Sleep: func(d time.Duration) { slept += d }}
	res, err := Land(context.Background(), o, 30)
	if err != nil || res.Merged {
		t.Fatalf("want a refusal, got %+v, %v", res, err)
	}
	if slept != 5*time.Minute {
		t.Errorf("waited %s, want the 5m budget", slept)
	}
	want := "merge state is UNKNOWN after waiting 5m0s"
	if len(res.Reasons) != 1 || !strings.HasPrefix(res.Reasons[0], want) {
		t.Errorf("reasons = %q, want one starting %q", res.Reasons, want)
	}
}

func TestLand_TheDefaultUnknownBudgetIsTheMeasuredOne(t *testing.T) {
	g := &fakeGH{checks: greenChecks, deps: `[]`}
	var slept time.Duration
	o := Options{Run: unknownViews(g, 1000), Untriaged: noneUntriaged, Wait: true, Sleep: func(d time.Duration) { slept += d }}
	if _, err := Land(context.Background(), o, 30); err != nil {
		t.Fatal(err)
	}
	if slept != DefaultUnknownWait || DefaultUnknownWait < 20*time.Minute {
		t.Errorf("waited %s with the default budget %s, want 20m or more", slept, DefaultUnknownWait)
	}
}

// BLOCKED with every check green can be a PR blocked for good (a required
// review), so it keeps the short budget rather than borrowing UNKNOWN's.
func TestLand_ABlockedStateKeepsTheChecksBudget(t *testing.T) {
	blocked := strings.Replace(readyView, `"CLEAN"`, `"BLOCKED"`, 1)
	g := &fakeGH{view: blocked, checks: greenChecks, deps: `[]`}
	pauses := 0
	res, _ := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged, Wait: true, Sleep: func(time.Duration) { pauses++ }}, 30)
	if pauses != maxWaitRounds {
		t.Errorf("paused %d times on BLOCKED, want %d", pauses, maxWaitRounds)
	}
	if len(res.Reasons) != 1 || res.Reasons[0] != "merge state is BLOCKED" {
		t.Errorf("reasons = %q", res.Reasons)
	}
}

// Without --wait nothing was waited for, so the refusal must not claim it was.
func TestDecide_AnUnwaitedUnknownStateIsPlain(t *testing.T) {
	f := readyFacts()
	f.MergeState = "UNKNOWN"
	if r := Decide(f); len(r) != 1 || r[0] != "merge state is UNKNOWN" {
		t.Errorf("reasons = %q", r)
	}
}

// #2105: no pull_request workflow ran on the head while its merge state was
// uncomputed, so the checks list stayed empty. That is GitHub's wait too.
func TestLand_AnUncomputedStateWithNoChecksGetsTheUnknownBudget(t *testing.T) {
	g := &fakeGH{checks: `[]`, deps: `[]`}
	var slept time.Duration
	o := Options{Run: unknownViews(g, 1000), Untriaged: noneUntriaged, Wait: true, UnknownWait: 10 * time.Minute, Sleep: func(d time.Duration) { slept += d }}
	res, err := Land(context.Background(), o, 30)
	if err != nil || res.Merged {
		t.Fatalf("want a refusal, got %+v, %v", res, err)
	}
	if slept != 10*time.Minute {
		t.Errorf("waited %s, want the 10m unknown budget", slept)
	}
}

func TestLand_TheUnknownBudgetIsExactWhenNotAWholeNumberOfPauses(t *testing.T) {
	g := &fakeGH{checks: greenChecks, deps: `[]`}
	var slept time.Duration
	o := Options{Run: unknownViews(g, 1000), Untriaged: noneUntriaged, Wait: true, UnknownWait: 45 * time.Second, Sleep: func(d time.Duration) { slept += d }}
	res, _ := Land(context.Background(), o, 30)
	if slept != 45*time.Second {
		t.Errorf("waited %s, want exactly 45s", slept)
	}
	if len(res.Reasons) != 1 || !strings.Contains(res.Reasons[0], "after waiting 45s") {
		t.Errorf("reasons = %q", res.Reasons)
	}
}

// #2056: gh 2.46 has no `pr checks --json`. Its usage error left stdout empty
// and read as "no checks reported", refusing a PR with every check green for
// a reason that was not true.
func TestLand_AGhWithoutChecksJSONIsAnErrorNamingTheRemedy(t *testing.T) {
	g := &fakeGH{view: readyView, deps: `[]`}
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		if args[0] == "pr" && args[1] == "checks" {
			return nil, errors.New("exit status 1: unknown flag: --json")
		}
		return g.run(ctx, args...)
	}
	res, err := Land(context.Background(), Options{Run: run, Untriaged: noneUntriaged}, 30)
	if err == nil {
		t.Fatalf("want an error, got a result %+v", res)
	}
	for _, want := range []string{"unknown flag: --json", "install a newer gh"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
	if len(g.calls) != 0 {
		t.Errorf("acted on a PR it could not read: %v", g.calls)
	}
}

func TestLand_AnyOtherChecksFailureWithNoOutputIsAnError(t *testing.T) {
	g := &fakeGH{view: readyView, deps: `[]`}
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		if args[0] == "pr" && args[1] == "checks" {
			return nil, errors.New("exit status 1: HTTP 502: Bad Gateway")
		}
		return g.run(ctx, args...)
	}
	_, err := Land(context.Background(), Options{Run: run, Untriaged: noneUntriaged}, 30)
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") || strings.Contains(err.Error(), "newer gh") {
		t.Errorf("err = %v, want the gh failure without the version remedy", err)
	}
}
