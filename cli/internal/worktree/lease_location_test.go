package worktree

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// linkedWorktree makes a repository with one commit and one linked worktree
// on branch, and returns both paths with symlinks resolved, as git answers.
func linkedWorktree(t *testing.T, branch string) (repo, wt string) {
	t.Helper()
	tmp := t.TempDir()
	repo = filepath.Join(tmp, "repo")
	gitT(t, tmp, "init", "-q", "-b", "main", repo)
	gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	wt = filepath.Join(tmp, "wt")
	gitT(t, repo, "worktree", "add", "-q", "-b", branch, wt)
	var err error
	if repo, err = filepath.EvalSymlinks(repo); err != nil {
		t.Fatal(err)
	}
	if wt, err = filepath.EvalSymlinks(wt); err != nil {
		t.Fatal(err)
	}
	return repo, wt
}

// AdminDir answers the worktree's own git admin dir from its .git pointer,
// absolute or relative, and refuses everything that is not a linked worktree.
func TestAdminDir(t *testing.T) {
	repo, wt := linkedWorktree(t, "feat/x")
	want := gitT(t, wt, "rev-parse", "--path-format=absolute", "--git-dir")
	if want, _ = filepath.EvalSymlinks(want); want == "" {
		t.Fatal("git answered no git dir")
	}

	got, err := AdminDir(wt)
	if err != nil || got != want {
		t.Fatalf("absolute pointer: got %q, %v; want %q", got, err, want)
	}

	gitT(t, repo, "worktree", "repair")
	rel, err := filepath.Rel(wt, want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+rel+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = AdminDir(wt)
	if err != nil || got != want {
		t.Fatalf("relative pointer: got %q, %v; want %q", got, err, want)
	}

	plain := t.TempDir()
	sub := filepath.Join(t.TempDir(), "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, ".git"), []byte("gitdir: ../.git/modules/sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"main checkout": repo, "plain directory": plain, "submodule": sub} {
		got, err := AdminDir(path)
		if !errors.Is(err, ErrNotLinkedWorktree) || got != "" {
			t.Errorf("%s: got %q, %v; want ErrNotLinkedWorktree", name, got, err)
		}
		if err != nil && !strings.Contains(err.Error(), path) {
			t.Errorf("%s: error %q does not name the path", name, err)
		}
	}
}

// The lease lives in the admin dir; the in-tree file is a mirror for one
// release and the fallback for a lease written before this change.
func TestLeaseLivesInTheWorktreeGitDir(t *testing.T) {
	_, wt := linkedWorktree(t, "feat/y")
	admin, err := AdminDir(wt)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	meta := Metadata{Creator: "t", Issue: 3307, CreatedAt: now, LeaseExpiresAt: now.Add(time.Hour), ReapOK: true}

	if err := SaveMetadata(wt, meta); err != nil {
		t.Fatalf("Save: %v", err)
	}
	adminBytes, err := os.ReadFile(filepath.Join(admin, LeaseFileName))
	if err != nil {
		t.Fatalf("Save did not write the admin dir: %v", err)
	}
	treeBytes, err := os.ReadFile(filepath.Join(wt, MetadataFileName))
	if err != nil {
		t.Fatalf("Save did not write the in-tree mirror: %v", err)
	}
	if string(adminBytes) != string(treeBytes) {
		t.Fatal("the mirror differs from the admin-dir lease")
	}

	// The admin dir wins when the two disagree.
	stale := meta
	stale.Issue = 1
	if err := os.WriteFile(filepath.Join(wt, MetadataFileName), mustJSON(t, stale), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadMetadata(wt)
	if err != nil || got == nil || got.Issue != 3307 {
		t.Fatalf("Load with both: got %+v, %v; want the admin-dir lease (issue 3307)", got, err)
	}

	// Legacy-only: the admin dir holds none, the in-tree file is read.
	if err := os.Remove(filepath.Join(admin, LeaseFileName)); err != nil {
		t.Fatal(err)
	}
	got, err = LoadMetadata(wt)
	if err != nil || got == nil || got.Issue != 1 {
		t.Fatalf("Load legacy-only: got %+v, %v; want the in-tree lease (issue 1)", got, err)
	}

	// Neither: nil, no error.
	if err := os.Remove(filepath.Join(wt, MetadataFileName)); err != nil {
		t.Fatal(err)
	}
	got, err = LoadMetadata(wt)
	if err != nil || got != nil {
		t.Fatalf("Load with neither: got %+v, %v; want nil, nil", got, err)
	}

	// A non-worktree path still reads an in-tree file, the same fallback.
	plain := t.TempDir()
	if err := os.WriteFile(filepath.Join(plain, MetadataFileName), mustJSON(t, stale), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = LoadMetadata(plain)
	if err != nil || got == nil || got.Issue != 1 {
		t.Fatalf("Load on a plain directory: got %+v, %v; want the in-tree lease", got, err)
	}

	// Save refuses a non-worktree path, naming it, and writes nothing there.
	other := t.TempDir()
	err = SaveMetadata(other, meta)
	if !errors.Is(err, ErrNotLinkedWorktree) || !strings.Contains(err.Error(), other) {
		t.Fatalf("Save on a plain directory: %v; want ErrNotLinkedWorktree naming %s", err, other)
	}
	if _, statErr := os.Stat(filepath.Join(other, MetadataFileName)); !os.IsNotExist(statErr) {
		t.Fatal("Save wrote an in-tree lease on a path it refused")
	}
}

func mustJSON(t *testing.T, meta Metadata) []byte {
	t.Helper()
	b, err := marshalMetadata(meta)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A lease that lives only at the legacy in-tree location is named by list, so
// the report says which worktrees the move has not reached; once the admin
// dir holds one, the mirror's presence is not reported (ADR-062).
func TestListNamesALegacyLocationLease(t *testing.T) {
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	now := time.Now()
	meta := Metadata{ReapOK: true, CreatedAt: now.Add(-time.Hour), LeaseExpiresAt: now.Add(time.Hour)}
	if err := os.WriteFile(filepath.Join(wtDir, MetadataFileName), mustJSON(t, meta), 0o644); err != nil {
		t.Fatal(err)
	}
	list := func() Info {
		t.Helper()
		infos, err := List(repoDir, func(string, string) ([]string, error) { return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
		return featInfo(t, infos)
	}

	legacy := list()
	if !legacy.LeaseLegacyOnly || !strings.Contains(legacy.StateReason, "legacy") {
		t.Errorf("legacy-only lease: LeaseLegacyOnly=%v reason=%q; want true and a reason naming the legacy location", legacy.LeaseLegacyOnly, legacy.StateReason)
	}

	if err := SaveMetadata(wtDir, meta); err != nil {
		t.Fatal(err)
	}
	moved := list()
	if moved.LeaseLegacyOnly || strings.Contains(moved.StateReason, "legacy") {
		t.Errorf("admin-dir lease with the mirror present: LeaseLegacyOnly=%v reason=%q; want false and no legacy note", moved.LeaseLegacyOnly, moved.StateReason)
	}
}

// A Save whose admin-dir write fails leaves both copies as they were: the
// mirror an old guard reads must not run ahead of the lease a new reader
// prefers (review finding on #3421).
func TestSaveMetadataPutsTheMirrorBackWhenTheAdminDirRefuses(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file does not refuse a write here")
	}
	_, wt := linkedWorktree(t, "feat/mirror")
	old := Metadata{Lane: "feat-lane-primary", State: string(StateActive), ReapOK: false}
	if err := SaveMetadata(wt, old); err != nil {
		t.Fatal(err)
	}
	admin, err := AdminDir(wt)
	if err != nil {
		t.Fatal(err)
	}
	adminLease := filepath.Join(admin, LeaseFileName)
	if err := os.Chmod(adminLease, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(adminLease, 0o644) })

	if err := SaveMetadata(wt, Metadata{Lane: "feat-lane-primary", State: string(StateParked)}); err == nil {
		t.Fatal("Save with an unwritable admin-dir lease returned nil")
	}
	mirror, err := os.ReadFile(filepath.Join(wt, MetadataFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mirror, mustJSON(t, old)) {
		t.Errorf("the mirror ran ahead of the admin-dir lease after a failed Save:\n%s", mirror)
	}
	if meta, err := LoadMetadata(wt); err != nil || meta == nil || meta.State != string(StateActive) {
		t.Errorf("LoadMetadata after the failed Save = %+v, %v; want the previous ACTIVE lease", meta, err)
	}

	// With no mirror before the failed Save, none is left behind.
	if err := os.Remove(filepath.Join(wt, MetadataFileName)); err != nil {
		t.Fatal(err)
	}
	if err := SaveMetadata(wt, Metadata{Lane: "feat-lane-primary", State: string(StateParked)}); err == nil {
		t.Fatal("Save with an unwritable admin-dir lease returned nil")
	}
	if _, err := os.Stat(filepath.Join(wt, MetadataFileName)); !os.IsNotExist(err) {
		t.Errorf("a failed Save left a mirror behind (err=%v)", err)
	}
}
