package harness

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstructionOutcome is what DeployInstructions did for one agents.presence
// target.
type InstructionOutcome struct {
	Agent  string
	File   string // absolute path
	Status string // "deployed" | "current" | "skipped"
	Reason string // why it was skipped
}

// DeployInstructions writes each agents.presence[] instruction file from its
// source: the source verbatim, followed by the deploy-only regions (the persona
// presence roster, the copilot skill catalog) the deployed file already
// carries, so the steps that own those regions replace them in place. A file
// that already holds exactly that is left alone, so a converged machine is not
// rewritten. A target whose requires_command is not on PATH is skipped; a
// symlink at the destination is replaced by a copy (BUG-100). A missing source
// is an error, reported after every other target has been deployed. This is
// the Go form of compile-harness.sh's deploy_instructions, which delegates to
// it when the installed dotf carries it.
func DeployInstructions(repoRoot, home string, has func(string) bool, dryRun bool) ([]InstructionOutcome, error) {
	_, targets, err := LoadPresence(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile)))
	if err != nil {
		return nil, err
	}
	var out []InstructionOutcome
	var errs []error
	for _, t := range targets {
		if t.Source == "" {
			continue
		}
		o := InstructionOutcome{Agent: t.Agent, File: filepath.Join(home, filepath.FromSlash(t.File))}
		if t.RequiresCommand != "" && !has(t.RequiresCommand) {
			o.Status, o.Reason = "skipped", t.RequiresCommand+" not on PATH"
			out = append(out, o)
			continue
		}
		status, err := deployInstruction(filepath.Join(repoRoot, filepath.FromSlash(t.Source)), o.File, dryRun)
		if err != nil {
			errs = append(errs, fmt.Errorf("instructions for %s (%s): %w", t.Agent, t.Source, err))
			continue
		}
		o.Status = status
		out = append(out, o)
	}
	return out, errors.Join(errs...)
}

func deployInstruction(src, dst string, dryRun bool) (string, error) {
	source, err := os.ReadFile(src) //nolint:gosec // a manifest-declared source in the checkout
	if err != nil {
		return "", err
	}
	isLink := false
	if fi, err := os.Lstat(dst); err == nil {
		isLink = fi.Mode()&os.ModeSymlink != 0
	}
	var current []byte
	if !isLink {
		current, _ = os.ReadFile(dst) //nolint:gosec // a manifest-declared target under HOME
	}
	want := withDeployOnlyRegions(string(source), string(current))
	if !isLink && string(current) == want {
		return "current", nil
	}
	if dryRun {
		return "deployed", nil
	}
	if isLink {
		if err := os.Remove(dst); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	return "deployed", os.WriteFile(dst, []byte(want), 0o644) //nolint:gosec // an instructions file, world-readable by design
}

// withDeployOnlyRegions is source with the deploy-only regions deployed
// carries, each where its writer puts it: the skill catalog in the source's
// empty slot, as compile-harness.sh's replace_region fills it, and the presence
// roster appended after one blank line, as `dotf harness presence` appends it.
// A catalog with no slot to fill is appended the same way. Only the first
// catalog counts, so a copy that an earlier deploy duplicated heals.
func withDeployOnlyRegions(source, deployed string) string {
	var catalog []string
	var kept [][]string
	mapRegions(deployed, func(region []string) []string {
		switch kindOf(region[0]) {
		case catalogRegion:
			if catalog == nil {
				catalog = region
				kept = append(kept, region)
			}
		case presenceRegion:
			kept = append(kept, region)
		}
		return region
	})
	filled := false
	out := mapRegions(source, func(region []string) []string {
		if catalog != nil && !filled && kindOf(region[0]) == catalogSlotRegion {
			filled = true
			return catalog
		}
		return region
	})
	var b strings.Builder
	b.WriteString(strings.TrimRight(out, "\n") + "\n")
	for _, region := range kept {
		if filled && kindOf(region[0]) == catalogRegion {
			continue
		}
		b.WriteString("\n" + strings.Join(region, "\n") + "\n")
	}
	return b.String()
}
