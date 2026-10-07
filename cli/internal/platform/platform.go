// Package platform is the one vocabulary for "which operating systems does this
// declaration apply to" (ADR-045 decision 7). The tool catalog, the converge
// reconcilers and every later manifest share it: absent means every OS, and an
// OS that is not listed is skipped, not failed.
package platform

import (
	"slices"
	"strings"
)

// known are the GOOS values a declaration may list, and knownArch the GOARCH
// values a per-architecture key may name.
var (
	known     = []string{"linux", "darwin", "windows"}
	knownArch = []string{"amd64", "arm64"}
)

// Supports reports whether a declaration listing platforms applies on goos:
// true when the list is empty, otherwise only for a listed GOOS.
func Supports(platforms []string, goos string) bool {
	return len(platforms) == 0 || slices.Contains(platforms, goos)
}

// ValidKey reports whether key names a platform as "goos" or "goos/goarch".
func ValidKey(key string) bool {
	goos, goarch, specific := strings.Cut(key, "/")
	if !slices.Contains(known, goos) {
		return false
	}
	return !specific || slices.Contains(knownArch, goarch)
}

// Unknown returns the first listed platform that is not a known GOOS, or "".
func Unknown(platforms []string) string {
	for _, p := range platforms {
		if !slices.Contains(known, p) {
			return p
		}
	}
	return ""
}
