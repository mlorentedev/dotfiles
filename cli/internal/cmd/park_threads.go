package cmd

import (
	"fmt"
	"io"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// reportParkThreads prints what a park did to its lane's handoff threads and
// turns an archive that failed into the command's error: the lane is parked
// either way, so the failure is said, never discarded (MEMORY-020).
func reportParkThreads(w io.Writer, lanePath string, a mem.ParkArchive, err error) error {
	for _, k := range a.Archived {
		_, _ = fmt.Fprintf(w, "archived   thread %q to %s\n", k, a.Path)
	}
	for _, k := range a.Kept {
		_, _ = fmt.Fprintf(w, "kept       thread %q in MEMORY.md: it still has Awaiting Manu items\n", k)
	}
	if err != nil {
		return parkArchiveError(lanePath, err)
	}
	return nil
}

// reportSweepThreads is reportParkThreads for a sweep's parks.
func reportSweepThreads(w io.Writer, r *worktree.SweepReport) error {
	for _, k := range r.ArchivedThreads {
		_, _ = fmt.Fprintf(w, "  - archived thread: %s\n", k)
	}
	for _, k := range r.KeptThreads {
		_, _ = fmt.Fprintf(w, "  - kept thread: %s (it still has Awaiting Manu items)\n", k)
	}
	if len(r.ArchiveFailures) == 0 {
		return nil
	}
	for _, f := range r.ArchiveFailures[1:] {
		_, _ = fmt.Fprintf(w, "  - %v\n", parkArchiveError(f.Path, fmt.Errorf("%s", f.Reason)))
	}
	f := r.ArchiveFailures[0]
	return parkArchiveError(f.Path, fmt.Errorf("%s", f.Reason))
}

func parkArchiveError(lanePath string, err error) error {
	return fmt.Errorf("ERROR: lane %s is parked, but its handoff threads were not archived. WHY: %w. FIX: clear the cause, then have each thread's writer archive it with `dotf mem handoff-write --finished`", lanePath, err)
}
