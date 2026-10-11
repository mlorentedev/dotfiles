package converge

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// recordsHarness deploys the agents' instruction files, skills and persona
// presence by running scripts/compile-harness.sh --deploy, which already does
// them in the order they need (instruction files, then the skill catalog and
// the presence regions injected into them). It is the instructions-first step
// (#2016): no agent is installed or run before its instructions exist.
//
// The plan and the probe are Go: an instruction file that is missing, or whose
// content outside the harness regions differs from its source, is a change.
// The deploy itself stays in the script until its Go port lands (PLAT-001b
// PR 2c), so linux and darwin only; Windows deploys these files from its own
// setup script, which runs as the legacy reconciler there.
type recordsHarness struct {
	run func(Env) error   // the deploy; CompileHarnessDeploy in production
	has func(string) bool // is a command on PATH
}

func (recordsHarness) Name() string { return "records-harness" }

// RevisitAfter re-runs the deploy after tools changed the machine: an agent
// tools installs had no instruction files when this step first ran (#2013 D11).
func (recordsHarness) RevisitAfter() string { return "tools" }
func (recordsHarness) Platforms() []string  { return []string{"linux", "darwin"} }

func (r recordsHarness) Reconcile(env Env, dryRun bool) (Result, error) {
	stale, err := r.staleInstructions(env)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: len(stale), Detail: instructionsDetail(stale, dryRun)}
	if dryRun {
		return res, nil
	}
	// Skills are not planned yet, so the deploy runs on every apply: it is
	// idempotent in content, and skipping it on a converged instruction set
	// would leave a changed skill record undeployed.
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

func instructionsDetail(stale []string, dryRun bool) string {
	if len(stale) == 0 {
		return "instruction files current; skills and presence re-rendered on apply"
	}
	verb := "deployed"
	if dryRun {
		verb = "to deploy"
	}
	return fmt.Sprintf("%d instruction file(s) %s: %s", len(stale), verb, strings.Join(stale, ", "))
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

// CompileHarnessDeploy runs scripts/compile-harness.sh --deploy from the
// checkout. Its output is returned with the error, so a failed deploy says why.
func CompileHarnessDeploy(env Env) error {
	cmd := exec.Command("bash", filepath.Join(env.RepoRoot, "scripts", "compile-harness.sh"), "--deploy") //nolint:gosec // the checkout's own script
	cmd.Dir = env.RepoRoot
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compile-harness.sh --deploy: %w\n%s", err, strings.TrimSpace(out.String()))
	}
	return nil
}

// onPath reports whether a command is on PATH.
func onPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
