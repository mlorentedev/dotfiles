package harness

import "strings"

// The GENERATED region pair, written by scripts/compile-harness.sh
// (BEGIN_PREFIX/END_MARKER) for the enforced patterns and the copilot skill
// catalog. The AGENT-PRESENCE pair lives in presence.go.
const (
	GeneratedBeginPrefix = "<!-- BEGIN HARNESS GENERATED"
	GeneratedEndMarker   = "<!-- END HARNESS GENERATED -->"
)

// StripRegions removes every harness-managed marker region (both the
// GENERATED and AGENT-PRESENCE kinds) from content, mirroring
// compile-harness.sh's region_content in reverse (strip instead of extract).
//
// Also drops the single blank line immediately preceding a BEGIN marker:
// both inject_agent_presence and replace_region's append branch write a
// region as "\n" + BEGIN + body + END + "\n" (compile-harness.sh), so an
// appended region always leaves that blank separator behind in the deployed
// file with nothing to match it in the un-appended repo source. Without
// dropping it, doctor's checkInstructionDrift reported drift on every target
// immediately after a clean --deploy — caught in review before merge.
//
// Line endings are normalised first. A deployed copy written by a Windows
// tool arrives CRLF while the repo source is LF (`.gitattributes` `*.md
// eol=lf`), and with a bare "\n" split every line kept a trailing "\r": the
// END marker never matched, so the strip swallowed the rest of the file, and
// every other line differed anyway — a drift FAIL no redeploy could clear
// (WIN-008/#1289). EOL is not content; the writer is fixed separately, and
// this keeps the comparator honest about the next writer that is not.
func StripRegions(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	skip, endMarker := false, ""
	dropPrecedingBlank := func() {
		if n := len(out); n > 0 && out[n-1] == "" {
			out = out[:n-1]
		}
	}
	for _, l := range lines {
		if skip {
			if l == endMarker {
				skip = false
			}
			continue
		}
		switch {
		case strings.HasPrefix(l, GeneratedBeginPrefix):
			dropPrecedingBlank()
			skip, endMarker = true, GeneratedEndMarker
		case strings.HasPrefix(l, PresenceBeginPrefix):
			dropPrecedingBlank()
			skip, endMarker = true, PresenceEndMarker
		default:
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
