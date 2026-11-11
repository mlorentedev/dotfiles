package converge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// systemUpgrade keeps the catalog's system packages current (TOOL-036, ADR-036
// amendment 2026-11-10): it refreshes the manager's index, upgrades the
// catalog's entries the manager reports outdated, and cleans up after an apply
// that upgraded something. It runs after tools, which installs what is absent;
// this step moves what is already there.
//
// The catalog's entries are the targets. A package the catalog does not name is
// never upgraded by name, whatever the manager reports outdated. The step never
// prompts: apt runs behind `sudo -n`, and where that fails the step skips and
// says why. A refresh that fails (offline, a mirror down) is a skip as well,
// because the step runs unattended every day.
type systemUpgrade struct {
	run  tools.Runner      // ExecManager in production; nil: the step is skipped
	has  func(string) bool // is a command on PATH
	root func() bool       // does dotf run as root; nil means os.Geteuid() == 0
	arch string            // the Windows GOARCH that picks a winget_arch id; "" means runtime.GOARCH
}

// sysTarget is one catalog system entry as this OS's manager names it.
type sysTarget struct {
	manager   string // apt, brew, brew-cask or winget
	pkg       string
	needsSudo bool // a cask whose install escalates inside brew
}

// key is how a manager's listing and a target are compared: lower case, and a
// tap-qualified formula by its short name, which is what brew prints.
func (t sysTarget) key() string {
	pkg := strings.ToLower(t.pkg)
	if t.manager == "brew" {
		pkg = path.Base(pkg)
	}
	return t.manager + "/" + pkg
}

const needsRootSkip = "needs root: run `dotf converge --only system-upgrade` from a terminal"

func (systemUpgrade) Name() string        { return "system-upgrade" }
func (systemUpgrade) Platforms() []string { return nil }

// managerBin is the executable each OS's refresh and upgrade run.
var managerBin = map[string]string{"darwin": "brew", "linux": "apt-get", "windows": "winget"}

func (r systemUpgrade) Reconcile(env Env, dryRun bool) (Result, error) {
	if r.run == nil {
		return Result{Skip: "no package-manager runner is wired into this registry"}, nil
	}
	targets, err := r.targets(env)
	if err != nil || len(targets) == 0 {
		return Result{Detail: "the catalog names no system package for " + env.GOOS}, err
	}
	if bin := managerBin[env.GOOS]; !r.has(bin) {
		return Result{Skip: bin + " is not on PATH; the next run upgrades once it is there"}, nil
	}
	switch env.GOOS {
	case "darwin":
		return r.darwin(dryRun, targets)
	case "linux":
		return r.linux(dryRun, targets)
	default:
		return r.windows(dryRun, targets)
	}
}

// Probe holds the post-condition: no catalog entry is still outdated, except a
// cask left for sudo, which Reconcile names. It lists again and does not
// refresh: a second refresh could move the index under the apply it checks.
func (r systemUpgrade) Probe(env Env) error {
	if r.run == nil {
		return nil
	}
	targets, err := r.targets(env)
	if err != nil || len(targets) == 0 || !r.has(managerBin[env.GOOS]) {
		return err
	}
	behind, err := r.outdated(env.GOOS, targets)
	if err != nil {
		return err
	}
	due, _ := r.splitPrivilege(behind)
	if len(due) > 0 {
		return fmt.Errorf("still outdated after the upgrade: %s", pkgNames(due))
	}
	return nil
}

// darwin: brew update, the outdated formulae and casks of the catalog, one
// brew upgrade per kind, then cleanup and autoremove.
func (r systemUpgrade) darwin(dryRun bool, targets []sysTarget) (Result, error) {
	if reason := r.refresh("brew", "brew", "update"); reason != "" {
		return Result{Skip: reason}, nil
	}
	behind, err := r.outdated("darwin", targets)
	if err != nil {
		return Result{}, err
	}
	due, waiting := r.splitPrivilege(behind)
	res := Result{Changes: len(due), Detail: upgradeDetail(dryRun, "brew update", due, waiting)}
	if dryRun || len(due) == 0 {
		return res, nil
	}
	done, err := r.brewUpgrade(due)
	res.Changes = done
	if err != nil {
		return res, err
	}
	res.Detail += r.housekeeping([]string{"brew", "cleanup"}, []string{"brew", "autoremove"})
	return res, nil
}

