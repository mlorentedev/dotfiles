package doctor

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"slices"
	"strings"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/sshtarget"
)

// workstationRemedy is the one command that converges the SSH target. Doctor
// names it and never runs it: the apply asks for root, and doctor never
// prompts for root (ADR-045 amendment 2026-10-12, the env-persist precedent).
const workstationRemedy = "`dotf converge --only workstation-target`"

// checkWorkstationTarget (#2356, PLAT-001b PR 7d) reports whether a machine
// that declared the ssh-target role still matches its declared SSH posture. It
// runs the converge step's own read-only plan, with the step's gates: darwin
// only, skipped until the machine opts in, and skipped off the mesh. It does
// not run the proof; that asks sshd over the network and stays converge's, so
// doctor remains read-only and fast.
func checkWorkstationTarget(sys *System, cfg *Config, rep *Report) {
	if sys.GOOS != "darwin" {
		return
	}
	rep.Section("Workstation SSH target")
	home := sys.home()
	roles, err := envpkg.LoadMachineRoles(envpkg.MachinePath(home))
	if err != nil {
		rep.Warn(err.Error())
		return
	}
	if !slices.Contains(roles, envpkg.RoleSSHTarget) {
		rep.Skip("this machine has not opted in (dotf env role add " + envpkg.RoleSSHTarget + ")")
		return
	}
	tmpl := envpkg.ResolveRepoFirst(sshtarget.TemplateFile, sys.Getenv("DOTFILES_REPO_DIR"), cfg.DotfilesDir, "")
	if tmpl == "" {
		rep.Warn(sshtarget.TemplateFile + " not found in the checkout or the deploy mirror: set DOTFILES_REPO_DIR to the checkout to check the SSH target")
		return
	}
	user := sys.Getenv("USER")
	if user == "" {
		rep.Warn("USER is unset: cannot tell which account sshd is opened to")
		return
	}
	t, err := sshtarget.New(filepath.Dir(filepath.Dir(tmpl)), user, workstationPaths(sys, home), workstationRun(sys), workstationAddrs(sys))
	if err != nil {
		rep.Warn("read the SSH target's repo files: " + err.Error())
		return
	}
	p, err := t.Inspect()
	switch {
	case errors.Is(err, sshtarget.ErrNoMesh):
		rep.Skip(err.Error())
	case err != nil:
		rep.Warn(err.Error())
	case !p.Converged():
		rep.Warn(fmt.Sprintf("differs from the declared state: %s; run %s (it asks for root once)", strings.Join(p.Pending, ", "), workstationRemedy))
	default:
		rep.Pass(fmt.Sprintf("key, host keys, sshd drop-in and service as declared for the mesh address %s", p.Mesh))
	}
}

func workstationPaths(sys *System, home string) sshtarget.Paths {
	if sys.SSHTargetPaths != nil {
		return sys.SSHTargetPaths(home)
	}
	return sshtarget.DefaultPaths(home)
}

func workstationAddrs(sys *System) func() ([]net.IP, error) {
	if sys.MeshAddrs != nil {
		return sys.MeshAddrs
	}
	return sshtarget.InterfaceAddrs
}

func workstationRun(sys *System) sshtarget.Runner {
	return func(argv ...string) ([]byte, error) {
		out, err := sys.CommandOutput(argv[0], argv[1:]...)
		return []byte(out), err
	}
}
