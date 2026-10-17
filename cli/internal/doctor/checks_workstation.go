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
// runs the converge step's own read-only plan, with the step's gates: the
// step's platforms only, skipped until the machine opts in, and skipped off the mesh. It does
// not run the proof; that asks sshd over the network and stays converge's, so
// doctor remains read-only and fast.
func checkWorkstationTarget(sys *System, cfg *Config, rep *Report) {
	if sys.GOOS != "darwin" && sys.GOOS != "linux" && sys.GOOS != "windows" {
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
	user, err := sshtarget.CurrentUser()
	if err != nil {
		rep.Warn("cannot tell which account sshd is opened to: " + err.Error())
		return
	}
	t, err := sshtarget.New(filepath.Dir(filepath.Dir(tmpl)), user, sys.GOOS, workstationPaths(sys, home), workstationRun(sys), workstationAddrs(sys))
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
		what := "key, host keys, sshd drop-in and service"
		if t.Firewall != nil {
			what = "firewall, " + what
		}
		msg := fmt.Sprintf("%s as declared for the mesh address %s", what, p.Mesh)
		for _, n := range p.Notes {
			msg += "; " + n
		}
		rep.Pass(msg)
	}
	if err == nil && p.MeshAtBoot != "" {
		rep.Warn(p.MeshAtBoot)
	}
}

func workstationPaths(sys *System, home string) sshtarget.Paths {
	if sys.SSHTargetPaths != nil {
		return sys.SSHTargetPaths(home)
	}
	return sshtarget.PathsFor(sys.GOOS, home)
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
