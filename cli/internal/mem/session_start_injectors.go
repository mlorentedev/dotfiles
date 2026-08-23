package mem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// This file ports the Claude-specific session-start injectors from
// claude-session-start.sh (CLI-025 PR2b). Each injector returns the exact text it
// contributes to the hook's CONTEXT_LINES — including its leading "\n" framing — so
// the assembly (a later slice) is a thin ordered concatenation and every injector's
// byte-equivalence is pinned by its own unit test. An injector with nothing to say
// returns "".

// claudeJSONSize warns when ~/.claude/.claude.json has been truncated below
// threshold bytes (the upstream `claude plugin install` strip bug, SDD-021). Empty
// when the file is absent or a healthy size.
func claudeJSONSize(claudeJSON string, threshold int) string {
	info, err := os.Stat(claudeJSON)
	if err != nil || info.IsDir() {
		return ""
	}
	size := info.Size()
	if size > 0 && size < int64(threshold) {
		return fmt.Sprintf("\n[claude.json] WARNING: ~/.claude/.claude.json is %d bytes (threshold %d). Healthy state is ~75 KB; truncation bug (anthropics/claude-code#59870) reduces it to ~1.5 KB and silently drops subscription state. Recovery: ls -t ~/.claude/backups/.claude.json.backup.* | head -1 && cp <newest-backup> ~/.claude/.claude.json", size, threshold)
	}
	return ""
}

// knowledgeHealth surfaces MEMORY.md size and crystallize-staleness warnings for the
// project's memory file. Empty when the file is absent. Each warning carries its own
// leading newline, matching the shell's incremental CONTEXT_LINES appends.
func knowledgeHealth(memoryFile string, maxLines, staleMaxDays int, now time.Time) string {
	data, err := os.ReadFile(memoryFile)
	if err != nil {
		return ""
	}
	var b strings.Builder

	if lineCount := strings.Count(string(data), "\n"); lineCount > maxLines {
		fmt.Fprintf(&b, "\nMEMORY.md has %d lines (limit: %d) — run /crystallize to trim", lineCount, maxLines)
	}

	lastDate := lastCrystallized(string(data))
	switch lastDate {
	case "":
		b.WriteString("\nKnowledge crystallization never run — run: ./scripts/knowledge-crystallize.sh")
	default:
		if last, perr := time.Parse("2006-01-02", lastDate); perr == nil {
			if days := int(now.Sub(last) / (24 * time.Hour)); days > staleMaxDays {
				fmt.Fprintf(&b, "\nCRYSTALLIZE NEEDED (%d days stale)", days)
			}
		}
	}
	return b.String()
}

// lastCrystallized returns the date after the last "## Last Crystallized:" marker
// (leading whitespace tolerated, BUG-022), or "" when absent.
func lastCrystallized(content string) string {
	const marker = "## Last Crystallized:"
	last := ""
	sc := bufio.NewScanner(strings.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimLeft(sc.Text(), " \t")
		if rest, ok := strings.CutPrefix(line, marker); ok {
			last = strings.TrimPrefix(rest, " ")
		}
	}
	return last
}

