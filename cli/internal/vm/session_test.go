package vm

import (
	"encoding/json"
	"testing"
)

func intp(n int) *int { return &n }

// ADR-055 decision 8: `status` describes the verb, the process code follows
// it, and a guest command that exits 1 is `ok` with exit 1.
func TestRecordProcessCode(t *testing.T) {
	for _, c := range []struct {
		name string
		rec  Record
		want int
	}{
		{"ok, no guest command", Record{Status: StatusOK}, 0},
		{"ok, guest exit 0", Record{Status: StatusOK, Exit: intp(0)}, 0},
		{"ok, guest exit 1", Record{Status: StatusOK, Exit: intp(1)}, 1},
		{"ok, guest exit 7", Record{Status: StatusOK, Exit: intp(7)}, 1},
		{"failed", Record{Status: StatusFailed}, 1},
		{"timeout is 1, not 3", Record{Status: StatusTimeout}, 1},
		{"busy", Record{Status: StatusBusy}, 2},
		{"unsupported", Record{Status: StatusUnsupported}, 2},
		{"stale", Record{Status: StatusStale}, 2},
		{"no-session", Record{Status: StatusNoSession}, 3},
	} {
		if got := c.rec.ProcessCode(); got != c.want {
			t.Errorf("%s: ProcessCode = %d, want %d", c.name, got, c.want)
		}
	}
}

// The record's keys are the contract a consumer reads (ADR-055 decision 7):
// every one is present whatever the outcome, `exit` is null for a verb that
// runs no guest command, and state_dir appears only when there is one.
func TestRecordJSONShape(t *testing.T) {
	b, err := json.Marshal(NewRecord("linux-1", "linux", StatusOK))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"version":1,"session":"linux-1","os":"linux","status":"ok","exit":null,"stdout":"","stderr":"","truncated":false,"duration_ms":0}`
	if string(b) != want {
		t.Errorf("record = %s\nwant     %s", b, want)
	}
	r := NewRecord("linux-1", "linux", StatusOK)
	r.StateDir, r.Exit = "/state/x", intp(0)
	b, _ = json.Marshal(r)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["state_dir"] != "/state/x" || m["exit"] != float64(0) {
		t.Errorf("record = %s, want state_dir and exit 0", b)
	}
}

// A new status value is additive: `stale` joins the list without a version
// bump (ADR-055 decision 7).
func TestRecordVersionIsOne(t *testing.T) {
	if NewRecord("s", "linux", StatusStale).Version != 1 {
		t.Error("version is not 1")
	}
}
