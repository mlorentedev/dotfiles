package lane

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

func runGitLaneTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s failed: %v\n%s", args, dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func setupLaneTestRepo(t *testing.T) (repoDir, wtRoot string) {
	t.Helper()
	root := t.TempDir()
	originDir := filepath.Join(root, "origin.git")
	repoDir = filepath.Join(root, "dotfiles")
	wtRoot = filepath.Join(root, "worktrees")

	if err := os.MkdirAll(originDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, originDir, "init", "--bare", "-b", "main")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, repoDir, "init", "-b", "main")
	runGitLaneTest(t, repoDir, "remote", "add", "origin", originDir)
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, repoDir, "add", "README.md")
	runGitLaneTest(t, repoDir, "commit", "-m", "initial")
	runGitLaneTest(t, repoDir, "push", "-u", "origin", "main")
	return repoDir, wtRoot
}

func TestLaneInitIdempotent(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)

	changed, err := InitLanes(repoDir, wtRoot)
	if err != nil {
		t.Fatalf("first InitLanes failed: %v", err)
	}
	if changed != 3 {
		t.Fatalf("first InitLanes changed = %d, want 3", changed)
	}

	for _, name := range CanonicalLanes {
		lanePath := filepath.Join(wtRoot, "dotfiles", name)
		meta, err := worktree.LoadMetadata(lanePath)
		if err != nil || meta == nil {
			t.Fatalf("LoadMetadata(%s) failed: %v", lanePath, err)
		}
		if meta.Lane != name || meta.State != "PARKED" || meta.ReapOK {
			t.Errorf("lane %s metadata = %+v, want Lane=%s State=PARKED ReapOK=false", name, meta, name)
		}
	}

	// Second call must be idempotent (changed == 0).
	changed2, err := InitLanes(repoDir, wtRoot)
	if err != nil {
		t.Fatalf("second InitLanes failed: %v", err)
	}
	if changed2 != 0 {
		t.Fatalf("second InitLanes changed = %d, want 0 (idempotent)", changed2)
	}
}

// The marker the explicit writer produces carries no paths, so it is valid on
// any OS, host and seat, and a re-run leaves it byte for byte as it was.
func TestWriteLanesMarkerIsMachineIndependentAndStable(t *testing.T) {
	repoDir, _ := setupLaneTestRepo(t)
	if changed, err := WriteLanesMarker(repoDir); err != nil || !changed {
		t.Fatalf("first write = %v, %v; want changed", changed, err)
	}
	markerPath := filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))
	firstMarker, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	var marker lanesMarker
	if err := json.Unmarshal(firstMarker, &marker); err != nil {
		t.Fatalf("%s is not valid JSON: %v\n%s", LanesMarkerFile, err, firstMarker)
	}
	if len(marker.CanonicalLanes) != len(CanonicalLanes) {
		t.Errorf("%s canonical_lanes = %v, want %v", LanesMarkerFile, marker.CanonicalLanes, CanonicalLanes)
	}
	if len(marker.ThirdPartyRoots) == 0 {
		t.Errorf("%s third_party_roots is empty; the guard needs at least one declared root", LanesMarkerFile)
	}
	for _, v := range append(append([]string{}, marker.CanonicalLanes...), marker.ThirdPartyRoots...) {
		if strings.ContainsAny(v, `/\:`) {
			t.Errorf("%s value %q looks like a path; the marker must stay machine-independent", LanesMarkerFile, v)
		}
	}
	if changed, err := WriteLanesMarker(repoDir); err != nil || changed {
		t.Fatalf("second write = %v, %v; want unchanged", changed, err)
	}
	if secondMarker, _ := os.ReadFile(markerPath); !bytes.Equal(firstMarker, secondMarker) {
		t.Errorf("%s changed on re-run:\nfirst:  %s\nsecond: %s", LanesMarkerFile, firstMarker, secondMarker)
	}
}

