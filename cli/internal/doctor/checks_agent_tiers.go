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
	manifestPath := filepath.Join(cfg.DotfilesDir, "harness", "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		// Not a failure of this check: the deploy dir simply may not carry a
		// manifest yet. checkCompileHarnessDrift owns that diagnosis.
		return
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
		return
	}
	if len(manifest.Agents.Deploy) == 0 {
		return
	}
	recordDir := manifest.Agents.RecordDir
	if recordDir == "" {
		recordDir = "harness/agents"
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
		recPath := filepath.Join(cfg.DotfilesDir, recordDir, e.Name(), "AGENT.md")
		fm, err := readAgentFrontmatter(recPath)
		if err != nil {
			continue
		}
		tier := fm["model"]
		if tier == "" {
			// Declaring no tier is not an error; the render emits no model line.
			continue
		}
		for _, d := range manifest.Agents.Deploy {
			if !recordTargets(fm["targets"], d.Agent) {
				continue
			}
			checked++
			if _, err := harness.ResolveTier(parsed, tier, d.Agent); err != nil {
				failed++
				rep.Fail(fmt.Sprintf(
					"agent record %s declares model tier %q, which %s cannot answer for harness %q — "+
						"the render will fail on the next deploy",
					filepath.Join(recordDir, e.Name(), "AGENT.md"), tier, harness.ModelMapFile, d.Agent))
			}
		}
	}

	if checked > 0 && failed == 0 {
		rep.Pass(fmt.Sprintf("every declared agent tier resolves for its deploy targets (%d checked)", checked))
	}
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
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// recordTargets answers whether a record applies to one harness.
//
// An ABSENT targets list means every harness — the same default the render uses.
// Getting this backwards would make a persona scoped to one harness fail this
// check against every other, which is a false positive on correct data and the
// fastest way to make an operator stop reading a diagnostic.
func recordTargets(raw, agent string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	for _, t := range strings.Split(raw, ",") {
		if strings.TrimSpace(t) == agent {
			return true
		}
	}
	return false
}
