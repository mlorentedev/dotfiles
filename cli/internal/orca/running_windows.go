//go:build windows

package orca

import (
	"os/exec"
	"strings"
)

// running asks tasklist for orca.exe. Chromium's Windows singleton is a named
// mutex plus a "lockfile" held open in userData, not a symlink naming a pid, so
// reading it means an open attempt; until that is measured on a Windows box,
// the image name is the check.
func running(string) bool {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq orca.exe", "/NH").Output()
	return err == nil && strings.Contains(string(out), "orca.exe")
}
