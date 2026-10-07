package deploy

import (
	"strings"
	"testing"
)

// A reader that adds a field also reads the version before it, which by
// construction carries none of the field. So the reader ships a release before
// any entry uses it, and the dotf every machine has installed keeps reading the
// manifest in between (#1814 class).
func TestParseManifest_ReadsThePreviousVersionToo(t *testing.T) {
	for _, m := range []string{
		`{"version":3,"configs":[{"name":"x","src":"a","dst":"b"}]}`,
		`{"version":4,"configs":[{"name":"x","src":"a","dst":"b","platforms":["linux","darwin"]}]}`,
	} {
		if _, err := ParseManifest([]byte(m)); err != nil {
			t.Errorf("%s: %v", m, err)
		}
	}
}

func TestParseManifest_ValidatesPlatforms(t *testing.T) {
	cases := []struct{ name, manifest, want string }{
		// A version 3 reader would refuse the field; a version 3 manifest
		// that carries it is one no released dotf can read.
		{"platforms in a version 3 manifest", `{"version":3,"configs":[{"name":"x","src":"a","dst":"b","platforms":["linux"]}]}`, `config "x": platforms needs manifest version 4`},
		{"an unknown platform", `{"version":4,"configs":[{"name":"x","src":"a","dst":"b","platforms":["macos"]}]}`, `config "x": unknown platform "macos"`},
		{"an empty list", `{"version":4,"configs":[{"name":"x","src":"a","dst":"b","platforms":[]}]}`, `config "x": empty platforms`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tc.manifest))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want an error containing %q, got %v", tc.want, err)
			}
		})
	}
}

// Absent means every OS; an OS that is not listed is skipped, not failed: the
// meaning packages.json's platforms has, through the same package.
func TestConfig_AppliesOn(t *testing.T) {
	every := Config{}
	posix := Config{Platforms: []string{"linux", "darwin"}}
	for goos, want := range map[string][2]bool{
		"linux":   {true, true},
		"darwin":  {true, true},
		"windows": {true, false},
	} {
		if got := [2]bool{every.AppliesOn(goos), posix.AppliesOn(goos)}; got != want {
			t.Errorf("%s: got %v, want %v", goos, got, want)
		}
	}
}
