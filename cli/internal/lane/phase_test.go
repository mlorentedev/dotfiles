package lane

import (
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

var phaseNow = time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)

func leased(expiresIn time.Duration) *worktree.Metadata {
	return &worktree.Metadata{Issue: 7, CreatedAt: phaseNow.Add(-48 * time.Hour), LeaseExpiresAt: phaseNow.Add(expiresIn), ReapOK: true}
}

// Derive names the phase of one lane from facts alone. One row per phase, plus
// the rows that pin the precedence between them, and herdr absent in several:
// a pane decides nothing (ADR-032), so no fact about herdr may move a phase.
func TestDerive(t *testing.T) {
	cases := []struct {
		name string
		f    Facts
		want Phase
	}{
		{"queued issue with no worktree", Facts{Queued: true}, PhasePending},
		{"merged and the worktree is gone", Facts{PR: PRMerged}, PhaseReaped},
		{"merged and the worktree is still on disk", Facts{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour)}, PR: PRMerged}, PhaseLanded},
		{"merged with a live session is still landed", Facts{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour)}, PR: PRMerged, Live: true}, PhaseLanded},
		{"merged and dirty is still landed", Facts{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour), Dirty: true}, PR: PRMerged}, PhaseLanded},
		{"open PR, first in the landing order", Facts{Exists: true, Info: worktree.Info{Metadata: leased(time.Hour)}, PR: PROpen, Head: true}, PhaseLanding},
		{"open PR, behind another", Facts{Exists: true, Info: worktree.Info{Metadata: leased(time.Hour)}, PR: PROpen}, PhaseReview},
		{"open PR with herdr absent", Facts{Exists: true, Info: worktree.Info{Metadata: leased(time.Hour)}, PR: PROpen, Head: true, HerdrPresent: false}, PhaseLanding},
		{"live session, lease running, no PR", Facts{Exists: true, Info: worktree.Info{Metadata: leased(time.Hour)}, Live: true, HerdrPresent: true}, PhaseRunning},
		{"live session past its lease, no PR", Facts{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour)}, Live: true}, PhaseRunning},
		{"lease running, nobody live, herdr absent", Facts{Exists: true, Info: worktree.Info{Metadata: leased(time.Hour)}}, PhaseRunning},
		{"lease expired, nobody live, no PR", Facts{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour)}}, PhaseStalled},
		{"lease expired, nobody live, PR closed unmerged", Facts{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour)}, PR: PRClosed}, PhaseStalled},
		{"upstream gone and PR not merged", Facts{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour), IsOrphan: true}}, PhaseOrphan},
		{"no lease", Facts{Exists: true, Info: worktree.Info{}}, PhaseUnadopted},
		{"no lease outranks an open PR", Facts{Exists: true, Info: worktree.Info{}, PR: PROpen, Head: true}, PhaseUnadopted},
		{"nothing queued, nothing on disk, nothing merged", Facts{}, PhaseNone},
		{"the main checkout is not a lane", Facts{Exists: true, Info: worktree.Info{IsMain: true, Metadata: leased(time.Hour)}}, PhaseNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Derive(c.f, phaseNow); got != c.want {
				t.Errorf("Derive = %q, want %q", got, c.want)
			}
		})
	}
}

// herdr is a display fact: flipping it never changes a phase.
func TestDerive_HerdrNeverMovesAPhase(t *testing.T) {
	facts := []Facts{
		{Queued: true},
		{PR: PRMerged},
		{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour)}, PR: PRMerged},
		{Exists: true, Info: worktree.Info{Metadata: leased(time.Hour)}, PR: PROpen, Head: true},
		{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour)}},
		{Exists: true, Info: worktree.Info{Metadata: leased(-time.Hour), IsOrphan: true}},
		{Exists: true, Info: worktree.Info{}},
	}
	for _, f := range facts {
		with, without := f, f
		with.HerdrPresent, without.HerdrPresent = true, false
		if a, b := Derive(with, phaseNow), Derive(without, phaseNow); a != b {
			t.Errorf("%+v: herdr present gives %q, absent gives %q", f, a, b)
		}
	}
}
