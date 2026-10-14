package converge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/pi"
)

// piSettings seeds pi's settings.json from ai/pi/settings.json when it is
// absent, and keeps its enabledModels equal to the repo's once it exists
// (pi.InspectSettings). Every OS.
//
// It runs after tools, which installs pi, and before configs-deploy, whose
// pi-compaction entry merges into the same file: a merge onto a missing file
// creates one holding only compaction, and the setup script's seed, which
// runs last, then finds a file and never fills it (#1948, lesson 422). The
// setup scripts keep their own seed and sync until the release carrying this
// step is the DOTF_VERSION pin, because an older dotf does not have it.
type piSettings struct {
	has func(string) bool // is a command on PATH
}

func (piSettings) Name() string        { return "pi-settings" }
func (piSettings) Platforms() []string { return nil }

func (r piSettings) paths(env Env) (src, dst string) {
	return filepath.Join(env.RepoRoot, filepath.FromSlash(pi.SettingsRel)), pi.SettingsPath(env.Home)
}

func (r piSettings) Reconcile(env Env, dryRun bool) (Result, error) {
	if !r.has("pi") {
		return Result{Skip: "pi is not installed"}, nil
	}
	src, dst := r.paths(env)
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return Result{Skip: "the checkout has no " + pi.SettingsRel}, nil
	}
	st, err := pi.InspectSettings(src, dst)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: st.Changes(), Detail: piSettingsDetail(st, dryRun)}
	if dryRun || res.Changes == 0 {
		return res, nil
	}
	return res, pi.ApplySettings(src, dst, st)
}

// Probe holds the post-condition: the file exists and its model list is the
// repo's.
func (r piSettings) Probe(env Env) error {
	src, dst := r.paths(env)
	st, err := pi.InspectSettings(src, dst)
	if err != nil {
		return err
	}
	if st.Changes() > 0 {
		return fmt.Errorf("%s still differs from %s: %s", dst, pi.SettingsRel, piSettingsDetail(st, true))
	}
	return nil
}

func piSettingsDetail(st pi.SettingsState, dryRun bool) string {
	switch {
	case st.Missing && dryRun:
		return "settings.json to seed from the repo"
	case st.Missing:
		return "seeded settings.json from the repo"
	case st.ModelsDiffer && dryRun:
		return "enabledModels to sync with the repo's"
	case st.ModelsDiffer:
		return "synced enabledModels with the repo's; pi's own choices kept"
	}
	return "settings.json present, enabledModels in sync"
}
