//go:build !windows

package orca

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// The lock Chromium's process singleton keeps in userData decides whether Orca
// runs. Each case writes the lock the way Chromium does, a symlink to
// "<host>-<pid>", and never starts Orca.
func TestRunning_ReadsTheSingletonLock(t *testing.T) {
	reaped := exec.Command("true")
	if err := reaped.Run(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		target string // "" means no lock; "file" means a regular file
		want   bool
	}{
		{"no lock", "", false},
		{"live pid", "macmini-" + strconv.Itoa(os.Getpid()), true},
		{"hyphenated host, dead pid", "mac-mini.local-" + strconv.Itoa(reaped.Process.Pid), false},
		{"another user's pid", "macmini-1", true},
		{"dead pid", "macmini-" + strconv.Itoa(reaped.Process.Pid), false},
		{"unparseable target", "macmini-", true},
		{"not a symlink", "file", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			lock := filepath.Join(dir, "SingletonLock")
			switch tc.target {
			case "":
			case "file":
				if err := os.WriteFile(lock, nil, 0o600); err != nil {
					t.Fatal(err)
				}
			default:
				if err := os.Symlink(tc.target, lock); err != nil {
					t.Fatal(err)
				}
			}
			if got := RunningIn(dir)(); got != tc.want {
				t.Errorf("running with lock %q = %v, want %v", tc.target, got, tc.want)
			}
		})
	}
}
