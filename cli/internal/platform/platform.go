// Package platform is the one vocabulary for "which operating systems does this
// declaration apply to" (ADR-045 decision 7). The tool catalog, the converge
// reconcilers and every later manifest share it: absent means every OS, and an
// OS that is not listed is skipped, not failed.
package platform

import "slices"

// known are the GOOS values a declaration may list.
var known = []string{"linux", "darwin", "windows"}

// Supports reports whether a declaration listing platforms applies on goos:
// true when the list is empty, otherwise only for a listed GOOS.
func Supports(platforms []string, goos string) bool {
	return len(platforms) == 0 || slices.Contains(platforms, goos)
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
