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

// regionKind says who writes a harness region.
type regionKind int

const (
	// sourceRegion is part of the source, such as the enforced-pattern
	// GENERATED region with its sha and provenance.
	sourceRegion regionKind = iota
	// presenceRegion is the persona roster `dotf harness presence` appends.
	presenceRegion
	// catalogRegion is the copilot skill catalog compile-harness.sh writes.
	catalogRegion
	// catalogSlotRegion is the empty region a source reserves for the catalog.
	catalogSlotRegion
)

func kindOf(begin string) regionKind {
	switch {
	case strings.HasPrefix(begin, PresenceBeginPrefix):
		return presenceRegion
	case strings.HasPrefix(begin, GeneratedBeginPrefix) && strings.Contains(begin, CatalogMarker):
		return catalogRegion
	case strings.TrimSpace(begin) == catalogSlot:
		return catalogSlotRegion
	}
	return sourceRegion
}

// deployOnly reports whether a region with this BEGIN line belongs to the
// deploy, not to the source: the presence roster, the skill catalog and the
// empty slot a source reserves for it. Every other region is part of the
// source and compared like its text.
func deployOnly(begin string) bool {
	return kindOf(begin) != sourceRegion
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
	return mapRegions(content, func(region []string) []string {
		if drop(region[0]) {
			return nil
		}
		return region
	})
}

// mapRegions passes each harness region of content, BEGIN and END lines
// included, through f and puts back the lines f returns; nil removes the region
// with the blank separator line before it. It is the one scanner for the
// markers. Line endings are normalised first; an unterminated region runs to
// the end.
func mapRegions(content string, f func(region []string) []string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	var out, region []string
	endMarker := ""
	flush := func() {
		lines := f(region)
		if n := len(out); lines == nil && n > 0 && out[n-1] == "" {
			out = out[:n-1]
		}
		out = append(out, lines...)
		region, endMarker = nil, ""
	}
	for _, l := range strings.Split(content, "\n") {
		if endMarker != "" {
			region = append(region, l)
			if l == endMarker {
				flush()
			}
			continue
		}
		if end := regionEnd(l); end != "" {
			region, endMarker = []string{l}, end
			continue
		}
		out = append(out, l)
	}
	if endMarker != "" {
		flush()
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
