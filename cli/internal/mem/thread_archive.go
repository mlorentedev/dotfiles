package mem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// archiveDir holds archived handoff threads beside MEMORY.md: the entries in
// one dated file per day, the shape every hand archive used (2026-11-02, -05,
// -12), and the thread files under archive/threads/ (MEMORY-020). An archived
// entry keeps its pointer byte-identical: `threads/<key>.md` read from the
// dated file is archive/threads/<key>.md. A file left in threads/ without its
// entry is what `dotf doctor` reports as a lost handoff.
const archiveDir = "archive"

// ArchivedThread is what ArchiveThreadAs did: a StoredThread whose Content no
// longer holds the thread, and the entry to append to the dated archive.
type ArchivedThread struct {
	StoredThread
	Block string // the entry this write would have placed
	// Live is the thread file in threads/ the archived File replaces: the
	// caller removes it once MEMORY.md no longer points at it.
	Live string
}

// ArchiveThreadAs is StoreThreadAs for a thread its writer declares finished
// (MEMORY-019, #3393): the thread file is written as on any write, and the
// entry leaves MEMORY.md for the archive instead of staying there. Ownership
// and forks are StoreThreadAs's, so another agent's finished write archives
// only its own fork. File is the thread file under archive/threads/. Pure: the
// caller appends Block, writes File, writes MEMORY.md, then removes Live.
//
// A thread still awaiting Manu is refused: the archive would take his open
// decisions off the one surface a session start loads.
func ArchiveThreadAs(content, memoryPath, threadKey, agent, body string) (ArchivedThread, error) {
	if awaitsManu(body) {
		return ArchivedThread{}, fmt.Errorf("thread %q still has Awaiting Manu items: a finished thread is archived out of MEMORY.md, where a session start reads them. Answer or move them, then write it with --finished", threadKey)
	}
	stored, err := StoreThreadAs(content, memoryPath, threadKey, agent, body)
	if err != nil {
		return ArchivedThread{}, err
	}
	pruned, taken, _ := takeThreads(stored.Content, stored.Key, false, nil)
	if len(taken) == 0 {
		return ArchivedThread{}, fmt.Errorf("thread %q was not placed, so there is nothing to archive", stored.Key)
	}
	live := stored.File
	stored.File = archivedThreadFile(memoryPath, filepath.Base(live))
	stored.Content = pruned
	stored.Changed = pruned != content
	return ArchivedThread{StoredThread: stored, Block: taken[0].entry, Live: live}, nil
}

// ParkArchive is what a lane park did to its branch's handoff threads.
type ParkArchive struct {
	Path     string   // the dated archive the entries go to
	Archived []string // thread keys moved out of MEMORY.md
	Kept     []string // thread keys left in MEMORY.md: they still await Manu
	// Shared are archived keys whose thread file another key holds (fix/x and
	// fix-x name one file): the file stays, so the archived entry's pointer
	// does not lead to this thread's body (#3521).
	Shared []string
}

// ArchiveParkedThread archives the handoff threads of a parked lane's branch:
// its thread and every <key>+<agent> fork, since a thread is a branch and a
// parked branch's forks record the same finished work (MEMORY-020). A block
// whose thread still has Awaiting Manu items stays: the rule --finished
// applies. The dated archive is written first and the thread files moved
// last, so an interrupted park leaves an entry in both places rather than in
// neither, and at worst a file in threads/ that doctor reports. A key reused
// after an earlier archive replaces the archived file of the same name.
func ArchiveParkedThread(projectDir, branch string, now time.Time) (ParkArchive, error) {
	memoryPath, key, ok := parkTarget(projectDir, branch)
	if !ok {
		return ParkArchive{}, nil
	}
	unlock, err := LockHandoffMemory(memoryPath)
	if err != nil {
		return ParkArchive{}, err
	}
	defer unlock()
	return archiveParked(memoryPath, key, now)
}

// parkTarget is the MEMORY.md and thread key a park of branch archives from,
// or false when there is none: no project, the default branch, no MEMORY.md.
func parkTarget(projectDir, branch string) (string, string, bool) {
	key := sanitizeThread(branch)
	if strings.TrimSpace(projectDir) == "" || key == "" || key == "main" || key == "master" {
		return "", "", false
	}
	memoryPath := filepath.Join(projectDir, "memory", "MEMORY.md")
	if _, err := os.Stat(memoryPath); errors.Is(err, os.ErrNotExist) {
		return "", "", false
	}
	return memoryPath, key, true
}

// archiveParked is ArchiveParkedThread under the MEMORY.md lock.
func archiveParked(memoryPath, key string, now time.Time) (ParkArchive, error) {
	raw, err := os.ReadFile(memoryPath) // #nosec G304 -- the project's own MEMORY.md
	if err != nil {
		return ParkArchive{}, err
	}
	memoryDir := filepath.Dir(memoryPath)
	pruned, taken, kept := takeThreads(string(raw), key, true, func(block string) bool {
		return awaitsManu(threadText(block, memoryDir))
	})
	date := now.Format("2006-01-02")
	res := ParkArchive{Path: ArchivePath(memoryPath, date), Kept: kept}
	if len(taken) == 0 {
		return res, nil
	}
	if err := appendArchiveFile(res.Path, date, taken); err != nil {
		return res, err
	}
	if err := ReplaceFile(memoryPath, pruned, 0o644); err != nil {
		return res, err
	}
	return res, moveThreadFiles(memoryPath, taken, &res)
}

