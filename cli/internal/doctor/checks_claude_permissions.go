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
	c, dst, bad, ok := readClaudePermissionRules(sys, rep)
	if !ok {
		return
	}
	if len(bad) == 0 {
		rep.Pass("no permission rule mixes * with the :* prefix form")
		return
	}
	remedy := "run: dotf doctor --fix"
	if fix {
		removed, err := dropMalformedRules(c, dst)
		if err == nil {
			for _, r := range removed {
				rep.Fix("removed " + r + " (it matched nothing)")
			}
			return
		}
		remedy = "--fix could not rewrite the file: " + err.Error()
	}
	for _, r := range bad {
		rep.Fail(fmt.Sprintf("%s in %s mixes * with the :* prefix form and matches nothing (%s)", r, dst, remedy))
	}
}

// readClaudePermissionRules resolves the deployed settings.json and lists its
// malformed rules. ok is false when the section has already said why it
// cannot check.
func readClaudePermissionRules(sys *System, rep *Report) (c *deploy.Config, dst string, bad []string, ok bool) {
	if !sys.has("claude") {
		rep.Skip("claude not in PATH")
		return nil, "", nil, false
	}
	repo := resolveRepoDir(sys)
	if repo == "" {
		rep.Skip("repo not found: the settings entry lives in " + deploy.ManifestRel)
		return nil, "", nil, false
	}
	c, dst, err := claudeSettingsTarget(repo, sys.home())
	if err != nil {
		rep.Warn(err.Error())
		return nil, "", nil, false
	}
	raw, err := os.ReadFile(dst) //nolint:gosec // the manifest's destination
	switch {
	case errors.Is(err, os.ErrNotExist):
		rep.Skip(dst + " not deployed; the deploy manifest section reports it")
		return nil, "", nil, false
	case err != nil:
		rep.Warn(err.Error())
		return nil, "", nil, false
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		rep.Warn(fmt.Sprintf("%s does not parse as JSON, so its rules were not checked: %v", dst, err))
		return nil, "", nil, false
	}
	return c, dst, claude.MalformedRules(doc), true
}

// dropMalformedRules removes the malformed rules through deploy.EditJSON and
// returns the ones it removed, read from the document it edited: Claude Code
// rewrites this file at runtime, so the list read before may be stale.
func dropMalformedRules(c *deploy.Config, dst string) ([]string, error) {
	var removed []string
	_, err := deploy.EditJSON(*c, dst, func(doc map[string]any) bool {
		removed = claude.MalformedRules(doc)
		return claude.DropMalformedRules(doc)
	})
	return removed, err
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
