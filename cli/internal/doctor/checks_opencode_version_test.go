package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// The opencode banner that setup-windows.ps1 parsed as the version "locked."
// (AI-034/#1294) must not become a false drift report in doctor: the version
// is the first semver anywhere in the output, matched against the catalog pin.
func TestCheckOpenCode_VersionIsTheSemverNotTheBannerToken(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithOpencode)
	cases := []struct {
		name, banner, wantLine string
	}{
		{"banner line before the number", "OpenCode locked.\n1.16.2\n", "opencode version matches packages.json (1.16.2)"},
		{"plain version", "1.16.2\n", "opencode version matches packages.json (1.16.2)"},
		{"older version", "1.15.0\n", "opencode version drift: installed=1.15.0 pinned=1.16.2 (packages.json)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, []string{"opencode"},
				map[string]string{"opencode --version": tc.banner})
			var buf bytes.Buffer
			checkOpenCode(sys, &Config{DotfilesDir: t.TempDir(), Versions: map[string]string{}}, capture(&buf))
			if !strings.Contains(buf.String(), tc.wantLine) {
				t.Errorf("want %q in\n%s", tc.wantLine, buf.String())
			}
			if strings.Contains(buf.String(), "locked.") {
				t.Errorf("the banner token must never be reported as a version\n%s", buf.String())
			}
		})
	}
}