// moveThreadFiles moves every taken thread's file, continuing past a failure so
// one bad file does not strand the others in threads/. Archived lists only the
// threads whose entry and file both moved; the error names each other one.
func moveThreadFiles(memoryPath string, taken []takenThread, res *ParkArchive) error {
	var errs []error
	for _, t := range taken {
		shared, err := moveThreadFile(memoryPath, t.key)
		if err != nil {
			errs = append(errs, fmt.Errorf("thread %q: its entry is archived, but its file stays in %s/, where dotf doctor reports it as a lost handoff: %w", t.key, threadsDir, err))
			continue
		}
		res.Archived = append(res.Archived, t.key)
		if shared {
			res.Shared = append(res.Shared, t.key)
		}
	}
	return errors.Join(errs...)
}

// appendArchiveFile appends the taken entries to the dated archive at path.
func appendArchiveFile(path, date string, taken []takenThread) error {
	existing, err := os.ReadFile(path) // #nosec G304 -- beside the project's MEMORY.md
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := string(existing)
	for _, t := range taken {
		text, _ = AppendArchive(text, date, t.entry)
	}
	if text == string(existing) {
		return nil
	}
	return ReplaceFile(path, text, 0o644)
}

// moveThreadFile moves key's thread file, when it has one, from threads/ to
// archive/threads/. A file whose heading names another key is that thread's
// (fix/x and fix-x name one file): it stays where it is, and shared says so.
// os.Rename replaces an existing target on every platform.
func moveThreadFile(memoryPath, key string) (shared bool, err error) {
	name, err := threadFileName(key)
	if err != nil {
		return false, err
	}
	src := filepath.Join(filepath.Dir(memoryPath), threadsDir, name)
	held, err := ReadThreadFile(src)
	if err != nil || held == nil {
		return false, err // an inline thread has no file
	}
	if CheckThreadFile(held, key) != nil {
		return true, nil // another key's file is not ours to move
	}
	dst := archivedThreadFile(memoryPath, name)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	return false, os.Rename(src, dst)
}

// archivedThreadFile is where an archived thread's file lives: under
// archive/threads/, so the entry's own pointer resolves from the dated file.
func archivedThreadFile(memoryPath, name string) string {
	return filepath.Join(filepath.Dir(memoryPath), archiveDir, threadsDir, name)
}

// ArchivePath is the dated archive file for memoryPath's project.
func ArchivePath(memoryPath, date string) string {
	return filepath.Join(filepath.Dir(memoryPath), archiveDir, "superseded-handoffs-"+date+".md")
}

// AppendArchive adds block to the dated archive text existing, opening a new
// file with its heading. A block the archive already holds is not added again,
// so a repeated finished write leaves the archive as it was.
func AppendArchive(existing, date, block string) (string, bool) {
	if existing == "" {
		return "# Superseded handoffs " + date + "\n\n" + block + "\n", true
	}
	if strings.Contains(existing, block) {
		return existing, false
	}
	return strings.TrimRight(existing, "\r\n") + "\n\n" + block + "\n", true
}

// takenThread is one block takeThreads removed: its key and its archive entry.
type takenThread struct{ key, entry string }

// takeThreads removes key's block from content's handoff section, and with
// forks every <key>+<agent> fork of it too, except a block keep reports true
// for, whose key it returns instead. It is the one extraction both archive
// paths share: a writer archiving its own thread never takes another agent's
// fork (MEMORY-019), a park takes them all.
func takeThreads(content, key string, forks bool, keep func(block string) bool) (string, []takenThread, []string) {
	lines := strings.Split(content, "\n")
	start, end := handoffSection(lines)
	if start < 0 {
		return content, nil, nil
	}
	var taken []takenThread
	var kept []string
	out := append([]string{}, lines[:start+1]...)
	for i := start + 1; i < end; {
		hk, ok := threadHeadingKey(lines[i])
		if !ok || !takesThread(hk, key, forks) {
			out = append(out, lines[i])
			i++
			continue
		}
		j := blockEnd(lines, i, end)
		block := strings.Join(trimBlankEdges(lines[i:j]), "\n")
		if keep != nil && keep(block) {
			kept = append(kept, hk)
			out = append(out, lines[i:j]...)
		} else {
			taken = append(taken, takenThread{key: hk, entry: block})
		}
		i = j
	}
	if len(taken) == 0 {
		return content, nil, kept
	}
	return strings.Join(append(out, lines[end:]...), "\n"), taken, kept
}

// takesThread reports whether the block headed hk is key's, or with forks one
// of key's <key>+<agent> forks.
func takesThread(hk, key string, forks bool) bool {
	return hk == key || forks && strings.HasPrefix(hk, key+"+")
}

// blockEnd is the index of the first thread heading after line i, or end.
func blockEnd(lines []string, i, end int) int {
	j := i + 1
	for j < end {
		if _, next := threadHeadingKey(lines[j]); next {
			break
		}
		j++
	}
	return j
}

// awaitsManu reports whether a thread's text still has Awaiting Manu items.
func awaitsManu(text string) bool {
	return strings.TrimSpace(ParseThread(text).All("Awaiting Manu")) != ""
}
