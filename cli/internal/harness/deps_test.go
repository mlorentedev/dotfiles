package harness

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"testing"
)

// HARNESS-147: the compiled map is only the fallback for a machine with no
// records to read, so it has to say what the records say. It had drifted in
// three rows before this test existed.
func TestTheCompiledDependencyMapMatchesTheRecords(t *testing.T) {
	records, err := LoadSkillDependencies(filepath.Join("..", "..", "..", "harness", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatal("no record declares requires:, so the comparison would pass on nothing")
	}
	names := map[string]bool{}
	for n := range records {
		names[n] = true
	}
	for n := range DefaultSkillDependencies {
		names[n] = true
	}
	for n := range names {
		compiled, recorded := slices.Clone(DefaultSkillDependencies[n]), slices.Clone(records[n])
		sort.Strings(compiled)
		sort.Strings(recorded)
		if !slices.Equal(compiled, recorded) {
			t.Errorf("%s: the compiled map says %v, its record's requires: says %v", n, compiled, recorded)
		}
	}
}

// HARNESS-147: the router reads prerequisites from the records it runs beside,
// so they deploy with the skills rather than with a release. The compiled map
// answers only where there are no records to read.
func TestSkillDependenciesReadTheRecordsAtRunTime(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "harness", "skills", "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: alpha\nrequires: [beta]\n---\n# Alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := SkillDependencies(root)
	if !slices.Equal(got["alpha"], []string{"beta"}) {
		t.Errorf("alpha requires %v, want [beta] from its record", got["alpha"])
	}
	if _, ok := got["spec"]; ok {
		t.Error("the compiled map was mixed into a root that has records")
	}
	for _, r := range []string{"", t.TempDir()} {
		if !reflect.DeepEqual(SkillDependencies(r), DefaultSkillDependencies) {
			t.Errorf("root %q has no records, so the compiled map should answer", r)
		}
	}
}
