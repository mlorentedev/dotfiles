package spec

import (
	"strings"
	"testing"
)

const waivedProposal = "---\nstatus: verifying\nreview: waived\nreview_waived_reason: \"retroactive spec, nothing to review\"\n---\n"

func signedReview(verdict string) string {
	return "---\nspec: AI-001-x\nverdict: " + verdict + "\nreviewed_sha: abc1234\n---\n"
}

// TestArchiveWaiverNeverOverridesASignedFail pins GUARD-018 (#1644): a waiver
// stands in for a review nobody ran, never for one that ran and refused.
func TestArchiveWaiverNeverOverridesASignedFail(t *testing.T) {
	for name, slot := range map[string]string{"first signer": ReviewFile, "second signer": SecondSigner.Review} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeSpec(t, root, "AI-001-x", map[string]string{
				"proposal.md": waivedProposal,
				slot:          signedReview("FAIL"),
			})

			_, err := Archive(root, "AI-001-x", ArchiveOptions{})
			if err == nil {
				t.Fatal("a waiver archived a spec whose signed review says FAIL")
			}
			for _, want := range []string{slot, "verdict FAIL", "dotf spec review AI-001-x", "dotf spec archive --help"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusal should name %q, got: %v", want, err)
				}
			}
		})
	}
}

// TestArchiveSignedFailWithWaiverLeavesABypassRecord is the honest exit for an
// abandoned spec: the override is still possible, and it is written down.
func TestArchiveSignedFailWithWaiverLeavesABypassRecord(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": waivedProposal,
		ReviewFile:    signedReview("FAIL"),
	})

	target, err := Archive(root, "AI-001-x", ArchiveOptions{
		ForceWithoutReview: true, BypassReason: "abandoned; findings ticketed as #1", Date: "2026-10-22",
	})
	if err != nil {
		t.Fatalf("--force-without-review --reason should archive: %v", err)
	}
	if got := frontmatterFields(readProposal(t, target))["review_bypass"]; !strings.Contains(got, "verdict FAIL") {
		t.Errorf("review_bypass should record the overridden FAIL, got %q", got)
	}
}

// TestArchiveWaiverBesideAPassingReviewArchives: the waiver is redundant there,
// and nothing it skips was a refusal.
func TestArchiveWaiverBesideAPassingReviewArchives(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": waivedProposal,
		ReviewFile:    signedReview("PASS"),
	})

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{}); err != nil {
		t.Fatalf("a waiver beside a PASS should archive: %v", err)
	}
}
