package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// triggerPatternsSubdir is where the vault keeps the patterns harness/triggers.json
// names. It is the `vault_subpath` harness/manifest.json declares for the
// enforced records, which compile-harness.sh reads from the same place.
const triggerPatternsSubdir = "00_meta/patterns"

// checkTriggerTargets asserts that every pattern a trigger names exists in the
// vault.
//
// A trigger whose pattern is missing still fires and routes work: the prompt hook
// prints the missing name as its evidence and a session goes looking for it.
// Measured 2026-09-24, 8 of the 18 shipped triggers named patterns that were
// never written or had been renamed, and nothing said so, because the file is only
// read by code that treats the name as an opaque string.
//
// compile-harness.sh --refresh refuses the same thing, but only when someone runs
// it. This is the check that still fires after a pattern is renamed in the vault
// on a machine that already deployed the triggers.
//
// It reads the DEPLOYED file, because that is the one the hook consults, and it
// SKIPS rather than passes when it cannot look: a machine without the vault has
// not shown its triggers are sound, only that this check has nothing to compare
// them against.
func checkTriggerTargets(sys *System, cfg *Config, rep *Report) {
	triggersPath := filepath.Join(cfg.DotfilesDir, filepath.FromSlash(harness.TriggersFile))
	raw, err := os.ReadFile(triggersPath) // #nosec G304 -- path is the deploy dir's own triggers file
	if err != nil {
		rep.Skip("no triggers at " + triggersPath + " - trigger targets unchecked")
		return
	}

	vault := sys.env("VAULT_PATH", filepath.Join(sys.home(), "Projects", "knowledge"))
	patterns := filepath.Join(vault, filepath.FromSlash(triggerPatternsSubdir))
	if !isDir(patterns) {
		rep.Skip("no vault patterns at " + patterns + " - trigger targets unchecked")
		return
	}

	parsed, err := harness.ParseTriggers(raw)
	if err != nil {
		rep.Fail("triggers at " + triggersPath + " do not parse: " + err.Error())
		return
	}

	var dangling []string
	named := 0
	for _, rule := range parsed.Triggers {
		if rule.Pattern == "" {
			continue
		}
		named++
		if !pathExists(filepath.Join(patterns, rule.Pattern+".md")) {
			dangling = append(dangling, rule.ID+" -> "+rule.Pattern)
		}
	}

	if named == 0 {
		// A document that names no pattern has shown nothing about its targets.
		// Reporting "all 0 triggers name a pattern" was a PASS with nothing
		// checked (HARNESS-148), so it skips like the other cannot-look paths.
		rep.Skip("the triggers at " + triggersPath + " name no pattern - trigger targets unchecked")
		return
	}
	if len(dangling) == 0 {
		rep.Pass(fmt.Sprintf("all %d triggers name a pattern the vault has", named))
		return
	}
	rep.Fail(fmt.Sprintf("%d of %d triggers name a pattern the vault does not have: %s "+
		"(fix harness/triggers.json, re-run compile-harness.sh --refresh, then setup)",
		len(dangling), named, strings.Join(dangling, ", ")))
}
