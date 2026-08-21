package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// checkModelMap reports on harness/model-map.json — the first doctor check over
// a registry in this repository (ADR-035).
//
// It exists because of constraint C15, and the constraint is not abstract. Four
// times in two days this repository measured the same failure: a positive-looking
// signal standing in for a check that never ran — a vault write that landed in
// the wrong place and reported success, an attestation that skipped correctly on
// an already-wrong SHA, a validator reporting "0 misplaced" while looking at one
// directory, and a reviewer whose complete output parsed as zero characters.
//
// A routing map is the worst possible place for that failure. "No pools are
// declared" and "the file cannot be read" produce identical downstream behaviour
// and mean opposite things — one is a configuration, the other is a broken
// system. So the three broken states are reported distinctly, and none of them
// is ever phrased as an empty-but-valid map.
//
// There is deliberately no fallback here and none in the loader: unlike
// triggers.json next door, model-map.json is not embedded, because an absent
// file resolving to a build-time default is exactly what this check would then
// certify as healthy (#1137).
func checkModelMap(cfg *Config, rep *Report) {
	rep.Section("Routing registry")

	mapPath := filepath.Join(cfg.DotfilesDir, harness.ModelMapFile)
	schemaPath := filepath.Join(cfg.DotfilesDir, harness.ModelMapSchemaFile)

	doc, err := os.ReadFile(mapPath)
	if err != nil {
		// This check reads the DEPLOYED copy (cfg.DotfilesDir), like every other
		// harness check here, so the ordinary cause of an absent map is a repo
		// that has moved ahead of the deploy dir rather than a missing file. Say
		// so: the FAIL is correct per C15, but a diagnostic that names only the
		// symptom sends the reader hunting for a file that is right there in the
		// checkout. setup-linux.sh mirrors the whole directory
		// (`cp -rf harness/. $DOTFILES_DIR/harness/`), so re-running setup is the
		// fix and no per-file wiring is needed.
		rep.Fail(fmt.Sprintf(
			"%s not found at %s — this is not an empty routing map, and nothing falls back to a default.\n"+
				"    This check reads the deployed copy; if the file exists in your checkout, re-run setup to mirror it.",
			harness.ModelMapFile, mapPath))
		return
	}

	// Parse before validating, so "the JSON is malformed" and "the JSON is valid
	// but the contract is violated" never collapse into one message.
	var parsed map[string]any
	if err := json.Unmarshal(doc, &parsed); err != nil {
		rep.Fail(fmt.Sprintf("%s could not be parsed as JSON: %v", harness.ModelMapFile, err))
		return
	}

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		rep.Fail(fmt.Sprintf(
			"%s is present but %s is missing — the map cannot be trusted without the contract it declares against",
			harness.ModelMapFile, harness.ModelMapSchemaFile))
		return
	}

	if err := harness.ValidateModelMap(doc, schema); err != nil {
		rep.Fail(fmt.Sprintf("%s does not satisfy %s: %v",
			harness.ModelMapFile, harness.ModelMapSchemaFile, err))
		return
	}

	pools, _ := parsed["pools"].(map[string]any)
	harnesses, _ := parsed["harnesses"].(map[string]any)
	rep.Pass(fmt.Sprintf("%s is present, parses, and satisfies its schema (%d pools, %d harnesses)",
		harness.ModelMapFile, len(pools), len(harnesses)))

	reportDeclaredBudgets(parsed, pools, rep)
}

// reportDeclaredBudgets prints what the map DECLARES and says plainly that
// nothing enforces it. ADR-035 ships level 1 only; level 2 needs a dispatcher to
// do the decrementing and there is none. A doctor line that read "concurrency: 5"
// without that sentence would be read as a limit in force, which is the same
// class of overstatement the check exists to prevent.
func reportDeclaredBudgets(parsed map[string]any, pools map[string]any, rep *Report) {
	names := make([]string, 0, len(pools))
	for name := range pools {
		names = append(names, name)
	}
	sort.Strings(names)

	var declared []string
	shared := map[string][]string{}
	for _, name := range names {
		b, err := harness.DeclaredBudget(parsed, name)
		if err != nil || !b.ConcurrencyDeclared {
			continue
		}
		entry := fmt.Sprintf("%s %d", name, b.Concurrency)
		if b.ReserveInteractive > 0 {
			entry += fmt.Sprintf(" (reserve %d)", b.ReserveInteractive)
		}
		declared = append(declared, entry)
		if len(b.SharedWith) > 0 {
			shared[name] = b.SharedWith
		}
	}

	if len(declared) == 0 {
		rep.Info("no pool declares a concurrency budget")
		return
	}
	rep.Info("declared concurrency: " + strings.Join(declared, ", ") +
		" — DECLARED, not enforced: nothing decrements these today (ADR-035 level 1)")

	for _, name := range names {
		if with, ok := shared[name]; ok {
			rep.Info(fmt.Sprintf(
				"pool %q is shared with %s, none of which routes through dotf — so the eventual guarantee is "+
					"\"dotf alone will never be the cause of exhaustion\", never \"exhaustion will not happen\"",
				name, strings.Join(with, ", ")))
		}
	}
}
