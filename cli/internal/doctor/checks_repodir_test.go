package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckRepoDirResolves drives the #696 guard: the DOTFILES_REPO_DIR cascade
// must resolve to a real git checkout. DOTFILES_REPO_DIR is set via the env
// (tier-1 of the cascade), so envpkg.ResolvePath returns it deterministically
// without a contract on disk.
func TestCheckRepoDirResolves(t *testing.T) {
	realCheckout := t.TempDir()
	if err := os.Mkdir(filepath.Join(realCheckout, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	notGit := t.TempDir() // exists but carries no .git

	cases := []struct {
		name         string
		repoDir      string
		wantFailures int
		wantSubstr   string
	}{
		{"real checkout -> pass", realCheckout, 0, "resolves to a checkout"},
		{"missing path -> fail", filepath.Join(realCheckout, "nope"), 1, "missing path"},
		{"exists but not a git checkout -> fail", notGit, 1, "not a git checkout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOTFILES_REPO_DIR", tc.repoDir)
			var buf bytes.Buffer
			rep := capture(&buf)
			checkRepoDirResolves(rep)
			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}
