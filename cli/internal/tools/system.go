package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
// needs no privilege to find; a cask has the same shortcut in its app bundle
// (caskAppPresent). The record is the only answer for a library, and the
// post-condition of every install (a manager that exits 0
// without the package listed is an error, as an npm or uv install is).

// systemPresent is the sentinel Plan.Installed carries for a system package that
// is there: it has no version to report.
const systemPresent = "present"

// managerBinary is the executable that runs a manager's installs, which must be
// on PATH for an install to be possible.
func managerBinary(manager string) string {
	switch manager {
	case "apt":
		return "apt-get"
	case "brew-cask":
		return "brew"
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
	case "brew-cask":
		out, err := query("brew", "list", "--cask", "--versions", pkg)
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return true
		}
		return in.caskAppPresent(query, pkg)
	case "winget":
		// Builds differ on the exit code for "no match", and the sentence it
		// prints does not echo the id, so a hit is a listing that names it.
		out, err := query("winget", "list", "--id", pkg, "-e", "--accept-source-agreements")
		return err == nil && strings.Contains(strings.ToLower(string(out)), strings.ToLower(pkg))
	}
	return false
}

// caskAppPresent reports whether the app bundle a cask installs is already on
// disk from another channel: dragged into /Applications, or put there by the
// vendor's own installer. `brew list --cask` does not record such an app, and
// `brew install --cask` refuses to overwrite its bundle, so without this an
// entry for a hand-installed app failed on every run. The bundle name comes
// from the cask itself (`brew info --json=v2`), never from the entry's name.
func (in *Installer) caskAppPresent(query Runner, pkg string) bool {
	out, err := query("brew", "info", "--cask", "--json=v2", pkg)
	if err != nil {
		return false
	}
	var info struct {
		Casks []struct {
			Artifacts []map[string]json.RawMessage `json:"artifacts"`
		} `json:"casks"`
	}
	if json.Unmarshal(out, &info) != nil {
		return false
	}
	for _, c := range info.Casks {
		for _, artifact := range c.Artifacts {
			for _, bundle := range appBundles(artifact["app"]) {
				if in.AppExists(bundle) {
					return true
				}
			}
		}
	}
	return false
}

// appBundles reads a cask's `app` artifact: each element is the bundle name, or
// an object whose `target` renames it on install.
func appBundles(raw json.RawMessage) []string {
	var elems []json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &elems) != nil {
		return nil
	}
	var names []string
	for _, e := range elems {
		var name string
		if json.Unmarshal(e, &name) == nil {
			names = append(names, name)
			continue
		}
		var renamed struct {
			Target string `json:"target"`
		}
		if json.Unmarshal(e, &renamed) == nil && renamed.Target != "" {
			names = append(names, renamed.Target)
		}
	}
	return names
}

// appInApplications is the default AppExists: the two directories Homebrew and
// drag-installs put an app in.
func appInApplications(bundle string) bool {
	dirs := []string{"/Applications"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Applications"))
	}
	for _, d := range dirs {
		if _, err := os.Stat(filepath.Join(d, bundle)); err == nil {
			return true
		}
	}
	return false
}

// systemInstallArgv is the command that installs pkg through manager. apt needs
// root: run directly as root, through `sudo -n` otherwise. -n makes sudo fail
// instead of asking for a password, so a run with no terminal (a scheduled
// converge, CI) never hangs on a prompt; dotf neither caches credentials nor
// prompts for one. The setup scripts avoid sudo and ask the user to run it
// once, which is what installSystem falls back to when sudo needs a password.
//
// --no-remove makes apt abort instead of removing a package to satisfy the new
// one: with -y it would otherwise remove what conflicts without asking, and
// Ubuntu's docker-compose-v2 depends on docker.io, which displaces Docker's
// docker-ce on a box that runs it (#2013 P5b).
func (in *Installer) systemInstallArgv(manager, pkg string) []string {
	switch manager {
	case "apt":
		return in.asRoot(aptInstallArgv(pkg))
	case "brew":
		return []string{"brew", "install", pkg}
	case "brew-cask":
		return []string{"brew", "install", "--cask", pkg}
	default: // winget
		return []string{"winget", "install", "--id", pkg, "-e", "--accept-source-agreements", "--accept-package-agreements"}
	}
}

// asRoot prefixes argv with `sudo -n` unless dotf already runs as root.
func (in *Installer) asRoot(argv []string) []string {
	if in.IsRoot() {
		return argv
	}
	return append([]string{"sudo", "-n"}, argv...)
}

