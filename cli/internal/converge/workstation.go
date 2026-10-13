package converge

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/sshtarget"
)

// workstationTarget makes a machine that declares the ssh-target role accept
// SSH from the mesh and from nothing else (#2356, #2013 A1). The plan reads
// without root. The apply asks for root once on the terminal, and without a
// terminal it is skipped with the plan as its reason (ADR-045 amendment
// 2026-10-12). Every apply, and every probe, ends with sshd accepting the
// owner's key from the mesh address and refusing it from the LAN address.
type workstationTarget struct {
	root  sshtarget.Root   // sudo on the terminal in production
	run   sshtarget.Runner // the user's commands: launchctl print, ssh
	addrs func() ([]net.IP, error)
	paths func(home string) sshtarget.Paths
	user  func() (string, error)
}

func (workstationTarget) Name() string { return "workstation-target" }

// Platforms is where sshd has a service manager the target drives: launchd
// and systemd. Windows follows (#2356, PLAT-001b PR 7e).
func (workstationTarget) Platforms() []string { return []string{"darwin", "linux"} }

func (r workstationTarget) target(e Env) (sshtarget.Target, error) {
	u, err := r.user()
	if err != nil {
		return sshtarget.Target{}, err
	}
	t, err := sshtarget.New(e.RepoRoot, u, e.GOOS, r.paths(e.Home), r.run, r.addrs)
	t.Root = r.root
	return t, err
}

func (r workstationTarget) Reconcile(e Env, dryRun bool) (Result, error) {
	if r.root == nil || r.run == nil {
		return Result{Skip: "no root or user runner is wired into this registry"}, nil
	}
	roles, err := env.LoadMachineRoles(env.MachinePath(e.Home))
	if err != nil {
		return Result{}, err
	}
	if !slices.Contains(roles, env.RoleSSHTarget) {
		return Result{Skip: "this machine has not opted in (dotf env role add " + env.RoleSSHTarget + ")"}, nil
	}
	t, err := r.target(e)
	if err != nil {
		return Result{}, err
	}
	p, err := t.Inspect()
	if errors.Is(err, sshtarget.ErrNoMesh) {
		return Result{Skip: err.Error()}, nil
	}
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: len(p.Pending), Detail: workstationDetail(p, dryRun)}
	if dryRun {
		return res, nil
	}
	err = t.Apply(p)
	if errors.Is(err, sshtarget.ErrNoTerminal) {
		return Result{Skip: err.Error() + "; " + workstationDetail(p, true)}, nil
	}
	return res, err
}

// Probe re-inspects, then asks sshd again from both addresses: a later change
// to the machine that reopens SSH fails the next converge.
func (r workstationTarget) Probe(e Env) error {
	t, err := r.target(e)
	if err != nil {
		return err
	}
	p, err := t.Inspect()
	if err != nil {
		return err
	}
	if !p.Converged() {
		return fmt.Errorf("still differs: %s", strings.Join(p.Pending, ", "))
	}
	return t.Prove(p)
}

func workstationDetail(p sshtarget.Plan, dryRun bool) string {
	lan := "no LAN address to test the refusal from"
	if p.LAN != nil {
		lan = "refused from " + p.LAN.String()
	}
	proof := fmt.Sprintf("sshd accepts the key from %s, %s", p.Mesh, lan)
	for _, n := range p.Notes {
		proof += "; " + n
	}
	switch {
	case p.Converged() && dryRun:
		return "converged; a run proves: " + proof
	case p.Converged():
		return "converged; " + proof
	case dryRun:
		return "to apply as root: " + strings.Join(p.Pending, ", ") + "; then proves: " + proof
	}
	return "applied: " + strings.Join(p.Pending, ", ") + "; " + proof
}
