package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

type stubForge map[string]lane.PRInfo

func (s stubForge) PRFor(branch string) (lane.PRInfo, error) { return s[branch], nil }

type stubFinder struct{}

func (stubFinder) Find(string, string) (lane.Worktree, bool, error) {
	return lane.Worktree{}, false, errors.New("no herdr server")
}

func statusFixture(t *testing.T) {
	t.Helper()
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	prev := laneSources
	laneSources = func() (lane.Sources, error) {
		return lane.Sources{
			Now: now,
			List: func() ([]worktree.Info, error) {
				return []worktree.Info{
					{Path: "/repo", Branch: "main", IsMain: true},
					{Path: "/wt/a", Branch: "fix/2501-a", Metadata: &worktree.Metadata{Creator: "claude", Issue: 2501, LeaseExpiresAt: now.Add(time.Hour), ReapOK: true}},
					{Path: "/wt/b", Branch: "fix/b"},
				}, nil
			},
			Forge: stubForge{"fix/2501-a": {Number: 40, State: lane.PROpen}},
			Live: func(p string) lane.Liveness {
				if p == "/wt/a" {
					return lane.LiveYes
				}
				return lane.LiveIdle
			},
			Herdr: stubFinder{},
		}, nil
	}
	t.Cleanup(func() { laneSources = prev })
}

func runLaneOut(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := New("test", "test")
	root.SetArgs(append([]string{"lane"}, args...))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(new(strings.Builder))
	err := root.Execute()
	return out.String(), err
}

// --json is a list of rows, one per lane, with the derived phase and herdr's
// state; the main checkout is not a row.
func TestLaneStatus_JSONShape(t *testing.T) {
	statusFixture(t)
	out, err := runLaneOut(t, "status", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []lane.Row
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2: %s", len(rows), out)
	}
	a := rows[0]
	if a.Branch != "fix/2501-a" || a.Phase != lane.PhaseLanding || a.PR != 40 || a.Ticket != 2501 || a.Agent != "claude" || a.Live != lane.LiveYes || a.Herdr != lane.HerdrAbsent {
		t.Errorf("row a = %+v", a)
	}
	if rows[1].Phase != lane.PhaseUnadopted {
		t.Errorf("row b phase = %q, want Unadopted", rows[1].Phase)
	}
}

func TestLaneStatus_TableAndExitCode(t *testing.T) {
	statusFixture(t)
	out, err := runLaneOut(t, "status")
	if err != nil {
		t.Fatalf("a read-only status exits 0, got %v", err)
	}
	for _, want := range []string{"PHASE", "Landing", "#2501", "#40 open", "Unadopted", "absent"} {
		if !strings.Contains(out, want) {
			t.Errorf("table lacks %q:\n%s", want, out)
		}
	}
}

func TestLaneStatus_NoLanesAndFailure(t *testing.T) {
	prev := laneSources
	t.Cleanup(func() { laneSources = prev })
	laneSources = func() (lane.Sources, error) {
		return lane.Sources{List: func() ([]worktree.Info, error) { return nil, nil }, Forge: stubForge{}}, nil
	}
	if out, err := runLaneOut(t, "status", "--json"); err != nil || strings.TrimSpace(out) != "[]" {
		t.Errorf("empty status = %q, %v; want []", out, err)
	}
	if out, err := runLaneOut(t, "status"); err != nil || !strings.Contains(out, "no lanes") {
		t.Errorf("empty table = %q, %v", out, err)
	}
	laneSources = func() (lane.Sources, error) { return lane.Sources{}, errors.New("not in a git checkout") }
	if _, err := runLaneOut(t, "status"); err == nil {
		t.Error("a repository that cannot be read must fail, not print an empty table")
	}
}

