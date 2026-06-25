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

	var report strings.Builder
	hasArchive := false
	nowEpoch := now.Unix()
	for _, name := range names {
		info, err := os.Stat(filepath.Join(memoryDir, name))
		if err != nil {
			continue
		}
		daysAgo := int((nowEpoch - info.ModTime().Unix()) / 86400)
		label := temperatureLabel(daysAgo, hotDays, warmDays, coldDays)
		if label == "ARCHIVE" {
			hasArchive = true
		}
		fmt.Fprintf(&report, "\n  %s: %s (%dd ago)", label, name, daysAgo)
	}

	if report.Len() == 0 {
		return ""
	}
	out := "\nMemory temperature:" + report.String()
	if hasArchive {
		out += "\nARCHIVE NEEDED: Move memory files >60d old to memory/archive/ and update MEMORY.md index"
	}
	return out
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
