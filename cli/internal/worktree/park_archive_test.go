package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

// parkedLaneFixture is a named lane on branch feat/x whose tip has landed, a
// vault whose MEMORY.md holds that branch's thread, and the options to park it.
func parkedLaneFixture(t *testing.T) (DoneOptions, string) {
	t.Helper()
	root := t.TempDir()
	originDir := filepath.Join(root, "origin.git")
	repoDir := filepath.Join(root, "dotfiles")
	wtDir := filepath.Join(root, "worktrees", "dotfiles", "lane-a")
	for _, d := range []string{originDir, repoDir, filepath.Dir(wtDir)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	runGitParkTest(t, originDir, "init", "--bare", "-b", "main")
	runGitParkTest(t, repoDir, "init", "-b", "main")
	runGitParkTest(t, repoDir, "remote", "add", "origin", originDir)
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, repoDir, "add", "README.md")
	runGitParkTest(t, repoDir, "commit", "-m", "initial")
	runGitParkTest(t, repoDir, "push", "-u", "origin", "main")
	runGitParkTest(t, repoDir, "worktree", "add", "--no-track", "-b", "feat/x", wtDir, "origin/main")
	ensureExclude(repoDir, MetadataFileName)
	if err := os.WriteFile(filepath.Join(wtDir, "f.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, wtDir, "add", "f.txt")
	runGitParkTest(t, wtDir, "commit", "-m", "work")
	tip := runGitParkTest(t, wtDir, "rev-parse", "HEAD")
	if err := SaveMetadata(wtDir, Metadata{Lane: "lane-a", State: "ACTIVE", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	vaultDir := filepath.Join(root, "vault")
	memDir := filepath.Join(vaultDir, "10_projects", "dotfiles", "memory")
	if err := os.MkdirAll(filepath.Join(memDir, "threads"), 0o755); err != nil {
		t.Fatal(err)
	}
	memory := "# M\n\n## Session Handoff\n\n### thread: feat-x (writer: claude)\n\n**Next action:** None.<br>\n[full thread](threads/feat-x.md)\n"
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(memory), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "threads", "feat-x.md"), []byte("### thread: feat-x (writer: claude)\n\n**Next action:** None.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return DoneOptions{
		RepoRoot:            repoDir,
		WorktreePath:        wtDir,
		LockPath:            filepath.Join(root, "wt.lock"),
		IgnoreRecentSession: true,
		VaultPath:           vaultDir,
		LandedHeads:         func(_, _ string) ([]string, error) { return []string{tip}, nil },
	}, memDir
}

// A park archives its branch's thread to the dated archive and reports it
// (MEMORY-020 AC1).
func TestParkArchivesTheBranchThread(t *testing.T) {
	opts, memDir := parkedLaneFixture(t)
	res, err := Park(opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.ThreadsErr != nil || strings.Join(res.Threads.Archived, ",") != "feat-x" {
		t.Fatalf("threads %+v, err %v; want feat-x archived", res.Threads, res.ThreadsErr)
	}
	archive, err := os.ReadFile(res.Threads.Path)
	if err != nil || !strings.Contains(string(archive), "[full thread](threads/feat-x.md)") {
		t.Errorf("archive %s holds %q (%v)", res.Threads.Path, archive, err)
	}
	if _, err := os.Stat(filepath.Join(memDir, "archive", "threads", "feat-x.md")); err != nil {
		t.Errorf("thread file not moved to archive/threads/: %v", err)
	}
	if memory, _ := os.ReadFile(filepath.Join(memDir, "MEMORY.md")); strings.Contains(string(memory), "feat-x") {
		t.Errorf("MEMORY.md still holds the thread:\n%s", memory)
	}
}

// A park whose archive fails still parks, and carries the failure to its
// caller instead of discarding it (MEMORY-020 AC4).
func TestParkReportsAFailedArchiveAndStillParks(t *testing.T) {
	opts, memDir := parkedLaneFixture(t)
	// A file where the archive directory goes: the dated archive cannot be written.
	if err := os.WriteFile(filepath.Join(memDir, "archive"), []byte("in the way\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Park(opts)
	if err != nil {
		t.Fatalf("the park itself failed: %v", err)
	}
	if res.ThreadsErr == nil {
		t.Fatal("a failed archive was not reported")
	}
	if res.DeletedBranch != "feat/x" {
		t.Errorf("DeletedBranch %q, want the lane parked and feat/x deleted", res.DeletedBranch)
	}
	if meta, _ := LoadMetadata(opts.WorktreePath); meta == nil || meta.State != "PARKED" {
		t.Errorf("lane state %+v, want PARKED", meta)
	}
	if memory, _ := os.ReadFile(filepath.Join(memDir, "MEMORY.md")); !strings.Contains(string(memory), "feat-x") {
		t.Error("MEMORY.md lost the thread although its archive failed")
	}
}

// Without a vault, a park archives nothing and reports nothing.
func TestParkWithoutAVaultLeavesMemoryAlone(t *testing.T) {
	opts, memDir := parkedLaneFixture(t)
	opts.VaultPath = ""
	res, err := Park(opts)
	if err != nil || res.ThreadsErr != nil || len(res.Threads.Archived) != 0 {
		t.Fatalf("res %+v err %v", res, err)
	}
	if memory, _ := os.ReadFile(filepath.Join(memDir, "MEMORY.md")); !strings.Contains(string(memory), "feat-x") {
		t.Error("a park with no vault changed MEMORY.md")
	}
}

// A sweep's park folds its archive into the report: archived and kept threads
// are listed, and a failure names the lane (MEMORY-020 AC3, AC4).
func TestSweepReportRecordsThreadArchives(t *testing.T) {
	r := &SweepReport{}
	r.record(Info{Path: "/wt/a"}, sweepOutcome{reaped: true, threads: mem.ParkArchive{Archived: []string{"feat-a"}, Kept: []string{"feat-a+agy"}, Shared: []string{"feat-a"}}})
	r.record(Info{Path: "/wt/b"}, sweepOutcome{reaped: true, threadsErr: errors.New("disk full")})
	if strings.Join(r.ArchivedThreads, ",") != "feat-a" || strings.Join(r.KeptThreads, ",") != "feat-a+agy" {
		t.Errorf("archived %v kept %v", r.ArchivedThreads, r.KeptThreads)
	}
	if strings.Join(r.SharedThreads, ",") != "feat-a" {
		t.Errorf("shared %v, want feat-a (#3521)", r.SharedThreads)
	}
	if len(r.ArchiveFailures) != 1 || r.ArchiveFailures[0].Path != "/wt/b" || r.ArchiveFailures[0].Reason != "disk full" {
		t.Errorf("failures %+v", r.ArchiveFailures)
	}
	if len(r.Reaped) != 2 {
		t.Errorf("a failed archive unparked a lane: reaped %d", len(r.Reaped))
	}
}
