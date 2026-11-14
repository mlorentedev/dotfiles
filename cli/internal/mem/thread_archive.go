package mem

import (
	"fmt"
	"path/filepath"
	"strings"
)

// archiveDir holds archived handoff threads beside MEMORY.md, one dated file
// per day: the shape every hand archive used (2026-11-02, -05, -12).
const archiveDir = "archive"

// ArchivedThread is what ArchiveThreadAs did: a StoredThread whose Content no
// longer holds the thread, and the block to append to the dated archive.
type ArchivedThread struct {
	StoredThread
	Block string // the block this write would have placed in MEMORY.md
}

// ArchiveThreadAs is StoreThreadAs for a thread its writer declares finished
// (MEMORY-019, #3393): the thread file is written as on any write, and the
// entry leaves MEMORY.md for the archive instead of staying there. Ownership
// and forks are StoreThreadAs's, so another agent's finished write archives
// only its own fork. Pure: the caller appends Block, then writes the thread
// file, then MEMORY.md.
//
// A thread still awaiting Manu is refused: the archive would take his open
// decisions off the one surface a session start loads.
func ArchiveThreadAs(content, memoryPath, threadKey, agent, body string) (ArchivedThread, error) {
	if strings.TrimSpace(ParseThread(body).All("Awaiting Manu")) != "" {
		return ArchivedThread{}, fmt.Errorf("thread %q still has Awaiting Manu items: a finished thread is archived out of MEMORY.md, where a session start reads them. Answer or move them, then write it with --finished", threadKey)
	}
	stored, err := StoreThreadAs(content, memoryPath, threadKey, agent, body)
	if err != nil {
		return ArchivedThread{}, err
	}
	lines := strings.Split(stored.Content, "\n")
	start, end := handoffSection(lines)
	s, e := threadSpan(lines, start, end, stored.Key)
	if s < 0 {
		return ArchivedThread{}, fmt.Errorf("thread %q was not placed, so there is nothing to archive", stored.Key)
	}
	block := strings.Join(trimBlankEdges(lines[s:e]), "\n")
	pruned, _ := pruneThreadFromMemory(stored.Content, stored.Key, false)
	stored.Content = pruned
	stored.Changed = pruned != content
	return ArchivedThread{StoredThread: stored, Block: block}, nil
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
	return strings.TrimRight(existing, "\n") + "\n\n" + block + "\n", true
}