// linux: apt needs root, so the step asks `sudo -n true` before anything and
// skips when dotf is neither root nor allowed passwordless sudo. Then apt-get
// update, the outdated catalog packages, apt-get install --only-upgrade, and
// autoremove.
func (r systemUpgrade) linux(dryRun bool, targets []sysTarget) (Result, error) {
	if !r.isRoot() {
		if _, err := r.run("sudo", "-n", "true"); err != nil {
			return Result{Skip: needsRootSkip}, nil
		}
	}
	if reason := r.refresh("apt", r.asRoot("apt-get", "update")...); reason != "" {
		return Result{Skip: reason}, nil
	}
	due, err := r.outdated("linux", targets)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: len(due), Detail: upgradeDetail(dryRun, "apt-get update", due, nil)}
	if dryRun || len(due) == 0 {
		return res, nil
	}
	argv := append(r.asRoot("apt-get", "install", "--only-upgrade", "-y"), pkgList(due)...)
	if _, err := r.run(argv[0], argv[1:]...); err != nil {
		res.Changes = 0
		return res, fmt.Errorf("%s: %w", strings.Join(argv, " "), err)
	}
	res.Detail += r.housekeeping(r.asRoot("apt-get", "autoremove", "-y"))
	return res, nil
}

// windows: winget source update, the outdated catalog ids, and one winget
// upgrade per id. Winget has no housekeeping. An id that fails (a package that
// needs elevation) is reported and does not stop the others.
func (r systemUpgrade) windows(dryRun bool, targets []sysTarget) (Result, error) {
	if reason := r.refresh("winget", "winget", "source", "update"); reason != "" {
		return Result{Skip: reason}, nil
	}
	due, err := r.outdated("windows", targets)
	if err != nil {
		return Result{}, err
	}
	res := Result{Changes: len(due), Detail: upgradeDetail(dryRun, "winget source update", due, nil)}
	if dryRun {
		return res, nil
	}
	var failed []error
	res.Changes = 0
	for _, t := range due {
		argv := []string{"winget", "upgrade", "--id", t.pkg, "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements"}
		if _, err := r.run(argv[0], argv[1:]...); err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", t.pkg, err))
			continue
		}
		res.Changes++
	}
	return res, errors.Join(failed...)
}

// refresh runs the manager's index refresh and returns the skip reason when it
// fails: the manager's own error, so the report says what to look at.
func (r systemUpgrade) refresh(manager string, argv ...string) string {
	if _, err := r.run(argv[0], argv[1:]...); err != nil {
		return fmt.Sprintf("%s refresh failed: %v", manager, err)
	}
	return ""
}

// housekeeping runs the commands after an upgrade and returns what to append
// to the step's detail. A failed cleanup is a note, not a failure: the upgrade
// it follows is made.
func (r systemUpgrade) housekeeping(cmds ...[]string) string {
	var notes []string
	for _, argv := range cmds {
		if _, err := r.run(argv[0], argv[1:]...); err != nil {
			notes = append(notes, fmt.Sprintf("%s failed: %v", strings.Join(argv, " "), err))
		}
	}
	if len(notes) > 0 {
		return "; housekeeping: " + strings.Join(notes, "; ")
	}
	return "; cleaned up"
}

// brewUpgrade upgrades the formulae and the casks, each as one command, and
// counts what a succeeded command upgraded. A failure of one kind does not
// keep the other from running.
func (r systemUpgrade) brewUpgrade(due []sysTarget) (int, error) {
	var failed []error
	done := 0
	for _, g := range []struct {
		manager string
		args    []string
	}{{"brew", []string{"upgrade"}}, {"brew-cask", []string{"upgrade", "--cask"}}} {
		pkgs := pkgList(ofManager(due, g.manager))
		if len(pkgs) == 0 {
			continue
		}
		argv := append(g.args, pkgs...)
		if _, err := r.run("brew", argv...); err != nil {
			failed = append(failed, fmt.Errorf("brew %s: %w", strings.Join(argv, " "), err))
			continue
		}
		done += len(pkgs)
	}
	return done, errors.Join(failed...)
}

