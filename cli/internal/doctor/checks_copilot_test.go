package doctor

import (
	"bytes"
	"path/filepath"
	"testing"
)

const catalogWithCopilot = `{"tools":[{"name":"copilot","version":"1.0.81","profile":"full","source":{"type":"npm","package":"@github/copilot"}}]}`

// copilot is an npm catalog tool (AI-038, #1321): the version is the first
// semver in `copilot --version` ("GitHub Copilot CLI 1.0.80." on the Windows
// box) compared with the packages.json pin the way opencode is (exact match
// PASS, otherwise a drift WARN); absent is a SKIP, not a FAIL, because a box
// may deliberately carry no Copilot.
func TestCheckCopilot_PinMatchByStatus(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithCopilot)
	cases := []struct {
		name    string
		onPath  []string
		version string
		want    Status
		needle  string
	}{
		{"at the pin → PASS", []string{"copilot"}, "GitHub Copilot CLI 1.0.81.\n", StatusPass, "matches packages.json"},
		{"above the pin → WARN drift (doctor reports exact match; the floor is tools install's)", []string{"copilot"}, "GitHub Copilot CLI 1.0.90.\n", StatusWarn, "drift"},
		{"below the pin → WARN drift", []string{"copilot"}, "GitHub Copilot CLI 1.0.78.\n", StatusWarn, "drift"},
		{"no semver in the output → WARN, no drift line", []string{"copilot"}, "banner only\n", StatusWarn, "no semver"},
		{"absent → SKIP naming the catalog", nil, "", StatusSkip, "dotf tools install copilot"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			sys := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, tc.onPath,
				map[string]string{"copilot --version": tc.version})
			var buf bytes.Buffer
			rep := capture(&buf)

			checkCopilot(sys, &Config{DotfilesDir: t.TempDir(), Versions: map[string]string{}}, rep)

			if got := statusOfLine(buf.String(), tc.needle); got != tc.want {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.want), buf.String())
			}
			if tc.needle == "no semver" && statusOfLine(buf.String(), "drift") != -1 {
				t.Fatalf("a version-less binary must not be reported as drift\n%s", buf.String())
			}
		})
	}
}
