package evals

import (
	"strings"
	"testing"
)

// The corpus this repository commits: every case checks clean, and there are
// enough of them that a score means something (HARNESS-101 AC1, AC2). This is
// the test CI runs on every PR, so a key whose anchors rot fails the build.
func TestRepositoryCorpusChecksClean(t *testing.T) {
	results, err := Check(repoFile(t))
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	for _, r := range results {
		count[r.Kind]++
		if len(r.Problems) > 0 {
			t.Errorf("%s:\n  %s", r.Path, strings.Join(r.Problems, "\n  "))
		}
	}
	if count["replay"] < 5 || count["planted"] < 2 {
		t.Errorf("corpus holds %d replay and %d planted cases; HARNESS-101 needs at least 5 and 2",
			count["replay"], count["planted"])
	}
}
