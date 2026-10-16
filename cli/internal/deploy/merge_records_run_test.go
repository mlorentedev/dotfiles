package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Only a merge entry needs a state directory, so one that cannot be placed
// fails a run only when a target merges: a replace-only run behaves as it did
// before records existed.
func TestMergeRecords(t *testing.T) {
	broken := func() (string, error) { return "", errors.New("no absolute home") }
	replace := Config{Name: "r", Src: "r.json", Dst: "{HOME}/r.json"}
	merge := Config{Name: "m", Src: "m.json", Dst: "{HOME}/m.json", Strategy: StrategyMerge}

	if dir, err := MergeRecords([]Config{replace}, broken); err != nil || dir != "" {
		t.Errorf("replace only: dir=%q err=%v, want no records and no error", dir, err)
	}
	if _, err := MergeRecords([]Config{replace, merge}, broken); err == nil {
		t.Error("a merge entry with no state directory must fail the run, not deploy without its record")
	}
	state := t.TempDir()
	dir, err := MergeRecords([]Config{merge}, func() (string, error) { return state, nil })
	if err != nil || dir != filepath.Join(state, "deploy", "merged") {
		t.Errorf("dir=%q err=%v, want <state>/deploy/merged", dir, err)
	}
}

// Run hands its Records to every entry, so `dotf deploy` and converge record
// and retire the same way.
func TestRun_PassesRecordsToEachMerge(t *testing.T) {
	repo, home, records := t.TempDir(), t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "m.json"), []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	man, err := ParseManifest([]byte(`{"version": 4, "configs": [
	  {"name": "m", "src": "m.json", "dst": "{HOME}/.m/config.json", "mode": "0644", "strategy": "merge"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	o := runOptions(repo, home, false)
	o.Records = records
	if _, err := Run(man, man.Configs, o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(records, "m.json")); err != nil {
		t.Errorf("Run did not record the merge: %v", err)
	}
}
