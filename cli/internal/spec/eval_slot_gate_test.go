package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An evaluation answer copied into a spec folder must not archive it (C37). The
// repository here has no reviewer pool, which is the case the pool check cannot
// cover: with no pool it has no opinion, so only the eval refusal stands
// between a graded answer and a signed review.
func TestCheckReviewGate_RefusesEvalRecord(t *testing.T) {
	for _, tc := range []struct {
		name            string
		review, request string
	}{
		{"both name the eval reviewer", "eval:nan/glm5.3-flash", "eval:nan/glm5.3-flash"},
		{"only the launcher's request does", "nan/glm5.3-flash", "eval:nan/glm5.3-flash"},
		{"only the answer does", "eval:nan/glm5.3-flash", "nan/glm5.3-flash"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			specDir := filepath.Join(repoRoot, "specs", "X")
			if err := os.MkdirAll(specDir, 0o755); err != nil {
				t.Fatal(err)
			}
			// The request first: it digests the review.md it finds, and a
			// review written before it would read as "has not changed".
			if err := WriteReviewRequest(specDir, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", tc.request, "basebasebase"); err != nil {
				t.Fatal(err)
			}
			doc := strings.Replace(provenanceReviewDoc, `reviewer: "nan/deepseek-v4-flash"`, `reviewer: "`+tc.review+`"`, 1)
			if err := os.WriteFile(filepath.Join(specDir, ReviewFile), []byte(doc), 0o600); err != nil {
				t.Fatal(err)
			}
			err := checkReviewGate(repoRoot, "X", specDir, nil)
			if err == nil {
				t.Fatal("an evaluation answer archived a spec")
			}
			if !strings.Contains(err.Error(), "evaluation") {
				t.Errorf("the refusal does not say the record is an evaluation answer: %v", err)
			}
		})
	}
}
