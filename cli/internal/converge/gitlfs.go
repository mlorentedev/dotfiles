package converge

import (
	"fmt"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

// gitLFS writes git's lfs filter to the global config (#2201), so a clone of
// a repository that tracks files with LFS gets the files, not their pointers.
// The catalog installs git-lfs; this step is the `git lfs install` it needs
// once per user. It runs only when a key is absent: the command rewrites all
// four, so it cannot be the idempotence check itself. Every OS.
type gitLFS struct {
	run gitconfig.Runner  // gitconfig.ExecRunner in production
	has func(string) bool // is a command on PATH
}

func (gitLFS) Name() string        { return "git-lfs" }
func (gitLFS) Platforms() []string { return nil }

func (r gitLFS) inspect() (gitconfig.LFSState, string, error) {
	switch {
	case !r.has("git"):
		return gitconfig.LFSState{}, "git is not on PATH", nil
	case r.run == nil:
		return gitconfig.LFSState{}, "no git runner is wired into this registry", nil
	}
	st, err := gitconfig.InspectLFS(r.run)
	if err == nil && st.NotInstalled {
		return st, "git-lfs is not installed (dotf tools install git-lfs)", nil
	}
	return st, "", err
}

func (r gitLFS) Reconcile(_ Env, dryRun bool) (Result, error) {
	st, skip, err := r.inspect()
	if skip != "" || err != nil {
		return Result{Skip: skip}, err
	}
	var parts []string
	res := Result{}
	switch {
	case len(st.Absent) > 0 && len(st.Differ) > 0:
		// Measured with git-lfs 3.8.0: install exits 2 on a differing key and
		// writes nothing, so the absent ones wait for the owner's decision.
		parts = append(parts, "not installed while a key differs: "+strings.Join(st.Absent, ", "))
	case len(st.Absent) > 0:
		res.Changes = 1
		verb := "applied: "
		if dryRun {
			verb = "to apply: "
		}
		parts = append(parts, verb+"git lfs install --skip-repo ("+strings.Join(st.Absent, ", ")+")")
	}
	if len(st.Differ) > 0 {
		parts = append(parts, "left as it is: "+strings.Join(st.Differ, "; "))
	}
	if len(parts) == 0 {
		parts = append(parts, "lfs filter in place")
	}
	res.Detail = strings.Join(parts, "; ")
	if dryRun || res.Changes == 0 {
		return res, nil
	}
	return res, gitconfig.InstallLFS(r.run)
}

// Probe holds the post-condition: no filter key is absent where git-lfs is
// installed and none differs. A different value is left by design and
// already named, and keeps install from running.
func (r gitLFS) Probe(Env) error {
	st, skip, err := r.inspect()
	if skip != "" || err != nil {
		return err
	}
	if len(st.Absent) > 0 && len(st.Differ) == 0 {
		return fmt.Errorf("after git lfs install, still absent: %s", strings.Join(st.Absent, ", "))
	}
	return nil
}
