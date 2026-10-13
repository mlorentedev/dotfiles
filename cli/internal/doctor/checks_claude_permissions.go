package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/claude"
	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// claudeSettingsEntry is the ai/deploy.json entry that installs Claude Code's
// settings.json.
const claudeSettingsEntry = "claude-settings"

// checkClaudePermissionRules applies Claude Code's own startup validation to
// the deployed settings.json (#2350): a rule that mixes a `*` with the
// trailing `:*` prefix form matches nothing, and Claude warns about it at
// every start. The template stopped carrying such rules in #2349, but deploy
// merges by union, so a box that had them keeps them. Each one is a FAIL;
// --fix removes them through deploy.EditJSON, the staged write deploy uses,
// and keeps every other rule in its order.
func checkClaudePermissionRules(sys *System, rep *Report, fix bool) {
	rep.Section("Claude permission rules")
	if !sys.has("claude") {
		rep.Skip("claude not in PATH")
		return
	}
	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found: the settings entry lives in " + deploy.ManifestRel)
		return
	}
	c, dst, err := claudeSettingsTarget(repo, sys.home())
	if err != nil {
		rep.Warn(err.Error())
		return
	}
	raw, err := os.ReadFile(dst) //nolint:gosec // the manifest's destination
	if errors.Is(err, os.ErrNotExist) {
		rep.Skip(dst + " not deployed; the deploy manifest section reports it")
		return
	}
	if err != nil {
		rep.Warn(err.Error())
		return
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		rep.Warn(fmt.Sprintf("%s does not parse as JSON, so its rules were not checked: %v", dst, err))
		return
	}
	bad := claude.MalformedRules(doc)
	if len(bad) == 0 {
		rep.Pass("no permission rule mixes * with the :* prefix form")
		return
	}
	if fix {
		changed, err := deploy.EditJSON(*c, dst, claude.DropMalformedRules)
		if err == nil && changed {
			for _, r := range bad {
				rep.Fix("removed " + r + " (it matched nothing)")
			}
			return
		}
		if err != nil {
			rep.Warn("could not remove them: " + err.Error())
		}
	}
	for _, r := range bad {
		rep.Fail(fmt.Sprintf("%s in %s mixes * with the :* prefix form and matches nothing (run: dotf doctor --fix)", r, dst))
	}
}

// claudeSettingsTarget finds the claude-settings entry and resolves its
// destination the way deploy does.
func claudeSettingsTarget(repo, home string) (*deploy.Config, string, error) {
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(deploy.ManifestRel))) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return nil, "", fmt.Errorf("cannot read %s: %w", deploy.ManifestRel, err)
	}
	man, err := deploy.ParseManifest(raw)
	if err != nil {
		return nil, "", err
	}
	c := man.Lookup(claudeSettingsEntry)
	if c == nil {
		return nil, "", fmt.Errorf("%s declares no %s entry", deploy.ManifestRel, claudeSettingsEntry)
	}
	dst, err := deploy.ExpandDst(c.Dst, home, env.ResolvePath)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", claudeSettingsEntry, err)
	}
	return c, dst, nil
}
