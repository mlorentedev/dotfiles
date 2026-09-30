package doctor

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// loadCatalog reads packages.json checkout-first (ADR-030 precedence), then
// the deploy mirror. An empty catalog means no copy was readable.
func loadCatalog(sys *System, cfg *Config) tools.Catalog {
	cat, _, _ := readPackageCatalog(sys, cfg)
	return cat
}

// readPackageCatalog is loadCatalog plus the path the catalog was read from (""
// when no copy loaded) and the first copy that exists but could not be read.
// The pin checks read through loadCatalog; only the catalog check reports the
// error, so it appears once rather than once per pinned tool.
func readPackageCatalog(sys *System, cfg *Config) (tools.Catalog, string, error) {
	var paths []string
	if repo := resolveRepoDir(sys); repo != "" {
		paths = append(paths, filepath.Join(repo, "packages.json"))
	}
	if cfg != nil && cfg.DotfilesDir != "" {
		paths = append(paths, filepath.Join(cfg.DotfilesDir, "packages.json"))
	}
	var unreadable error
	for _, p := range paths {
		cat, err := tools.Load(p)
		if err == nil {
			return cat, p, unreadable
		}
		if unreadable == nil && pathExists(p) {
			unreadable = err
		}
	}
	return tools.Catalog{}, "", unreadable
}

// catalogPin returns the packages.json pin for name, or "" when the catalog or
// the tool is absent. packages.json is the pin SSOT for every catalog tool
// (bw, sops, and since AI-034/#1294 opencode); versions.conf keeps only the
// pins the shell layer still consumes.
func catalogPin(sys *System, cfg *Config, name string) string {
	for _, t := range loadCatalog(sys, cfg).Tools {
		if t.Name == name {
			return t.Version
		}
	}
	return ""
}

// checkShadowedCatalogTools reports an npm-distributed catalog tool that
// resolves from more than one PATH directory (AI-034/#1294). The copy that
// wins is not necessarily the one `dotf tools install` converges: the Windows
// work box carried an npm-global opencode (first on PATH) and a winget one,
// and setup reported "still locked. after winget install 1.16.2" forever. The
// catalog cannot converge a binary it does not own, so the extra channel is
// named for the operator to remove. WARN, not FAIL: the tool does run.
//
// It also names a catalog copy that exists but cannot be read. Without that, a
// corrupt packages.json made every pin check SKIP and this check say nothing,
// which reads as a clean result.
func checkShadowedCatalogTools(sys *System, cfg *Config, rep *Report) {
	cat, from, err := readPackageCatalog(sys, cfg)
	switch {
	case err != nil && from == "":
		rep.Warn(fmt.Sprintf("%v — the catalog pin and shadowed-copy checks have nothing to read", err))
	case err != nil:
		rep.Warn(fmt.Sprintf("%v — pins are read from %s instead", err, from))
	}
	for _, t := range cat.Tools {
		if t.Source.Type != "npm" {
			continue
		}
		dirs := sys.dirsProviding(t.Name)
		if len(dirs) > 1 {
			rep.Warn(fmt.Sprintf("%s resolves from %d PATH directories (%s) — `dotf tools install` converges the npm copy only; remove the other channel's copy",
				t.Name, len(dirs), strings.Join(dirs, ", ")))
		}
	}
}

// dirsProviding lists the PATH directories that carry an executable named
// name (on Windows also name.exe / .cmd / .ps1, the shapes npm, scoop and
// winget each leave), in PATH order, without duplicates.
func (s *System) dirsProviding(name string) []string {
	candidates := []string{name}
	if runtime.GOOS == "windows" || s.GOOS == "windows" {
		candidates = append(candidates, name+".exe", name+".cmd", name+".ps1")
	}
	seen := map[string]bool{}
	var dirs []string
	for _, dir := range s.pathEntries() {
		if dir == "" {
			continue
		}
		clean := filepath.Clean(dir)
		if seen[clean] {
			continue
		}
		for _, c := range candidates {
			if isExecFile(filepath.Join(clean, c)) {
				seen[clean] = true
				dirs = append(dirs, clean)
				break
			}
		}
	}
	return dirs
}
