package orca

import (
	"path/filepath"
	"testing"
)

// Orca's state lives in Electron's userData for the name "orca", which follows
// each OS's appData. A Linux path on a Mac made tune and export miss it (#2013
// F-024).
func TestUserDataDir_FollowsElectronAppData(t *testing.T) {
	home := filepath.FromSlash("/home/u")
	for _, tc := range []struct {
		name, goos string
		env        map[string]string
		want       string
	}{
		{"darwin", "darwin", map[string]string{"XDG_CONFIG_HOME": "/x"}, filepath.Join(home, "Library", "Application Support", "orca")},
		{"windows APPDATA", "windows", map[string]string{"APPDATA": "/roam"}, filepath.Join("/roam", "orca")},
		{"windows without APPDATA", "windows", nil, filepath.Join(home, "AppData", "Roaming", "orca")},
		{"linux XDG", "linux", map[string]string{"XDG_CONFIG_HOME": "/x"}, filepath.Join("/x", "orca")},
		{"linux default", "linux", nil, filepath.Join(home, ".config", "orca")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := UserDataDir(home, tc.goos, func(k string) string { return tc.env[k] })
			if got != tc.want {
				t.Errorf("UserDataDir(%s) = %q, want %q", tc.goos, got, tc.want)
			}
		})
	}
}
