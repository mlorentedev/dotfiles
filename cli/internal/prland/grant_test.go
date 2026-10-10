package prland

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v3"
)

// grant is the shape of GrantFile.
type grant struct {
	Grant      string   `yaml:"grant"`
	Conditions []string `yaml:"conditions"`
	Excluded   []string `yaml:"excluded"`
	Merge      string   `yaml:"merge"`
	Disclose   string   `yaml:"disclose"`
}

// This repository's grant is pinned whole: a condition dropped, an exclusion
// removed, or a key added fails here, so widening the grant takes an edit the
// owner reviews in this test, not only in the file (#2178).
func TestTheRepositoryGrantIsExactlyTheDecidedOne(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", GrantFile))
	if err != nil {
		t.Fatalf("this repository opts in to the merge grant, so %s must exist: %v", GrantFile, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	var got grant
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("%s: %v", GrantFile, err)
	}
	want := grant{
		Grant:      "prs-the-agent-opened",
		Conditions: []string{"ci-green", "no-conflicts", "reviewer-triaged"},
		Excluded:   []string{"release-please"},
		Merge:      "squash --match-head-commit",
		Disclose:   "who reviewed, or that no review ran, in the triage comment",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s declares\n%+v\nwant\n%+v", GrantFile, got, want)
	}
}

// Each declared condition is one Decide enforces: a grant that names a
// condition the actor does not check would be a promise nothing keeps.
func TestDecideRefusesForEveryDeclaredCondition(t *testing.T) {
	breaks := map[string]func(*Facts){
		"ci-green":         func(f *Facts) { f.Checks = []Check{{"test", "fail"}} },
		"no-conflicts":     func(f *Facts) { f.MergeState = "DIRTY" },
		"reviewer-triaged": func(f *Facts) { f.Untriaged = true },
		"release-please":   func(f *Facts) { f.HeadRef = releasePrefix + "branches--main" },
		"the grant itself": func(f *Facts) { f.GrantDeclared = false },
	}
	for name, mutate := range breaks {
		f := readyFacts()
		mutate(&f)
		if len(Decide(f)) == 0 {
			t.Errorf("%s: Decide lands a PR that breaks it", name)
		}
	}
}