// splitPrivilege separates the casks that need a password from the rest.
// brew refuses root and asks inside the cask's installer, which no unattended
// run can answer, so a cask marked needs_sudo waits unless `sudo -n true`
// works. It asks once, and only when such a cask is outdated.
func (r systemUpgrade) splitPrivilege(behind []sysTarget) (due []sysTarget, waiting []string) {
	sudoOK, asked := false, false
	for _, t := range behind {
		if t.manager != "brew-cask" || !t.needsSudo {
			due = append(due, t)
			continue
		}
		if !asked {
			_, err := r.run("sudo", "-n", "true")
			sudoOK, asked = err == nil, true
		}
		if sudoOK {
			due = append(due, t)
		} else {
			waiting = append(waiting, t.pkg)
		}
	}
	return due, waiting
}

// targets is the catalog's system entries that name a package for this OS.
func (r systemUpgrade) targets(env Env) ([]sysTarget, error) {
	entries, err := loadCatalog(env)
	if err != nil {
		return nil, err
	}
	arch := r.arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	var out []sysTarget
	for _, t := range entries {
		if t.Source.Type != "system" || !t.SupportsOS(env.GOOS) {
			continue
		}
		if manager, pkg := t.Source.SystemPackage(env.GOOS, arch); pkg != "" {
			out = append(out, sysTarget{manager: manager, pkg: pkg, needsSudo: t.Source.NeedsSudo})
		}
	}
	return out, nil
}

// outdated is the targets the manager lists as behind, in catalog order. The
// listing is only ever matched against the targets, so a package the catalog
// does not name cannot reach an upgrade.
func (r systemUpgrade) outdated(goos string, targets []sysTarget) ([]sysTarget, error) {
	var behind func([]sysTarget) (func(sysTarget) bool, error)
	switch goos {
	case "darwin":
		behind = r.brewBehind
	case "linux":
		behind = r.aptBehind
	default:
		behind = r.wingetBehind
	}
	isBehind, err := behind(targets)
	if err != nil {
		return nil, err
	}
	var out []sysTarget
	for _, t := range targets {
		if isBehind(t) {
			out = append(out, t)
		}
	}
	return out, nil
}

// brewBehind reads `brew outdated --json=v2` for formulae and for casks, as
// {"formulae": [{"name", "pinned", ...}], "casks": [{"name", ...}]}. A pinned
// formula is listed but brew will not move it, so it is not behind. Casks are
// listed as brew's default sees them: one that updates itself, or has no
// version, is not (no --greedy).
func (r systemUpgrade) brewBehind([]sysTarget) (func(sysTarget) bool, error) {
	behind := map[string]bool{}
	for _, kind := range []struct{ flag, key, manager string }{
		{"--formula", "formulae", "brew"}, {"--cask", "casks", "brew-cask"},
	} {
		out, err := r.run("brew", "outdated", "--json=v2", kind.flag)
		if err != nil {
			return nil, fmt.Errorf("brew outdated %s: %w", kind.flag, err)
		}
		var doc map[string][]struct {
			Name   string `json:"name"`
			Pinned bool   `json:"pinned"`
		}
		if err := json.Unmarshal(out, &doc); err != nil {
			return nil, fmt.Errorf("brew outdated %s: reading its JSON: %w", kind.flag, err)
		}
		for _, e := range doc[kind.key] {
			if !e.Pinned {
				behind[sysTarget{manager: kind.manager, pkg: e.Name}.key()] = true
			}
		}
	}
	return func(t sysTarget) bool { return behind[t.key()] }, nil
}

