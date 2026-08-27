// Package mem implements the `dotf mem` domain — the session-start/end hook
// cluster ported from the twin shell scripts (CLI-025). It folds the drifting
// cross-OS twins into one Go noun so the SessionStart/SessionEnd hooks shrink to
// thin shims.
//
// session-end is the first piece (PR1): it archives the /handoff block from a
// project's MEMORY.md into a durable cross-session record (MEMORY-001 / ADR-014),
// converging session-handoff.{sh,ps1}.
package mem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sessionEndPayload is the slice of the SessionEnd hook JSON we consume.
type sessionEndPayload struct {
	Cwd       string `json:"cwd"`
	SessionID string `json:"session_id"`
}

// SessionEnd archives the "## Session Handoff" block that /handoff wrote into the
// project's MEMORY.md (under vaultPath) into an append-only, timestamped record at
//
//	<vaultPath>/10_projects/<project>/sessions/<date>-<project>-claude.md
//
// It is the Go port of session-handoff.{sh,ps1} (MEMORY-001, ADR-014). The agent
// authors the handoff (via /handoff, with reasoning); this only persists it.
//
// Resilience contract: a SessionEnd hook must NEVER crash a session. Every
// trivial / missing / malformed input is a clean no-op — it returns ("", nil) and
// writes nothing. A non-nil error surfaces only for an unexpected write failure
// once the record is known to be wanted; the command wrapper swallows even that to
// exit 0.
//
// vaultPath is injected (the caller resolves it via vault.ResolveVault, which runs
// the ADR-025 cascade and retires the old hardcoded ~/Projects/knowledge literal,
// #463); "" means no vault → no-op. now is injected for deterministic stamping.
func SessionEnd(payload []byte, vaultPath string, now time.Time) (string, error) {
	if len(payload) == 0 || vaultPath == "" {
		return "", nil
	}

	var p sessionEndPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return "", nil // malformed JSON -> no-op
	}
	if p.Cwd == "" {
		return "", nil
	}
	project := filepath.Base(p.Cwd)

	memory := filepath.Join(vaultPath, "10_projects", project, "memory", "MEMORY.md")
	content, err := os.ReadFile(memory)
	if err != nil {
		return "", nil // absent / unreadable MEMORY.md -> no-op
	}

	block := extractHandoffBlock(string(content))
	if strings.TrimSpace(block) == "" {
		return "", nil // trivial session -> no-op
	}

	sid := p.SessionID
	if sid == "" {
		sid = "unknown"
	}
	date := now.Format("2006-01-02")

	outDir := filepath.Join(vaultPath, "10_projects", project, "sessions")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	// PER WORKTREE, and through the SAME namer the skill uses.
	//
	// This was `<date>-<project>-claude.md`: hardcoded, thread-less, and written
	// with a TRUNCATING os.WriteFile. With several worktrees running — five on
	// this machine — every SessionEnd wrote the same path and the last one
	// destroyed the others' durable records. That is HARNESS-088's defect in the
	// path that was meant to be the safe copy.
	//
	// It also assembled its own filename while `dotf mem thread` assembled a
	// different one, so the hook's archive and the skill's journal would drift
	// into two files per session. Sharing JournalName removes the second
	// convention rather than documenting it.
	out := filepath.Join(outDir, JournalName(date, project, "claude", ThreadKey(p.Cwd)))
	if err := os.WriteFile(out, []byte(buildRecord(date, project, sid, block)), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

// extractHandoffBlock returns the lines after a "## Session Handoff" heading up to
// the next "## " heading or EOF — the awk/foreach logic of the shell twins.
func extractHandoffBlock(content string) string {
	var b strings.Builder
	capturing := false
	for line := range strings.SplitSeq(content, "\n") {
		if !capturing {
			if isHandoffHeading(line) {
				capturing = true
			}
			continue
		}
		if strings.HasPrefix(line, "## ") {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// isHandoffHeading matches "## Session Handoff" with optional trailing whitespace,
// mirroring the shell's /^## Session Handoff[[:space:]]*$/.
func isHandoffHeading(line string) bool {
	return strings.TrimRight(line, " \t\r") == "## Session Handoff"
}

// buildRecord renders the durable session record (frontmatter + heading + block).
// This is the single converged form of the two drifting shell twins: the .sh used
// an em-dash in the heading, the .ps1 a hyphen — the em-dash wins.
//
// The frontmatter satisfies the vault Frontmatter Law (id, type, status, created,
// owner) with id/type/status as the first three keys, per handoff/SKILL.md §1b —
// vault_health and vault-validate.py §1 both report a missing field as an error.
func buildRecord(date, project, sid, block string) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: \"session-%s-%s-claude\"\n", date, project)
	b.WriteString("type: session\n")
	b.WriteString("status: active\n")
	fmt.Fprintf(&b, "created: \"%s\"\n", date)
	b.WriteString("owner: manu\n")
	fmt.Fprintf(&b, "session_id: %s\n", sid)
	b.WriteString("agent: claude\n")
	fmt.Fprintf(&b, "project: %s\n", project)
	fmt.Fprintf(&b, "date: \"%s\"\n", date)
	fmt.Fprintf(&b, "tags: [session, handoff, %s]\n", project)
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# Session %s — %s (claude)\n\n", date, project)
	b.WriteString(block)
	b.WriteByte('\n')
	return b.String()
}
