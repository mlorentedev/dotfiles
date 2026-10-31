package prreview

import "testing"

// The exact output of each edge case, as the script printed it when
// TestGateMatchesTheScript held the two equal. The differential test goes with
// the script; this table stays, so these rules keep a test after that.
var edgeGateWants = map[string]string{
	"the baseline is the last review by position, not by date": "run=false\nreason=1 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"a marker older than the latest review is spent":           "run=false\nreason=1 new commit(s) since the review of 2026-09-25T12:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"a commit dated at the review is not new":                  "run=true\nreason=3 new commit(s) since the review of 2026-09-25T10:00:00Z, at or past the threshold of 3\nmode=incremental\n",
	"a commit dated at the review, three of them":              "run=false\nreason=0 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"the committer date wins over a later author date":         "run=false\nreason=0 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"commits without parents count":                            "run=false\nreason=1 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"a commit with exactly two parents is a merge":             "run=false\nreason=0 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"a root commit counts":                                     "run=false\nreason=1 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"merges after the head_sha do not count":                   "run=true\nreason=3 new commit(s) since the reviewed commit c0, at or past the threshold of 3\nmode=incremental\n",
	"a head_sha at the tip has nothing new":                    "run=false\nreason=0 new commit(s) since the reviewed commit c0, below the threshold of 3; comment /review to ask for one now\n",
	"an identity on line 5 marks a review":                     "run=false\nreason=1 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"an identity on line 6 does not":                           "run=true\nreason=no previous review, so PR-Agent reviews the whole PR\nmode=full\n",
	"an indented heading is not a review":                      "run=true\nreason=no previous review, so PR-Agent reviews the whole PR\nmode=full\n",
	"an identity behind NBSP and NEL":                          "run=false\nreason=1 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"a state block under the link-reference identity":          "run=false\nreason=1 new commit(s) since the reviewed commit c0, below the threshold of 3; comment /review to ask for one now\n",
	"the last of two state blocks is the one read":             "run=false\nreason=0 new commit(s) since the reviewed commit c0, below the threshold of 3; comment /review to ask for one now\n",
	"a block with trailing blank lines":                        "run=false\nreason=0 new commit(s) since the reviewed commit c0, below the threshold of 3; comment /review to ask for one now\n",
	"malformed state JSON counts by date":                      "run=true\nreason=3 new commit(s) since the review of 2026-09-25T10:00:00Z, at or past the threshold of 3\nmode=incremental\n",
	"an empty head_sha counts by date":                         "run=true\nreason=3 new commit(s) since the review of 2026-09-25T10:00:00Z, at or past the threshold of 3\nmode=incremental\n",
	"a null head_sha counts by date":                           "run=true\nreason=3 new commit(s) since the review of 2026-09-25T10:00:00Z, at or past the threshold of 3\nmode=incremental\n",
	"a last_run that is not an object counts by date":          "run=true\nreason=3 new commit(s) since the review of 2026-09-25T10:00:00Z, at or past the threshold of 3\nmode=incremental\n",
	"keys are case-sensitive":                                  "run=true\nreason=3 new commit(s) since the review of 2026-09-25T10:00:00Z, at or past the threshold of 3\nmode=incremental\n",
	"a comment by a deleted account":                           "run=true\nreason=no previous review, so PR-Agent reviews the whole PR\nmode=full\n",
	"a comment with a null body":                               "run=true\nreason=no previous review, so PR-Agent reviews the whole PR\nmode=full\n",
	"a comments listing of nulls":                              "run=true\nreason=no previous review, so PR-Agent reviews the whole PR\nmode=full\n",
	"comments that are null":                                   "run=true\nreason=the PR comments cannot be read, so the push is reviewed\nmode=full\n",
	"comments that are numbers":                                "run=true\nreason=the PR comments cannot be read, so the push is reviewed\nmode=full\n",
	"commits that are numbers":                                 "run=true\nreason=the PR commits cannot be read, so the push is reviewed\nmode=full\n",
	"commits that are null":                                    "run=true\nreason=the PR commits cannot be read, so the push is reviewed\nmode=full\n",
	"absent commits with a review":                             "run=true\nreason=the PR commits cannot be read, so the push is reviewed\nmode=full\n",
	"absent commits with no review":                            "run=true\nreason=no previous review, so PR-Agent reviews the whole PR\nmode=full\n",
	"a bot push never reads the input":                         "run=false\nreason=a bot pushed, and PR-Agent skips bot pushes (push_trigger_ignore_bot_commits)\n",
	"the sender type is case-sensitive":                        "run=false\nreason=0 new commit(s) since the review of 2026-09-25T10:00:00Z, below the threshold of 3; comment /review to ask for one now\n",
	"a threshold of zero always runs":                          "run=true\nreason=0 new commit(s) since the review of 2026-09-25T10:00:00Z, at or past the threshold of 0\nmode=incremental\n",
}

func TestGate_EdgeCases(t *testing.T) {
	for _, c := range edgeGateCases() {
		t.Run(c.name, func(t *testing.T) {
			want, ok := edgeGateWants[c.name]
			if !ok {
				t.Fatal("edgeGateWants has no entry for this case")
			}
			if got := runGateCase(t, c); got != want {
				t.Errorf("output = %q, want %q", got, want)
			}
		})
	}
}
