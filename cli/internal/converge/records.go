package converge

import (
	"errors"
	"fmt"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// Options carries the side effects a registry needs from its caller, so a test
// can drive the registry without running the real tools (lesson 335).
type Options struct {
	// RunHarnessDeploy deploys the instruction files, skills and presence;
	// CompileHarnessDeploy in production.
	RunHarnessDeploy func(Env) error
	// MiseRun and MiseStdout run mise and the tools it installs;
	// tools.HomeRunners in production. Unset, the tools step is skipped.
	MiseRun, MiseStdout tools.Runner
	// GitRun runs git and gh and returns stdout; gitconfig.ExecRunner in
	// production. Unset, the checkout and git-config steps are skipped.
	GitRun gitconfig.Runner
	// CloneURL is what the checkout step clones when the checkout is absent;
	// empty means DefaultCloneURL.
	CloneURL string
	// RenderConfigs substitutes {env:VAR} in a staged config and returns
	// deploy.ErrRenderIncomplete when the secret store could not answer;
	// ResolvePath resolves {VAR} in a destination (env.ResolvePath). Unset,
	// the configs step fails rather than install unrendered placeholders.
	RenderConfigs deploy.Renderer
	ResolvePath   func(string) string
	// Launchctl runs launchctl and returns stdout; env.ExecLaunchctl in
	// production, and UID is the user whose gui domain it targets. Unset, the
	// env-persist step is skipped.
	Launchctl env.LaunchctlRunner
	UID       int
	// RunSetup runs the setup script of this OS; ExecSetup in production.
	// Unset, the legacy-setup step is skipped.
	RunSetup func(Env) error
}

// Registry is the ordered list of reconcilers a run drives. The order extends
// ADR-041 decision 4: the checkout comes first, since every step reads it, and
// records come before anything that reads them (ADR-045 decision 4). The
// setup script runs last, after every native step, for whatever the native
// steps do not cover yet; each row that ports a block of it adds a native
// reconciler before it.
func Registry(o Options) []Reconciler {
	url := o.CloneURL
	if url == "" {
		url = DefaultCloneURL
	}
	return []Reconciler{
		checkout{run: o.GitRun, url: url, has: onPath},
		recordsMirror{},
		recordsHarness{run: o.RunHarnessDeploy, has: onPath},
		toolsSync{run: o.MiseRun, stdout: o.MiseStdout, has: onPath},
		configsDeploy{render: o.RenderConfigs, resolve: o.ResolvePath, has: onPath},
		gitConfig{run: o.GitRun, has: onPath},
		envPersist{launchctl: o.Launchctl, uid: o.UID},
		legacySetup{run: o.RunSetup},
	}
}

// recordsMirror mirrors harness/, every manifest target and the deploy-dir set
// (rc files, versions.conf, scripts/ …) from the checkout into the deploy dir
// (harness.Mirror). On macOS it is the only thing that refreshes them (#2224). It is the first step of the records
// phase: the instruction files and the hook bindings read what it mirrors.
type recordsMirror struct{}

func (recordsMirror) Name() string        { return "records-mirror" }
func (recordsMirror) Platforms() []string { return nil }

func (recordsMirror) Reconcile(env Env, dryRun bool) (Result, error) {
	mirror, verb := harness.Mirror, "updated"
	if dryRun {
		mirror, verb = harness.PlanMirror, "to write"
	}
	res, err := mirror(env.RepoRoot, env.DeployDir)
	if errors.Is(err, harness.ErrCheckoutIsDeployDir) {
		return Result{Detail: "the checkout is the deploy dir; nothing to mirror"}, nil
	}
	if err != nil {
		return Result{}, err
	}
	return Result{
		Changes: res.Updated,
		Detail: fmt.Sprintf("harness/ + %d target(s) + the deploy-dir set → %s (%d %s, %d unchanged)",
			len(res.Targets), env.DeployDir, res.Updated, verb, res.Unchanged),
	}, nil
}

// Probe re-plans the mirror: after an apply, nothing may still differ.
func (recordsMirror) Probe(env Env) error {
	res, err := harness.PlanMirror(env.RepoRoot, env.DeployDir)
	if errors.Is(err, harness.ErrCheckoutIsDeployDir) {
		return nil
	}
	if err != nil {
		return err
	}
	if res.Updated > 0 {
		return fmt.Errorf("%d file(s) in %s still differ from the checkout", res.Updated, env.DeployDir)
	}
	return nil
}
