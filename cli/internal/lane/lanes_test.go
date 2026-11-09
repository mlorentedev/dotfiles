package lane

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

	// The opt-in marker is the lane system's own: written by init, carrying no
	// paths so it is valid on any OS, host and seat.
	markerPath := filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))
	firstMarker, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("lane init did not write %s: %v", LanesMarkerFile, err)
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
	secondMarker, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("marker vanished after second InitLanes: %v", err)
	}
	if !bytes.Equal(firstMarker, secondMarker) {
		t.Errorf("%s changed on re-run:\nfirst:  %s\nsecond: %s", LanesMarkerFile, firstMarker, secondMarker)
	}
}

// TestLaneInitPreservesDeclaredThirdPartyRoots pins the ownership split: the
// marker's canonical_lanes is policy rewritten from code on every init, while
// third_party_roots is host-owned data whose manual entries survive a re-run.
// Without that, "adding a session host is a data change" would be false.
func TestLaneInitPreservesDeclaredThirdPartyRoots(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)

	markerPath := filepath.Join(repoDir, filepath.FromSlash(LanesMarkerFile))
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markerPath, []byte(`{"version":1,"canonical_lanes":["stale-lane"],"third_party_roots":["acme-sessions"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes: %v", err)
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
