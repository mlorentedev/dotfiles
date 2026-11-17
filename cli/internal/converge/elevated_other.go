//go:build !windows

package converge

import "os"

// processElevated reports whether dotf runs as root.
func processElevated() bool {
	return os.Geteuid() == 0
}