// plan prints each action with its class and, for an ambiguous one, the
// command that resolves it; --json carries the actions and the landing order.
func TestLanePlan_PrintsClassesAndResolvingCommands(t *testing.T) {
	stubLandCheck(t, func(int) ([]string, error) { return nil, nil })
	statusFixture(t)
	out, err := runLaneOut(t, "plan")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"CLASS", "ambiguous", "flag", "fix/b", "landing order:", "#40 fix/2501-a", "ready: dotf pr triage-queue --pr 40 && dotf pr land 40"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan lacks %q:\n%s", want, out)
		}
	}
	js, err := runLaneOut(t, "plan", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Actions []lane.Action  `json:"actions"`
		Landing []lane.Landing `json:"landing"`
	}
	if err := json.Unmarshal([]byte(js), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, js)
	}
	if len(got.Actions) != 1 || got.Actions[0].Class != lane.ClassAmbiguous || len(got.Landing) != 1 || !got.Landing[0].Head {
		t.Errorf("plan = %+v", got)
	}
}

func stubLandCheck(t *testing.T, check lane.LandCheck) {
	t.Helper()
	prev := laneLandCheck
	t.Cleanup(func() { laneLandCheck = prev })
	laneLandCheck = func(*cobra.Command) lane.LandCheck { return check }
}

// The head's landing conditions are printed with the reasons that stand in the
// way, and a reader that fails says the head was not checked, never ready.
func TestLanePlan_PrintsTheHeadsLandingConditions(t *testing.T) {
	statusFixture(t)
	stubLandCheck(t, func(int) ([]string, error) {
		return []string{"test-windows: pending", "reviewer output awaits triage"}, nil
	})
	out, err := runLaneOut(t, "plan")
	if err != nil || !strings.Contains(out, "blocked: test-windows: pending; reviewer output awaits triage") || strings.Contains(out, "ready:") {
		t.Errorf("blocked head printed %q, %v", out, err)
	}
	stubLandCheck(t, func(int) ([]string, error) { return nil, errors.New("HTTP 502") })
	out, err = runLaneOut(t, "plan")
	if err != nil || !strings.Contains(out, "not checked:") || strings.Contains(out, "ready:") {
		t.Errorf("unreadable head printed %q, %v", out, err)
	}
}

func TestLanePlan_NothingToDo(t *testing.T) {
	prev := laneSources
	t.Cleanup(func() { laneSources = prev })
	laneSources = func() (lane.Sources, error) {
		return lane.Sources{List: func() ([]worktree.Info, error) { return nil, nil }, Forge: stubForge{}}, nil
	}
	if out, err := runLaneOut(t, "plan"); err != nil || !strings.Contains(out, "nothing to do") {
		t.Errorf("empty plan = %q, %v", out, err)
	}
	if out, err := runLaneOut(t, "plan", "--json"); err != nil || !strings.Contains(out, `"actions": []`) {
		t.Errorf("empty JSON plan = %q, %v", out, err)
	}
}

func TestLaneApply_PrintsChangedAndRunsOnlyAutoVerbs(t *testing.T) {
	prevS, prevV := laneSources, laneVerbs
	t.Cleanup(func() { laneSources, laneVerbs = prevS, prevV })
	laneSources = func() (lane.Sources, error) {
		return lane.Sources{List: func() ([]worktree.Info, error) { return nil, nil }, Forge: stubForge{}}, nil
	}
	var sweeps, nexts int
	laneVerbs = func(*cobra.Command, lane.Sources) lane.Verbs {
		return lane.Verbs{
			Sweep: func() (int, error) { sweeps++; return 1, nil },
			Next:  func() (bool, error) { nexts++; return true, nil },
		}
	}
	out, err := runLaneOut(t, "apply")
	if err != nil || out != "changed=0\n" {
		t.Errorf("apply over an empty plan = %q, %v; want changed=0", out, err)
	}
	if sweeps != 0 || nexts != 0 {
		t.Errorf("sweeps=%d nexts=%d: an empty plan ran a verb", sweeps, nexts)
	}
	if out, err := runLaneOut(t, "apply", "--json"); err != nil || !strings.Contains(out, `"changed":0`) {
		t.Errorf("JSON apply = %q, %v", out, err)
	}
}
