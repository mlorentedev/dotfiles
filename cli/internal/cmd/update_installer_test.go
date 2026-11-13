package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// #3395: after a failed installer run, dotf update reads the converge report
// the installer's own converge left, and only one that finished after the
// installer started: an older last.json describes some earlier run.
func TestFailedConvergeSince(t *testing.T) {
	start := time.Date(2026, 11, 12, 7, 32, 42, 500e6, time.UTC)
	report := func(finished, result, entries string) string {
		return `{"finished_at":"` + finished + `","goos":"darwin","result":"` + result + `","error":"1 step failed","changed":1,"entries":[` + entries + `]}`
	}
	const failedUpgrade = `{"name":"system-upgrade","status":"failed"},{"name":"cache-gc","status":"ok"}`
	tests := []struct {
		name string
		body string // "" writes no report
		want []string
	}{
		{name: "fresh failed report names its failed steps", body: report("2026-11-12T07:40:00Z", "failed", failedUpgrade), want: []string{"system-upgrade"}},
		{name: "finished in the installer's start second", body: report("2026-11-12T07:32:42Z", "failed", failedUpgrade), want: []string{"system-upgrade"}},
		{name: "a report older than the installer is not its converge", body: report("2026-11-12T07:00:00Z", "failed", failedUpgrade), want: nil},
		{name: "a passing converge is no converge failure", body: report("2026-11-12T07:40:00Z", "ok", `{"name":"system-upgrade","status":"ok"}`), want: nil},
		{name: "a failed run with no failed step names the run", body: report("2026-11-12T07:40:00Z", "failed", `{"name":"system-upgrade","status":"ok"}`), want: []string{"the run (1 step failed)"}},
		{name: "no report", body: "", want: nil},
		{name: "an unreadable report", body: "{", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdg := t.TempDir()
			t.Setenv("XDG_STATE_HOME", xdg)
			if tt.body != "" {
				dir := filepath.Join(xdg, "dotfiles", "converge")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "last.json"), []byte(tt.body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if got := failedConvergeSince(start); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("failedConvergeSince = %q, want %q", got, tt.want)
			}
		})
	}
}
