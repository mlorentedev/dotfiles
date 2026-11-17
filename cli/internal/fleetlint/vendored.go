package fleetlint

import (
	"errors"
	"os"
	"path/filepath"
)

// RuleVendoredScript is a hygiene script a repository still carries after the
// ci-workflows hygiene composite replaced it with a dotf command (CI-005).
const RuleVendoredScript = "fleet/vendored-script"

// treeRules read the repository's files rather than its workflows, so they
// have no workflow fixtures.
var treeRules = map[string]bool{RuleVendoredScript: true}

// vendoredScripts are the copies the composite replaced, by path, with what
// replaces each. pin-actions.sh is not here: nothing in dotf replaces it yet.
var vendoredScripts = []struct{ path, what, command string }{
	{"scripts/check-actions-pinned.sh", "pin check", "dotf lint fleet --rule fleet/sha-pin"},
	{"scripts/check-lessons.sh", "lessons check", "dotf lessons fmt --check"},
}

// vendoredFindings reports each vendored script present in dir, a symlink
// included. Only absence is skipped: any other failure to stat one is an
// error, since skipping it would read as a clean check.
func vendoredFindings(dir string) ([]Finding, error) {
	var out []Finding
	for _, s := range vendoredScripts {
		_, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(s.path)))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, Finding{Rule: RuleVendoredScript, Path: s.path, Line: 1,
			Message: "a vendored copy of the fleet's " + s.what + "; run `" + s.command + "` (the ci-workflows hygiene composite) and delete it"})
	}
	return out, nil
}
