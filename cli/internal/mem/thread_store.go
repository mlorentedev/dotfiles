package mem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// MemoryByteBudget is how much of a MEMORY.md a session start reads. The rest
// is dropped, and the handoff section is the file's last, so the newest thread
// is the first thing lost: a write reports success and the next session never
// sees it (#1477, measured at 44 KB with the banner saying "Only part of it was
// loaded").
const MemoryByteBudget = 25_000

// threadsDir holds one file per thread beside MEMORY.md (MEMORY-008, ADR-056).
// The file is the thread; its entry in MEMORY.md is derived from it.
const threadsDir = "threads"

// entryNextActionMax is how many characters of the Next action an entry quotes.
// The thread file holds the rest.
const entryNextActionMax = 160

// entryUpdatedMax and entryJournalMax cap the other two quoted fields, so no
// field, however long, rebuilds the block the store exists to shrink. Caps are
// in characters; with every field at its cap, ASCII text and a key of up to 30
// characters, an entry is at most 450 B.
const (
	entryUpdatedMax = 24
	entryJournalMax = 72
)

// threadFileSafe is what a thread file may be named: journalUnsafe's class,
// never opening with a dot, so no key can name a path outside threads/.
const threadFileSafe = `[A-Za-z0-9_@+-][A-Za-z0-9._@+-]*`

// threadPointer is the line a stored thread's entry ends with. It is
// structure: its presence is what says the entry is derived, and checkBody
// refuses one in a body. A markdown link, so the vault's graph sees it too.
var threadPointer = regexp.MustCompile(`^\[full thread\]\(` + threadsDir + `/(` + threadFileSafe + `\.md)\)$`)

var threadFileNameRe = regexp.MustCompile(`^` + threadFileSafe + `$`)

// StoredThread is what StoreThreadAs did.
type StoredThread struct {
	ThreadWrite
	File     string // the thread file's path
	FileBody string // what the thread file holds after the write
}

// StoreThreadAs is WriteThreadAs for the per-thread store (MEMORY-008): the
// body goes to the thread's own file, and MEMORY.md gets an entry derived from
// it, a few hundred bytes instead of the 1.5 KB block that put the file past
// its budget. Ownership and forks are WriteThreadAs's, so an agent never
// replaces another's thread here either. Pure: the caller writes both files,
// the thread file first.
//
// A block a release from before the store wrote is inline, and stays readable
// as it is: a reader follows a pointer only where an entry carries one. This
// write replaces the block with an entry, and leaves every other thread alone.
func StoreThreadAs(content, memoryPath, threadKey, agent, body string) (StoredThread, error) {
	if err := checkThreadKey(threadKey); err != nil {
		return StoredThread{}, err
	}
	key, kept, stamp, err := threadOwner(content, threadKey, agent)
	if err != nil {
		return StoredThread{}, err
	}
	if err := checkBodyStructure(body); err != nil {
		return StoredThread{}, err
	}
	name, err := threadFileName(key)
	if err != nil {
		return StoredThread{}, err
	}
	out, changed, err := placeThread(content, key, stamp, deriveEntry(body, threadsDir+"/"+name))
	if err != nil {
		return StoredThread{}, err
	}
	return StoredThread{
		ThreadWrite: ThreadWrite{Content: out, Key: key, Kept: kept, Changed: changed},
		File:        filepath.Join(filepath.Dir(memoryPath), threadsDir, name),
		FileBody:    strings.Join(renderThread(key+stamp, body), "\n"),
	}, nil
}

// threadFileName maps a key to its file. A slash maps the way a branch does
// (sanitizeThread), since an explicit --thread is taken as given; anything that
// could leave threads/ is refused, not escaped.
func threadFileName(key string) (string, error) {
	name := sanitizeThread(key)
	if !threadFileNameRe.MatchString(name) {
		return "", fmt.Errorf("thread key %q names no safe file under %s/: a key is one branch-like word of letters, digits and . _ @ + -", key, threadsDir)
	}
	return name + ".md", nil
}

// deriveEntry is a stored thread's MEMORY.md entry: when it was updated, the
// first line of its next action, how many decisions await Manu, the journal
// (threadWriter still attributes an unstamped entry by it) and the pointer.
func deriveEntry(body, pointer string) string {
	th := ParseThread(body)
	var out []string
	if v := th.All("Updated"); v != "" {
		out = append(out, "> Updated: "+entryLine(v, entryUpdatedMax)+"<br>")
	}
	if v := th.All("Next action"); v != "" {
		out = append(out, "**Next action:** "+entryLine(v, entryNextActionMax)+"<br>")
	}
	if v := th.All("Awaiting Manu"); v != "" {
		out = append(out, fmt.Sprintf("**Awaiting Manu:** %d, see the thread<br>", countItems(v)))
	}
	if v := th.All("Journal"); v != "" {
		out = append(out, "Journal: "+entryLine(v, entryJournalMax))
	}
	return strings.Join(append(out, "[full thread]("+pointer+")"), "\n")
}

// entryLine is a field's first non-blank line without its `<br>`, cut to max
// characters when max is positive.
func entryLine(v string, limit int) string {
	var line string
	for l := range strings.SplitSeq(v, "\n") {
		if line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(l), "<br>")); line != "" {
			break
		}
	}
	if r := []rune(line); limit > 0 && len(r) > limit {
		line = strings.TrimSpace(string(r[:limit])) + "…"
	}
	return line
}

