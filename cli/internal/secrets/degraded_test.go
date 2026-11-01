package secrets

import (
	"path/filepath"
	"testing"
	"time"
)

func TestUnitFromCgroup(t *testing.T) {
	cases := map[string]string{
		// Read from the hive daemon on msi, 2026-10-31.
		"0::/user.slice/user-1000.slice/user@1000.service/app.slice/hive.service\n": "hive.service",
		"0::/user.slice/user-1000.slice/session-3.scope\n":                          "",
		"12:pids:/system.slice/foo.service\n0::/system.slice/foo.service\n":         "foo.service",
		"": "",
	}
	for in, want := range cases {
		if got := UnitFromCgroup(in); got != want {
			t.Errorf("UnitFromCgroup(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDegradedDir_NeedsARuntimeDir(t *testing.T) {
	if got := DegradedDir(func(string) string { return "" }); got != "" {
		t.Errorf("no XDG_RUNTIME_DIR: got %q, want none", got)
	}
	got := DegradedDir(func(string) string { return "/run/user/1000" })
	if want := filepath.Join("/run/user/1000", "dotf", "skipped-secrets"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDegradedTraces_RoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "skipped-secrets")
	if units, err := DegradedUnits(dir); err != nil || len(units) != 0 {
		t.Fatalf("missing dir: got %v, %v; want none", units, err)
	}
	at := time.Date(2026, 10, 30, 22, 54, 0, 0, time.UTC)
	if err := MarkDegraded(dir, "hive.service", at); err != nil {
		t.Fatal(err)
	}
	units, err := DegradedUnits(dir)
	if err != nil || units["hive.service"] != "2026-10-30T22:54:00Z" || len(units) != 1 {
		t.Fatalf("got %v, %v", units, err)
	}
	if err := ClearDegraded(dir, "hive.service"); err != nil {
		t.Fatal(err)
	}
	if err := ClearDegraded(dir, "hive.service"); err != nil {
		t.Fatalf("clearing twice: %v", err)
	}
	if units, _ := DegradedUnits(dir); len(units) != 0 {
		t.Fatalf("after clear: %v", units)
	}
}
