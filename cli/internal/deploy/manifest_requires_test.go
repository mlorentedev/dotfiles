package deploy

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// requiresExempt lists the entries allowed to omit the `requires` every
// sibling in their directory declares, each with the reason.
var requiresExempt = map[string]string{
	// bash sources it as well as zsh, so a box without zsh still reads it.
	"zsh-functions-sh": "sourced by bash too",
}

// An entry that shares a directory with siblings all gated on one command is a
// config for that tool, and is gated the same way. pi-compaction was the one
// ~/.pi/agent entry without `requires: pi`: on a box with no pi (a fresh CI
// runner, #2013 X1) it created a settings.json holding only the compaction
// keys, and doctor's pin on defaultModel then failed against a file nothing
// reads.
func TestManifest_AnEntryRequiresWhatEverySiblingInItsDirectoryRequires(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("../../..", ManifestRel))
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	// The positive half: on a clean manifest the loop below never runs, so a
	// read that came back empty, or an entry renamed away, would pass silently.
	byName := map[string]Config{}
	for _, c := range m.Configs {
		byName[c.Name] = c
	}
	if c, ok := byName["pi-compaction"]; !ok || c.Requires != "pi" {
		t.Errorf("pi-compaction: want declared with requires pi, got %+v (present: %t)", c, ok)
	}
	for name := range requiresExempt {
		if _, ok := byName[name]; !ok {
			t.Errorf("requiresExempt names %s, which %s no longer declares; drop the exemption", name, ManifestRel)
		}
	}
	for _, name := range missingSiblingRequires(m.Configs) {
		if _, ok := requiresExempt[name]; !ok {
			t.Errorf("%s omits the requires every other entry in its directory declares; add it, or exempt it in requiresExempt with the reason", name)
		}
	}
}

// missingSiblingRequires names each entry that declares no `requires` while
// every other entry in its destination directory declares one and the same.
// A directory holding configs for different tools ({HOME}) imposes nothing.
// Sorted, so the result does not depend on map order.
func missingSiblingRequires(configs []Config) []string {
	byDir := map[string][]Config{}
	for _, c := range configs {
		byDir[path.Dir(c.Dst)] = append(byDir[path.Dir(c.Dst)], c)
	}
	var missing []string
	for _, entries := range byDir {
		for i, c := range entries {
			if c.Requires == "" && siblingsAgree(entries, i) {
				missing = append(missing, c.Name)
			}
		}
	}
	slices.Sort(missing)
	return missing
}

// siblingsAgree reports whether the entries other than entries[skip] exist and
// all declare the same non-empty `requires`.
func siblingsAgree(entries []Config, skip int) bool {
	want := ""
	for i, c := range entries {
		if i == skip {
			continue
		}
		if c.Requires == "" || (want != "" && c.Requires != want) {
			return false
		}
		want = c.Requires
	}
	return want != ""
}