// memoryTemperature classifies each non-MEMORY.md file in the project's memory dir
// as HOT/WARM/COLD/ARCHIVE by mtime, and flags when any file is archive-cold. Empty
// when the dir is absent or holds no classifiable files.
func memoryTemperature(memoryDir string, hotDays, warmDays, coldDays int, now time.Time) string {
	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "MEMORY.md" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // the shell's glob expands in sorted order

	var report, candidates strings.Builder
	archivable := 0
	nowEpoch := now.Unix()
	for _, name := range names {
		path := filepath.Join(memoryDir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		daysAgo := int((nowEpoch - info.ModTime().Unix()) / 86400)
		label := temperatureLabel(daysAgo, hotDays, warmDays, coldDays)
		if label != "ARCHIVE" {
			fmt.Fprintf(&report, "\n  %s: %s (%dd ago)", label, name, daysAgo)
			continue
		}
		typ := memoryType(path)
		if !archivableTypes[typ] {
			if typ == "" {
				typ = "unknown"
			}
			fmt.Fprintf(&report, "\n  STANDING: %s (%dd ago, type=%s — not archivable on age)", name, daysAgo, typ)
			continue
		}
		archivable++
		fmt.Fprintf(&report, "\n  ARCHIVE: %s (%dd ago, type=%s)", name, daysAgo, typ)
		fmt.Fprintf(&candidates, "\n  - %s (%dd, type=%s)", name, daysAgo, typ)
	}

	if report.Len() == 0 {
		return ""
	}
	out := "\nMemory temperature:" + report.String()
	if archivable > 0 {
		out += fmt.Sprintf(
			"\nARCHIVE NEEDED (%d): move to memory/archive/ and drop each file's MEMORY.md pointer in the same edit%s",
			archivable, candidates.String())
	}
	return out
}

// archivableTypes are the memory kinds an age sweep may retire, and the list is
// deliberately short (HARNESS-073, #967).
//
// mtime records when a memory was last EDITED, never when it was last relied on,
// so a standing guardrail untouched for ninety days is the most settled entry in
// the directory rather than the most disposable one. On 2026-08-14 the age sweep
// proposed four `feedback` memories at once — the no-AI-attribution rule, the
// worktree rule, the incident-to-guard rule and the two-tier deploy lesson — all
// four load-bearing, all four followed that same session.
//
// A missing or unrecognised type is exempt too, and that is the fail-closed half:
// archiving a memory also drops its pointer from MEMORY.md, the only surface
// loaded at session start, so the rule does not merely move — it stops being seen.
// The unknown case must therefore degrade to "kept", never to "moved".
var archivableTypes = map[string]bool{"project": true, "reference": true}

// memoryType returns the `metadata.type` of an auto-memory file, or "" when the
// file carries no frontmatter or no such key.
//
// The key is matched exactly after trimming, so the sibling `node_type:` present
// in every one of these files never satisfies it — a substring match would report
// every memory as type "memory" and re-open the defect from the other side.
func memoryType(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return ""
		}
		if key, val, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(key) == "type" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

// temperatureLabel buckets an age in days against the HOT/WARM/COLD thresholds.
func temperatureLabel(daysAgo, hotDays, warmDays, coldDays int) string {
	switch {
	case daysAgo <= hotDays:
		return "HOT"
	case daysAgo <= warmDays:
		return "WARM"
	case daysAgo <= coldDays:
		return "COLD"
	default:
		return "ARCHIVE"
	}
}

// triageQueue renders the PR-triage checkpoint.
//
// GUARD-002 made a green check mean "a review happened". It says nothing about
// whether anyone acted on the review, and nothing pushes that fact into an agent
// session: a workflow_run re-evaluates the gate and GitHub notifies the human,
// but no channel reaches here. So the loop closes at the one moment every session
// has in common, which is this brief — determinism by code, not by an agent
// remembering to ask.
//
// `summary` is the compact list of pending PRs; empty means the queue is clear.
// An error is REPORTED, never swallowed. `dotf pr triage-queue` exits non-zero
// when it cannot answer precisely so that a queue which could not be computed
// never reads as an empty one, and that rule has to survive the trip into here —
// silence on failure would recreate exactly the blind spot the queue exists to
// remove.
func triageQueue(summary string, err error) string {
	if err != nil {
		return "\n[pr-triage] queue could not be computed: " + err.Error() +
			"\n  This is not an empty queue. Run `dotf pr triage-queue` to see why."
	}
	if strings.TrimSpace(summary) == "" {
		return ""
	}
	return "\n[pr-triage] awaiting a disposition: " + summary +
		"\n  Dispose with /pr-review-triage. Nothing counts as triaged until its" +
		"\n  table lands on the PR under the registry's triage marker."
}
