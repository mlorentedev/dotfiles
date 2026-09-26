package doctor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// checkAgentTiersResolve catches a disagreement between two COMMITTED files: an
// agent record declares a neutral tier, and model-map.json decides which
// harnesses that tier covers. They can drift apart in the repository, and until
// this check existed nothing noticed until someone ran a deploy on a machine.
//
// Why this is a doctor check and not part of `compile-harness.sh --check`: that
// mode is the offline drift gate and it runs in the CI `lint` job, which installs
// no Go and has no `dotf`. Resolving tiers there would report drift on a
// perfectly good record purely because the machine lacks the resolver —
// conflating a property of the deploy ENVIRONMENT with a property of the
// committed RECORD. `dotf doctor` has neither problem: it already loads this
// registry, and it runs where `dotf` exists by definition.
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
		// manifest yet. checkCompileHarnessDrift owns that diagnosis.
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
// returns how many pairs it checked and how many failed.
func checkRecordTier(cfg *Config, parsed map[string]any, rep *Report, record string, deploy []string) (checked, failed int) {
	fm, err := readAgentFrontmatter(filepath.Join(cfg.DotfilesDir, record))
	if err != nil {
		return 0, 0
	}
	tier := fm["model"]
	if tier == "" {
		// Declaring no tier is not an error; the render emits no model line.
		return 0, 0
	}
	targets, declared := fm["targets"]
	for _, agent := range deploy {
		if !recordTargets(targets, declared, agent) {
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

// readAgentFrontmatter reads the single-line frontmatter values an agent record
// declares. Deliberately minimal: it answers only what this check asks, and the
// authoritative parse lives in the render pipeline.
//
// Values are returned raw, including any YAML flow-sequence brackets, because
// the caller decides what a given key's shape means.
func readAgentFrontmatter(path string) (map[string]string, error) {
	f, err := os.Open(path) // #nosec G304 -- path is built from the manifest's own record_dir
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	out := map[string]string{}
	sc := bufio.NewScanner(f)
	delims := 0
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			delims++
			if delims >= 2 {
				break
			}
			continue
		}
		if delims != 1 {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found || strings.HasPrefix(key, " ") || strings.HasPrefix(key, "\t") {
			continue
		}
		// The first occurrence wins, as it does in the render's awk (`exit` on
		// the first match).
		if k := strings.TrimSpace(key); !seen(out, k) {
			out[k] = strings.TrimSpace(value)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func seen(m map[string]string, k string) bool {
	_, ok := m[k]
	return ok
}

// recordTargets answers whether the render deploys a record to one harness.
//
// It is the render's rule, not a better one, because this check exists to
// predict what the next deploy does. The render (`skill_targets_agent` in
// scripts/compile-harness.sh) reads the first frontmatter line that starts
// with `targets:` and treats the record as targeting a harness when that line
// CONTAINS the harness name:
//
//   - key absent: every harness. Getting this backwards would make a persona
//     scoped to one harness fail against every other, a false positive on
//     correct data and the fastest way to make an operator stop reading.
//   - key present: a substring match, so `["opencode"]` targets opencode, and
//     a block-style list (`targets:` with the entries on later lines) targets
//     nobody, because its first line names no harness.
//
// The substring rule is imprecise (`copilot` contains `pi`), and a block list
// silently deploying nowhere is a render defect. Both are tracked in #1733
// (HARNESS-159); TestRecordTargetsAgreesWithTheRender holds this function to
// the render until then.
func recordTargets(raw string, declared bool, agent string) bool {
	if !declared {
		return true
	}
	return strings.Contains(raw, agent)
}
