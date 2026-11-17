package converge

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// userScopeState is the Windows half of env-persist: what HKCU\Environment
// holds against the contract and the tool directories.
type userScopeState struct {
	vars        []env.ResolvedVar
	drift       []env.ResolvedVar
	retired     []string
	markerStale bool
	path        env.PathPlan
}

func (s userScopeState) changes() int {
	n := len(s.drift) + len(s.retired)
	for _, stale := range []bool{s.markerStale, s.path.Changed, s.path.MarkerStale} {
		if stale {
			n++
		}
	}
	return n
}

func (s userScopeState) detail(dryRun bool) string {
	if s.changes() == 0 {
		return fmt.Sprintf("%d variable(s) and the tool directories in the user environment", len(s.vars))
	}
	var parts []string
	if len(s.drift) > 0 {
		names := make([]string, 0, len(s.drift))
		for _, v := range s.drift {
			names = append(names, v.Name)
		}
		parts = append(parts, "set "+strings.Join(names, ", "))
	}
	if len(s.retired) > 0 {
		parts = append(parts, "unset "+strings.Join(s.retired, ", "))
	}
	if s.markerStale {
		parts = append(parts, "record "+env.ManagedMarker)
	}
	if s.path.Changed {
		parts = append(parts, "put the tool directories at the head of the user Path")
	}
	if s.path.MarkerStale {
		parts = append(parts, "record "+env.ManagedPathMarker)
	}
	verb := "applied: "
	if dryRun {
		verb = "to apply: "
	}
	return verb + strings.Join(parts, "; ")
}

func (r envPersist) stores() (env.UserEnvStore, env.UserPathStore, error) {
	store, err := r.userEnv()
	if err != nil {
		return nil, nil, err
	}
	path, err := r.userPath()
	return store, path, err
}

func (r envPersist) inspectUser(e Env, store env.UserEnvStore, path env.UserPathStore) (userScopeState, error) {
	var st userScopeState
	vars, err := env.ResolveVars(filepath.Join(e.RepoRoot, "env-contract.json"), env.MachinePath(e.Home), e.GOOS, e.Home)
	if err != nil {
		return st, err
	}
	if err := env.ValidateNames(vars); err != nil {
		return st, err
	}
	st.vars = vars
	if st.drift, err = env.Drift(vars, store); err != nil {
		return st, err
	}
	if st.retired, err = env.Retired(store, vars); err != nil {
		return st, err
	}
	if st.markerStale, err = env.MarkerStale(store, vars); err != nil {
		return st, err
	}
	st.path, err = env.PlanPath(r.bins, store, path, r.expand)
	return st, err
}

func (r envPersist) reconcileUser(e Env, dryRun bool) (Result, error) {
	if r.userEnv == nil || r.userPath == nil {
		return Result{Skip: "no user environment store is wired into this registry"}, nil
	}
	store, path, err := r.stores()
	if err != nil {
		return Result{}, err
	}
	st, err := r.inspectUser(e, store, path)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: st.changes(), Detail: st.detail(dryRun)}
	if dryRun || res.Changes == 0 {
		return res, nil
	}
	if _, err := env.Persist(st.vars, store); err != nil {
		return res, err
	}
	_, err = env.PersistPath(r.bins, store, path, r.expand)
	return res, err
}

// probeUser holds the post-condition in the store, not in this process: a
// registry write changes the next process's environment, never the running one.
func (r envPersist) probeUser(e Env) error {
	if r.userEnv == nil || r.userPath == nil {
		return nil
	}
	store, path, err := r.stores()
	if err != nil {
		return err
	}
	st, err := r.inspectUser(e, store, path)
	if err != nil {
		return err
	}
	if st.changes() > 0 {
		return fmt.Errorf("the user environment still differs: %s", st.detail(true))
	}
	return nil
}
