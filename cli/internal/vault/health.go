package vault

// health.go — the Go port of scripts/vault-health.sh (CLI-021 / #490, increment
// 2). Built beside the shell twin, exactly like crystallize.go was in increment
// 1. CLI-023 (#492) cut every caller over and deleted the script, so this is the
// only implementation.
//
// Two seams increment 1 did not have, both called out in the spec
// (specs/CLI-021-dotf-vault-build-knowledge/tasks.md §3):
//
//  1. An EXTERNAL BINARY contract. Four of seven sections shell out to
//     `obsidian`, which talks to a running GUI over IPC. The golden corpus
//     (tests/golden/vault-health/) stubs that binary on PATH and pins the exact
//     argv this code sends it — not just stdout — because a port could drift in
//     HOW it calls obsidian while stdout stayed byte-identical.
//  2. The Backlog Integrity section's two checks, check-backlog-integrity.sh
//     and check-backlog-merged.sh (SDD-012 / SDD-012b), were exec'd through
//     bash at first. They are ported in backlog.go (#492), so this file runs
//     no shell at all.
//
// The shell was the oracle: every observable byte is pinned by the golden corpus
// and reproduced faithfully, including its VAULT_DIR/VAULT_PATH/default
// fallback (a plain env-var cascade, NOT the ADR-025 machine.json cascade
// ResolveVault() uses elsewhere — vault-health.sh predated that cascade, and
// matching the oracle takes priority over "improving" it here).
//
// One deliberate departure, made once the session banner ran this port rather
// than the script: sections 3 and 4 no longer count what cannot be fixed by
// design (#1979). Orphans exclude session journals, agent memory and the
// archive, from the list and the population alike; unresolved links exclude
// those whose every source is a template placeholder or an archived note.
// Dead-ends exclude journals, templates, course notes and the archive, and both
// link-graph counts set attachments apart (#2197); linkExemptions holds the
// zones. The goldens for those sections are the Go contract from there on.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
)

// HealthOptions configures a `vault health` run. VaultDir/VaultName are already
// resolved by the caller (the shell's own env-var cascade lives in the cmd
// layer, mirroring how crystallize's path resolution lives there too).
type HealthOptions struct {
	VaultDir  string
	VaultName string
	Verbose   bool
	// GOOS decides the argv sent to obsidian; empty means runtime.GOOS. A seam
	// so the golden corpus pins one platform's argv on every CI leg.
	GOOS string
}

// deletedLineRe mirrors `grep '^.D '`: git status --short's Y-column (unstaged
// worktree state) is D — a file removed from disk but still tracked in HEAD.
// The X-column (staged state, an intentional `git rm`) is deliberately NOT
// matched here — that is what the leading `.` skips.
var deletedLineRe = regexp.MustCompile(`^.D `)

// healthRun carries the running counters and options across the 7 sections, the
// same shape the shell's global CHECKS_PASSED/FAILED/SKIPPED play.
type healthRun struct {
	w       io.Writer
	opts    HealthOptions
	passed  int
	failed  int
	skipped int

	// populated by section 2, consumed by sections 3 and 5.
	mdFiles    []string
	totalFiles int
}

func (h *healthRun) pass(format string, a ...any) {
	emit(h.w, "  PASS: "+format+"\n", a...)
	h.passed++
}

func (h *healthRun) fail(format string, a ...any) {
	emit(h.w, "  FAIL: "+format+"\n", a...)
	h.failed++
}

// warn counts toward "passed", mirroring the shell's warn(): a WARN is not a
// FAIL. Reproduced as-is rather than "corrected" — it is the oracle's contract.
func (h *healthRun) warn(format string, a ...any) {
	emit(h.w, "  WARN: "+format+"\n", a...)
	h.passed++
}

func (h *healthRun) skip(title, reason string) {
	emit(h.w, "  SKIP: %s - %s\n", title, reason)
	h.skipped++
}

func (h *healthRun) info(format string, a ...any) {
	emit(h.w, "  INFO: "+format+"\n", a...)
}

func (h *healthRun) section(n, title string) {
	emit(h.w, "\n[%s] %s\n", n, title)
}