// refreshAptIndex runs `apt-get update` once per Installer, before its first
// apt install. A fresh machine's index is stale: a hosted runner's lists named
// a libgit2 build the mirror had dropped, so the first install 404ed (#2013
// X1). A run with nothing to install never gets here, so a converged machine
// still runs no manager command.
//
// When sudo wants a password it is not attempted: the install that follows
// is refused the same way, and the command that path names refreshes the
// index before it installs (SudoInstallCommand). A refresh
// that fails for another reason (a third-party source that 404s) is a
// warning, not the run's failure, and the install tries the index as it is.
func (in *Installer) refreshAptIndex() {
	if in.aptIndexAsked {
		return
	}
	in.aptIndexAsked = true
	argv := in.asRoot([]string{"apt-get", "update"})
	if in.needsSudoPassword(argv) {
		return
	}
	if err := in.Run(argv[0], argv[1:]...); err != nil {
		_, _ = fmt.Fprintf(in.Out, "apt index refresh failed (%s: %v); installing from the index as it is\n", strings.Join(argv, " "), err)
	}
}

// aptInstallArgv installs pkgs through apt-get, without sudo.
func aptInstallArgv(pkgs ...string) []string {
	return append([]string{"apt-get", "install", "-y", "--no-remove"}, pkgs...)
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
		p.Action, p.Note = PlanMissingManager, "waits on "+in.missingSystemTool(t)
	default:
		p.Action = PlanInstall
		manager, pkg := t.Source.SystemPackage(in.GOOS)
		if argv := in.systemInstallArgv(manager, pkg); in.needsSudoPassword(argv) {
			// The classifier the apply uses after sudo -n refuses, asked up
			// front, so a plan never promises an install the apply skips.
			p.Action, p.Note, p.Package = PlanNeedsSudo, "run: "+SudoInstallCommand([]string{pkg}), pkg
		}
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
	if manager == "apt" {
		in.refreshAptIndex()
	}
	argv := in.systemInstallArgv(manager, pkg)
	if err := in.Run(argv[0], argv[1:]...); err != nil {
		if in.needsSudoPassword(argv) {
			// A named outcome, not a failure: nothing is wrong with the entry
			// or the machine, only the privilege dotf may not ask for. It says
			// what to run, and the other tools still converge.
			_, _ = fmt.Fprintf(in.Out, "%s: needs sudo; run: %s\n", t.Name, SudoInstallCommand([]string{pkg}))
			in.sudoDeferred = append(in.sudoDeferred, pkg)
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

// SudoInstallCommand is the one command that installs every apt package in
// pkgs, for a person to run once instead of one command per package (#2308).
// apt is the only manager that escalates: brew refuses to run as root, and
// winget elevates per installer through UAC, which no batch can collect. ""
// for no packages.
//
// --no-remove holds for the batch as it does for one package, so a single
// conflicting package (docker.io on a box that runs docker-ce) aborts the
// whole command; apt names it, and the person installs the rest without it.
//
// It refreshes the index first: when sudo wants a password, dotf skips its own
// refresh too (refreshAptIndex), so an install alone would meet the same stale
// lists that 404ed on a fresh machine (#2013 X1).
func SudoInstallCommand(pkgs []string) string {
	if len(pkgs) == 0 {
		return ""
	}
	return "sudo apt-get update && sudo " + strings.Join(aptInstallArgv(pkgs...), " ")
}

// SudoDeferred is the command for every apt package Install skipped because
// sudo wanted a password, in the order Install met them; "" when none was.
func (in *Installer) SudoDeferred() string {
	return SudoInstallCommand(in.sudoDeferred)
}

// needsSudoPassword reports whether a failed install failed because sudo wanted
// a password. sudo -n exits 1 for that and for any other failure of the command,
// so the failure is classified by asking sudo again with a command that cannot
// fail for another reason: `sudo -n true` fails only when sudo itself refuses.
func (in *Installer) needsSudoPassword(argv []string) bool {
	if len(argv) < 2 || argv[0] != "sudo" || argv[1] != "-n" {
		return false
	}
	if in.sudoRefuses == nil {
		query := in.Query
		if query == nil {
			query = ExecRunner
		}
		_, err := query("sudo", "-n", "true")
		refuses := err != nil
		in.sudoRefuses = &refuses
	}
	return *in.sudoRefuses
}
