package initrepo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The lane contract files (ADR-050 §6). laneConfigPath is lane.ConfigPath: the
// package does not import lane, so a test (TestBootstrapLane_ConfigLoads) loads
// the scaffolded file with the real loader to keep the two from drifting.
const (
	laneConfigPath = ".dotf/lane.json"
	lanePromptPath = ".dotf/lane-prompt.md"
	lanesHeading   = "## Lanes"
)

// LaneResult reports what BootstrapLane did to each of the three files. Each
// value is "created", "appended", "replaced" or "unchanged".
type LaneResult struct {
	Config string // .dotf/lane.json
	Prompt string // .dotf/lane-prompt.md
	Agents string // the "## Lanes" section of AGENTS.md
}

// String is the one-line report of a run, for dotf init's step log.
func (r LaneResult) String() string {
	return fmt.Sprintf("lane.json %s, lane-prompt.md %s, AGENTS.md %s", r.Config, r.Prompt, r.Agents)
}

// BootstrapLane scaffolds the lane contract under repoRoot: an empty-queue
// .dotf/lane.json, the shared prompt template and the "## Lanes" section of
// AGENTS.md, all from templates embedded in the binary.
//
// It is idempotent. Existing files are left alone; force replaces the prompt
// and the AGENTS.md section in place, never lane.json, whose queue belongs to
// the repository. dryRun reports the same result and writes nothing.
func BootstrapLane(repoRoot string, force, dryRun bool) (LaneResult, error) {
	var (
		res LaneResult
		err error
	)
	if res.Config, err = writeLaneFile(repoRoot, laneConfigPath, "lane.json", false, dryRun); err != nil {
		return res, err
	}
	if res.Prompt, err = writeLaneFile(repoRoot, lanePromptPath, "lane-prompt.md", force, dryRun); err != nil {
		return res, err
	}
	if res.Agents, err = writeLanesSection(repoRoot, force, dryRun); err != nil {
		return res, err
	}
	return res, nil
}

// writeLaneFile writes the embedded template tmpl to repoRoot/rel. An existing
// file is kept unless replace is set and its content differs.
func writeLaneFile(repoRoot, rel, tmpl string, replace, dryRun bool) (string, error) {
	want, err := ReadTemplate(tmpl)
	if err != nil {
		return "", err
	}
	path := filepath.Join(repoRoot, filepath.FromSlash(rel))
	action := "created"
	switch have, err := os.ReadFile(path); {
	case err == nil && (!replace || string(have) == string(want)):
		return "unchanged", nil
	case err == nil:
		action = "replaced"
	case !errors.Is(err, os.ErrNotExist):
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	if dryRun {
		return action, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, want, 0o644); err != nil {
		return "", err
	}
	return action, nil
}

// writeLanesSection adds the "## Lanes" section to repoRoot/AGENTS.md, in the
// same shapes as BootstrapAgentsOpts: create the file, append the section, or
// (with force) replace it in place.
func writeLanesSection(repoRoot string, force, dryRun bool) (string, error) {
	raw, err := ReadTemplate("agents-lanes-section.md")
	if err != nil {
		return "", err
	}
	section := strings.Trim(string(raw), "\n")
	path := filepath.Join(repoRoot, "AGENTS.md")

	action, next := "created", agentsHeader+"\n"+section+"\n"
	switch existing, err := os.ReadFile(path); {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return "", fmt.Errorf("reading %s: %w", path, err)
	default:
		content := string(existing)
		switch {
		case hasSection(content, lanesHeading) && !force:
			return "unchanged", nil
		case hasSection(content, lanesHeading):
			action, next = "replaced", replaceSection(content, lanesHeading, section)
			if next == content {
				return "unchanged", nil
			}
		default:
			action, next = "appended", strings.TrimRight(content, "\n")+"\n\n"+section+"\n"
		}
	}
	if dryRun {
		return action, nil
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return "", err
	}
	return action, nil
}