// errorLine/infoLine mirror utils.sh's log_error/log_info — top-level, no
// leading two-space indent, distinct from the section-local info() above.
func (h *healthRun) errorLine(format string, a ...any) {
	emit(h.w, "[ERROR] "+format+"\n", a...)
}

func (h *healthRun) infoLine(format string, a ...any) {
	emit(h.w, "[INFO] "+format+"\n", a...)
}

// pct mirrors the shell's integer `count * 100 / total`, with the same
// zero-total guard as the two `if [ "$TOTAL_FILES" -gt 0 ]` sites.
func pct(count, total int) int {
	if total <= 0 {
		return 0
	}
	return count * 100 / total
}

// hasAnyChar mirrors `grep -q '.'`: true when at least one line has length > 0
// (a line containing only whitespace still matches — '.' matches any byte).
func hasAnyChar(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		if len(line) > 0 {
			return true
		}
	}
	return false
}

// countNonBlank mirrors `grep -c '[^[:space:]]'`: the count of lines holding at
// least one NON-whitespace character — deliberately different from hasAnyChar.
func countNonBlank(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

// namesVault reports whether the obsidian CLI's `vault` answer is the vault's
// own record: a `name<TAB><vault>` line, as the real CLI prints it. The CLI
// exits 0 and prints to stdout on its own errors too ("Vault not found.",
// `Error: Command "--no-sandbox" not found.`), so neither output nor a mention
// of the name is a connection.
func namesVault(out, name string) bool {
	for _, line := range strings.Split(out, "\n") {
		if key, value, ok := strings.Cut(line, "\t"); ok && key == "name" && strings.TrimSpace(value) == name {
			return true
		}
	}
	return false
}

// obsidianArgs builds `[--no-sandbox] vault=<name> <sub...>`. `vault=<name>`
// first is the CLI's documented form (https://obsidian.md/help/cli);
// `--vault <name>` is not a parameter, and the CLI ignored it and answered for
// whichever vault was active, or "Vault not found." when none was.
// `--no-sandbox` is for the Linux AppImage's Electron sandbox, which every
// non-darwin call carried before; the macOS binary is a separate CLI
// (obsidian-cli) that reads it as a command and fails every call (measured on
// obsidian 1.14.4).
func (h *healthRun) obsidianArgs(sub ...string) []string {
	goos := h.opts.GOOS
	if goos == "" {
		goos = runtime.GOOS //nolint:forbidigo // the seam's production default
	}
	var args []string
	if goos != "darwin" {
		args = append(args, "--no-sandbox")
	}
	return append(append(args, "vault="+h.opts.VaultName), sub...)
}

// obsidianCmd runs `obsidian <obsidianArgs>` and returns its stdout with
// trailing newlines stripped — the same trim bash's `$(...)` command
// substitution performs, load-bearing because callers reuse this value both
// for counting AND for the --verbose listing.
func (h *healthRun) obsidianCmd(sub ...string) string {
	out, _ := h.obsidianRun(sub...) // exit code ignored: `2>/dev/null || true`
	return out
}

// obsidianRun is obsidianCmd plus how the CLI exited; stderr is discarded.
func (h *healthRun) obsidianRun(sub ...string) (string, error) {
	out, err := exec.Command("obsidian", h.obsidianArgs(sub...)...).Output()
	return strings.TrimRight(string(out), "\n"), err
}

// obsidianList runs a listing subcommand and returns its answer, or why it is
// not one. A CLI that exits non-zero has no answer, even an empty one: an empty
// list would count as zero findings and pass. The CLI also reports a failure on
// stdout and exits 0 (measured: `Error: Command "x" not found. It may require a
// plugin to be enabled.`), so an answer that starts with "Error:" is refused
// rather than counted as one listed file.
func (h *healthRun) obsidianList(sub ...string) (string, error) {
	out, err := h.obsidianRun(sub...)
	if err != nil {
		return "", err
	}
	first, _, _ := strings.Cut(strings.TrimLeft(out, " \t\r\n"), "\n")
	if strings.HasPrefix(first, "Error:") {
		return "", errors.New(strings.TrimSpace(first))
	}
	return out, nil
}

// printTruncated mirrors `echo "$VAR" | head -N` plus the "... and M more"
// line, keyed on cleanCount (the NON-blank count) rather than the raw line
// count — reproducing the shell's own arithmetic exactly.
func (h *healthRun) printTruncated(label, raw string, cleanCount, limit int) {
	emit(h.w, "  --- %s ---\n", label)
	lines := strings.Split(raw, "\n")
	if len(lines) > limit {
		lines = lines[:limit]
	}
	for _, l := range lines {
		emit(h.w, "%s\n", l)
	}
	if cleanCount > limit {
		emit(h.w, "  ... and %d more\n", cleanCount-limit)
	}
}

// collectMarkdownFiles mirrors `find "$vaultDir" -name '*.md' -not -path
// '*/.obsidian/*'`: every *.md file, skipping the .obsidian subtree entirely
// (which also excludes anything nested under it, matching the path-substring
// exclusion in the shell).
func collectMarkdownFiles(vaultDir string) []string {
	var files []string
	_ = filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // the shell's `find` swallows walk errors too
		}
		if d.IsDir() {
			if d.Name() == ".obsidian" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

// frontmatterCounts mirrors six separate `grep -rl "^${field}:" --include='*.md'`
// sweeps, collapsed into one pass over the file set section 2 already
// collected: a file counts toward a field the first time any line starts with
// "field:", never twice.
func (h *healthRun) frontmatterCounts(fields []string) map[string]int {
	prefixes := make(map[string]string, len(fields))
	for _, f := range fields {
		prefixes[f] = f + ":"
	}
	counts := make(map[string]int, len(fields))
	for _, path := range h.mdFiles {
		data, err := os.ReadFile(path) //nolint:gosec // path came from our own vault walk, not user input
		if err != nil {
			continue
		}
		found := make(map[string]bool, len(fields))
		for _, line := range strings.Split(string(data), "\n") {
			for _, f := range fields {
				if !found[f] && strings.HasPrefix(line, prefixes[f]) {
					found[f] = true
				}
			}
		}
		for f, ok := range found {
			if ok {
				counts[f]++
			}
		}
	}
	return counts
}

// printPrefixed mirrors `sed 's|^|<prefix>|'`: every line of s gets prefix
// prepended, and empty input produces no output at all (not one empty line).
func printPrefixed(w io.Writer, s, prefix string) {
	if s == "" {
		return
	}
	for _, line := range strings.Split(strings.TrimSuffix(s, "\n"), "\n") {
		emit(w, "%s%s\n", prefix, line)
	}
}

func (h *healthRun) section1WorkingTree() {
	h.section("1/7", "Working Tree Integrity")
	vd := h.opts.VaultDir

	if !isDir(vd) {
		h.skip("Working tree integrity", "vault dir not found")
		return
	}
	if !isDir(filepath.Join(vd, ".git")) {
		h.skip("Working tree integrity", "vault is not a git repo")
		return
	}

	out, _ := exec.Command("git", "-C", vd, "status", "--short").Output()
	var deleted []string
	for _, line := range strings.Split(string(out), "\n") {
		if deletedLineRe.MatchString(line) {
			deleted = append(deleted, line)
		}
	}

	if len(deleted) == 0 {
		h.pass("Working tree clean — no files deleted from disk")
		return
	}
	h.fail("%d file(s) deleted from working tree but still in HEAD — likely sync/watcher race", len(deleted))
	for _, l := range deleted {
		emit(h.w, "        %s\n", l)
	}
	emit(h.w, "        Recovery: cd %s && git checkout -- <file>\n", vd)
}

// section2Connectivity returns (code, true) when the shell would have exited
// immediately — obsidian absent from PATH (1) or its GUI unreachable (2) —
// short-circuiting every later section, footer included.
func (h *healthRun) section2Connectivity() (int, bool) {
	h.section("2/7", "Vault Connectivity")

	if _, err := exec.LookPath("obsidian"); err != nil {
		h.errorLine("Obsidian CLI not found in PATH")
		return 1, true
	}

	answer := h.obsidianCmd("vault")
	if !hasAnyChar(answer) {
		h.errorLine("Cannot reach Obsidian GUI. Is Obsidian running?")
		h.infoLine("Start Obsidian, then re-run.")
		return 2, true
	}
	if !namesVault(answer, h.opts.VaultName) {
		first, _, _ := strings.Cut(strings.TrimSpace(answer), "\n")
		h.errorLine("Obsidian CLI answered, but not for vault '" + h.opts.VaultName + "': " + first)
		h.infoLine("Is the vault registered and open in Obsidian under that name?")
		return 2, true
	}
	h.pass("Obsidian CLI connected to vault '%s'", h.opts.VaultName)

	if isDir(h.opts.VaultDir) {
		h.mdFiles = collectMarkdownFiles(h.opts.VaultDir)
		h.totalFiles = len(h.mdFiles)
		h.info("Total markdown files: %d", h.totalFiles)
	} else {
		h.fail("Vault directory not found: %s", h.opts.VaultDir)
		h.totalFiles = 0
	}
	return 0, false
}

// linkCheck names one of the link-graph counts a genre can be left out of.
type linkCheck int

const (
	checkOrphans linkCheck = iota
	checkDeadEnds
	checkUnresolved
)

// linkExemptions declares, once, the notes each link-graph count leaves out
// because they lack the links it measures by design (#1979, #2197). The zones
// are those of the vault's 00_meta/_ssot.md, which points here for this list.
// A rule matches a path prefix, or with segment set, any directory of that
// name at any depth. The report names the zones from this table, so a rule
// added here is a rule the report discloses.
var linkExemptions = []struct {
	match   string
	segment bool
	checks  []linkCheck
}{
	// Archived notes: links to what has moved or gone, and nothing links in.
	{"90_archive/", false, []linkCheck{checkOrphans, checkDeadEnds, checkUnresolved}},
	// Template skeletons: placeholders such as {{client_slug}} instead of links.
	{"00_meta/templates/", false, []linkCheck{checkDeadEnds, checkUnresolved}},
	// Course study notes: not linked knowledge (owner, 2026-10-09).
	{"20_certifications/", false, []linkCheck{checkDeadEnds}},
	// Work product and client records: leaf notes kept per product or client,
	// not linked knowledge (owner, 2026-10-09).
	{"50_work/20-products/", false, []linkCheck{checkDeadEnds}},
	{"50_work/30-clients/", false, []linkCheck{checkDeadEnds}},
	// Session journals: a chronological record, not a linked note.
	{"sessions", true, []linkCheck{checkOrphans, checkDeadEnds}},
	// Agent memory files: read by the agent, never linked to.
	{"memory", true, []linkCheck{checkOrphans}},
}

// linkExempt reports whether the vault-relative path rel is left out of check.
func linkExempt(check linkCheck, rel string) bool {
	rel = filepath.ToSlash(rel)
	dirs := strings.Split(rel, "/")
	dirs = dirs[:len(dirs)-1]
	for _, e := range linkExemptions {
		if !slices.Contains(e.checks, check) {
			continue
		}
		if e.segment && slices.Contains(dirs, e.match) || !e.segment && strings.HasPrefix(rel, e.match) {
			return true
		}
	}
	return false
}

// exemptZones names the zones check leaves out, joined for a sentence:
// "90_archive/, sessions/ and memory/".
func exemptZones(check linkCheck, conj string) string {
	var zones []string
	for _, e := range linkExemptions {
		if slices.Contains(e.checks, check) {
			zones = append(zones, strings.TrimSuffix(e.match, "/")+"/")
		}
	}
	if len(zones) < 2 {
		return strings.Join(zones, "")
	}
	return strings.Join(zones[:len(zones)-1], ", ") + " " + conj + " " + zones[len(zones)-1]
}

// orphanExempt reports a vault-relative path that has no incoming links by design.
func orphanExempt(rel string) bool { return linkExempt(checkOrphans, rel) }

// unresolvedExempt reports a link source whose broken links are expected.
func unresolvedExempt(source string) bool { return linkExempt(checkUnresolved, source) }

// linkTally is one link-graph count over the markdown population.
type linkTally struct {
	listed       []string // notes the CLI listed that count
	exemptListed int      // notes the CLI listed in an exempt zone
	attachments  []string // non-markdown files the CLI listed
	population   int      // markdown files that count
	exemptFiles  int      // markdown files in an exempt zone
}

// tally splits a CLI listing for check. Attachments are set apart: the CLI
// lists images and PDFs too, which the markdown population never held, so
// counting them inflated the dead-ends reading from 56% to 88% (#2197).
func (h *healthRun) tally(check linkCheck, out string) linkTally {
	var t linkTally
	for _, l := range nonBlankLines(out) {
		rel := strings.TrimSpace(l)
		switch ext := filepath.Ext(rel); {
		case ext != "" && ext != ".md":
			t.attachments = append(t.attachments, rel)
		case linkExempt(check, rel):
			t.exemptListed++
		default:
			t.listed = append(t.listed, rel)
		}
	}
	for _, f := range h.mdFiles {
		if rel, err := filepath.Rel(h.opts.VaultDir, f); err == nil && linkExempt(check, rel) {
			t.exemptFiles++
			continue
		}
		t.population++
	}
	return t
}

// topFolders names the n folders, two levels deep, holding most of paths.
func topFolders(paths []string, n int) string {
	counts := map[string]int{}
	for _, p := range paths {
		dirs := strings.Split(filepath.ToSlash(p), "/")
		dirs = dirs[:len(dirs)-1]
		if len(dirs) > 2 {
			dirs = dirs[:2]
		}
		dir := strings.Join(dirs, "/")
		if dir == "" {
			dir = "(vault root)"
		}
		counts[dir]++
	}
	dirs := make([]string, 0, len(counts))
	for d := range counts {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		if counts[dirs[i]] != counts[dirs[j]] {
			return counts[dirs[i]] > counts[dirs[j]]
		}
		return dirs[i] < dirs[j]
	})
	if len(dirs) > n {
		dirs = dirs[:n]
	}
	parts := make([]string, len(dirs))
	for i, d := range dirs {
		parts[i] = fmt.Sprintf("%s (%d)", d, counts[d])
	}
	return strings.Join(parts, ", ")
}

func nonBlankLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func (h *healthRun) section3OrphansDeadEnds() {
	h.section("3/7", "Orphans & Dead-Ends")

	orphansOut, orphansErr := h.obsidianList("orphans")
	orphans := h.tally(checkOrphans, orphansOut)
	orphanCount := len(orphans.listed)
	orphanPct := pct(orphanCount, orphans.population)

	switch {
	case orphansErr != nil:
		h.fail("Orphans: the obsidian CLI answered with an error: %s", orphansErr)
	case orphanPct <= 30:
		h.pass("Orphans: %d/%d (%d%%)", orphanCount, orphans.population, orphanPct)
	case orphanPct <= 50:
		h.warn("Orphans: %d/%d (%d%%) — consider adding backlinks", orphanCount, orphans.population, orphanPct)
	default:
		h.fail("Orphans: %d/%d (%d%%) — too many isolated files", orphanCount, orphans.population, orphanPct)
	}
	if orphansErr == nil {
		if orphans.exemptFiles > 0 {
			h.info("Not counted: %d file(s) under %s (%d orphaned), which have no incoming links by design",
				orphans.exemptFiles, exemptZones(checkOrphans, "and"), orphans.exemptListed)
		}
		if len(orphans.attachments) > 0 {
			h.info("Not counted: %d attachment(s) no note links to", len(orphans.attachments))
		}
	}

	deadOut, deadErr := h.obsidianList("deadends")
	deadEnds := h.tally(checkDeadEnds, deadOut)
	deadCount := len(deadEnds.listed)
	deadPct := pct(deadCount, deadEnds.population)

	switch {
	case deadErr != nil:
		h.fail("Dead-ends: the obsidian CLI answered with an error: %s", deadErr)
	case deadPct <= 30:
		h.pass("Dead-ends: %d/%d (%d%%)", deadCount, deadEnds.population, deadPct)
	case deadPct <= 50:
		h.warn("Dead-ends: %d/%d (%d%%) — consider adding outgoing links", deadCount, deadEnds.population, deadPct)
	default:
		h.fail("Dead-ends: %d/%d (%d%%) — too many files without outgoing links", deadCount, deadEnds.population, deadPct)
	}
	if deadErr == nil {
		if deadPct > 30 {
			hint := " (--verbose lists them)"
			if h.opts.Verbose {
				hint = ""
			}
			h.info("Most dead-ends: %s%s", topFolders(deadEnds.listed, 3), hint)
		}
		if deadEnds.exemptFiles > 0 {
			h.info("Not counted: %d file(s) under %s (%d dead-ends), which have no outgoing links by design",
				deadEnds.exemptFiles, exemptZones(checkDeadEnds, "and"), deadEnds.exemptListed)
		}
	}

	if h.opts.Verbose && orphanCount > 0 {
		h.printTruncated("Orphan files", strings.Join(orphans.listed, "\n"), orphanCount, 20)
	}
	if h.opts.Verbose && orphansErr == nil && len(orphans.attachments) > 0 {
		h.printTruncated("Unlinked attachments", strings.Join(orphans.attachments, "\n"), len(orphans.attachments), 20)
	}
	if h.opts.Verbose && deadErr == nil && deadCount > 0 {
		h.printTruncated("Dead-end files", strings.Join(deadEnds.listed, "\n"), deadCount, 20)
	}
}

// unresolvedLink is one entry of `obsidian unresolved verbose format=json`.
// The CLI joins several sources into one string with ", ". It is split on the
// bare comma and trimmed, so a joiner without the space still separates the
// sources. A comma inside a file name only produces fragments without an
// exempt prefix, which count: the split can over-count, never exempt a live
// link.
type unresolvedLink struct {
	Link    string `json:"link"`
	Sources string `json:"sources"`
}

func (h *healthRun) section4Unresolved() {
	h.section("4/7", "Unresolved Links")

	out, err := h.obsidianList("unresolved", "verbose", "format=json")
	if err != nil {
		h.fail("Unresolved links: the obsidian CLI answered with an error: %s", err)
		return
	}
	var all []unresolvedLink
	if strings.TrimSpace(out) != "" {
		if err := json.Unmarshal([]byte(out), &all); err != nil {
			first, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
			h.fail("Unresolved links: cannot read the obsidian CLI's JSON (%v); it printed: %s", err, first)
			return
		}
	}
	var live []string
	for _, l := range all {
		counted := strings.TrimSpace(l.Sources) == "" // no source named: count it
		for _, src := range strings.Split(l.Sources, ",") {
			if src = strings.TrimSpace(src); src != "" && !unresolvedExempt(src) {
				counted = true
			}
		}
		if counted {
			live = append(live, l.Link)
		}
	}
	count := len(live)

	switch {
	case count == 0:
		h.pass("No unresolved links")
	case count <= 10:
		h.warn("Unresolved links: %d", count)
	default:
		h.fail("Unresolved links: %d", count)
	}
	if exempt := len(all) - count; exempt > 0 {
		h.info("Not counted: %d unresolved link(s) found only in %s (placeholders and archived notes)", exempt, exemptZones(checkUnresolved, "or"))
	}

	if h.opts.Verbose && count > 0 {
		h.printTruncated("Unresolved links", strings.Join(live, "\n"), count, 20)
	}
}

func (h *healthRun) section5Frontmatter() {
	h.section("5/7", "Frontmatter Coverage")

	if h.totalFiles == 0 {
		h.skip("Frontmatter", "no markdown files found")
		return
	}

	fields := []string{"id", "type", "status", "tags", "created", "owner"}
	counts := h.frontmatterCounts(fields)
	for _, f := range fields {
		c := counts[f]
		p := pct(c, h.totalFiles)
		switch {
		case p >= 80:
			h.pass("%s: %d/%d (%d%%)", f, c, h.totalFiles, p)
		case p >= 50:
			h.warn("%s: %d/%d (%d%%)", f, c, h.totalFiles, p)
		default:
			h.fail("%s: %d/%d (%d%%)", f, c, h.totalFiles, p)
		}
	}
}

func (h *healthRun) section6Tags() {
	h.section("6/7", "Tag Hygiene")

	out, err := h.obsidianList("tags")
	if err != nil {
		h.fail("Tags: the obsidian CLI answered with an error: %s", err)
		return
	}
	count := countNonBlank(out)
	h.info("Total unique tags: %d", count)

	if h.opts.Verbose && count > 0 {
		h.printTruncated("Tags", out, count, 30)
	}
}

// runIntegrityChecks is the FIRST of section 7's two passes. It returns
// (code, true) the moment a file drifts — the ORACLE DEFECT (#1314): the shell
// re-execs check-backlog-integrity.sh a SECOND time, piped straight into `sed`,
// to print output it already has a verdict for — instead of capturing to a
// variable first, the way runMergedChecks below correctly does. Under
// `set -euo pipefail` that unnegated pipeline's non-zero exit (inherited from
// the script it just reported failing) aborts the WHOLE SCRIPT right here: no
// later file in this same loop, no merged-check pass, no closing footer.
// Pinned by the backlog-drift golden, whose expected/stdout simply stops after
// this file's detail. Not "fixed" here — see #1314.
func (h *healthRun) runIntegrityChecks(matches []string) (int, bool) {
	for _, tasks := range matches {
		out, drift, err := BacklogIntegrity(tasks)
		if err != nil {
			// The script printed nothing here and exited 2; the reason is
			// what a reader needs, so the port says it.
			h.fail("Backlog integrity: cannot read %s/11-tasks.md: %v", filepath.Base(filepath.Dir(tasks)), err)
			return 2, true
		}
		if drift {
			h.fail("Backlog drift in %s/11-tasks.md (duplicate IDs / status contradictions)",
				filepath.Base(filepath.Dir(tasks)))
			printPrefixed(h.w, out, "        ")
			return 1, true
		}
	}
	h.pass("Backlog integrity: %d task file(s) clean (one ticket = one entry)", len(matches))
	return 0, false
}

// runMergedChecks is section 7's SECOND, independent pass: semantic drift is
// ADVISORY (warn, never fail). It captures each script's output to a variable
// before printing, so — unlike runIntegrityChecks above — it does NOT share
// section 7's pipefail landmine: a captured string cannot fail a pipeline.
func (h *healthRun) runMergedChecks(matches []string) {
	for _, tasks := range matches {
		out, stale, err := BacklogMerged(tasks, "")
		if err != nil {
			h.warn("Stale-merged ticks: cannot read %s/11-tasks.md: %v", filepath.Base(filepath.Dir(tasks)), err)
			continue
		}
		if stale {
			h.warn("Stale-merged ticks in %s/11-tasks.md — work shipped, tick still [ ]:",
				filepath.Base(filepath.Dir(tasks)))
			printPrefixed(h.w, out, "        ")
		}
	}
}

// section7Backlog returns (code, true) when the shell would have aborted the
// entire script right here (runIntegrityChecks' oracle defect), short-
// circuiting everything after it, footer included — exactly like
// section2Connectivity's two abort points.
func (h *healthRun) section7Backlog() (int, bool) {
	h.section("7/7", "Backlog Integrity")

	vd := h.opts.VaultDir
	if !isDir(vd) {
		h.skip("Backlog integrity", "vault dir not found")
		return 0, false
	}

	matches, _ := filepath.Glob(filepath.Join(vd, "10_projects", "*", "11-tasks.md"))
	sort.Strings(matches)

	if len(matches) == 0 {
		h.skip("Backlog integrity", "no 10_projects/*/11-tasks.md found")
		return 0, false
	}

	if code, aborted := h.runIntegrityChecks(matches); aborted {
		return code, true
	}
	h.runMergedChecks(matches)
	return 0, false
}

// RunHealth runs the full 7-section report and returns the exit code the shell
// would produce: 0 all pass, 1 a check failed (or obsidian is entirely absent
// from PATH), 2 the GUI is unreachable. A non-nil error is reserved for a
// failure this function cannot itself express as one of those three codes.
func RunHealth(w io.Writer, opts HealthOptions) (int, error) {
	h := &healthRun{w: w, opts: opts}

	emit(w, "========================================\n")
	emit(w, "   VAULT HEALTH REPORT\n")
	emit(w, "========================================\n")
	emit(w, "Vault: %s (%s)\n", opts.VaultName, opts.VaultDir)

	h.section1WorkingTree()

	if code, aborted := h.section2Connectivity(); aborted {
		return code, nil
	}

	h.section3OrphansDeadEnds()
	h.section4Unresolved()
	h.section5Frontmatter()
	h.section6Tags()

	if code, aborted := h.section7Backlog(); aborted {
		return code, nil
	}

	emit(w, "\n========================================\n")
	emit(w, "Results: %d passed, %d failed, %d skipped\n", h.passed, h.failed, h.skipped)
	emit(w, "========================================\n")

	if h.failed > 0 {
		return 1, nil
	}
	return 0, nil
}
