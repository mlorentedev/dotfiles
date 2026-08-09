package vault

// crystallize.go — the Go port of scripts/knowledge-crystallize.{sh,ps1}
// (CLI-021 / #490, AUDIT-007 PR5). Built BESIDE the twins: nothing is deleted
// and no caller is repointed here — that is CLI-023 (#492).
//
// The shell is the oracle. Every observable behaviour below is pinned by
// tests/golden/crystallize/, captured at the revisions in that directory's
// ORACLE file. Where the shell is buggy this code reproduces the bug and the
// defect is ticketed separately (#873, #874) — a port that improves while
// translating cannot be characterization-tested.
//
// The one deliberate divergence: the shell colours its log tags via utils.sh;
// no Go command in this CLI emits ANSI, so this does not either. The goldens
// strip ANSI and therefore CANNOT catch that difference, which is exactly why
// it is written down here.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

const (
	// lineLimit mirrors check_line_count()'s `limit=150`.
	lineLimit = 150

	markerCurrentDate    = "# currentDate"
	markerLastCrystal    = "## Last Crystallized:"
	markerSessionHandoff = "## Session Handoff"
)

// ErrRefused marks the BUG-062 refusal: the file was left byte-identical on
// purpose. The four [ERROR] lines already name the file and the tracking issue,
// so the caller silences cobra's own "Error:" line for this case only — the shell
// exits 1 having printed exactly those four lines, and byte-parity on the golden
// corpus is the contract. Genuine errors still print normally.
var ErrRefused = errors.New("refused")

// dateLineRe mirrors awk's /^Today.s date is / — an unescaped dot, so it matches
// any character where the apostrophe sits. Reproduced exactly rather than
// tightened: a stricter pattern would silently change which lines are replaced.
var dateLineRe = regexp.MustCompile(`^Today.s date is `)

