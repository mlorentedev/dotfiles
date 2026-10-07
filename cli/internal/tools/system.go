package tools

import (
	"fmt"
	"strings"
)

// System packages (source.type "system") are converged by the OS package
// manager, not downloaded by dotf: linux goes through apt, darwin through
// Homebrew, windows through winget. There is no pin to compare, so the
// reconcile policy is presence alone and the answer is install or skip, never
// upgrade; a second run therefore runs no manager command.
//
// Presence is one rule, asked in one order (systemInstalled): the entry's
// declared Command is on PATH, otherwise the manager's own record lists the
// package. The command shortcut makes a copy from any other channel count and
// needs no privilege to find; the record is the only answer for a library or a
// GUI app, and the post-condition of every install (a manager that exits 0
// without the package listed is an error, as an npm or uv install is).

// systemPresent is the sentinel Plan.Installed carries for a system package that
// is there: it has no version to report.
const systemPresent = "present"

// managerBinary is the executable that runs a manager's installs, which must be
// on PATH for an install to be possible.
func managerBinary(manager string) string {
	if manager == "apt" {
		return "apt-get"
	}
	return manager
}

// systemInstalled returns systemPresent when the package is on this machine and
// "" when it is absent or the entry names no package for this OS.
func (in *Installer) systemInstalled(t Tool) string {
	manager, pkg := t.Source.SystemPackage(in.GOOS)
	if pkg == "" {
		return ""
	}
	if cmd := t.Source.Command; cmd != "" && in.HasCommand(cmd) {
		return systemPresent
	}
	if in.managerLists(manager, pkg) {
		return systemPresent
	}
	return ""
}

// managerLists asks the manager whether pkg is installed. A missing manager
// binary fails the query, which reads as absent.
func (in *Installer) managerLists(manager, pkg string) bool {
	query := in.Query
	if query == nil {
		query = ExecRunner
	}
	switch manager {
	case "apt":
		// `dpkg -s` exits 0 for a package that was removed but kept its
		// configuration, so the status string is what says "installed".
		out, err := query("dpkg-query", "-W", "-f=${Status}", pkg)
		return err == nil && strings.Contains(string(out), "install ok installed")
	case "brew":
		out, err := query("brew", "list", "--versions", pkg)
		return err == nil && strings.TrimSpace(string(out)) != ""
	case "winget":
		// Builds differ on the exit code for "no match", and the sentence it
		// prints does not echo the id, so a hit is a listing that names it.
		out, err := query("winget", "list", "--id", pkg, "-e", "--accept-source-agreements")
		return err == nil && strings.Contains(strings.ToLower(string(out)), strings.ToLower(pkg))
	}
	return false
}

// systemInstallArgv is the command that installs pkg through manager. apt needs
// root: run directly as root, through `sudo -n` otherwise. -n makes sudo fail
// instead of asking for a password, so a run with no terminal (a scheduled
// converge, CI) never hangs on a prompt; dotf neither caches credentials nor
// prompts for one. The setup scripts avoid sudo and ask the user to run it
// once, which is what installSystem falls back to when sudo needs a password.
func (in *Installer) systemInstallArgv(manager, pkg string) []string {
	switch manager {
	case "apt":
		argv := []string{"apt-get", "install", "-y", pkg}
		if !in.IsRoot() {
			argv = append([]string{"sudo", "-n"}, argv...)
		}
		return argv
	case "brew":
		return []string{"brew", "install", pkg}
	default: // winget
		return []string{"winget", "install", "--id", pkg, "-e", "--accept-source-agreements", "--accept-package-agreements"}
	}
}

// missingSystemTool names the first executable the install needs and PATH lacks
// ("" when it can run): the manager, and sudo when apt must escalate.
func (in *Installer) missingSystemTool(t Tool) string {
	manager, pkg := t.Source.SystemPackage(in.GOOS)
	if pkg == "" {
		return ""
	}
	argv := in.systemInstallArgv(manager, pkg)
	for _, bin := range []string{argv[0], managerBinary(manager)} {
		if !in.HasCommand(bin) {
			return bin
		}
	}
	return ""
}

// planSystem is Plan for a system entry, after the shared platform check: skip
// what is present, install what is absent, never upgrade.
func (in *Installer) planSystem(p Plan, t Tool) Plan {
	switch {
	case p.Installed != "":
		p.Action = PlanSkip
	case in.missingSystemTool(t) != "":
		p.Action = PlanMissingManager
	default:
		p.Action = PlanInstall
	}
	return p
}

// installSystem converges a system package through the OS manager.
func (in *Installer) installSystem(t Tool) (Result, error) {
	manager, pkg := t.Source.SystemPackage(in.GOOS)
	if in.systemInstalled(t) != "" {
		_, _ = fmt.Fprintf(in.Out, "%s already installed; skipping\n", t.Name)
		return Skipped, nil
	}
	if missing := in.missingSystemTool(t); missing != "" {
		_, _ = fmt.Fprintf(in.Out, "%s: %s is not on PATH; skipping (the next run installs it once %s is there)\n", t.Name, missing, missing)
		return Skipped, nil
	}
	argv := in.systemInstallArgv(manager, pkg)
	if err := in.Run(argv[0], argv[1:]...); err != nil {
		if in.needsSudoPassword(argv) {
			// A named outcome, not a failure: nothing is wrong with the entry
			// or the machine, only the privilege dotf may not ask for. It says
			// what to run, and the other tools still converge.
			_, _ = fmt.Fprintf(in.Out, "%s: needs sudo; run: %s\n", t.Name, strings.Join(append([]string{"sudo"}, argv[2:]...), " "))
			return Skipped, nil
		}
		return Skipped, fmt.Errorf("%s: %s: %w", t.Name, strings.Join(argv, " "), err)
	}
	if in.systemInstalled(t) == "" {
		return Skipped, fmt.Errorf("%s: %s exited 0 but does not list %s as installed", t.Name, manager, pkg)
	}
	_, _ = fmt.Fprintf(in.Out, "%s installed via %s (%s)\n", t.Name, manager, pkg)
	return Installed, nil
}

// needsSudoPassword reports whether a failed install failed because sudo wanted
// a password. sudo -n exits 1 for that and for any other failure of the command,
// so the failure is classified by asking sudo again with a command that cannot
// fail for another reason: `sudo -n true` fails only when sudo itself refuses.
func (in *Installer) needsSudoPassword(argv []string) bool {
	if len(argv) < 2 || argv[0] != "sudo" || argv[1] != "-n" {
		return false
	}
	query := in.Query
	if query == nil {
		query = ExecRunner
	}
	_, err := query("sudo", "-n", "true")
	return err != nil
}
