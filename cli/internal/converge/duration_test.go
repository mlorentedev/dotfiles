package converge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// stepClock advances by step on every reading, so each runOne measures step.
func stepClock(t *testing.T, step time.Duration) {
	t.Helper()
	saved := now
	t.Cleanup(func() { now = saved })
	at := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	now = func() time.Time { at = at.Add(step); return at }
}

// CLI-114 AC4: every step that ran carries how long it took; a step the run
// never reached carries none.
func TestRun_RecordsEachStepsDuration(t *testing.T) {
	stepClock(t, 1500*time.Millisecond)
	a, b, c := &fake{name: "a", changes: 1}, &fake{name: "b", err: errors.New("boom")}, &fake{name: "c"}
	rep, _ := Run([]Reconciler{a, b, c}, Env{GOOS: "linux"}, false)
	for i, want := range []time.Duration{1500 * time.Millisecond, 1500 * time.Millisecond, 0} {
		if got := rep.Entries[i].Duration; got != want {
			t.Errorf("%s: Duration = %v, want %v", rep.Entries[i].Name, got, want)
		}
	}
}

func TestWriteReport_HoldsEachStepsDurationInMilliseconds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "last.json")
	rep := Report{GOOS: "linux", Entries: []Entry{
		{Name: "tools", Status: StatusChange, Changes: 1, Duration: 12345 * time.Millisecond},
		{Name: "identity", Status: StatusSkipped, Detail: "not reached"},
	}}
	if err := WriteReport(path, rep, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // test file
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if ms, ok := got.Entries[0]["duration_ms"].(float64); !ok || ms != 12345 {
		t.Errorf("tools: duration_ms = %v, want 12345\n%s", got.Entries[0]["duration_ms"], raw)
	}
	if ms, ok := got.Entries[1]["duration_ms"].(float64); !ok || ms != 0 {
		t.Errorf("a step not reached: duration_ms = %v, want 0\n%s", got.Entries[1]["duration_ms"], raw)
	}
}
