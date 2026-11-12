//go:build !windows

package diskfree

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func free(path string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, fmt.Errorf("statfs %s: %w", path, err)
	}
	return st.Bavail * uint64(st.Bsize), nil //nolint:gosec // a block size is never negative
}
