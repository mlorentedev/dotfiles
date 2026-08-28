package vault

// health.go — the Go port of scripts/vault-health.sh (CLI-021 / #490, increment
// 2). Built BESIDE the shell twin, exactly like crystallize.go was in increment
// 1: nothing here is wired up as a caller yet (that is CLI-023 / #492). The
// shell stays canonical until that cutover.
//
// Two seams increment 1 did not have, both called out in the spec
// (specs/CLI-021-dotf-vault-build-knowledge/tasks.md §3):
//
//  1. An EXTERNAL BINARY contract. Four of seven sections shell out to
//     `obsidian`, which talks to a running GUI over IPC. The golden corpus
//     (tests/golden/vault-health/) stubs that binary on PATH and pins the exact
//     argv this code sends it — not just stdout — because a port could drift in
//     HOW it calls obsidian while stdout stayed byte-identical.
//  2. A SUBPROCESS seam onto two sibling scripts, check-backlog-integrity.sh and
//     check-backlog-merged.sh (SDD-012 / SDD-012b). They are EXECED, not ported:
//     porting them is out of #490's three increments, and they survive the
//     CLI-023 cutover as their own `vault` subcommands. A Go binary has no
//     $SCRIPT_DIR, so ScriptsDir/BashPath are an explicit location seam
//     (ADR-025) rather than a guess — and an unresolved seam FAILS the section
//     rather than silently skipping it, per the spec.
//
// The shell is the oracle: every observable byte is pinned by the golden corpus
// and reproduced faithfully, including its VAULT_DIR/VAULT_PATH/default
// fallback (a plain env-var cascade, NOT the ADR-025 machine.json cascade
// ResolveVault() uses elsewhere — vault-health.sh predates that cascade, and
// matching the oracle takes priority over "improving" it here).

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// HealthOptions configures a `vault health` run. VaultDir/VaultName are already
// resolved by the caller (the shell's own env-var cascade lives in the cmd
// layer, mirroring how crystallize's path resolution lives there too).
type HealthOptions struct {
	VaultDir   string
	VaultName  string
	Verbose    bool
	ScriptsDir string // hosts check-backlog-integrity.sh / check-backlog-merged.sh
	BashPath   string // interpreter for the two backlog scripts (mem.ResolveBash())
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

// obsidianCmd runs `obsidian --no-sandbox <sub> --vault <name>` and returns its
// stdout with trailing newlines stripped — the same trim bash's `$(...)`
// command substitution performs, load-bearing because callers reuse this value
// both for counting AND for the --verbose listing.
func (h *healthRun) obsidianCmd(sub string) string {
	cmd := exec.Command("obsidian", "--no-sandbox", sub, "--vault", h.opts.VaultName)
	out, _ := cmd.Output() // stderr discarded, exit code ignored: `2>/dev/null || true`
	return strings.TrimRight(string(out), "\n")
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

// runScript execs a backlog script through the resolved bash interpreter (the
// Windows-safe convention session_start.go already uses for vault-health.sh
// itself) rather than relying on the shebang + executable bit the shell
// invokes directly — there is no .ps1 twin for either script to fall back to.
func (h *healthRun) runScript(script, arg string) (string, int) {
	cmd := exec.Command(h.opts.BashPath, script, arg)
	out, err := cmd.Output()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = 1
		}
	}
	return string(out), code
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

	if !hasAnyChar(h.obsidianCmd("vault")) {
		h.errorLine("Cannot reach Obsidian GUI. Is Obsidian running?")
		h.infoLine("Start Obsidian or run: obsidian --no-sandbox &")
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

func (h *healthRun) section3OrphansDeadEnds() {
	h.section("3/7", "Orphans & Dead-Ends")

	orphanOut := h.obsidianCmd("orphans")
	orphanCount := countNonBlank(orphanOut)
	deadOut := h.obsidianCmd("dead-ends")
	deadCount := countNonBlank(deadOut)

	orphanPct := pct(orphanCount, h.totalFiles)
	deadPct := pct(deadCount, h.totalFiles)

	switch {
	case orphanPct <= 30:
		h.pass("Orphans: %d/%d (%d%%)", orphanCount, h.totalFiles, orphanPct)
	case orphanPct <= 50:
		h.warn("Orphans: %d/%d (%d%%) — consider adding backlinks", orphanCount, h.totalFiles, orphanPct)
	default:
		h.fail("Orphans: %d/%d (%d%%) — too many isolated files", orphanCount, h.totalFiles, orphanPct)
	}

	switch {
	case deadPct <= 30:
		h.pass("Dead-ends: %d/%d (%d%%)", deadCount, h.totalFiles, deadPct)
	case deadPct <= 50:
		h.warn("Dead-ends: %d/%d (%d%%) — consider adding outgoing links", deadCount, h.totalFiles, deadPct)
	default:
		h.fail("Dead-ends: %d/%d (%d%%) — too many files without outgoing links", deadCount, h.totalFiles, deadPct)
	}

	// The shell lists orphan files only — there is no dead-ends listing.
	if h.opts.Verbose && orphanCount > 0 {
		h.printTruncated("Orphan files", orphanOut, orphanCount, 20)
	}
}

func (h *healthRun) section4Unresolved() {
	h.section("4/7", "Unresolved Links")

	out := h.obsidianCmd("unresolved")
	count := countNonBlank(out)

	switch {
	case count == 0:
		h.pass("No unresolved links")
	case count <= 10:
		h.warn("Unresolved links: %d", count)
	default:
		h.fail("Unresolved links: %d", count)
	}

	if h.opts.Verbose && count > 0 {
		h.printTruncated("Unresolved links", out, count, 20)
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

	out := h.obsidianCmd("tags")
	count := countNonBlank(out)
	h.info("Total unique tags: %d", count)

	if h.opts.Verbose && count > 0 {
		h.printTruncated("Tags", out, count, 30)
	}
}

// section7Backlog returns (code, true) when the shell would have aborted the
// entire script right here — the ORACLE DEFECT below — short-circuiting
// everything after it, footer included, exactly like section2Connectivity's
// two abort points.
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

	integrityScript := filepath.Join(h.opts.ScriptsDir, "check-backlog-integrity.sh")
	mergedScript := filepath.Join(h.opts.ScriptsDir, "check-backlog-merged.sh")
	if h.opts.ScriptsDir == "" || !fileExists(integrityScript) || !fileExists(mergedScript) {
		h.fail("Backlog integrity: cannot locate check-backlog-integrity.sh / check-backlog-merged.sh " +
			"(scripts dir unresolved) — run from a dotfiles checkout or set DOTFILES_REPO_DIR")
		return 0, false
	}

	for _, tasks := range matches {
		out, code := h.runScript(integrityScript, tasks)
		if code != 0 {
			h.fail("Backlog drift in %s/11-tasks.md (duplicate IDs / status contradictions)",
				filepath.Base(filepath.Dir(tasks)))
			printPrefixed(h.w, out, "        ")
			// ORACLE DEFECT reproduced (#1314): the shell re-execs the check a
			// SECOND time piped straight into `sed` to print its output,
			// instead of capturing to a variable first the way the
			// merged-check loop below correctly does. Under `set -euo
			// pipefail` that unnegated pipeline's non-zero exit (inherited
			// from the script it just reported failing) aborts the WHOLE
			// SCRIPT right here: no later files in this same loop, no
			// merged-check pass, no closing footer. Pinned by the
			// backlog-drift golden, whose expected/stdout simply stops after
			// this file's detail. Not "fixed" here — see #1314.
			return code, true
		}
	}
	h.pass("Backlog integrity: %d task file(s) clean (one ticket = one entry)", len(matches))

	// Semantic drift is ADVISORY (warn, never fail) — a second, independent
	// pass over the same file set. This one captures to a variable before
	// printing, so it does NOT share section 7's pipefail landmine above.
	for _, tasks := range matches {
		out, code := h.runScript(mergedScript, tasks)
		if code != 0 {
			h.warn("Stale-merged ticks in %s/11-tasks.md — work shipped, tick still [ ]:",
				filepath.Base(filepath.Dir(tasks)))
			printPrefixed(h.w, out, "        ")
		}
	}
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
