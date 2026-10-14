//go:build !windows

package secrets

import "testing"

// /dev is its own filesystem on macOS (devfs) and Linux (devtmpfs), so it is
// the mount point every Unix CI runner has: the positive half of the check.
func TestIsMountPoint_DevIs(t *testing.T) {
	mounted, err := isMountPoint("/dev")
	if err != nil || !mounted {
		t.Fatalf("isMountPoint(/dev) = %v, %v", mounted, err)
	}
}
