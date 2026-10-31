package secrets

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A systemd unit that `dotf secrets run --skip-locked` started without its
// secrets, because Bitwarden was locked, leaves one file here named after the
// unit and holding the time it started. `dotf secrets unlock` restarts each
// such unit so it picks its secrets up, and `dotf doctor` reports the units
// still waiting. The directory is under XDG_RUNTIME_DIR, so a reboot clears it
// along with the units it describes (dotfiles#3005).

// DegradedDir is where the traces live, or "" when getenv yields no
// XDG_RUNTIME_DIR (no user session, so no unit to restart either).
func DegradedDir(getenv func(string) string) string {
	rt := getenv("XDG_RUNTIME_DIR")
	if rt == "" {
		return ""
	}
	return filepath.Join(rt, "dotf", "skipped-secrets")
}

// UnitFromCgroup returns the systemd service a process belongs to, read from
// its /proc/<pid>/cgroup content, or "" when it runs outside a service (a
// shell, a scope, a test). The service is the last path segment, which ends
// in ".service": "0::/user.slice/.../app.slice/hive.service".
func UnitFromCgroup(cgroup string) string {
	for _, line := range strings.Split(strings.TrimSpace(cgroup), "\n") {
		path := line[strings.LastIndex(line, ":")+1:]
		unit := path[strings.LastIndex(path, "/")+1:]
		if strings.HasSuffix(unit, ".service") {
			return unit
		}
	}
	return ""
}

// MarkDegraded records that unit started without its secrets at now.
func MarkDegraded(dir, unit string, now time.Time) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, unit), []byte(now.UTC().Format(time.RFC3339)+"\n"), 0o600)
}

// ClearDegraded removes unit's trace. A missing trace is not an error.
func ClearDegraded(dir, unit string) error {
	err := os.Remove(filepath.Join(dir, unit))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// DegradedUnits maps each unit with a trace to the time it recorded. A missing
// directory reads as none.
func DegradedUnits(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	units := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".service") {
			continue
		}
		since, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		units[e.Name()] = strings.TrimSpace(string(since))
	}
	return units, nil
}
