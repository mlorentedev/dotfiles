package deploy

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// worktreeRoot resolves the one contract variable the herdr entry renders.
func worktreeRoot(name string) string {
	if name == "WORKTREE_ROOT" {
		return "/r/worktrees"
	}
	return ""
}

func herdrWithPaths() Config {
	c := herdrConfig()
	c.Paths = PathsSlash
	return c
}

// #2503: herdr's worktrees.directory is the root `dotf worktree add` uses, so
// it must be rendered from WORKTREE_ROOT, not hardcoded beside it. The source
// is TOML; a string that does not begin with a token (herdr's own `~`) stays
// as it was, and the destination's unmanaged keys survive the merge.
func TestDeploy_PathsRenderATOMLMergeSource(t *testing.T) {
	root := repoWithHerdrConfig(t, "[worktrees]\ndirectory = \"{WORKTREE_ROOT}\"\n\n[ui]\nshell = \"~/bin/zsh\"\n")
	home := t.TempDir()
	dst := writeHerdrDst(t, home, "onboarding = false\n\n[worktrees]\ndirectory = \"~/.herdr/worktrees\"\n")

	out, err := Deploy(herdrWithPaths(), root, home, worktreeRoot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Changed {
		t.Error("a rendered directory that differs is a change")
	}
	got := readTOML(t, dst)
	if dir := got["worktrees"].(map[string]any)["directory"]; dir != "/r/worktrees" {
		t.Errorf("worktrees.directory = %v, want the resolved WORKTREE_ROOT", dir)
	}
	if shell := got["ui"].(map[string]any)["shell"]; shell != "~/bin/zsh" {
		t.Errorf("ui.shell = %v, want the untokened string untouched", shell)
	}
	if got["onboarding"] != false {
		t.Errorf("the unmanaged key was lost: %v", got)
	}
	p, err := PlanConfig(herdrWithPaths(), root, home, worktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	if p.Changed {
		t.Error("a rendered TOML destination must read as in sync")
	}
}

// #2638: `dotf worktree add` follows a new WORKTREE_ROOT at once, while herdr
// keeps the root rendered at deploy time. The plan re-renders the source with
// the root in force now, so the stale destination reads as drift (which
// `dotf doctor` reports) instead of as in sync.
func TestDeploy_AWorktreeRootChangedAfterADeployIsDrift(t *testing.T) {
	root := repoWithHerdrConfig(t, "[worktrees]\ndirectory = \"{WORKTREE_ROOT}\"\n")
	home := t.TempDir()
	writeHerdrDst(t, home, "onboarding = false\n")
	if _, err := Deploy(herdrWithPaths(), root, home, worktreeRoot, nil, false); err != nil {
		t.Fatal(err)
	}
	moved := func(name string) string {
		if name == "WORKTREE_ROOT" {
			return "/r/moved"
		}
		return ""
	}
	p, err := PlanConfig(herdrWithPaths(), root, home, moved)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Changed {
		t.Error("herdr's rendered root is stale after WORKTREE_ROOT moved, and the plan reads it as in sync")
	}
}

// Rendering re-encodes the source, so a value the source declared no interest
// in must come back as it was read: an integer above 2^53 and a datetime.
func TestExpandPaths_TOMLKeepsValuesItDoesNotRender(t *testing.T) {
	out, err := expandPathsAs(tomlPaths, []byte("id = 9007199254740993\nat = 2026-10-16T01:11:00Z\ndir = \"{WORKTREE_ROOT}/x\"\n"), PathsSlash, "/h", worktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal(out, &got); err != nil {
		t.Fatalf("rendered source is not TOML: %v\n%s", err, out)
	}
	if got["id"] != int64(9007199254740993) {
		t.Errorf("id = %v (%T), want the integer as read", got["id"], got["id"])
	}
	if at, ok := got["at"].(time.Time); !ok || !at.Equal(time.Date(2026, 10, 16, 1, 11, 0, 0, time.UTC)) {
		t.Errorf("at = %v (%T), want the datetime as read", got["at"], got["at"])
	}
	if got["dir"] != "/r/worktrees/x" {
		t.Errorf("dir = %v, want the rendered root", got["dir"])
	}
}

func TestExpandPaths_TOMLRejectsWhatItCannotRender(t *testing.T) {
	if _, err := expandPathsAs(tomlPaths, []byte("a = ["), PathsSlash, "/h", noResolve); err == nil || !strings.Contains(err.Error(), "TOML source") {
		t.Errorf("non-TOML source: %v", err)
	}
	if _, err := expandPathsAs(tomlPaths, []byte(`a = "{NOPE}/x"`), PathsSlash, "/h", noResolve); err == nil || !strings.Contains(err.Error(), "NOPE") {
		t.Errorf("an unresolvable token must be named: %v", err)
	}
}

// A released reader (0.71.0, manifest version 5) refuses paths on a TOML
// source, so the entry that declares it must carry a version that reader
// refuses as "newer": the operator then reads "update dotf", not a complaint
// about the entry.
func TestParseManifest_PathsOnATOMLSourceNeedsVersion6(t *testing.T) {
	const entry = `"configs":[{"name":"herdr","src":"ai/herdr/config.toml","dst":"{HOME}/c.toml","strategy":"merge","paths":"slash"}]}`
	for _, version := range []string{"3", "4", "5"} {
		_, err := ParseManifest([]byte(`{"version":` + version + `,` + entry))
		if err == nil || !strings.Contains(err.Error(), "needs manifest version 6") {
			t.Errorf("version %s: want a refusal naming version 6, got %v", version, err)
		}
	}
	if _, err := ParseManifest([]byte(`{"version":6,` + entry)); err != nil {
		t.Errorf("version 6 must accept paths on a TOML source: %v", err)
	}
}

// The source's format decides how it is rendered, never the destination's
// extension: a replace installs the source's bytes wherever they go.
func TestDeploy_PathsRenderBySourceFormatNotDestination(t *testing.T) {
	root := repoWithTrust(t, "ai/x/src.json", `{"dir":"{WORKTREE_ROOT}"}`)
	home := t.TempDir()
	c := Config{Name: "x", Src: "ai/x/src.json", Dst: "{HOME}/x.toml", Mode: "0644", Paths: PathsSlash}
	if _, err := Deploy(c, root, home, worktreeRoot, nil, false); err != nil {
		t.Fatalf("a JSON source must render as JSON whatever the destination is called: %v", err)
	}
	if got := readObject(t, filepath.Join(home, "x.toml")); got["dir"] != "/r/worktrees" {
		t.Errorf("dir = %v, want the rendered root", got["dir"])
	}
}
