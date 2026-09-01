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

// checkAgentSkillsMigrated reports persona skills that carry no declared
// enforcement severity, so the gate's inaction is VISIBLE rather than inferred.
//
// THIS CHECK EXISTED ONLY AS A PROMISE UNTIL NOW. `harness.Decide`'s
// EnforceUnset branch says, in a comment, that such a skill "is surfaced by
// `dotf doctor` instead" — and `UnmigratedSkills()` was written, exported and
// unit-tested to that end. Measured 2026-08-31: NOTHING in production called it.
// The whole binding chain (bind -> hook -> gate -> Decide) was live and decided
// nothing for all 35 skills, and the one mechanism meant to say so did not run.
//
// That is precisely the failure mode this repository keeps cataloguing: a guard
// whose absence is indistinguishable from a clean result. It matters most right
// now, during the migration to declared severity, because this is the only
// surface that reports how far that migration has actually got.
//
// It is a WARN, never a FAIL. An unmigrated skill is a known, deliberate state —
// EnforceUnset exists so that neither enforcement nor silence is the default —
// and failing the machine's health command over planned work would train the
// reader to ignore it. It becomes a Pass only when every record is migrated.
func checkAgentSkillsMigrated(cfg *Config, rep *Report) {
	rep.Section("Persona skill enforcement")

	recordDir, ok := agentRecordDir(cfg)
	if !ok {
		rep.Skip("no harness manifest in " + cfg.DotfilesDir + " — no persona records to check")
		return
	}
	entries, err := os.ReadDir(filepath.Join(cfg.DotfilesDir, recordDir))
	if err != nil {
		rep.Skip("no persona records under " + recordDir)
		return
	}

	total, unmigrated := 0, 0
	var pending []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(cfg.DotfilesDir, recordDir, e.Name(), "AGENT.md")
		// LoadPersona, not readAgentFrontmatter: the latter reads single-line
		// values only and its own docstring says it never reads `skills`. Using
		// it here would report every record as having no skills at all — a
		// check that passes on the broken thing.
		p, err := harness.LoadPersona(path)
		if err != nil {
			rep.Warn(fmt.Sprintf("persona record %s does not parse, so its skills were not checked: %v",
				filepath.Join(recordDir, e.Name(), "AGENT.md"), err))
			continue
		}
		total += len(p.Skills)
		if u := p.UnmigratedSkills(); len(u) > 0 {
			unmigrated += len(u)
			pending = append(pending, fmt.Sprintf("%s (%d: %s)", p.Name, len(u), strings.Join(u, ", ")))
		}
	}

	switch {
	case total == 0:
		rep.Skip("no persona declares any skill")
	case unmigrated == 0:
		rep.Pass(fmt.Sprintf("every persona skill declares an enforcement severity (%d)", total))
	default:
		sort.Strings(pending)
		rep.Warn(fmt.Sprintf(
			"%d of %d persona skills carry no `enforce:` severity, so `dotf harness gate` does NOT act on them — %s",
			unmigrated, total, strings.Join(pending, "; ")))
	}
}

// agentRecordDir reads `agents.record_dir` from the deployed manifest, with the
// same default the render pipeline uses. Returns false when there is no manifest
// to read, which is a valid state (a machine that has not deployed yet) rather
// than an error this check should diagnose.
func agentRecordDir(cfg *Config) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(cfg.DotfilesDir, "harness", "manifest.json"))
	if err != nil {
		return "", false
	}
	var manifest struct {
		Agents struct {
			RecordDir string `json:"record_dir"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", false
	}
	if manifest.Agents.RecordDir == "" {
		return "harness/agents", true
	}
	return manifest.Agents.RecordDir, true
}