// yamlBlockOpenRe mirrors the shell's block-scalar opener test. Structural, not
// an indent probe: indent width is NOT fixed across projects (four spaces for
// hive, six for pollex), so nothing here may key on a literal width.
var yamlBlockOpenRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*:[ \t]*\|[-+0-9]*[ \t]*$`)

var yamlDocStartRe = regexp.MustCompile(`^---[ \t]*$`)

// splitLines models awk's record splitting: records are separated by "\n", and a
// trailing newline does not produce a final empty record.
func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	trimmed := strings.TrimSuffix(content, "\n")
	return strings.Split(trimmed, "\n")
}

// joinLines models awk's output: every record is emitted followed by ORS ("\n").
// This is why awk NORMALISES a file that lacked a trailing newline, and the port
// must normalise identically.
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func anyLineHasPrefix(lines []string, prefix string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return true
		}
	}
	return false
}

func countLinesWithPrefix(lines []string, prefix string) int {
	n := 0
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			n++
		}
	}
	return n
}

// DedupCurrentDate drops EVERY "# currentDate" line and the "Today's date is"
// line that follows it.
//
// Note the shell's actual semantics, which the docstring on the shell function
// misstates: it does not "keep the latest" — it removes all of them, and
// UpdateCurrentDate then re-adds one. The net effect is a single block, but at a
// NEW position (before the handoff), not where any original sat. That relocation
// is pinned by the handoff-with-duplicates golden.
func DedupCurrentDate(lines []string) []string {
	out := make([]string, 0, len(lines))
	skip := false
	for _, l := range lines {
		if strings.HasPrefix(l, markerCurrentDate) {
			skip = true
			continue
		}
		if skip && dateLineRe.MatchString(l) {
			skip = false
			continue
		}
		// awk's bare `skip { skip=0 }` has no `next`, so the line still prints.
		skip = false
		out = append(out, l)
	}
	return out
}

// AppendBeforeHandoff inserts block before the FIRST "## Session Handoff" line,
// or appends at EOF when there is none.
//
// HARNESS-029 requires that block to stay last: it is rewritten every session, so
// keeping it out of the auto-loaded KV-cache prefix stops it busting the provider
// prompt cache on every new session. A bare append breaks that invariant, which
// is what BUG-060 was.
func AppendBeforeHandoff(lines []string, block string) []string {
	if !anyLineHasPrefix(lines, markerSessionHandoff) {
		return append(append([]string{}, lines...), splitLines(block+"\n")...)
	}
	out := make([]string, 0, len(lines)+2)
	done := false
	for _, l := range lines {
		if !done && strings.HasPrefix(l, markerSessionHandoff) {
			out = append(out, splitLines(block+"\n")...)
			done = true
		}
		out = append(out, l)
	}
	return out
}

// UpdateCurrentDate rewrites the date line following a "# currentDate" marker, or
// adds the whole section when the marker is absent. It returns the log line the
// shell emits.
//
// ORACLE DEFECT reproduced (#873): when the marker is present but no
// "Today's date is" line follows it anywhere, nothing is written — yet the shell
// still logs "Updated currentDate to <today>". Pinned by the
// marker-without-dateline golden. Do not "fix" this here; fixing it requires a
// deliberate recapture.
func UpdateCurrentDate(lines []string, today string) ([]string, string) {
	if anyLineHasPrefix(lines, markerCurrentDate) {
		out := make([]string, 0, len(lines))
		inDate := false
		for _, l := range lines {
			if strings.HasPrefix(l, markerCurrentDate) {
				out = append(out, l)
				inDate = true
				continue
			}
			// awk keeps in_date set until a date line turns up — not merely the
			// immediately-following line.
			if inDate && dateLineRe.MatchString(l) {
				out = append(out, fmt.Sprintf("Today's date is %s.", today))
				inDate = false
				continue
			}
			out = append(out, l)
		}
		return out, fmt.Sprintf("[INFO] Updated currentDate to %s", today)
	}
	block := fmt.Sprintf("%s\nToday's date is %s.", markerCurrentDate, today)
	return AppendBeforeHandoff(lines, block),
		fmt.Sprintf("[INFO] Added currentDate section (%s)", today)
}

// StampLastCrystallized updates, inserts before the currentDate marker, or
// appends the "## Last Crystallized:" stamp — the shell's three-branch cascade.
func StampLastCrystallized(lines []string, today string) ([]string, string) {
	stamp := fmt.Sprintf("%s %s", markerLastCrystal, today)

	if anyLineHasPrefix(lines, markerLastCrystal) {
		out := make([]string, 0, len(lines))
		for _, l := range lines {
			if strings.HasPrefix(l, markerLastCrystal) {
				out = append(out, stamp)
				continue
			}
			out = append(out, l)
		}
		return out, fmt.Sprintf("[INFO] Updated Last Crystallized to %s", today)
	}

	if anyLineHasPrefix(lines, markerCurrentDate) {
		// awk's `print "## Last Crystallized: " today "\n"` emits the stamp AND a
		// blank line, because print appends ORS to a string already ending in \n.
		out := make([]string, 0, len(lines)+2)
		for _, l := range lines {
			if strings.HasPrefix(l, markerCurrentDate) {
				out = append(out, stamp, "")
			}
			out = append(out, l)
		}
		return out, fmt.Sprintf("[INFO] Added Last Crystallized stamp (%s)", today)
	}

	return AppendBeforeHandoff(lines, stamp),
		fmt.Sprintf("[INFO] Added Last Crystallized stamp (%s)", today)
}

// IsYAMLBlockScalar reports whether the file stores its body inside a YAML block
// scalar — the shape BUG-062 (#857) refuses rather than corrupts, because every
// marker above is anchored at column 0 and would therefore miss, sending the text
// outside the block while printing success.
func IsYAMLBlockScalar(content string) bool {
	lines := splitLines(content)
	if len(lines) == 0 || !yamlDocStartRe.MatchString(lines[0]) {
		return false
	}
	for _, l := range lines {
		if yamlBlockOpenRe.MatchString(l) {
			return true
		}
	}
	return false
}

// newlineCount mirrors `wc -l`, which counts newline characters — NOT lines. A
// file whose last line lacks a trailing newline therefore counts one fewer.
func newlineCount(content string) int {
	return strings.Count(content, "\n")
}

// ProcessProject applies the full transform to one MEMORY.md.
func ProcessProject(w io.Writer, memoryFile, today string) error {
	raw, err := os.ReadFile(memoryFile) //nolint:gosec // path built from HOME + project key
	if err != nil {
		return err
	}
	content := string(raw)

	if IsYAMLBlockScalar(content) {
		fmt.Fprintf(w, "[ERROR] Refusing to stamp %s\n", memoryFile)
		fmt.Fprintf(w, "[ERROR] Its body sits inside a YAML block scalar, which this script cannot edit\n")
		fmt.Fprintf(w, "[ERROR] without corrupting the file (#857). Stamp it by hand until the\n")
		fmt.Fprintf(w, "[ERROR] YAML-aware 'dotf vault crystallize' lands (#490).\n")
		return fmt.Errorf("%w: %s is a YAML block scalar", ErrRefused, memoryFile)
	}

	lines := splitLines(content)

	if n := countLinesWithPrefix(lines, markerCurrentDate); n > 1 {
		fmt.Fprintf(w, "[INFO] Removing %d duplicate # currentDate entries...\n", n-1)
		lines = DedupCurrentDate(lines)
	}

	lines, logLine := UpdateCurrentDate(lines, today)
	fmt.Fprintln(w, logLine)

	lines, logLine = StampLastCrystallized(lines, today)
	fmt.Fprintln(w, logLine)

	out := joinLines(lines)
	if err := os.WriteFile(memoryFile, []byte(out), 0o600); err != nil {
		return err
	}

	if n := newlineCount(out); n > lineLimit {
		fmt.Fprintf(w, "[WARNING] MEMORY.md has %d lines (limit: %d) — run /crystallize to trim\n", n, lineLimit)
	} else {
		fmt.Fprintf(w, "[SUCCESS] MEMORY.md line count: %d / %d\n", n, lineLimit)
	}

	fmt.Fprintf(w, "[SUCCESS] Updated: %s\n", memoryFile)
	return nil
}

// PrintChecklist reproduces print_checklist() byte for byte, em-dashes included.
func PrintChecklist(w io.Writer) {
	fmt.Fprint(w, "\n=== Knowledge Crystallization Checklist ===\n\n")
	fmt.Fprint(w, "Manual steps (AI-assisted, run in Claude Code):\n")
	fmt.Fprint(w, "  [ ] /insights  — audit observation backlog\n")
	fmt.Fprint(w, "  [ ] /crystallize — promote observations to vault lessons\n")
	fmt.Fprint(w, "  [ ] Check ~/Projects/knowledge/00_meta/patterns/ for new patterns\n")
	fmt.Fprint(w, "  [ ] Update 11-tasks.md backlog progress bar\n\n")
	fmt.Fprint(w, "Automated (done by this script):\n")
	fmt.Fprint(w, "  [x] currentDate updated\n")
	fmt.Fprint(w, "  [x] Last Crystallized stamped\n")
	fmt.Fprint(w, "  [x] Duplicate date entries removed\n\n")
}

// memoryFileFor is find_memory_file(): <home>/.claude/projects/<key>/memory/MEMORY.md.
//
// The key comes from memlink.ClaudeProjectKey rather than a local reimplementation
// (tasks.md is explicit about this): that function also maps '\' and ':', which is
// the #689 drive-colon fix. On the POSIX paths the shell handles, the two encodings
// are identical.
func memoryFileFor(home, projectDir string) string {
	return filepath.Join(memlink.ClaudeMemoryTarget(home, projectDir), "MEMORY.md")
}

// decodePath is decode_path(): reverse the key, else scan <home>/Projects for a
// directory whose encoding matches. Returns "" when nothing resolves.
func decodePath(home, encoded string) string {
	simple := strings.ReplaceAll(encoded, "-", "/")
	if isDir(simple) {
		return simple
	}
	var found string
	root := filepath.Join(home, "Projects")
	// maxdepth 5 relative to root, matching the shell's `find -maxdepth 5`.
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || found != "" {
			return nil //nolint:nilerr // the shell swallows find's errors too
		}
		if rel, rerr := filepath.Rel(root, path); rerr == nil {
			if depth := len(strings.Split(filepath.ToSlash(rel), "/")); rel != "." && depth > 5 {
				return filepath.SkipDir
			}
		}
		if memlink.ClaudeProjectKey(path) == encoded {
			found = path
		}
		return nil
	})
	return found
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// CrystallizeOne is the single-project entrypoint. It returns false when the
// project has no MEMORY.md — the shell warns and exits 0 in that case, so this is
// not an error.
func CrystallizeOne(w io.Writer, home, projectDir, today string) (ok bool, err error) {
	fmt.Fprintf(w, "[INFO] Project: %s\n", projectDir)
	fmt.Fprintf(w, "[INFO] Date: %s\n", today)

	memoryFile := memoryFileFor(home, projectDir)
	if _, statErr := os.Stat(memoryFile); statErr != nil {
		fmt.Fprintf(w, "[WARNING] No MEMORY.md found for %s\n", projectDir)
		fmt.Fprintf(w, "[WARNING] Expected: %s\n", memoryFile)
		fmt.Fprintf(w, "[WARNING] Run Claude Code in this project first to initialize the memory directory.\n")
		return false, nil
	}

	fmt.Fprintf(w, "[INFO] MEMORY.md: %s\n", memoryFile)
	if err := ProcessProject(w, memoryFile, today); err != nil {
		return true, err
	}
	PrintChecklist(w)
	return true, nil
}

// CrystallizeAll is run_all(): walk every project key under
// <home>/.claude/projects and process the ones that resolve.
//
// A refusal counts as SKIPPED, never processed. Reporting "5 / 5" while having
// declined one is the same "prints success while doing nothing" failure the
// BUG-062 guard exists to stop, so the arithmetic is load-bearing.
func CrystallizeAll(w io.Writer, home, today string) error {
	projectsDir := filepath.Join(home, ".claude", "projects")
	fmt.Fprintf(w, "[INFO] Discovering all projects in %s...\n", projectsDir)
	fmt.Fprint(w, "\n")

	matches, _ := filepath.Glob(filepath.Join(projectsDir, "*", "memory", "MEMORY.md"))
	sort.Strings(matches)

	found, skipped := 0, 0
	for _, memoryFile := range matches {
		if st, err := os.Stat(memoryFile); err != nil || st.IsDir() {
			continue
		}
		found++

		encoded := filepath.Base(filepath.Dir(filepath.Dir(memoryFile)))
		projectDir := decodePath(home, encoded)

		if projectDir == "" {
			fmt.Fprintf(w, "[WARNING] [%s] → not found on disk (different machine or deleted — skipping)\n", encoded)
			skipped++
			fmt.Fprint(w, "\n")
			continue
		}

		fmt.Fprintf(w, "[INFO] [%s] → %s\n", encoded, projectDir)
		if err := ProcessProject(w, memoryFile, today); err != nil {
			fmt.Fprintf(w, "[WARNING] Failed to process %s — skipping\n", projectDir)
			skipped++
		}
		fmt.Fprint(w, "\n")
	}

	if found == 0 {
		fmt.Fprintf(w, "[WARNING] No MEMORY.md files found in %s\n", projectsDir)
		fmt.Fprintf(w, "[WARNING] Open any project in Claude Code first to initialize its memory directory.\n")
		return nil
	}

	fmt.Fprintf(w, "[SUCCESS] Processed %d / %d projects (%d skipped)\n", found-skipped, found, skipped)
	PrintChecklist(w)
	return nil
}
