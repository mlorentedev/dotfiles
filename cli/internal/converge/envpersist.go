package converge

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// envPersist gives GUI apps on macOS the contract's path variables (#2013 S3,
// F-021). An app launched from the Dock reads no rc file, only the user's
// launchd session, so VAULT_PATH and the rest are written there through
// `launchctl setenv` (env.LaunchdUserEnv). That session is cleared at logout,
// so a LaunchAgent re-runs `dotf env persist` at every login.
type envPersist struct {
	launchctl env.LaunchctlRunner // env.ExecLaunchctl in production
	uid       int
}

func (envPersist) Name() string        { return "env-persist" }
func (envPersist) Platforms() []string { return []string{"darwin"} }

type envPersistState struct {
	vars        []env.ResolvedVar
	plistStale  bool // the agent's plist is missing or differs
	loaded      bool // launchd knows the agent
	drift       []env.ResolvedVar
	retired     []string
	markerStale bool
}

func (s envPersistState) changes() int {
	n := len(s.drift) + len(s.retired)
	if s.plistStale {
		n++
	}
	if s.plistStale || !s.loaded {
		n++ // (re)load the agent
	}
	if s.markerStale {
		n++
	}
	return n
}

func (r envPersist) service() string {
	return "gui/" + strconv.Itoa(r.uid) + "/" + env.LaunchAgentLabel
}

func (r envPersist) inspect(e Env) (envPersistState, error) {
	var st envPersistState
	vars, err := env.ResolveVars(filepath.Join(e.RepoRoot, "env-contract.json"), env.MachinePath(e.Home), e.GOOS, e.Home)
	if err != nil {
		return st, err
	}
	if err := env.ValidateNames(vars); err != nil {
		return st, err
	}
	st.vars = vars
	have, err := os.ReadFile(env.LaunchAgentPath(e.Home))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return st, err
	}
	st.plistStale = !bytes.Equal(have, env.LaunchAgentPlist(e.Home))
	// `launchctl print` exits non-zero (113) for a service launchd does not know.
	_, err = r.launchctl("print", r.service())
	st.loaded = err == nil

	store := env.LaunchdUserEnv{Run: r.launchctl}
	if st.drift, err = env.Drift(vars, store); err != nil {
		return st, err
	}
	if st.retired, err = env.Retired(store, vars); err != nil {
		return st, err
	}
	st.markerStale, err = env.MarkerStale(store, vars)
	return st, err
}

func (r envPersist) Reconcile(e Env, dryRun bool) (Result, error) {
	if r.launchctl == nil {
		return Result{Skip: "no launchctl runner is wired into this registry"}, nil
	}
	st, err := r.inspect(e)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: st.changes(), Detail: envPersistDetail(st, dryRun)}
	if dryRun || res.Changes == 0 {
		return res, nil
	}
	if err := r.loadAgent(e, st); err != nil {
		return res, err
	}
	_, err = env.Persist(st.vars, env.LaunchdUserEnv{Run: r.launchctl})
	return res, err
}

// loadAgent writes the plist and has launchd load it. A loaded agent whose
// plist changed is booted out first, because bootstrap refuses a loaded label
// and launchd keeps the definition it read at load time.
func (r envPersist) loadAgent(e Env, st envPersistState) error {
	path := env.LaunchAgentPath(e.Home)
	if st.plistStale {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // ~/Library/LaunchAgents is world-readable by convention
			return err
		}
		if err := secrets.AtomicWriteMode(path, env.LaunchAgentPlist(e.Home), 0o644); err != nil {
			return err
		}
		if st.loaded {
			if _, err := r.launchctl("bootout", r.service()); err != nil {
				return fmt.Errorf("launchctl bootout %s: %w", r.service(), err)
			}
		}
	}
	if st.plistStale || !st.loaded {
		domain := "gui/" + strconv.Itoa(r.uid)
		if _, err := r.launchctl("bootstrap", domain, path); err != nil {
			return fmt.Errorf("launchctl bootstrap %s %s: %w", domain, path, err)
		}
	}
	return nil
}

// Probe holds the post-condition: the agent is current and loaded, so the next
// login re-applies the scope, and this session already holds every variable.
func (r envPersist) Probe(e Env) error {
	if r.launchctl == nil {
		return nil
	}
	st, err := r.inspect(e)
	if err != nil {
		return err
	}
	switch {
	case st.plistStale:
		return fmt.Errorf("%s is still missing or out of date", env.LaunchAgentPath(e.Home))
	case !st.loaded:
		return fmt.Errorf("launchd has not loaded %s", r.service())
	case len(st.drift) > 0 || len(st.retired) > 0 || st.markerStale:
		return fmt.Errorf("the launchd session still differs from the contract: %s", envPersistDetail(st, true))
	}
	return nil
}

func envPersistDetail(st envPersistState, dryRun bool) string {
	if st.changes() == 0 {
		return fmt.Sprintf("%d variable(s) in the launchd session, login agent loaded", len(st.vars))
	}
	var parts []string
	if n := len(st.drift); n > 0 {
		names := make([]string, 0, n)
		for _, v := range st.drift {
			names = append(names, v.Name)
		}
		parts = append(parts, "set "+strings.Join(names, ", "))
	}
	if len(st.retired) > 0 {
		parts = append(parts, "unset "+strings.Join(st.retired, ", "))
	}
	if st.markerStale {
		parts = append(parts, "record "+env.ManagedMarker)
	}
	if st.plistStale {
		parts = append(parts, "write the login agent")
	}
	if st.plistStale || !st.loaded {
		parts = append(parts, "load the login agent")
	}
	verb := "applied: "
	if dryRun {
		verb = "to apply: "
	}
	return verb + strings.Join(parts, "; ")
}