// countItems counts a field's list items; a field written as prose is one.
func countItems(v string) int {
	n := 0
	for l := range strings.SplitSeq(v, "\n") {
		if t := strings.TrimSpace(l); strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
			n++
		}
	}
	return max(n, 1)
}

// CheckThreadFile refuses a write over a thread file another key holds. fix/x
// and fix-x name one file, and the file's own heading says whose it is.
func CheckThreadFile(existing []byte, key string) error {
	lines := trimBlankEdges(strings.Split(string(existing), "\n"))
	if len(lines) == 0 {
		return nil
	}
	if held, ok := threadHeadingKey(lines[0]); ok && held != key {
		return fmt.Errorf("thread %q would be stored in the file thread %q holds; pass --thread with a key that names a file of its own", key, held)
	}
	return nil
}

// ReadThreadFile reads a thread file, with nil for one not written yet.
func ReadThreadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- derived from the operator's --memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

// threadText is a thread block's full text: the file its entry points at, or
// the block itself when it is inline. A pointer to a file that cannot be read
// leaves the entry, which is still a handoff.
func threadText(block, memoryDir string) string {
	for l := range strings.SplitSeq(block, "\n") {
		m := threadPointer.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(memoryDir, threadsDir, m[1])) // #nosec G304 -- name matched threadFileSafe
		if err != nil {
			return block
		}
		return strings.TrimRight(string(data), "\n")
	}
	return block
}

// CheckMemoryBudget refuses a write that takes MEMORY.md past MemoryByteBudget,
// naming the threads to archive, least recently updated first, and how to
// archive each. Only a write that grows the file is refused: one that shrinks or
// holds a file already past the budget passes, or no such file could ever be
// brought back under it.
//
// Only a thread with a date is suggested. One with none cannot be called old:
// the pre-thread block and some live threads carry no Updated line, and naming
// them first would archive work nobody had finished (#1477).
func CheckMemoryBudget(before, after string) error {
	if len(after) <= MemoryByteBudget || len(after) <= len(before) {
		return nil
	}
	dated, undated := threadsByAge(after)
	if len(dated) > 3 {
		dated = dated[:3]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "this write takes MEMORY.md to %d bytes, past the %d a session start reads, so the newest thread would be the one cut off; archive threads first, then write again.", len(after), MemoryByteBudget)
	if len(dated) == 0 {
		b.WriteString(" No thread carries a date, so none is suggested: choose by hand.")
	} else {
		b.WriteString(" Least recently updated:")
	}
	for _, t := range dated {
		if t.file != "" {
			fmt.Fprintf(&b, "\n  %s (%s): move %s/%s to archive/ and delete its entry from MEMORY.md", t.key, t.date, threadsDir, t.file)
		} else {
			fmt.Fprintf(&b, "\n  %s (%s): move its block from MEMORY.md to archive/", t.key, t.date)
		}
	}
	if len(undated) > 0 {
		fmt.Fprintf(&b, "\nUndated, so not suggested: %s", strings.Join(undated, ", "))
	}
	return errors.New(b.String())
}

// threadAge is a thread's key, the date it was last known to change, and its
// thread file when its entry points at one.
type threadAge struct{ key, date, file string }

// isoDate finds a YYYY-MM-DD date in a field.
var isoDate = regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2})\b`)

// threadsByAge splits the section's threads into the dated ones, oldest first,
// and the keys of the undated ones in file order. A thread's date is its
// Updated line, or failing that the date its Journal file is named with.
func threadsByAge(content string) ([]threadAge, []string) {
	lines := strings.Split(content, "\n")
	start, end := handoffSection(lines)
	var dated []threadAge
	var undated []string
	for i := start + 1; start >= 0 && i < end; i++ {
		key, ok := threadHeadingKey(lines[i])
		if !ok {
			continue
		}
		s, e := threadSpan(lines, start, end, key)
		block := strings.Join(lines[s:e], "\n")
		th := ParseThread(block)
		var date string
		for _, label := range []string{"Updated", "Journal"} {
			if v := th.All(label); v != "" {
				if m := isoDate.FindStringSubmatch(v); m != nil {
					date = m[1]
					break
				}
			}
		}
		if date == "" {
			undated = append(undated, key)
			continue
		}
		dated = append(dated, threadAge{key, date, pointerIn(block)})
	}
	sort.SliceStable(dated, func(a, b int) bool { return dated[a].date < dated[b].date })
	return dated, undated
}

// ReplaceFile writes content over path through a temp file in the same
// directory, creating the directory: a half-written MEMORY.md or thread file is
// the one outcome worse than a clobbered one, and both are read at the start of
// every session.
//
// The file keeps the mode it had, and a new one takes newMode. os.CreateTemp
// makes the temp file 0600, and renaming it over MEMORY.md used to narrow every
// file handoff-write wrote; a file shared with other tools is not ours to
// re-permission, the rule harness bind follows for settings files.
func ReplaceFile(path, content string, newMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("stage the write: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".handoff-*")
	if err != nil {
		return fmt.Errorf("stage the write: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // a no-op once the rename succeeds
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("stage the write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("stage the write: %w", err)
	}
	if fi, err := os.Stat(path); err == nil {
		newMode = fi.Mode().Perm()
	}
	if err := os.Chmod(tmpName, newMode); err != nil {
		return fmt.Errorf("stage the write: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
