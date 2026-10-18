package converge

import (
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/sshtarget"
)

// workstationTarget makes a machine that declares the ssh-target role accept
// SSH from the mesh and from nothing else (#2356, #2013 A1). The plan reads
// without root. The apply asks for root once on the terminal, and without a
// terminal it is skipped with the plan as its reason (ADR-045 amendment
// 2026-10-12); on Windows it runs only in an elevated process, and is skipped
// the same way in any other. Every apply, and every probe, ends with sshd accepting the
// owner's key from the mesh address and refusing it from the LAN address.
type workstationTarget struct {
	root  sshtarget.Root   // sudo on the terminal in production
	run   sshtarget.Runner // the user's commands: launchctl print, ssh
	addrs func() ([]net.IP, error)
	paths func(goos, home string) sshtarget.Paths
	user  func() (string, error)
}

func (workstationTarget) Name() string { return "workstation-target" }

// Platforms is where sshd has a service manager the target drives: launchd,
// systemd and the Windows service control manager (#2356, PLAT-001b PR 7e).
func (workstationTarget) Platforms() []string { return []string{"darwin", "linux", "windows"} }

func (r workstationTarget) target(e Env) (sshtarget.Target, error) {
	u, err := r.user()
	if err != nil {
		return sshtarget.Target{}, err
	}
	t, err := sshtarget.New(e.RepoRoot, u, e.GOOS, r.paths(e.GOOS, e.Home), r.run, r.addrs)
	t.Root = r.root
	t.Dotf = e.Dotf
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
		return r.retire(e, dryRun)
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
	if errors.Is(err, sshtarget.ErrNoTerminal) || errors.Is(err, sshtarget.ErrNotElevated) {
		return Result{Skip: err.Error() + "; " + workstationDetail(p, true)}, nil
	}
	if err != nil {
		// Apply undoes what it changed, so nothing is kept and the line is the
		// error alone: the plan it tried, then what failed and whether the undo
		// ran. Never a change kept or a proof that did not run (#2765).
		return Result{}, fmt.Errorf("planned as root: %s; %w", strings.Join(p.Pending, ", "), err)
	}
	return res, nil
}

// retire undoes, on a machine whose role was removed, what the step's record
// says it created (SEC-014, #2388). Without a record there is nothing the
// step can tell is its own, and the step is skipped as before.
func (r workstationTarget) retire(e Env, dryRun bool) (Result, error) {
	skip := Result{Skip: "this machine has not opted in (dotf env role add " + env.RoleSSHTarget + ")"}
	if rec := r.paths(e.GOOS, e.Home).Record; rec == "" {
		return skip, nil
	} else if _, err := os.Stat(rec); errors.Is(err, os.ErrNotExist) {
		return skip, nil
	}
	t, err := r.target(e)
	if err != nil {
		return Result{}, err
	}
	p, err := t.InspectRetire()
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: len(p.Pending), Detail: retireDetail(p, dryRun)}
	if dryRun {
		return res, nil
	}
	err = t.Retire(p)
	if errors.Is(err, sshtarget.ErrNoTerminal) || errors.Is(err, sshtarget.ErrNotElevated) {
		return Result{Skip: err.Error() + "; " + retireDetail(p, true)}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("planned to retire as root: %s; %w", strings.Join(p.Pending, ", "), err)
	}
	return res, nil
}

func retireDetail(p sshtarget.Retirement, dryRun bool) string {
	d := "retired"
	switch {
	case p.Converged():
		d = "nothing left to retire"
	case dryRun:
		d = "to retire as root: " + strings.Join(p.Pending, ", ")
	}
	for _, n := range append(p.Notes, "the firewall and power policy stay (#2794)") {
		d += "; " + n
	}
	return d
}

// Probe re-inspects, then asks sshd again from both addresses: a later change
// to the machine that reopens SSH fails the next converge. On a machine whose
// role was removed, it holds the retirement to its plan instead.
func (r workstationTarget) Probe(e Env) error {
	roles, err := env.LoadMachineRoles(env.MachinePath(e.Home))
	if err != nil {
		return err
	}
	t, err := r.target(e)
	if err != nil {
		return err
	}
	if !slices.Contains(roles, env.RoleSSHTarget) {
		p, err := t.InspectRetire()
		if err == nil && !p.Converged() {
			err = fmt.Errorf("still to retire: %s", strings.Join(p.Pending, ", "))
		}
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
	if p.MeshAtBoot != "" {
		proof += "; " + p.MeshAtBoot
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
