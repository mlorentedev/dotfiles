package prreview

import (
	"path/filepath"
)

// repoFile is a path under the repository root, from this package's directory.
func repoFile(parts ...string) string {
	return filepath.Join(append([]string{"..", "..", ".."}, parts...)...)
}
