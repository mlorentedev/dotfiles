package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// isLegacyHiveShim reports whether path is the launcher the retired hive
// installer wrote: <...>\hive\bin\hive.cmd holding exactly `@echo off` and a
// line forwarding to <...>\hive\runtime\current\Scripts\hive.exe. It answers
// for hive ahead of the uv-installed copy and nothing else uses it; any other
// file on PATH is the user's and is never touched.
func isLegacyHiveShim(name, path string) bool {
	if name != "hive" || !strings.EqualFold(filepath.Base(path), "hive.cmd") {
		return false
	}
	bin := filepath.Dir(path)
	if !strings.EqualFold(filepath.Base(bin), "bin") || !strings.EqualFold(filepath.Base(filepath.Dir(bin)), "hive") {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var lines []string
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, strings.ToLower(l))
		}
	}
	return len(lines) == 2 && lines[0] == "@echo off" &&
		strings.HasPrefix(lines[1], `"`) &&
		strings.HasSuffix(lines[1], `\hive\runtime\current\scripts\hive.exe" %*`)
}

// retiredShim is a legacy launcher moved aside, not yet deleted.
type retiredShim struct{ path, backup string }

// retireLegacyShims moves aside the recognised legacy launchers that sit ahead
// of the first copy that is not one: those shadow the install, anything after
// it is inert. They are moved, not deleted, so a failed re-probe can put them
// back. A failure to move one restores the rest and is returned.
func retireLegacyShims(name string, copies []string) ([]retiredShim, error) {
	var moved []retiredShim
	for _, c := range copies {
		if !isLegacyHiveShim(name, c) {
			break
		}
		s := retiredShim{path: c, backup: c + ".retired"}
		if err := os.Rename(c, s.backup); err != nil {
			restoreShims(moved)
			return nil, fmt.Errorf("could not remove the legacy %s launcher %s (close anything using it): %w", name, c, err)
		}
		moved = append(moved, s)
	}
	return moved, nil
}

func restoreShims(moved []retiredShim) {
	for _, s := range moved {
		_ = os.Rename(s.backup, s.path)
	}
}

func dropShims(moved []retiredShim) {
	for _, s := range moved {
		_ = os.Remove(s.backup)
	}
}
