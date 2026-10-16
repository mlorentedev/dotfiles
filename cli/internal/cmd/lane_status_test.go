package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

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