// aptBehind reads `apt list --upgradable`, whose lines read
// `git/jammy-updates 1:2.34.1-1 amd64 [upgradable from: 1:2.34.0-1]`: the
// package is what precedes the first slash. The `Listing...` header and any
// warning have none.
func (r systemUpgrade) aptBehind([]sysTarget) (func(sysTarget) bool, error) {
	out, err := r.run("apt", "list", "--upgradable")
	if err != nil {
		return nil, fmt.Errorf("apt list --upgradable: %w", err)
	}
	behind := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if name, _, ok := strings.Cut(strings.TrimSpace(line), "/"); ok && name != "" && !strings.ContainsAny(name, " \t") {
			behind[sysTarget{manager: "apt", pkg: name}.key()] = true
		}
	}
	return func(t sysTarget) bool { return behind[t.key()] }, nil
}

// wingetBehind reads `winget upgrade`'s table and matches a catalog id as a
// whole column, without regard to case, as managerLists does. A table cut to
// the console width ends an id with an ellipsis: that column matches the ids
// it is a prefix of.
func (r systemUpgrade) wingetBehind([]sysTarget) (func(sysTarget) bool, error) {
	out, err := r.run("winget", "upgrade", "--accept-source-agreements")
	listing := strings.ToLower(string(out))
	if err != nil && !strings.Contains(listing, "no installed package") {
		return nil, fmt.Errorf("winget upgrade: %w", err)
	}
	return func(t sysTarget) bool {
		id := strings.ToLower(t.pkg)
		for _, col := range strings.Fields(listing) {
			if col == id || strings.HasSuffix(col, "…") && strings.HasPrefix(id, strings.TrimSuffix(col, "…")) {
				return true
			}
		}
		return false
	}, nil
}

func (r systemUpgrade) isRoot() bool {
	if r.root != nil {
		return r.root()
	}
	return os.Geteuid() == 0
}

// asRoot runs argv through `sudo -n` unless dotf is root: -n fails instead of
// asking, so a run with no terminal never waits on a password.
func (r systemUpgrade) asRoot(argv ...string) []string {
	if r.isRoot() {
		return argv
	}
	return append([]string{"sudo", "-n"}, argv...)
}

func ofManager(list []sysTarget, manager string) []sysTarget {
	var out []sysTarget
	for _, t := range list {
		if t.manager == manager {
			out = append(out, t)
		}
	}
	return out
}

func pkgList(list []sysTarget) []string {
	out := make([]string, 0, len(list))
	for _, t := range list {
		out = append(out, t.pkg)
	}
	return out
}

func pkgNames(list []sysTarget) string { return strings.Join(pkgList(list), ", ") }

// upgradeDetail is the step's report line: the refresh as info (it is never
// counted as a change), then what is, or would be, upgraded, then what waits
// on a password.
func upgradeDetail(dryRun bool, refresh string, due []sysTarget, waiting []string) string {
	parts := []string{"index refreshed (" + refresh + ")"}
	switch {
	case len(due) == 0:
		parts = append(parts, "catalog entries up to date")
	case dryRun:
		parts = append(parts, "to upgrade: "+pkgNames(due))
	default:
		parts = append(parts, "upgraded: "+pkgNames(due))
	}
	if len(waiting) > 0 {
		parts = append(parts, "left, needs sudo: "+strings.Join(waiting, ", ")+" (run `dotf converge --only system-upgrade` from a terminal)")
	}
	return strings.Join(parts, "; ")
}

// ExecManager is the production runner of the step: stdout alone on success, so
// a manager's warning on stderr cannot corrupt the JSON or the listing the step
// parses, and on failure an error that ends with the line the manager said it
// on (stderr, else stdout: winget reports on stdout).
func ExecManager(name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...) //nolint:gosec // name is the OS's package manager, chosen by the step
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err != nil {
		if line := lastLine(stderr.String(), stdout.String()); line != "" {
			err = fmt.Errorf("%w: %s", err, line)
		}
	}
	return stdout.Bytes(), err
}

// lastLine is the last non-empty line of the first stream that has one.
func lastLine(streams ...string) string {
	for _, s := range streams {
		lines := strings.Split(strings.TrimSpace(s), "\n")
		if line := strings.TrimSpace(lines[len(lines)-1]); line != "" {
			return line
		}
	}
	return ""
}
