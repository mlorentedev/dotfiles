package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const catalogWithOpencode = `{"tools":[{"name":"opencode","version":"1.16.2","profile":"full","source":{"type":"npm","package":"opencode-ai"}},{"name":"sops","version":"3.13.1","profile":"full","source":{"type":"github-release","repo":"getsops/sops","asset":{"linux":"x"},"checksums":"c"}}]}`

func TestCatalogPin_ReadsTheCheckoutBeforeTheMirror(t *testing.T) {
	repo, mirror := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithOpencode)
	writeFile(t, filepath.Join(mirror, "packages.json"), strings.ReplaceAll(catalogWithOpencode, "1.16.2", "1.0.0"))
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil)
	cfg := &Config{DotfilesDir: mirror}

	if got := catalogPin(sys, cfg, "opencode"); got != "1.16.2" {
		t.Errorf("checkout pin must win, got %q", got)
	}
	if got := catalogPin(sys, cfg, "nope"); got != "" {
		t.Errorf("an unknown tool has no pin, got %q", got)
	}
}

// Two opencode copies on PATH — the Windows work box's npm-global + winget
// pair — must be named; one copy must stay quiet.
func TestCheckShadowedCatalogTools_NamesEveryDirectoryProvidingTheTool(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithOpencode)
	a, b := filepath.Join(t.TempDir(), "npm"), filepath.Join(t.TempDir(), "winget")
	writeExec(t, filepath.Join(a, "opencode"))
	writeExec(t, filepath.Join(b, "opencode"))
	pathSep := string(os.PathListSeparator)

	t.Run("two directories -> WARN naming both", func(t *testing.T) {
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "PATH": a + pathSep + b}, nil, nil)
		var buf bytes.Buffer
		rep := capture(&buf)
		checkShadowedCatalogTools(sys, &Config{DotfilesDir: t.TempDir()}, rep)
		out := buf.String()
		if !strings.Contains(out, "[WARN]") || !strings.Contains(out, "opencode resolves from 2 PATH directories") {
			t.Fatalf("expected a WARN naming the count\n%s", out)
		}
		if !strings.Contains(out, a) || !strings.Contains(out, b) {
			t.Errorf("both directories must be named\n%s", out)
		}
	})

	t.Run("one directory -> quiet", func(t *testing.T) {
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "PATH": a}, nil, nil)
		var buf bytes.Buffer
		rep := capture(&buf)
		checkShadowedCatalogTools(sys, &Config{DotfilesDir: t.TempDir()}, rep)
		if strings.Contains(buf.String(), "opencode") {
			t.Errorf("a single copy is not shadowed\n%s", buf.String())
		}
	})

	t.Run("a duplicated PATH entry is one directory", func(t *testing.T) {
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "PATH": a + pathSep + a}, nil, nil)
		var buf bytes.Buffer
		rep := capture(&buf)
		checkShadowedCatalogTools(sys, &Config{DotfilesDir: t.TempDir()}, rep)
		if strings.Contains(buf.String(), "opencode") {
			t.Errorf("the same directory twice on PATH is not two copies\n%s", buf.String())
		}
	})
}
