package doctor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/pi"
)

// checkPiPackageRequirements FAILs for a declared pi package whose `requires`
// does not resolve on PATH (HARNESS-139, #1628).
//
// pi-memory was the case: declared, installed, and counted, while its retrieval
// dependency qmd was never provisioned, so memory_search answered with an
// install hint. A feature that reads as present and does not work is worse
// than none. The manifest now says what each package needs, and this check
// holds the machine to it. It reads the manifest through the reconciler's own
// loader, so the two cannot disagree about what is declared.
func checkPiPackageRequirements(sys *System, cfg *Config, rep *Report) {
	m, err := pi.LoadManifest(piManifestRoot(sys, cfg))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			rep.Skip("no pi package manifest — nothing declares a requirement")
			return
		}
		rep.Warn("pi package manifest unreadable, so requirements were not checked: " + err.Error())
		return
	}
	checked := 0
	var missing []string
	for _, p := range m.Packages {
		for _, req := range p.Requires {
			checked++
			if !sys.has(req) {
				missing = append(missing, fmt.Sprintf("%s needs %s", p.Source, req))
			}
		}
	}
	if len(missing) > 0 {
		rep.Fail(fmt.Sprintf("declared pi packages whose requirement is not on PATH: %s — provision it, or remove the package from %s",
			strings.Join(missing, "; "), pi.ManifestFile))
		return
	}
	rep.Pass(fmt.Sprintf("every declared pi package requirement resolves (%d checked)", checked))
}

// piManifestRoot is the root whose manifest doctor reads: the checkout when it
// has one, as piPackagesManifest does, and the deploy dir otherwise.
func piManifestRoot(sys *System, cfg *Config) string {
	if repo := resolveRepoDir(sys); repo != "" && pathExists(filepath.Join(repo, pi.ManifestFile)) {
		return repo
	}
	return cfg.DotfilesDir
}
