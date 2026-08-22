package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryDoctrineMarkerIsInItsOwnRecord is the guard for a defect that shipped
// and then bit within the hour.
//
// enforcedRegionMarkers answers "did region X reach this deployed file?" by
// searching for a phrase. The phrase for `definition-of-done` was
// "Definition of Done" — which appears **nowhere in that record**. Its only
// occurrence in the whole enforced set was inside a DIFFERENT record's
// provenance blockquote: pr-stewardship opens with "It elaborates Definition of
// Done §4 …".
//
// So the check passed for years by proxy, verifying one region's presence by
// finding meta-text belonging to another. It broke the moment #1181 compacted
// provenance blockquotes out of the capped payload — the doctrine was entirely
// intact and the check failed anyway.
//
// A marker must identify the region it NAMES, from that region's own rule text.
// Verifying a thing by a proxy that lives somewhere else is the failure this
// repository keeps re-measuring in new costumes.
func TestEveryDoctrineMarkerIsInItsOwnRecord(t *testing.T) {
	root := repoRootForDoctorTest(t)
	for id, marker := range enforcedRegionMarkers {
		body, err := os.ReadFile(filepath.Join(root, "harness", "enforced", id+".md"))
		if err != nil {
			t.Errorf("enforced record for marker %q is missing: %v", id, err)
			continue
		}
		if !strings.Contains(string(body), marker) {
			t.Errorf("marker for %q is %q, which does not appear in harness/enforced/%s.md.\n"+
				"A marker must come from the rule text of the region it names — otherwise the check "+
				"passes by finding someone else's prose, and breaks when that prose moves.",
				id, marker, id)
		}
	}
}

// TestDoctrineMarkersSurviveCompaction pins the specific interaction that broke:
// the compacted payload drops provenance blockquotes, so no marker may live in
// one. Keying on a line the renderer is designed to remove is a check that
// reports a deployment failure for a deployment that succeeded.
func TestDoctrineMarkersSurviveCompaction(t *testing.T) {
	root := repoRootForDoctorTest(t)
	for id, marker := range enforcedRegionMarkers {
		body, err := os.ReadFile(filepath.Join(root, "harness", "enforced", id+".md"))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(body), "\n") {
			if !strings.Contains(line, marker) {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), "> Injected verbatim into every agent") {
				t.Errorf("marker for %q (%q) is inside the provenance blockquote, which "+
					"render_region_compact removes from capped surfaces — the check would then fail "+
					"on a file whose doctrine is entirely present", id, marker)
			}
		}
	}
}

// repoRootForDoctorTest walks up to the repository root so these guards read the
// records the repo actually ships.
func repoRootForDoctorTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "harness", "enforced")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find harness/enforced walking up from %s", dir)
		}
		dir = parent
	}
}
