//go:build !linux

package worktree

import "testing"

// This file exists on the platforms the previous code was broken on, and it is
// the only place the unsupported branch can be asserted against the real
// implementation rather than a seam. It runs on the `test (windows-latest)`
// leg, which is where the original defect was reachable and nowhere else.
//
// The Linux tests in sweep_proc_test.go drive the seam and prove the CALLER
// refuses; these prove the PLATFORM answer that feeds it. Neither one alone is
// enough: a seam test passes on a platform whose implementation is wrong, and
// this test cannot see whether the caller honours the answer.

func TestUnsupportedPlatformAnswersTrueForEveryPath(t *testing.T) {
	for _, path := range []string{"", ".", `C:\Users\someone\Projects\repo-wt-feat`, "/tmp/repo-wt-feat"} {
		reading := isHostProcessInside(path)
		if !reading.Inside {
			t.Errorf("isHostProcessInside(%q).Inside = false on a platform with no process "+
				"discovery; the caller deletes the worktree on a false, so the only safe "+
				"answer is true", path)
		}
		// Nothing was inspected, so a non-zero count would claim a scan that
		// never ran. The absence is carried by processDiscoverySupported.
		if reading.Uninspectable != 0 {
			t.Errorf("isHostProcessInside(%q).Uninspectable = %d, want 0: no scan happened here",
				path, reading.Uninspectable)
		}
	}
}

func TestUnsupportedPlatformDoesNotAdvertiseDiscovery(t *testing.T) {
	if processDiscoverySupported {
		t.Error("processDiscoverySupported is true on a platform with no implementation; " +
			"sweep would report a liveness check it never performed")
	}
}
