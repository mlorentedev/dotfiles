package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// checkAgentTiersResolve catches a disagreement between two COMMITTED files: an
// agent record declares a neutral tier, and model-map.json decides which
// harnesses that tier covers. They can drift apart in the repository, and until
// this check existed nothing noticed until someone ran a deploy on a machine.
//
// Why this is a doctor check and not part of `dotf harness check`: that is the
// offline drift gate over the records' shape, and a tier is a question about the
// model map a deploy resolves against. `dotf doctor` already loads that map.
//
// Scoped to the harnesses `agents.deploy` actually renders to. A tier gap for a
// harness nothing deploys to is a real question (see #1170 for copilot's) but it
// is not drift, and reporting it here would train the reader to ignore this line.
func checkAgentTiersResolve(cfg *Config, parsed map[string]any, rep *Report) {
	recordDir, deploy, ok := agentDeployTargets(cfg, rep)
	if !ok {
		return
	}
	entries, err := os.ReadDir(filepath.Join(cfg.DotfilesDir, recordDir))
	if err != nil {
		return
	}

	checked, failed := 0, 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		c, f := checkRecordTier(cfg, parsed, rep, filepath.Join(recordDir, e.Name(), "AGENT.md"), deploy)
		checked, failed = checked+c, failed+f
	}

	if checked > 0 && failed == 0 {
		rep.Pass(fmt.Sprintf("every declared agent tier resolves for its deploy targets (%d checked)", checked))
	}
}

// agentDeployTargets reads the manifest's agent record dir and the harnesses
// `agents.deploy` renders to. ok is false when there is nothing to check.
func agentDeployTargets(cfg *Config, rep *Report) (recordDir string, deploy []string, ok bool) {
	raw, err := os.ReadFile(filepath.Join(cfg.DotfilesDir, "harness", "manifest.json"))
	if err != nil {
		// Not a failure of this check: the deploy dir simply may not carry a
		// manifest yet. checkRecordDrift owns that diagnosis.
		return "", nil, false
	}
	var manifest struct {
		Agents struct {
			RecordDir string `json:"record_dir"`
			Deploy    []struct {
				Agent string `json:"agent"`
			} `json:"deploy"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		rep.Warn(fmt.Sprintf("harness/manifest.json does not parse, so agent tiers were not checked: %v", err))
		return "", nil, false
	}
	for _, d := range manifest.Agents.Deploy {
		deploy = append(deploy, d.Agent)
	}
	recordDir = manifest.Agents.RecordDir
	if recordDir == "" {
		recordDir = "harness/agents"
	}
	return recordDir, deploy, len(deploy) > 0
}

// checkRecordTier resolves one record's tier for every harness it targets, and
// returns how many pairs it checked and how many failed. It reads the record
// with the render's own predicates (harness.SkillTargetsAgent,
// harness.RecordModelTier), because this check exists to predict what the next
// deploy does: a rule merely more reasonable than the render's would be wrong.
func checkRecordTier(cfg *Config, parsed map[string]any, rep *Report, record string, deploy []string) (checked, failed int) {
	raw, err := os.ReadFile(filepath.Join(cfg.DotfilesDir, record)) // #nosec G304 -- path is built from the manifest's own record_dir
	if err != nil {
		return 0, 0
	}
	tier := harness.RecordModelTier(raw)
	if tier == "" {
		// Declaring no tier is not an error; the render emits no model line.
		return 0, 0
	}
	for _, agent := range deploy {
		targeted, err := harness.SkillTargetsAgent(raw, agent)
		if err != nil {
			// The render refuses the record on every harness for the same
			// reason, so it is reported once.
			rep.Fail(fmt.Sprintf("agent record %s cannot be read for its targets (%v) — the render will fail on the next deploy", record, err))
			return checked, failed + 1
		}
		if !targeted {
			continue
		}
		checked++
		if _, err := harness.ResolveTier(parsed, tier, agent); err != nil {
			failed++
			rep.Fail(fmt.Sprintf(
				"agent record %s declares model tier %q, which %s cannot answer for harness %q — "+
					"the render will fail on the next deploy",
				record, tier, harness.ModelMapFile, agent))
		}
	}
	return checked, failed
}
