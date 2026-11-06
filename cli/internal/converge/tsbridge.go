package converge

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

const tsBridgeManagedHeader = "# Managed by dotf for the ts-bridge-client role.\n"

var tsBridgeGitContent = []byte(tsBridgeManagedHeader + `[http "https://gitea.kubelab.live"]
	proxy = socks5h://127.0.0.1:1081
`)

type tsBridgeClient struct{}

type tsBridgeView struct {
	enabled, missing, current bool
	path                      string
}

func (tsBridgeClient) Name() string        { return "ts-bridge-client" }
func (tsBridgeClient) Platforms() []string { return nil }

func (tsBridgeClient) Reconcile(e Env, dryRun bool) (Result, error) {
	v, err := inspectTsBridgeClient(e.Home)
	if err != nil {
		return Result{}, err
	}
	switch {
	case v.enabled && v.current:
		return Result{Detail: "Gitea Git proxy drop-in is current"}, nil
	case !v.enabled && v.missing:
		return Result{Skip: "this machine has not opted in (dotf env role add " + envpkg.RoleTSBridgeClient + ")"}, nil
	case v.enabled:
		res := Result{Changes: 1, Detail: "write " + v.path}
		if dryRun {
			return res, nil
		}
		return res, writeTsBridgeGitConfig(v.path)
	default:
		res := Result{Changes: 1, Detail: "remove " + v.path + " (role removed)"}
		if dryRun {
			return res, nil
		}
		return res, os.Remove(v.path)
	}
}

func (r tsBridgeClient) Probe(e Env) error {
	res, err := r.Reconcile(e, true)
	if err != nil || res.Skip != "" {
		return err
	}
	if res.Changes > 0 {
		return fmt.Errorf("ts-bridge client still differs: %s", res.Detail)
	}
	return nil
}

func tsBridgeRoleEnabled(home string) (bool, error) {
	roles, err := envpkg.LoadMachineRoles(envpkg.MachinePath(home))
	return slices.Contains(roles, envpkg.RoleTSBridgeClient), err
}

func inspectTsBridgeClient(home string) (tsBridgeView, error) {
	enabled, err := tsBridgeRoleEnabled(home)
	if err != nil {
		return tsBridgeView{}, err
	}
	path := tsBridgeGitConfig(home)
	raw, readErr := os.ReadFile(path) //nolint:gosec // role-owned user config
	missing := errors.Is(readErr, os.ErrNotExist)
	if readErr != nil && !missing {
		return tsBridgeView{}, readErr
	}
	if !missing && !bytes.HasPrefix(raw, []byte(tsBridgeManagedHeader)) {
		return tsBridgeView{}, fmt.Errorf(
			"%s exists but is not managed by dotf; refusing to reconcile because Git includes this path regardless of role state",
			path,
		)
	}
	return tsBridgeView{
		enabled: enabled,
		missing: missing,
		current: !missing && bytes.Equal(raw, tsBridgeGitContent),
		path:    path,
	}, nil
}

func tsBridgeGitConfig(home string) string {
	return filepath.Join(home, ".config", "git", "ts-bridge.gitconfig")
}

func writeTsBridgeGitConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ts-bridge-*.gitconfig")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(tsBridgeGitContent); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
