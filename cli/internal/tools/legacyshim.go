package tools

import (
	"os"
	"path/filepath"
	"strings"
)

// isLegacyHiveShim reports whether path is the launcher the retired hive
// installer wrote: a hive.cmd that forwards to <...>\hive\runtime\current. It
// answers for hive ahead of the uv-installed copy and nothing else uses it, so
// removing it is the converge's to do; any other file on PATH is the user's.
func isLegacyHiveShim(name, path string) bool {
	if name != "hive" || !strings.EqualFold(filepath.Base(path), "hive.cmd") {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), `hive\runtime\`)
}

// retireLegacyShims removes the recognised legacy launchers among copies and
// reports whether it removed any.
func retireLegacyShims(name string, copies []string) bool {
	removed := false
	for _, c := range copies {
		if isLegacyHiveShim(name, c) && os.Remove(c) == nil {
			removed = true
		}
	}
	return removed
}
