package converge

import (
	"fmt"
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// lanes activates the persistent named lanes of the checkout this run
// converges (HARNESS-195). It acts on env.RepoRoot and on nothing else:
// DeployDir is never read, and a RepoRoot that is the deploy clone is refused,
// so the deploy clone cannot become a lane repository by being converged. The
// signal is the marker committed on the default branch; see lane.Activate.
type lanes struct {
	resolve func(string) string // env.ResolvePath in production
	has     func(string) bool   // is a command on PATH
}

func (lanes) Name() string        { return "lanes" }
func (lanes) Platforms() []string { return nil }

// target answers where to act, or why this step does not apply right now.
func (r lanes) target(env Env) (root, skip string) {
	if !r.has("git") {
		return "", "git is not on PATH"
	}
	if r.resolve == nil {
		return "", "no path resolver is wired into this registry"
	}
	if root = r.resolve("WORKTREE_ROOT"); root == "" {
		return "", "WORKTREE_ROOT could not be resolved: run `dotf doctor`"
	}
	main, err := worktree.ResolveMainRepoRoot(env.RepoRoot)
	if err != nil {
		return "", env.RepoRoot + " is not a git checkout yet"
	}
	// Physical identity, never a name: the deploy clone is a configured field
	// of this run, so the comparison is between two fields. A linked worktree
	// of the deploy clone resolves to it too.
	if sameFile(env.RepoRoot, env.DeployDir) || sameFile(main, env.DeployDir) {
		return "", "the checkout is the deploy dir, which never runs lanes"
	}
	return root, ""
}

func sameFile(a, b string) bool {
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(fa, fb)
}

func (r lanes) Reconcile(env Env, dryRun bool) (Result, error) {
	root, skip := r.target(env)
	if skip != "" {
		return Result{Skip: skip}, nil
	}
	act, err := lane.Activate(env.RepoRoot, root, dryRun)
	if err != nil {
		return Result{}, err
	}
	switch {
	case !act.Active:
		return Result{Detail: "left as it is: " + lane.LanesMarkerFile + " is not committed on origin's default branch, so this repository does not run lanes"}, nil
	case act.Changes == 0:
		return Result{Detail: "lanes active (" + lane.LanesMarkerFile + " on " + act.Ref + ")"}, nil
	case dryRun:
		return Result{Changes: act.Changes, Detail: fmt.Sprintf("to apply: create the lanes and the activation record (%d change(s))", act.Changes)}, nil
	}
	return Result{Changes: act.Changes, Detail: fmt.Sprintf("applied: lanes created and activated (%d change(s))", act.Changes)}, nil
}

// Probe holds the post-condition: a repository whose default branch carries
// the marker has its lanes and its activation record.
func (r lanes) Probe(env Env) error {
	root, skip := r.target(env)
	if skip != "" {
		return nil
	}
	act, err := lane.Activate(env.RepoRoot, root, true)
	if err != nil {
		return err
	}
	if act.Active && act.Changes > 0 {
		return fmt.Errorf("%d lane or record change(s) are still pending after the apply", act.Changes)
	}
	return nil
}
