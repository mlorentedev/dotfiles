package harness

import "strings"

// The GENERATED region pair, written by scripts/compile-harness.sh
// (BEGIN_PREFIX/END_MARKER) for the enforced patterns and the copilot skill
// catalog. The AGENT-PRESENCE pair lives in presence.go.
const (
	GeneratedBeginPrefix = "<!-- BEGIN HARNESS GENERATED"
	GeneratedEndMarker   = "<!-- END HARNESS GENERATED -->"
)

// CatalogMarker identifies the copilot skill-catalog region among the GENERATED
// ones: compile-harness.sh writes it into the BEGIN line ("— skill catalog from
// vault …"). TestCatalogMarkerMatchesCompileHarness pins the two together.
const CatalogMarker = "skill catalog"

// catalogSlot is the empty BEGIN line a source carries where the catalog is
// injected at deploy time (ai/copilot/copilot-instructions.md).
const catalogSlot = GeneratedBeginPrefix + " -->"

// deployOnly reports whether a region with this BEGIN line belongs to the
// deploy, not to the source: the persona presence roster, the copilot skill
// catalog, and the empty slot a source reserves for that catalog. Every other
// region (the enforced-pattern GENERATED region, with its sha and provenance)
// is part of the source and compared like its text.
func deployOnly(begin string) bool {
	switch {
	case strings.HasPrefix(begin, PresenceBeginPrefix):
		return true
	case strings.HasPrefix(begin, GeneratedBeginPrefix):
		return strings.Contains(begin, CatalogMarker) || strings.TrimSpace(begin) == catalogSlot
	}
	return false
}

// DeployedMatchesSource reports whether a deployed instruction file holds its
// source, ignoring the deploy-only regions on both sides. A region the source
// carries, such as the enforced-pattern GENERATED region, is compared like any
// other text, so a refreshed source region reads as drift on a stale deployed
// copy; stripping every region on both sides could not see that (F-064, #2013).
//
// Line endings are normalised first (WIN-008/#1289), the blank line before a
// stripped region goes with it, and trailing newlines are not content (#1308).
func DeployedMatchesSource(deployed, source string) bool {
	d := strings.TrimRight(stripRegions(deployed, deployOnly), "\n")
	s := strings.TrimRight(stripRegions(source, deployOnly), "\n")
	return d == s
}

// stripRegions removes the harness marker regions whose BEGIN line drop
// accepts, with the blank separator line before each.
func stripRegions(content string, drop func(begin string) bool) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	endMarker := ""
	for _, l := range lines {
		if endMarker != "" {
			if l == endMarker {
				endMarker = ""
			}
			continue
		}
		end := regionEnd(l)
		if end == "" || !drop(l) {
			out = append(out, l)
			continue
		}
		if n := len(out); n > 0 && out[n-1] == "" {
			out = out[:n-1]
		}
		endMarker = end
	}
	return strings.Join(out, "\n")
}

// regionEnd returns the END marker closing a region whose BEGIN line is l, or
// "" when l opens no region.
func regionEnd(l string) string {
	switch {
	case strings.HasPrefix(l, GeneratedBeginPrefix):
		return GeneratedEndMarker
	case strings.HasPrefix(l, PresenceBeginPrefix):
		return PresenceEndMarker
	}
	return ""
}
