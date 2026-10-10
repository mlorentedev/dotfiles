//go:build windows

package orca

import (
	"os/exec"
	"strings"
)

// running asks tasklist for orca.exe. Chromium's Windows singleton is a named
// mutex, not a file in userData, so the image name is what there is to check.
func running(string) bool {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq orca.exe", "/NH").Output()
	return err == nil && strings.Contains(string(out), "orca.exe")
}
