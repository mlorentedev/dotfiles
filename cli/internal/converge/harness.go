package converge

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// recordsHarness deploys the agents' instruction files, skills, agent records,
// persona presence and doctrine with harness.Deploy, in the order they need
// (instruction files, then the skill catalog and the presence regions injected
// into them). It is the instructions-first step (#2016): no agent is installed
// or run before its instructions exist.
//
// The plan and the probe read the instruction files: one that is missing, or
// whose content outside the harness regions differs from its source, is a
// change. They also judge the skill records, agent records and doctrine by
// their rendered form, through a dry run of the same deploy: an output to
// write or prune is a change (#2013 AC2). It runs on every OS: setup-windows.ps1 and setup-linux.sh deploy
// through the same harness.Deploy, as `dotf harness deploy`.
type recordsHarness struct {
	run     func(Env) error                       // the deploy; HarnessDeploy in production
	preview func(Env) (harness.Deployment, error) // its dry run; HarnessPreview in production, nil previews nothing
	has     func(string) bool                     // is a command on PATH
}

func (recordsHarness) Name() string { return "records-harness" }

// RevisitAfter re-runs the deploy after tools changed the machine: an agent
// tools installs had no instruction files when this step first ran (#2013 D11).
func (recordsHarness) RevisitAfter() string { return "tools" }
func (recordsHarness) Platforms() []string  { return []string{"linux", "darwin", "windows"} }

func (r recordsHarness) Reconcile(env Env, dryRun bool) (Result, error) {
	stale, err := r.staleInstructions(env)
	if err != nil {
		return Result{}, err
	}
	records, err := r.staleRecords(env)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: len(stale) + records.total(), Detail: instructionsDetail(stale, records, dryRun)}
	if dryRun {
		return res, nil
	}
	// The deploy still runs on every apply, a plan of 0 changes included:
	// presence has no preview, so 0 is not yet proof that nothing is stale.
	// The deploy rewrites only what differs, so a converged machine is left
	// as it is.
	if r.run == nil {
		return res, errors.New("no harness deploy runner is wired into this registry")
	}
	if err := r.run(env); err != nil {
		return res, err
	}
	return res, nil
}

// Probe holds the post-condition: every applicable instruction file exists and
// matches its source outside the harness regions, and each presence region the
// records render is current.
func (r recordsHarness) Probe(env Env) error {
	stale, err := r.staleInstructions(env)
	if err != nil {
		return err
	}
	if len(stale) > 0 {
		return fmt.Errorf("instruction files still missing or stale after the deploy: %s", strings.Join(stale, ", "))
	}
	records, err := r.staleRecords(env)
	if err != nil {
		return err
	}
	if records.total() > 0 {
		return fmt.Errorf("records still to deploy after the deploy: %s", records)
	}
	return presenceCurrent(env, r.has)
}

// staleInstructions lists the home-relative instruction files that are missing
// or do not hold their source (harness.DeployedMatchesSource, the comparison
// doctor's instruction-drift check makes).
func (r recordsHarness) staleInstructions(env Env) ([]string, error) {
	_, targets, err := harness.LoadPresence(filepath.Join(env.RepoRoot, filepath.FromSlash(harness.ManifestFile)))
	if err != nil {
		return nil, err
	}
	var stale []string
	for _, t := range targets {
		if t.Source == "" || (t.RequiresCommand != "" && !r.has(t.RequiresCommand)) {
			continue
		}
		src, err := os.ReadFile(filepath.Join(env.RepoRoot, filepath.FromSlash(t.Source)))
		if err != nil {
			return nil, fmt.Errorf("instruction source for %s: %w", t.Agent, err)
		}
		dst, err := os.ReadFile(filepath.Join(env.Home, filepath.FromSlash(t.File)))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err != nil || !harness.DeployedMatchesSource(string(dst), string(src)) {
			stale = append(stale, t.File)
		}
	}
	return stale, nil
}

// recordChanges counts the outputs a dry run of the deploy would write or
// prune, per surface.
type recordChanges struct{ skills, agents, doctrine int }

func (c recordChanges) total() int { return c.skills + c.agents + c.doctrine }

func (c recordChanges) String() string {
	return fmt.Sprintf("%d skill, %d agent and %d doctrine output(s)", c.skills, c.agents, c.doctrine)
}

// staleRecords previews the deploy and counts what it would change. The
// instruction outcomes are left out: staleInstructions judges those files
// with the harness regions stripped, which the deploy's own outcome does not.
func (r recordsHarness) staleRecords(env Env) (recordChanges, error) {
	if r.preview == nil {
		return recordChanges{}, nil
	}
	d, err := r.preview(env)
	if err != nil {
		return recordChanges{}, fmt.Errorf("preview the harness records: %w", err)
	}
	var c recordChanges
	for _, o := range d.Skills {
		if o.Status == harness.SkillDeployed || o.Status == harness.SkillPruned {
			c.skills++
		}
	}
	for _, o := range d.Agents {
		if o.Status == harness.SkillDeployed || o.Status == harness.SkillPruned {
			c.agents++
		}
	}
	for _, o := range d.Doctrine {
		if o.Status == harness.SkillDeployed {
			c.doctrine++
		}
	}
	return c, nil
}

func instructionsDetail(stale []string, records recordChanges, dryRun bool) string {
	verb := "deployed"
	if dryRun {
		verb = "to deploy"
	}
	var parts []string
	if len(stale) == 0 {
		parts = append(parts, "instruction files current")
	} else {
		parts = append(parts, fmt.Sprintf("%d instruction file(s) %s: %s", len(stale), verb, strings.Join(stale, ", ")))
	}
	if records.total() == 0 {
		parts = append(parts, "skill, agent and doctrine records current")
	} else {
		parts = append(parts, records.String()+" "+verb)
	}
	return strings.Join(parts, "; ")
}

// presenceCurrent checks each presence region the records render against the
// deployed file, for the targets the plan applies to (requires_command on
// PATH): a file an uninstalled agent left behind is not this run's to fix. A
// manifest without agent records renders no roster, so there is nothing to
// check.
func presenceCurrent(env Env, has func(string) bool) error {
	recordDir, targets, err := harness.LoadPresence(filepath.Join(env.RepoRoot, filepath.FromSlash(harness.ManifestFile)))
	if err != nil || recordDir == "" {
		return err
	}
	for _, t := range targets {
		if t.RequiresCommand != "" && !has(t.RequiresCommand) {
			continue
		}
		block, err := harness.RenderPresence(env.RepoRoot, t.Agent)
		if err != nil {
			return err
		}
		path := filepath.Join(env.Home, filepath.FromSlash(t.File))
		if block == "" || !fileExists(path) {
			continue
		}
		state, err := harness.PresenceStatus(path, block)
		if err != nil {
			return err
		}
		if state != harness.PresenceCurrent {
			return fmt.Errorf("presence region in %s is %s", t.File, state)
		}
	}
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// HarnessPreview is HarnessDeploy's dry run: what each surface would write or
// prune, with nothing written. Presence has no preview, so it reports none.
func HarnessPreview(env Env) (harness.Deployment, error) {
	return harness.Deploy(env.RepoRoot, env.Home, onPath, true)
}

// HarnessDeploy deploys every harness surface from the checkout into the home.
// The error names each item that failed, so a failed deploy says why.
func HarnessDeploy(env Env) error {
	_, err := harness.Deploy(env.RepoRoot, env.Home, onPath, false)
	return err
}

// onPath reports whether a command is on PATH.
func onPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
