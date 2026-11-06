package copilotsettings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
)

// SettingsEntry is the deploy manifest entry whose destination Copilot reads.
const SettingsEntry = "copilot-settings"

// Result describes retired keys found or removed from one settings file.
type Result struct {
	Settings string
	Keys     []string
	Missing  bool
	Changed  bool
}

// Inspect reports retired keys without changing the settings file.
func Inspect(repo, home string, resolve func(string) string) (Result, error) {
	c, dst, policy, err := loadTarget(repo, home, resolve)
	if err != nil {
		return Result{}, err
	}
	return inspectConfig(*c, dst, policy)
}

// Repair removes retired keys through deploy's guarded JSON edit.
func Repair(repo, home string, resolve func(string) string, dryRun bool) (Result, error) {
	c, dst, policy, err := loadTarget(repo, home, resolve)
	if err != nil {
		return Result{}, err
	}
	if dryRun {
		res, err := inspectConfig(*c, dst, policy)
		res.Changed = len(res.Keys) > 0
		return res, err
	}
	return repairConfig(*c, dst, policy)
}

// RepairConfig is deploy.Run's optional repair seam for copilot-settings.
func RepairConfig(c deploy.Config, repo, home string, resolve func(string) string, dryRun bool) (bool, error) {
	if c.Name != SettingsEntry {
		return false, nil
	}
	policy, err := LoadPolicy(repo)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	dst, err := deploy.ExpandDst(c.Dst, home, resolve)
	if err != nil {
		return false, fmt.Errorf("%s: %w", SettingsEntry, err)
	}
	if dryRun {
		res, err := inspectConfig(c, dst, policy)
		return len(res.Keys) > 0, err
	}
	res, err := repairConfig(c, dst, policy)
	return res.Changed, err
}

func loadTarget(repo, home string, resolve func(string) string) (*deploy.Config, string, Policy, error) {
	policy, err := LoadPolicy(repo)
	if err != nil {
		return nil, "", Policy{}, err
	}
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(deploy.ManifestRel))) //nolint:gosec // checkout-relative, fixed name
	if err != nil {
		return nil, "", Policy{}, fmt.Errorf("read %s: %w", deploy.ManifestRel, err)
	}
	manifest, err := deploy.ParseManifest(raw)
	if err != nil {
		return nil, "", Policy{}, err
	}
	c := manifest.Lookup(SettingsEntry)
	if c == nil {
		return nil, "", Policy{}, fmt.Errorf("%s declares no %s entry", deploy.ManifestRel, SettingsEntry)
	}
	dst, err := deploy.ExpandDst(c.Dst, home, resolve)
	if err != nil {
		return nil, "", Policy{}, fmt.Errorf("%s: %w", SettingsEntry, err)
	}
	return c, dst, policy, nil
}

func inspectConfig(c deploy.Config, dst string, policy Policy) (Result, error) {
	res := Result{Settings: dst}
	_, err := deploy.EditJSON(c, dst, func(doc map[string]any) bool {
		res.Keys = findPaths(doc, policy.RetiredPaths)
		return false
	})
	if errors.Is(err, os.ErrNotExist) {
		res.Missing = true
		return res, nil
	}
	return res, err
}

func repairConfig(c deploy.Config, dst string, policy Policy) (Result, error) {
	res := Result{Settings: dst}
	changed, err := deploy.EditJSON(c, dst, func(doc map[string]any) bool {
		res.Keys = dropPaths(doc, policy.RetiredPaths)
		return len(res.Keys) > 0
	})
	if errors.Is(err, os.ErrNotExist) {
		res.Missing = true
		return res, nil
	}
	res.Changed = changed
	return res, err
}

func findPaths(doc map[string]any, paths []string) []string {
	var found []string
	for _, path := range paths {
		if hasPath(doc, strings.Split(path, ".")) {
			found = append(found, path)
		}
	}
	sort.Strings(found)
	return found
}

func dropPaths(doc map[string]any, paths []string) []string {
	var removed []string
	for _, path := range paths {
		if dropPath(doc, strings.Split(path, ".")) {
			removed = append(removed, path)
		}
	}
	sort.Strings(removed)
	return removed
}

func hasPath(doc map[string]any, parts []string) bool {
	value, ok := doc[parts[0]]
	if !ok {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	child, ok := value.(map[string]any)
	return ok && hasPath(child, parts[1:])
}

func dropPath(doc map[string]any, parts []string) bool {
	if len(parts) == 1 {
		if _, ok := doc[parts[0]]; !ok {
			return false
		}
		delete(doc, parts[0])
		return true
	}
	child, ok := doc[parts[0]].(map[string]any)
	if !ok || !dropPath(child, parts[1:]) {
		return false
	}
	if len(child) == 0 {
		delete(doc, parts[0])
	}
	return true
}
