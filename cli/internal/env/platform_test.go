package env

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestForOSReadsTheOwnKeyThenTheFallback(t *testing.T) {
	def := map[string]string{"linux": "$HOME/x", "windows": `$env:USERPROFILE\x`}
	withDarwin := map[string]string{"linux": "$HOME/x", "darwin": "$HOME/Library/x"}
	optOut := map[string]string{"linux": "$HOME/x", "darwin": ""}
	for _, tc := range []struct {
		name    string
		m       map[string]string
		goos    string
		want    string
		wantKey string
		wantOK  bool
	}{
		{"linux reads its own", def, "linux", "$HOME/x", "linux", true},
		{"windows reads its own", def, "windows", `$env:USERPROFILE\x`, "windows", true},
		{"darwin inherits linux", def, "darwin", "$HOME/x", "linux", true},
		{"a darwin key wins", withDarwin, "darwin", "$HOME/Library/x", "darwin", true},
		{"an empty darwin key opts out", optOut, "darwin", "", "darwin", true},
		{"windows never inherits", withDarwin, "windows", "", "", false},
		{"an unset GOOS reads linux", def, "", "$HOME/x", "linux", true},
		{"nil map", nil, "darwin", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, key, ok := ForOS(tc.m, tc.goos)
			if got != tc.want || key != tc.wantKey || ok != tc.wantOK {
				t.Errorf("ForOS(%v, %q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.m, tc.goos, got, key, ok, tc.want, tc.wantKey, tc.wantOK)
			}
		})
	}
}

func TestForOSResolvesListsLikeValues(t *testing.T) {
	entries := map[string][]string{"linux": {"$HOME/.local/bin"}}
	got, key, ok := ForOS(entries, "darwin")
	if !ok || key != "linux" || len(got) != 1 || got[0] != "$HOME/.local/bin" {
		t.Errorf("ForOS(entries, darwin) = (%v, %q, %v), want the linux list", got, key, ok)
	}
}

func TestAppliesOn(t *testing.T) {
	for _, tc := range []struct {
		scope, goos string
		want        bool
	}{
		{"", "windows", true},
		{"linux", "linux", true},
		{"linux", "darwin", true},
		{"linux", "", true},
		{"linux", "windows", false},
		{"windows", "darwin", false},
		{"darwin", "darwin", true},
		{"darwin", "linux", false},
	} {
		if got := AppliesOn(tc.scope, tc.goos); got != tc.want {
			t.Errorf("AppliesOn(%q, %q) = %v, want %v", tc.scope, tc.goos, got, tc.want)
		}
	}
}

// TestTheRealContractKeysByKnownOSAndCopiesNothing reads env-contract.json and
// fails on a key that is not a GOOS the contract serves (a `macos` key would be
// read by no OS, silently), and on a darwin value equal to the linux one it
// would inherit anyway: a darwin key carries only what differs.
func TestTheRealContractKeysByKnownOSAndCopiesNothing(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "env-contract.json"))
	if err != nil {
		t.Fatalf("read env-contract.json: %v", err)
	}
	var c struct {
		EnvVars []struct {
			Name       string            `json:"name"`
			RequiredOn string            `json:"required_on"`
			Default    map[string]string `json:"default"`
		} `json:"env_vars"`
		RequiredPathEntries map[string][]string `json:"required_path_entries"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("parse env-contract.json: %v", err)
	}
	known := map[string]bool{"linux": true, "darwin": true, "windows": true}
	check := func(where string, keys []string) {
		for _, k := range keys {
			if !known[k] {
				t.Errorf("%s: %q is not a contract OS key (linux, darwin, windows)", where, k)
			}
		}
	}
	for _, v := range c.EnvVars {
		check(v.Name+".default", slices.Collect(maps.Keys(v.Default)))
		if v.RequiredOn != "" {
			check(v.Name+".required_on", []string{v.RequiredOn})
		}
		if d, ok := v.Default["darwin"]; ok && d == v.Default["linux"] {
			t.Errorf("%s.default.darwin repeats the linux value it inherits", v.Name)
		}
	}
	check("required_path_entries", slices.Collect(maps.Keys(c.RequiredPathEntries)))
	if d, ok := c.RequiredPathEntries["darwin"]; ok && slices.Equal(d, c.RequiredPathEntries["linux"]) {
		t.Errorf("required_path_entries.darwin repeats the linux list it inherits")
	}
}