// TestLaneInitPreservesDeclaredThirdPartyRoots pins the ownership split: the
// marker's canonical_lanes is policy rewritten from code on every write, while
// third_party_roots is host-owned data whose manual entries survive a re-run.
// Without that, "adding a session host is a data change" would be false.
func TestLaneInitPreservesDeclaredThirdPartyRoots(t *testing.T) {
	repoDir, _ := setupLaneTestRepo(t)

	markerPath := filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markerPath, []byte(`{"version":1,"canonical_lanes":["stale-lane"],"third_party_roots":["acme-sessions"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := WriteLanesMarker(repoDir); err != nil {
		t.Fatalf("WriteLanesMarker: %v", err)
	}

	body, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("reading %s: %v", LanesMarkerFile, err)
	}
	var marker lanesMarker
	if err := json.Unmarshal(body, &marker); err != nil {
		t.Fatalf("%s is not valid JSON: %v\n%s", LanesMarkerFile, err, body)
	}

	for _, want := range append(append([]string{}, ThirdPartyWorktreeRoots...), "acme-sessions") {
		if !slices.Contains(marker.ThirdPartyRoots, want) {
			t.Errorf("third_party_roots = %v, want it to retain %q", marker.ThirdPartyRoots, want)
		}
	}
	if slices.Contains(marker.CanonicalLanes, "stale-lane") {
		t.Errorf("canonical_lanes = %v, want the code-owned list to replace the stale entry", marker.CanonicalLanes)
	}
	if !slices.Equal(marker.CanonicalLanes, CanonicalLanes) {
		t.Errorf("canonical_lanes = %v, want %v", marker.CanonicalLanes, CanonicalLanes)
	}

	// The pre-commit guard is a line-based POSIX shell parser, so the marker must
	// stay indented with one array element per line. A compact rewrite would
	// silently empty every list the guard reads.
	if !strings.Contains(string(body), "\n    \"") {
		t.Errorf("%s is not indented one element per line, which the guard parser requires:\n%s", LanesMarkerFile, body)
	}
}

func TestLaneSwitchRoutingAndWIPCap(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes: %v", err)
	}

	// 1. First feat/ branch routes to feat-lane-primary.
	lane1, path1, err := SwitchLane(repoDir, wtRoot, "feat/101-first", "")
	if err != nil {
		t.Fatalf("SwitchLane(feat/101-first) failed: %v", err)
	}
	if lane1 != LaneFeatPrimary {
		t.Errorf("lane1 = %q, want %q", lane1, LaneFeatPrimary)
	}
	meta1, err := worktree.LoadMetadata(path1)
	if err != nil || meta1 == nil {
		t.Fatalf("LoadMetadata(%s): %v", path1, err)
	}
	if meta1.State != "ACTIVE" || meta1.Issue != 101 {
		t.Errorf("meta1 = %+v, want State=ACTIVE Issue=101", meta1)
	}

	// 2. Second refactor/ branch routes to feat-lane-secondary.
	lane2, _, err := SwitchLane(repoDir, wtRoot, "refactor/102-second", "")
	if err != nil {
		t.Fatalf("SwitchLane(refactor/102-second) failed: %v", err)
	}
	if lane2 != LaneFeatSecondary {
		t.Errorf("lane2 = %q, want %q", lane2, LaneFeatSecondary)
	}

	// 3. Third feat/ branch must fail closed with WIP cap error (both feat lanes ACTIVE).
	if _, _, err := SwitchLane(repoDir, wtRoot, "feat/103-overflow", ""); err == nil {
		t.Fatalf("expected SwitchLane(feat/103-overflow) to fail closed when both feat lanes are ACTIVE")
	}

	// 4. fix/ branch routes to fast-track-lane even when both feat lanes are ACTIVE.
	lane3, _, err := SwitchLane(repoDir, wtRoot, "fix/104-urgent", "")
	if err != nil {
		t.Fatalf("SwitchLane(fix/104-urgent) failed: %v", err)
	}
	if lane3 != LaneFastTrack {
		t.Errorf("lane3 = %q, want %q", lane3, LaneFastTrack)
	}
}

func TestLaneInitExcludesMetadataThroughTheCommonGitDir(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)

	// A linked worktree's .git is a pointer FILE, so resolving info/exclude as
	// <repo>/.git/info/exclude names a path that cannot exist and the exclusion
	// silently did nothing there — BUG-043's shape, on the Go side. Init from
	// the linked worktree: it must not fail, and the exclusion must land in the
	// COMMON dir, which is the one git actually reads.
	linked := filepath.Join(t.TempDir(), "linked")
	runGitLaneTest(t, repoDir, "worktree", "add", "-b", "side", linked)

	if _, err := InitLanes(linked, wtRoot); err != nil {
		t.Fatalf("InitLanes from a linked worktree failed: %v", err)
	}

	common := runGitLaneTest(t, linked, "rev-parse", "--path-format=absolute", "--git-common-dir")
	data, err := os.ReadFile(filepath.Join(common, "info", "exclude"))
	if err != nil {
		t.Fatalf("reading %s/info/exclude: %v", common, err)
	}
	for _, name := range []string{worktree.MetadataFileName, PromptFile} {
		if !strings.Contains(string(data), name) {
			t.Errorf("the common dir's info/exclude is missing %q:\n%s", name, data)
		}
	}
	if _, err := os.Stat(filepath.Join(linked, ".git", "info", "exclude")); err == nil {
		t.Errorf("%s/.git is a pointer file; the exclusion was written there instead of the common dir", linked)
	}
}

// Each lane's identity lives in its git admin dir, written by init on every
// exit of provisionLane: create, adopt, and the "already healthy" early return
// (so a record deleted by hand is restored).
func TestLaneInitRecordsTheIdentityInEachLanesGitDir(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes failed: %v", err)
	}
	for _, name := range CanonicalLanes {
		lanePath := filepath.Join(wtRoot, "dotfiles", name)
		got, ok, err := Identity(lanePath)
		if err != nil || !ok || got != name {
			t.Errorf("Identity(%s) = %q, %v, %v; want %q, true, nil", name, got, ok, err, name)
		}
	}
	if changed, err := InitLanes(repoDir, wtRoot); err != nil || changed != 0 {
		t.Fatalf("second InitLanes = %d, %v; want 0, nil (the records are already there)", changed, err)
	}

	// The record is git-owned: it follows the worktree, not the leaf.
	moved := filepath.Join(wtRoot, "dotfiles", "renamed-by-hand")
	runGitLaneTest(t, repoDir, "worktree", "move", filepath.Join(wtRoot, "dotfiles", LaneFastTrack), moved)
	if got, ok, _ := Identity(moved); !ok || got != LaneFastTrack {
		t.Errorf("Identity after worktree move = %q, %v; want %q, true", got, ok, LaneFastTrack)
	}
}

func TestLaneInitRestoresADeletedIdentityRecord(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes failed: %v", err)
	}
	lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatSecondary)
	gitDir := runGitLaneTest(t, lanePath, "rev-parse", "--absolute-git-dir")
	if err := os.Remove(filepath.Join(gitDir, LaneIdentityFile)); err != nil {
		t.Fatal(err)
	}

	changed, err := InitLanes(repoDir, wtRoot)
	if err != nil {
		t.Fatalf("InitLanes after deleting the record failed: %v", err)
	}
	if changed != 1 {
		t.Errorf("InitLanes changed = %d, want 1 (the restored record)", changed)
	}
	if got, ok, _ := Identity(lanePath); !ok || got != LaneFeatSecondary {
		t.Errorf("Identity after re-init = %q, %v; want %q, true", got, ok, LaneFeatSecondary)
	}
	if changed, err := InitLanes(repoDir, wtRoot); err != nil || changed != 0 {
		t.Errorf("third InitLanes = %d, %v; want 0, nil", changed, err)
	}
}

// Two repositories sharing a basename under one WORKTREE_ROOT resolve to the
// same lane paths. The second init must refuse the first's lanes, naming the
// path and the repository they belong to, and leave them untouched.
func TestLaneInitRefusesALaneDirectoryOfAnotherRepository(t *testing.T) {
	firstRepo, wtRoot := setupLaneTestRepo(t)
	secondRepo, _ := setupLaneTestRepo(t)
	if filepath.Base(firstRepo) != filepath.Base(secondRepo) {
		t.Fatalf("fixture: basenames differ (%s, %s)", firstRepo, secondRepo)
	}
	if _, err := InitLanes(firstRepo, wtRoot); err != nil {
		t.Fatalf("first InitLanes failed: %v", err)
	}
	lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
	before, err := os.ReadFile(filepath.Join(lanePath, worktree.MetadataFileName))
	if err != nil {
		t.Fatal(err)
	}

	_, err = InitLanes(secondRepo, wtRoot)
	if err == nil {
		t.Fatal("InitLanes for the second repository succeeded; want a refusal")
	}
	// git answers physical paths; the fixture path may be the logical one.
	physicalFirstRepo, err2 := filepath.EvalSymlinks(firstRepo)
	if err2 != nil {
		t.Fatal(err2)
	}
	for _, want := range []string{lanePath, physicalFirstRepo} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
	// A refused init leaves the second repository unarmed: no marker, no lanes.
	if _, serr := os.Stat(filepath.Join(secondRepo, filepath.FromSlash(LanesMarkerFile))); serr == nil {
		t.Errorf("the refused init wrote %s into the second repository; the guard would be armed with no lanes", LanesMarkerFile)
	}
	// Nor can the second repository claim or park the first's lanes by path.
	if _, _, serr := SwitchLane(secondRepo, wtRoot, "feat/1-stranger", ""); serr == nil || !strings.Contains(serr.Error(), physicalFirstRepo) {
		t.Errorf("SwitchLane from the second repository = %v; want a refusal naming the owner", serr)
	}
	if _, perr := ParkLane(secondRepo, lanePath, "", false); perr == nil || !strings.Contains(perr.Error(), physicalFirstRepo) {
		t.Errorf("ParkLane from the second repository = %v; want a refusal naming the owner", perr)
	}
	if head := runGitLaneTest(t, lanePath, "rev-parse", "--abbrev-ref", "HEAD"); head != "HEAD" {
		t.Errorf("the first repository's lane is no longer detached (HEAD=%s); the stranger checked a branch out in it", head)
	}
	after, err := os.ReadFile(filepath.Join(lanePath, worktree.MetadataFileName))
	if err != nil || !bytes.Equal(before, after) {
		t.Errorf("the first repository's lane metadata changed (err=%v):\nbefore: %s\nafter:  %s", err, before, after)
	}
	if got, ok, _ := Identity(lanePath); !ok || got != LaneFeatPrimary {
		t.Errorf("the first repository's lane identity = %q, %v; want %q, true", got, ok, LaneFeatPrimary)
	}
	if HasNamedLanes(secondRepo, wtRoot) {
		t.Error("HasNamedLanes(secondRepo) = true; the second repository must not have adopted the first's lanes")
	}
}

func TestLaneInitRefusesAPlainDirectoryAtALanePath(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
	if err := os.MkdirAll(lanePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lanePath, "notes.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := InitLanes(repoDir, wtRoot)
	if err == nil {
		t.Fatal("InitLanes over a plain directory succeeded; want a refusal")
	}
	if !strings.Contains(err.Error(), lanePath) {
		t.Errorf("error %q does not name %q", err, lanePath)
	}
	if _, serr := os.Stat(filepath.Join(lanePath, "notes.txt")); serr != nil {
		t.Errorf("the plain directory's content was touched: %v", serr)
	}
	if _, serr := os.Stat(filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))); serr == nil {
		t.Errorf("the refused init wrote %s; the guard would be armed with no lanes", LanesMarkerFile)
	}
}

// A lane root that denies traversal makes Lstat fail with something other
// than absence. Reading that as "absent" would write the marker and then
// fail to provision, so it is an error and nothing is written.
func TestLaneInitRefusesAnUntraversableLaneRootBeforeWritingTheMarker(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("directory modes do not deny the owner on Windows or root")
	}
	repoDir, wtRoot := setupLaneTestRepo(t)
	repoWTRoot := filepath.Join(wtRoot, "dotfiles")
	if err := os.MkdirAll(repoWTRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(repoWTRoot, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(repoWTRoot, 0o755) })

	_, err := InitLanes(repoDir, wtRoot)
	if err == nil || !strings.Contains(err.Error(), "inspecting") {
		t.Fatalf("InitLanes under an untraversable lane root: err=%v, want a refusal naming the inspection", err)
	}
	if _, serr := os.Stat(filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))); serr == nil {
		t.Errorf("the refused init wrote %s; the guard would be armed with no lanes", LanesMarkerFile)
	}
}

// A dangling symlink at a lane path is a directory entry os.Stat does not
// see, so a preflight that skips what Stat cannot reach writes the marker
// and then fails at git worktree add: the refused init the guard must not
// arm behind.
func TestLaneInitRefusesADanglingSymlinkAtALanePathBeforeWritingTheMarker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs a privilege on Windows")
	}
	repoDir, wtRoot := setupLaneTestRepo(t)
	lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
	if err := os.MkdirAll(filepath.Dir(lanePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(wtRoot, "nowhere"), lanePath); err != nil {
		t.Fatal(err)
	}

	_, err := InitLanes(repoDir, wtRoot)
	if err == nil {
		t.Fatal("InitLanes over a dangling symlink succeeded; want a refusal")
	}
	if !strings.Contains(err.Error(), lanePath) {
		t.Errorf("error %q does not name %q", err, lanePath)
	}
	if _, serr := os.Stat(filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))); serr == nil {
		t.Errorf("the refused init wrote %s; the guard would be armed with no lanes", LanesMarkerFile)
	}
}

// A symlink at a lane path that resolves to a worktree of this repository is
// adopted, and coherently: every path comparison resolves symlinks, so the
// lease and the record land in the target and the lane reads back by either
// name. Characterises the review question of whether the adoption is safe.
func TestLaneInitAdoptsASymlinkToThisRepositorysWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs a privilege on Windows")
	}
	repoDir, wtRoot := setupLaneTestRepo(t)
	target := filepath.Join(wtRoot, "elsewhere", LaneFeatPrimary)
	runGitLaneTest(t, repoDir, "worktree", "add", "--detach", target, "HEAD")
	lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatPrimary)
	if err := os.MkdirAll(filepath.Dir(lanePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, lanePath); err != nil {
		t.Fatal(err)
	}

	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes over a symlink to this repository's worktree: %v", err)
	}
	meta, err := worktree.LoadMetadata(target)
	if err != nil || meta == nil || meta.Lane != LaneFeatPrimary {
		t.Fatalf("lease in the target: meta=%+v err=%v", meta, err)
	}
	for _, p := range []string{lanePath, target} {
		name, ok, ierr := Identity(p)
		if ierr != nil || !ok || name != LaneFeatPrimary {
			t.Errorf("Identity(%s) = %q, %v, %v; want %q", p, name, ok, ierr, LaneFeatPrimary)
		}
	}
	changed, err := InitLanes(repoDir, wtRoot)
	if err != nil || changed != 0 {
		t.Errorf("second init: changed=%d err=%v; want 0, nil", changed, err)
	}
}

// A lane whose lease a binary before ADR-062 wrote in the tree gets it copied
// into the worktree's git dir by the next init, counted as a change; the
// mirror's presence or absence, once the admin dir holds the lease, is not.
func TestLaneInitMigratesInTreeLeases(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes failed: %v", err)
	}
	lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatSecondary)
	admin, err := worktree.AdminDir(lanePath)
	if err != nil {
		t.Fatal(err)
	}
	adminLease := filepath.Join(admin, worktree.LeaseFileName)
	before, err := os.ReadFile(adminLease)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(adminLease); err != nil {
		t.Fatal(err)
	}

	changed, err := InitLanes(repoDir, wtRoot)
	if err != nil {
		t.Fatalf("InitLanes over a legacy-only lease failed: %v", err)
	}
	if changed != 1 {
		t.Errorf("InitLanes changed = %d, want 1 (the migrated lease)", changed)
	}
	after, err := os.ReadFile(adminLease)
	if err != nil {
		t.Fatalf("the lease was not copied into the admin dir: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("the migrated lease differs from the in-tree one:\nbefore: %s\nafter:  %s", before, after)
	}
	if changed, err := InitLanes(repoDir, wtRoot); err != nil || changed != 0 {
		t.Errorf("InitLanes with the admin-dir lease and the mirror still present = %d, %v; want 0, nil", changed, err)
	}

	if err := os.Remove(filepath.Join(lanePath, worktree.MetadataFileName)); err != nil {
		t.Fatal(err)
	}
	if changed, err := InitLanes(repoDir, wtRoot); err != nil || changed != 0 {
		t.Errorf("InitLanes with the admin-dir lease and no mirror = %d, %v; want 0, nil", changed, err)
	}
}
