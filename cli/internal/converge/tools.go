package converge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// toolsSync converges the tool layer on every OS: the packages.json catalog
// (`dotf tools install`: brew, apt and winget packages, release binaries, npm
// and uv tools, mise itself) and the pinned CLIs mise installs (`dotf tools
// sync`, ADR-044, the versions.conf pins marked "# mise: cli"). It runs after
// the records, so an agent that starts once a tool lands already has its
// instructions, and before configs-deploy, whose entries `require` a tool.
//
// An apply walks install, sync, install, the order setup-linux.sh uses: the
// first pass places mise, the sync installs uv, and the second pass installs
// the uv tools that waited on it, so a fresh machine converges in one run.
type toolsSync struct {
	run, stdout tools.Runner      // tools.HomeRunners in production
	has         func(string) bool // is a command on PATH
	getenv      func(string) string
	catalog     CatalogInstaller // nil: the catalog half is not wired
	bins        []string         // where the tool layer places binaries; see reach
}

func (toolsSync) Name() string        { return "tools" }
func (toolsSync) Platforms() []string { return nil }

func (r toolsSync) Reconcile(env Env, dryRun bool) (Result, error) {
	if err := r.reach(); err != nil {
		return Result{}, err
	}
	if r.catalog == nil && r.unavailable() != "" {
		return Result{Skip: r.unavailable()}, nil
	}
	entries, err := r.entries(env)
	if err != nil {
		return Result{}, err
	}
	cat := r.walk(entries, !dryRun)
	res, serr := r.syncHalf(env, dryRun)
	if !dryRun {
		cat.then(r.walk(cat.retry(), true))
	}
	res.Changes += len(cat.changed)
	res.Detail = joinDetail(cat.detail(dryRun), res.Detail)
	return res, errors.Join(cat.err(), serr)
}

// Probe holds the post-condition: no catalog entry is left to install or
// upgrade (a wait on a manager and a needs-sudo are reported, not failed),
// the rendered mise config is current, and every pinned CLI runs at its pin.
func (r toolsSync) Probe(env Env) error {
	if err := r.reach(); err != nil {
		return err
	}
	entries, err := r.entries(env)
	if err != nil {
		return err
	}
	left := r.walk(entries, false)
	if len(left.changed) > 0 {
		return fmt.Errorf("catalog entries still to install: %s", strings.Join(left.changed, ", "))
	}
	if err := left.err(); err != nil {
		return err
	}
	s, pins, err := r.sync(env)
	if err != nil {
		return err
	}
	if r.catalog != nil {
		if lost := unreachable(left, provided(entries, pins, env.GOOS), !r.has("mise")); len(lost) > 0 {
			return fmt.Errorf("installed by this machine's tools but not on PATH: %s", strings.Join(lost, ", "))
		}
	}
	if r.unavailable() != "" {
		return nil
	}
	p, err := s.Plan(pins)
	if err != nil {
		return err
	}
	if p.Pending() {
		return fmt.Errorf("after the sync, config current=%v, not at their pin: %s", !p.ConfigChanged, strings.Join(append(append(p.Missing, p.MissingPackages...), p.Outdated...), ", "))
	}
	return nil
}

func (r toolsSync) entries(env Env) ([]tools.Tool, error) {
	if r.catalog == nil {
		return nil, nil
	}
	return loadCatalog(env)
}

func (r toolsSync) walk(entries []tools.Tool, apply bool) catalogPass {
	if r.catalog == nil {
		return catalogPass{}
	}
	return walkCatalog(r.catalog, entries, apply)
}

// syncHalf plans or applies the mise sync. mise absent after the first
// catalog pass is a note, not a failure: the catalog probe already fails when
// mise was due to be installed and was not.
func (r toolsSync) syncHalf(env Env, dryRun bool) (Result, error) {
	if skip := r.unavailable(); skip != "" {
		return Result{Detail: skip}, nil
	}
	s, pins, err := r.sync(env)
	if err != nil {
		return Result{}, err
	}
	plan := s.Plan
	if !dryRun {
		plan = s.Apply
	}
	p, err := plan(pins)
	res := Result{Changes: len(p.Missing) + len(p.MissingPackages) + len(p.Outdated), Detail: syncDetail(p, len(pins), dryRun)}
	if p.ConfigChanged {
		res.Changes++
	}
	return res, err
}

// reach appends to this process's PATH each directory the tool layer places
// binaries in that PATH lacks, keeping the caller's own entries first. Every
// lookup after it, in this step and in the steps that follow, then finds what
// the step installed: the hourly `dotf update` runs with a minimal PATH, and
// a fresh machine has no rc that adds ~/.local/bin or mise's shims yet.
func (r toolsSync) reach() error {
	path := os.Getenv("PATH")
	have := map[string]bool{}
	for _, dir := range filepath.SplitList(path) {
		have[dir] = true
	}
	for _, dir := range r.bins {
		if have[dir] {
			continue
		}
		have[dir] = true
		if path != "" {
			path += string(os.PathListSeparator)
		}
		path += dir
	}
	return os.Setenv("PATH", path)
}

// unavailable names what the mise half lacks on this machine, or "".
func (r toolsSync) unavailable() string {
	switch {
	case !r.has("mise"):
		return "mise is not on PATH; `dotf tools install mise` installs it"
	case r.run == nil || r.stdout == nil:
		return "no mise runner is wired into this registry"
	}
	return ""
}

func joinDetail(parts ...string) string {
	kept := parts[:0]
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "; ")
}

func (r toolsSync) sync(env Env) (tools.MiseSync, []tools.MiseTool, error) {
	raw, err := os.ReadFile(filepath.Join(env.RepoRoot, "versions.conf")) //nolint:gosec // the checkout's own pin file
	if err != nil {
		return tools.MiseSync{}, nil, err
	}
	all, err := tools.ParseMisePins(raw)
	if err != nil {
		return tools.MiseSync{}, nil, err
	}
	getenv := r.getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	s := tools.MiseSync{ConfigDir: tools.MiseConfigDir(env.Home, getenv), Run: r.run, Stdout: r.stdout, PythonPackages: all.PythonPackages, Latest: all.Latest, CheckUpgrades: true}
	return s, all.Tools, nil
}

func syncDetail(p tools.SyncPlan, total int, dryRun bool) string {
	if !p.Pending() {
		return fmt.Sprintf("%d pinned CLI(s) at their pin", total)
	}
	verb, up := "installed", "upgraded"
	if dryRun {
		verb, up = "to install", "to upgrade"
	}
	parts := []string{}
	if p.ConfigChanged {
		parts = append(parts, "mise config to write")
	}
	if len(p.Missing) > 0 {
		parts = append(parts, verb+": "+strings.Join(p.Missing, ", "))
	}
	if len(p.MissingPackages) > 0 {
		parts = append(parts, "python packages "+verb+": "+strings.Join(p.MissingPackages, ", "))
	}
	if len(p.Outdated) > 0 {
		parts = append(parts, up+" to their newest release: "+strings.Join(p.Outdated, ", "))
	}
	return strings.Join(parts, "; ")
}