func TestMissingSiblingRequires(t *testing.T) {
	cases := []struct {
		name    string
		configs []Config
		want    []string
	}{
		{"one entry missing it", []Config{
			{Name: "a", Dst: "{HOME}/.t/a", Requires: "t"},
			{Name: "b", Dst: "{HOME}/.t/b", Requires: "t"},
			{Name: "c", Dst: "{HOME}/.t/c"},
		}, []string{"c"}},
		{"two directories, one miss each, sorted", []Config{
			{Name: "z", Dst: "{HOME}/.u/z"},
			{Name: "u1", Dst: "{HOME}/.u/u1", Requires: "u"},
			{Name: "a", Dst: "{HOME}/.t/a"},
			{Name: "t1", Dst: "{HOME}/.t/t1", Requires: "t"},
		}, []string{"a", "z"}},
		{"siblings disagree", []Config{
			{Name: "a", Dst: "{HOME}/a", Requires: "tmux"},
			{Name: "b", Dst: "{HOME}/b", Requires: "zsh"},
			{Name: "c", Dst: "{HOME}/c"},
		}, nil},
		{"no sibling gated", []Config{
			{Name: "a", Dst: "{HOME}/.ssh/a"},
			{Name: "b", Dst: "{HOME}/.ssh/b"},
		}, nil},
		{"alone in its directory", []Config{
			{Name: "a", Dst: "{HOME}/.orca/a"},
		}, nil},
	}
	for _, tc := range cases {
		got := missingSiblingRequires(tc.configs)
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// requiredButNotCataloged lists the commands an entry may require on an OS
// where packages.json installs nothing, each with the reason. A key is
// "<command>" for every OS or "<command>/<goos>" for one.
var requiredButNotCataloged = map[string]string{
	"git":            "install.sh's prerequisite: the checkout step needs it before the catalog runs",
	"claude":         "the setup twins install it until #2013 PR-B moves it to the catalog",
	"pi":             "the setup twins install it until #2278 moves it to the catalog",
	"zsh/darwin":     "macOS ships zsh as its login shell",
	"zsh/windows":    "the zsh configs are for POSIX shells; Windows runs pwsh",
	"tmux/windows":   "the tmux config is for POSIX terminals; Windows has no tmux",
	"colima/linux":   "colima is the macOS container runtime; Linux runs the docker engine",
	"colima/windows": "colima is the macOS container runtime; Windows runs Docker Desktop",
}

// An entry gated on `requires: X` is deployed only where X is on PATH. On an
// OS where nothing installs X the entry never deploys, and nothing reports it:
// the gate reads as "not this machine's tool". #2013 X1 found the zsh configs
// that way on ubuntu-latest, where packages.json had no zsh. So every command
// an entry requires is installed by the catalog on each OS the entry targets,
// or named in requiredButNotCataloged with the reason.
func TestManifest_EveryRequiredCommandIsInstalledWhereTheEntryDeploys(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("../../..", ManifestRel))
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := tools.Load(filepath.Join("../../..", "packages.json"))
	if err != nil {
		t.Fatal(err)
	}
	missing := uncatalogedRequires(m.Configs, cat)
	// The positive half: zsh is the case this guard was written for.
	if slices.Contains(missing, "zsh/linux") {
		t.Error("nothing installs zsh on linux, so the zsh configs never deploy there")
	}
	used := map[string]bool{}
	for _, key := range missing {
		cmd, _, _ := strings.Cut(key, "/")
		switch {
		case requiredButNotCataloged[key] != "":
			used[key] = true
		case requiredButNotCataloged[cmd] != "":
			used[cmd] = true
		default:
			t.Errorf("%s: an entry requires %s, and packages.json installs nothing there; add a catalog entry, or exempt it in requiredButNotCataloged with the reason", key, cmd)
		}
	}
	for key := range requiredButNotCataloged {
		if !used[key] {
			t.Errorf("requiredButNotCataloged exempts %s, which is no longer needed; drop the exemption", key)
		}
	}
}

// uncatalogedRequires names, as "<command>/<goos>", each command an entry
// requires on an OS it deploys to where no catalog tool installs that command.
// Sorted and without duplicates.
func uncatalogedRequires(configs []Config, cat tools.Catalog) []string {
	var out []string
	for _, c := range configs {
		if c.Requires == "" {
			continue
		}
		for _, goos := range []string{"linux", "darwin", "windows"} {
			if c.AppliesOn(goos) && !catalogInstalls(cat, c.Requires, goos) {
				out = append(out, c.Requires+"/"+goos)
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// catalogInstalls reports whether a catalog tool puts cmd on PATH on goos. A
// system entry names its command when it has one; every other source puts the
// tool's name on PATH.
func catalogInstalls(cat tools.Catalog, cmd, goos string) bool {
	for _, t := range cat.Tools {
		bin := t.Name
		if t.Source.Command != "" {
			bin = t.Source.Command
		}
		if bin == cmd && t.SupportsOS(goos) {
			return true
		}
	}
	return false
}
