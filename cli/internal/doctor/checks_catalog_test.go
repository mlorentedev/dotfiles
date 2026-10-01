package doctor

import (
	"bytes"
	"fmt"
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

	// One directory reached through two spellings (usrmerge's /bin -> /usr/bin,
	// scoop's current -> versioned pair) is one install, not a shadowed copy.
	t.Run("a symlinked alias of a PATH directory is one directory", func(t *testing.T) {
		alias := filepath.Join(t.TempDir(), "alias")
		if err := os.Symlink(a, alias); err != nil {
			t.Skipf("cannot create a directory symlink here: %v", err)
		}
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "PATH": a + pathSep + alias}, nil, nil)
		var buf bytes.Buffer
		rep := capture(&buf)
		checkShadowedCatalogTools(sys, &Config{DotfilesDir: t.TempDir()}, rep)
		if strings.Contains(buf.String(), "opencode") {
			t.Errorf("a directory and its symlink alias are not two copies\n%s", buf.String())
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

// An unreadable catalog used to make the pin checks SKIP and the shadow check
// say nothing, which reads as "all clear". It is named once, from the catalog
// check, and a readable fallback copy is still used.
func TestCheckShadowedCatalogTools_NamesAnUnreadableCatalog(t *testing.T) {
	run := func(t *testing.T, repo, mirror string) string {
		t.Helper()
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo, "PATH": ""}, nil, nil)
		var buf bytes.Buffer
		checkShadowedCatalogTools(sys, &Config{DotfilesDir: mirror}, capture(&buf))
		return buf.String()
	}

	t.Run("the only copy is corrupt -> WARN naming it", func(t *testing.T) {
		repo := t.TempDir()
		writeFile(t, filepath.Join(repo, "packages.json"), `{"tools":[`)
		out := run(t, repo, t.TempDir())
		// tools.Load quotes the path, which doubles a Windows backslash.
		if !strings.Contains(out, "[WARN]") || !strings.Contains(out, fmt.Sprintf("%q", filepath.Join(repo, "packages.json"))) {
			t.Fatalf("expected a WARN naming the unreadable file\n%s", out)
		}
		if !strings.Contains(out, "nothing to read") {
			t.Errorf("with no readable copy the WARN must say the checks have nothing to read\n%s", out)
		}
	})

	t.Run("a corrupt checkout falls back to the mirror, and says so", func(t *testing.T) {
		repo, mirror := t.TempDir(), t.TempDir()
		writeFile(t, filepath.Join(repo, "packages.json"), `{"tools":[`)
		writeFile(t, filepath.Join(mirror, "packages.json"), catalogWithOpencode)
		out := run(t, repo, mirror)
		if !strings.Contains(out, "[WARN]") || !strings.Contains(out, fmt.Sprintf("%q", filepath.Join(repo, "packages.json"))) {
			t.Fatalf("expected a WARN naming the corrupt checkout copy\n%s", out)
		}
		if !strings.Contains(out, "pins are read from "+filepath.Join(mirror, "packages.json")) {
			t.Errorf("the WARN must name the copy the pins come from\n%s", out)
		}
		if got := catalogPin(newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, nil, nil), &Config{DotfilesDir: mirror}, "opencode"); got != "1.16.2" {
			t.Errorf("the mirror's pin must still be read, got %q", got)
		}
	})

	// A readable catalog with no tools is still a readable one.
	t.Run("an empty but valid mirror behind a corrupt checkout is named as the source", func(t *testing.T) {
		repo, mirror := t.TempDir(), t.TempDir()
		writeFile(t, filepath.Join(repo, "packages.json"), `{"tools":[`)
		writeFile(t, filepath.Join(mirror, "packages.json"), `{"tools":[]}`)
		out := run(t, repo, mirror)
		if strings.Contains(out, "nothing to read") || !strings.Contains(out, "pins are read from "+filepath.Join(mirror, "packages.json")) {
			t.Errorf("an empty mirror was read, and the WARN must say so\n%s", out)
		}
	})

	t.Run("no copy at all -> quiet", func(t *testing.T) {
		if out := run(t, t.TempDir(), t.TempDir()); out != "" {
			t.Errorf("an absent catalog is not an unreadable one\n%s", out)
		}
	})
}
