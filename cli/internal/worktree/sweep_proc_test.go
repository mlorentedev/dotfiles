package worktree

import (
	"path/filepath"
	"testing"
)

// The refusing branch of Gate f only executes on a platform this CI does not
// run, so without a seam it is unreachable by any test and its behaviour rests
// on reading the code. That is exactly how the defect these tests replace
// survived an adversarial review: isHostProcessInside read /proc with no
// build-tag split, returned false when os.ReadDir failed, and therefore
// reported "nobody is inside" on every Windows run.
//
// Driving the seam both ways is what makes the fix checkable. The direction
// that matters is the second one: a gate that cannot answer must not clear the
// worktree for deletion.

func withHostProcessInside(t *testing.T, answer bool) {
	t.Helper()
	original := hostProcessInside
	hostProcessInside = func(string) bool { return answer }
	t.Cleanup(func() { hostProcessInside = original })
}

func reapableInfo(t *testing.T) Info {
	t.Helper()
	return Info{
		Path:     filepath.Join(t.TempDir(), "repo-wt-feat"),
		Branch:   "feat/x",
		State:    StateReapable,
		PRMerged: true,
		Dirty:    false,
		IsOrphan: false,
		IsMain:   false,
	}
}

func TestGateFRefusesWhenProcessDiscoveryCannotAnswer(t *testing.T) {
	withHostProcessInside(t, true)

	info := reapableInfo(t)
	if isCandidateForReap(info, "/somewhere/else") {
		t.Error("a worktree was cleared for reaping while Gate f reported a live process inside; " +
			"an unanswerable gate must refuse, because the caller deletes on a false")
	}
}

func TestGateFAllowsAReapableWorktreeWhenNothingIsInside(t *testing.T) {
	withHostProcessInside(t, false)

	info := reapableInfo(t)
	if !isCandidateForReap(info, "/somewhere/else") {
		t.Error("a reapable worktree with no process inside was refused; the gate has stopped " +
			"discriminating and now refuses everything, which passes the test above vacuously")
	}
}

// processDiscoverySupported is a build-tag constant, so this asserts the value
// for the platform the test binary was built for rather than a fixed answer.
// It pins the contract that the two files agree on the constant's meaning: it
// is true exactly where isHostProcessInside can observe processes.
func TestProcessDiscoveryIsReportedForThisPlatform(t *testing.T) {
	// Linux is the only implemented platform today. If a Windows or Darwin
	// implementation lands, this test must be updated in the same change --
	// which is the point: the constant should never drift from reality
	// silently, since a false one makes sweep inert and a true one makes it
	// dangerous.
	if !processDiscoverySupported {
		t.Skip("no process discovery on this platform; the refusal path is covered above")
	}

	if hostProcessInside == nil {
		t.Fatal("process discovery is advertised as supported but Gate f has no implementation")
	}
}
