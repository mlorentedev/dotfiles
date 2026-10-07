package harness

import "strings"

// The GENERATED region pair, written by scripts/compile-harness.sh
// (BEGIN_PREFIX/END_MARKER) for the enforced patterns and the copilot skill
// catalog. The AGENT-PRESENCE pair lives in presence.go.
const (
	GeneratedBeginPrefix = "<!-- BEGIN HARNESS GENERATED"
	GeneratedEndMarker   = "<!-- END HARNESS GENERATED -->"
)

// DeployedMatchesSource reports whether a deployed instruction file holds its
// source: the source verbatim, plus regions written only at deploy time (the
// AGENT-PRESENCE roster, the copilot skill catalog). A region is deploy-only
// when its BEGIN line does not appear in the source. A region the source
// carries, such as the enforced-pattern GENERATED region, is compared like any
// other text, so a refreshed source region reads as drift on a stale deployed
// copy. Stripping every region on both sides could not see that (F-064, #2013).
//
// Line endings are normalised first (WIN-008/#1289), the blank line an
// appended region leaves before its BEGIN line is dropped with it, and
// trailing newlines are not content (#1308).
func DeployedMatchesSource(deployed, source string) bool {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	inSource := map[string]bool{}
	for _, l := range strings.Split(source, "\n") {
		if strings.HasPrefix(l, GeneratedBeginPrefix) || strings.HasPrefix(l, PresenceBeginPrefix) {
			inSource[l] = true
		}
	}
	kept := stripRegions(deployed, func(begin string) bool { return !inSource[begin] })
	return strings.TrimRight(kept, "\n") == strings.TrimRight(source, "\n")
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
