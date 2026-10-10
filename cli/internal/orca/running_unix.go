//go:build !windows

package orca

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// running reads the lock Chromium's process singleton holds in userData while
// the app runs: a symlink named SingletonLock whose target is "<host>-<pid>".
// Orca takes it (app.requestSingleInstanceLock), so the check needs neither
// the binary's name, which differs per OS and package (AppImage, .deb, .app),
// nor a process listing.
//
// The host half is ignored: macOS hostnames drift (macmini, macmini.local, a
// network-assigned name), and matching it would let a stale lock from an old
// name block forever. A pid reused by another process reads as running, which
// refuses the write: the safe direction. Anything unreadable short of "no lock"
// also reads as running.
func running(userDataDir string) bool {
	target, err := os.Readlink(filepath.Join(userDataDir, "SingletonLock"))
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		return true
	}
	pid, err := strconv.Atoi(target[strings.LastIndex(target, "-")+1:])
	if err != nil || pid <= 0 {
		return true
	}
	err = syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
