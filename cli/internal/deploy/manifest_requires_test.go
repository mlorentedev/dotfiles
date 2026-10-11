package deploy

import (
	"os"
	"path"
	"path/filepath"
	"testing"
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
	for _, name := range missingSiblingRequires(m.Configs) {
		if _, ok := requiresExempt[name]; !ok {
			t.Errorf("%s omits the requires every other entry in its directory declares; add it, or exempt it in requiresExempt with the reason", name)
		}
	}
}

// missingSiblingRequires names each entry that declares no `requires` while
// every other entry in its destination directory declares one and the same.
// A directory holding configs for different tools ({HOME}) imposes nothing.
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
		if len(got) != len(tc.want) || (len(got) == 1 && got[0] != tc.want[0]) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
