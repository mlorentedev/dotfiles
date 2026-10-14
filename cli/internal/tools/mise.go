package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// MiseMarker marks the versions.conf pin on the next line as a CLI that
// `dotf tools sync` installs through mise (ADR-044). It sits on its own line,
// not after the value, because every other reader of versions.conf (the
// shell, doctor, setup-windows.ps1) skips comment lines but would read a
// trailing comment as part of the version.
const MiseMarker = "# mise: cli"

// MisePythonPackageMarker marks the pin on the next line as a package the
// Python that mise installs must carry (#2062): FOO_VERSION is the PyPI
// distribution `foo`. `dotf tools sync` lists it in the file mise's
// python.default_packages_file setting names, so mise installs it into every
// Python it installs, and installs it into the pinned Python itself when that
// one was installed before the package was declared.
const MisePythonPackageMarker = "# mise: python-package"

// MiseLatestMarker marks the pin on the next line as a CLI mise installs and
// keeps at its newest release (#2013 D9): the coding agents, which upstream
// ships daily. The value stays a version, read as a floor, because the other
// readers of versions.conf compare it as one. The sync renders the tool as
// `latest` and upgrades it whenever `mise outdated` reports a newer release.
const MiseLatestMarker = "# mise: latest"

// nearMissRE matches a comment that reads like the marker (`#mise: cli`,
// `# MISE: cli`, `# mise:cli`). Left alone it would be an ordinary comment and
// the pin under it would silently never install.
var nearMissRE = regexp.MustCompile(`(?i)^\s*#\s*mise\s*:`)

// MiseTool is one pinned CLI: the mise tool name and its pin.
type MiseTool struct {
	Name    string
	Version string
}

// MisePins is everything versions.conf marks for mise.
type MisePins struct {
	Tools          []MiseTool // MiseMarker and MiseLatestMarker: installed by mise
	PythonPackages []MiseTool // MisePythonPackageMarker: installed into mise's Python
	Latest         []string   // MiseLatestMarker: the Tools mise keeps at their newest release
}

// ParseMisePins returns the pins versions.conf marks, each list sorted by
// name. The name comes from the variable: FOO_BAR_VERSION is mise's `foo-bar`,
// whose short name resolves to the aqua backend, or the PyPI distribution
// `foo-bar`. A marker that is not directly followed by a non-empty *_VERSION
// pin is an error, and so is a comment that reads like a marker but is not
// exactly one, so a marker can never silently mark nothing. A python package
// with no python pin is an error too: nothing would install it.
func ParseMisePins(versionsConf []byte) (MisePins, error) {
	lines := strings.Split(strings.ReplaceAll(string(versionsConf), "\r\n", "\n"), "\n")
	var p MisePins
	for i, l := range lines {
		marker := strings.TrimSpace(l)
		var dst *[]MiseTool
		latest := false
		switch marker {
		case MiseMarker:
			dst = &p.Tools
		case MiseLatestMarker:
			dst, latest = &p.Tools, true
		case MisePythonPackageMarker:
			dst = &p.PythonPackages
		default:
			if nearMissRE.MatchString(l) {
				return MisePins{}, fmt.Errorf("versions.conf line %d: %q looks like a marker but is none of %q, %q or %q; fix it, or the pin under it is never installed", i+1, marker, MiseMarker, MiseLatestMarker, MisePythonPackageMarker)
			}
			continue
		}
		if i+1 >= len(lines) {
			return MisePins{}, fmt.Errorf("versions.conf line %d: %q marks no pin", i+1, marker)
		}
		key, value, ok := strings.Cut(strings.TrimSpace(lines[i+1]), "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !ok || !strings.HasSuffix(key, "_VERSION") || value == "" {
			return MisePins{}, fmt.Errorf("versions.conf line %d: %q must be followed by a NAME_VERSION=<pin> line", i+1, marker)
		}
		name := strings.ToLower(strings.ReplaceAll(strings.TrimSuffix(key, "_VERSION"), "_", "-"))
		*dst = append(*dst, MiseTool{Name: name, Version: value})
		if latest {
			p.Latest = append(p.Latest, name)
		}
	}
	if len(p.Tools) == 0 {
		// A versions.conf with no marker (one deployed before the markers
		// existed, say) would render an empty config over a converged one and
		// report success.
		return MisePins{}, fmt.Errorf("versions.conf has no %q pins: nothing to sync", MiseMarker)
	}
	if len(p.PythonPackages) > 0 && !hasTool(p.Tools, "python") {
		return MisePins{}, fmt.Errorf("versions.conf marks python packages (%s) but no PYTHON_VERSION under %q: nothing would install them", names(p.PythonPackages), MiseMarker)
	}
	for _, l := range [][]MiseTool{p.Tools, p.PythonPackages} {
		sort.Slice(l, func(a, b int) bool { return l[a].Name < l[b].Name })
	}
	sort.Strings(p.Latest)
	return p, nil
}

// PythonSeries is the major.minor of the Python versions.conf pins for mise
// ("3.13"), which uv-tool installs run on; "" when there is none to read. A
// series rather than the patch, so uv takes mise's interpreter and a patch
// bump does not make it fetch another.
func PythonSeries(versionsConf []byte) string {
	p, err := ParseMisePins(versionsConf)
	if err != nil {
		return ""
	}
	for _, t := range p.Tools {
		if t.Name == "python" {
			if parts := strings.SplitN(t.Version, ".", 3); len(parts) >= 2 {
				return parts[0] + "." + parts[1]
			}
		}
	}
	return ""
}

// ParseMiseTools returns the tools ParseMisePins reads, for a caller that
// needs only those.
func ParseMiseTools(versionsConf []byte) ([]MiseTool, error) {
	p, err := ParseMisePins(versionsConf)
	if err != nil {
		return nil, err
	}
	return p.Tools, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func hasTool(tools []MiseTool, name string) bool {
	for _, t := range tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

func names(tools []MiseTool) string {
	n := make([]string, len(tools))
	for i, t := range tools {
		n[i] = t.Name
	}
	return strings.Join(n, ", ")
}

// RenderMiseConfig is the mise config `dotf tools sync` owns: exact pins,
// sorted, under a header naming the generator. A tool named in latest renders
// as `latest` instead of its pin (MiseLatestMarker). A non-empty
// pythonPackages is the file mise reads the packages for every Python it
// installs from.
func RenderMiseConfig(tools []MiseTool, pythonPackages string, latest ...string) []byte {
	var b bytes.Buffer
	b.WriteString("# Generated by `dotf tools sync` from versions.conf (lines marked \"" + MiseMarker + "\").\n")
	b.WriteString("# Do not edit: change the pin in versions.conf and re-run the sync.\n\n[tools]\n")
	for _, t := range tools {
		v := t.Version
		if contains(latest, t.Name) {
			v = "latest"
		}
		name := t.Name
		if runtime.GOOS == "windows" && name == "direnv" {
			fmt.Fprintf(&b, "\"github:direnv/direnv\" = { version = %q, bin = \"direnv.exe\" }\n", v)
			continue
		}
		if runtime.GOOS == "windows" && name == "bats" {
			name = `"npm:bats"`
		}
		fmt.Fprintf(&b, "%s = %q\n", name, v)
	}
	// mise is pinned in packages.json; its update notice says `mise self-update`,
	// which would move it past that pin.
	b.WriteString("\n[settings]\ndisable_update_warning = true\n")
	if pythonPackages != "" {
		fmt.Fprintf(&b, "python.default_packages_file = %q\n", filepath.ToSlash(pythonPackages))
	}
	var alias []string
	for _, t := range tools {
		if backend, ok := attestedBackend[t.Name]; ok && runtime.GOOS != "windows" {
			alias = append(alias, fmt.Sprintf("%s = %q", t.Name, backend))
		}
	}
	if len(alias) > 0 {
		b.WriteString("\n[tool_alias]\n" + strings.Join(alias, "\n") + "\n")
	}
	var env []string
	for _, t := range tools {
		if kv, ok := ownUpdaterOff[t.Name]; ok {
			env = append(env, kv)
		}
	}
	if len(env) > 0 {
		b.WriteString("\n[env]\n" + strings.Join(env, "\n") + "\n")
	}
	return b.Bytes()
}

// attestedBackend routes a tool's short name to a backend that verifies more
// than aqua's registry entry for it does (#2013 H3). aqua:herdrdev/herdr
// checks the release digest only; mise's github backend checks the digest and
// the GitHub artifact attestations upstream publishes (measured with mise
// 2026.10.3: "GitHub artifact attestations verified"). An alias, not a
// renamed tool, so `mise which`, `where` and `bin-paths` keep the short name.
// Not on Windows: the herdr zip there carries no attestation, and mise
// refuses an asset whose attestation it cannot find.
var attestedBackend = map[string]string{
	"herdr": "github:herdrdev/herdr",
}

// ownUpdaterOff turns off the self-updater of a tool mise installs: mise keeps
// it current (MiseLatestMarker), and a second updater would replace the binary
// mise installed outside the pin's floor and outside converge's report (#2013
// D9). mise sets [env] under `mise activate` and in its shims, so the setting
// reaches every copy mise runs and no other. pi has no updater, but its startup
// check tells the user to run `pi update`, which installs a second pi outside
// mise; PI_SKIP_VERSION_CHECK is the switch pi 1.2.0 reads.
var ownUpdaterOff = map[string]string{
	"claude": `DISABLE_AUTOUPDATER = "1"`,
	"pi":     `PI_SKIP_VERSION_CHECK = "1"`,
}

// RenderPythonPackages is the requirements file mise installs into every
// Python it installs (`pip install -r`): one exact pin per line.
func RenderPythonPackages(pkgs []MiseTool) []byte {
	var b bytes.Buffer
	b.WriteString("# Generated by `dotf tools sync` from versions.conf (lines marked \"" + MisePythonPackageMarker + "\").\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "%s==%s\n", p.Name, p.Version)
	}
	return b.Bytes()
}

// MiseConfigDir is the directory mise reads its global config from:
// MISE_CONFIG_DIR, else $XDG_CONFIG_HOME/mise, else ~/.config/mise.
func MiseConfigDir(home string, getenv func(string) string) string {
	if d := getenv("MISE_CONFIG_DIR"); d != "" {
		return d
	}
	if x := getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "mise")
	}
	return filepath.Join(home, ".config", "mise")
}

// MiseShimsDir is where mise keeps its shims: MISE_DATA_DIR/shims, else
// $XDG_DATA_HOME/mise/shims, else the OS default data dir's shims:
// %LOCALAPPDATA%\mise\shims on Windows, ~/.local/share/mise/shims elsewhere. A
// shim runs the tool at the version mise's config selects, with or without
// `mise activate`.
func MiseShimsDir(home, goos string, getenv func(string) string) string {
	if d := getenv("MISE_DATA_DIR"); d != "" {
		return filepath.Join(d, "shims")
	}
	if x := getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "mise", "shims")
	}
	if goos == "windows" {
		if l := getenv("LOCALAPPDATA"); l != "" {
			return filepath.Join(l, "mise", "shims")
		}
		return filepath.Join(home, "AppData", "Local", "mise", "shims")
	}
	return filepath.Join(home, ".local", "share", "mise", "shims")
}

// MiseSync converges the pinned CLIs through mise. It owns one file,
// <ConfigDir>/conf.d/dotfiles.toml, and never touches the rest of the mise
// config: a hand-written config.toml keeps working beside it.
type MiseSync struct {
	ConfigDir string
	Run       Runner // `mise install`, and a `<tool> --version` silent on stdout; stdout and stderr merged
	// Stdout runs `mise which`, whose stdout alone is the path, and each
	// tool's `--version` first (probeVersion): a warning on stderr must not
	// become part of either. Nil means Run.
	Stdout Runner
	// PythonPackages must be importable from mise's Python at their pin
	// (MisePythonPackageMarker). Nil means the sync owns no packages file.
	PythonPackages []MiseTool
	// Latest names the tools mise keeps at their newest release
	// (MiseLatestMarker). Their pin is a floor, rendered as `latest`.
	Latest []string
	// CheckUpgrades asks `mise outdated` whether a Latest tool has a newer
	// release, which reaches the network. The converge and `dotf tools sync`
	// set it; doctor does not, so a release upstream never turns doctor red.
	CheckUpgrades bool
	// Prune removes each installed tool no pin asks for and no other tracked
	// mise config requires (#2375): a tool installed by hand and since
	// abandoned, whose shim would otherwise dangle. Only the converge sets it.
	Prune bool
}

// SyncPlan is what a sync would change.
type SyncPlan struct {
	ConfigChanged   bool     // the rendered config or packages file differs from the file
	Missing         []string // tools that are absent or below their pin
	MissingPackages []string // python packages mise's Python lacks at their pin
	Outdated        []string // Latest tools installed with a newer release available
	Orphans         []string // installs no pin asks for, as tool@version (Prune only)
}

// Pending reports whether Apply has anything to do.
func (p SyncPlan) Pending() bool {
	return p.ConfigChanged || len(p.Missing) > 0 || len(p.MissingPackages) > 0 || len(p.Outdated) > 0 || len(p.Orphans) > 0
}

// PythonPackagesPath is the requirements file the sync owns beside its config.
func (s MiseSync) PythonPackagesPath() string {
	return filepath.Join(s.ConfigDir, "conf.d", "dotfiles-python-packages.txt")
}

func (s MiseSync) render(tools []MiseTool) []byte {
	if len(s.PythonPackages) == 0 {
		return RenderMiseConfig(tools, "", s.Latest...)
	}
	return RenderMiseConfig(tools, s.PythonPackagesPath(), s.Latest...)
}

// ConfigPath is the file the sync owns.
func (s MiseSync) ConfigPath() string {
	return filepath.Join(s.ConfigDir, "conf.d", "dotfiles.toml")
}

// Plan reports what Apply would do and writes nothing.
func (s MiseSync) Plan(tools []MiseTool) (SyncPlan, error) {
	p, unlisted, err := s.plan(tools)
	if err == nil {
		err = unlisted
	}
	return p, err
}

// plan is Plan with a failed orphan listing returned on its own, so that Apply
// still installs and upgrades when only the prune cannot run.
func (s MiseSync) plan(tools []MiseTool) (p SyncPlan, unlisted, err error) {
	have, err := os.ReadFile(s.ConfigPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return SyncPlan{}, nil, err
	}
	p = SyncPlan{ConfigChanged: !bytes.Equal(have, s.render(tools))}
	if len(s.PythonPackages) > 0 {
		pkgs, err := os.ReadFile(s.PythonPackagesPath())
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return SyncPlan{}, nil, err
		}
		p.ConfigChanged = p.ConfigChanged || !bytes.Equal(pkgs, RenderPythonPackages(s.PythonPackages))
	}
	for _, t := range tools {
		if !s.runsAtPin(t) {
			p.Missing = append(p.Missing, t.Name)
		}
	}
	p.MissingPackages = s.missingPythonPackages()
	if p.Outdated, err = s.outdated(p.Missing); err != nil {
		return p, nil, err
	}
	if s.Prune {
		p.Orphans, unlisted = s.Orphans(tools)
	}
	return p, unlisted, nil
}

// Orphans names, as tool@version, each install `mise ls --prunable` reports
// whose tool is not one of tools. It only lists: doctor reports them, and
// only a sync with Prune set removes them. Staying downstream of mise's own list keeps
// two things out by construction: a tool another project's tracked config
// requires, and a superseded version of a pinned tool, which `mise upgrade`
// has already scheduled for removal.
func (s MiseSync) Orphans(tools []MiseTool) ([]string, error) {
	run := s.Stdout
	if run == nil {
		run = s.Run
	}
	out, err := run("mise", "ls", "--prunable", "--json")
	if err != nil {
		return nil, fmt.Errorf("mise ls --prunable: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	var report map[string][]struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out, &report); err != nil {
		return nil, fmt.Errorf("mise ls --prunable: unreadable report %q: %w", strings.TrimSpace(string(out)), err)
	}
	var found []string
	for name, versions := range report {
		if hasTool(tools, name) {
			continue
		}
		for _, v := range versions {
			found = append(found, name+"@"+v.Version)
		}
	}
	sort.Strings(found)
	return found, nil
}

// orphanTools is the distinct tool names of orphans. The version follows the
// last "@", since an npm tool's own name may hold one (npm:@scope/pkg).
func orphanTools(orphans []string) []string {
	var names []string
	for _, o := range orphans {
		if name := o[:strings.LastIndex(o, "@")]; !contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

// outdatedEntry is one tool in `mise outdated --json`.
type outdatedEntry struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

// outdated names the Latest tools that are installed and have a newer release,
// as `mise outdated` reports it. A missing tool is left to `mise install`,
// which installs the newest release of a tool requested as `latest`.
func (s MiseSync) outdated(missing []string) ([]string, error) {
	if !s.CheckUpgrades {
		return nil, nil
	}
	var ask []string
	for _, name := range s.Latest {
		if !contains(missing, name) {
			ask = append(ask, name)
		}
	}
	if len(ask) == 0 {
		return nil, nil
	}
	run := s.Stdout
	if run == nil {
		run = s.Run
	}
	out, err := run("mise", append([]string{"outdated", "--json"}, ask...)...)
	if err != nil {
		return nil, fmt.Errorf("mise outdated: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	var report map[string]outdatedEntry
	if err := json.Unmarshal(out, &report); err != nil {
		return nil, fmt.Errorf("mise outdated: unreadable report %q: %w", strings.TrimSpace(string(out)), err)
	}
	var names []string
	for _, name := range ask {
		if e, ok := report[name]; ok && e.Latest != "" && e.Latest != e.Current {
			names = append(names, name)
		}
	}
	return names, nil
}

// Apply writes the config when it differs, runs `mise install` when anything is
// pending, and then requires every tool to run at or above its pin, through
// `mise which`, not PATH (the shims reach PATH in a later row). It returns the
// plan it acted on; a tool that still does not run is an error naming it. When
// mise cannot list the orphans, the rest still runs, nothing is pruned, and the
// listing's failure is the error.
func (s MiseSync) Apply(tools []MiseTool) (p SyncPlan, err error) {
	if runtime.GOOS == "windows" && hasTool(tools, "bats") && !hasTool(tools, "node") {
		return SyncPlan{}, errors.New("windows bats requires node: mark NODE_VERSION for mise before using the npm installer")
	}
	p, unlisted, err := s.plan(tools)
	if err != nil {
		return p, err
	}
	// A listing that fails before the prune means nothing was pruned; one
	// that fails only after it means the prune ran and could not be checked.
	var unverified error
	defer func() {
		switch {
		case err != nil:
		case unlisted != nil:
			err = fmt.Errorf("nothing pruned: %w", unlisted)
		case unverified != nil && len(p.Orphans) > 0:
			err = fmt.Errorf("pruned %s; cannot verify: %w", strings.Join(p.Orphans, ", "), unverified)
		case unverified != nil:
			err = fmt.Errorf("cannot verify that no orphan remains: %w", unverified)
		}
	}()
	if p.ConfigChanged {
		// The packages file first: the config names it, and mise reads it
		// when it installs a Python.
		if len(s.PythonPackages) > 0 {
			if err := writeFileAtomic(s.PythonPackagesPath(), RenderPythonPackages(s.PythonPackages)); err != nil {
				return p, err
			}
		}
		if err := writeFileAtomic(s.ConfigPath(), s.render(tools)); err != nil {
			return p, err
		}
	}
	if !p.Pending() {
		return p, nil
	}
	if p.ConfigChanged || len(p.Missing) > 0 {
		if runtime.GOOS == "windows" && contains(p.Missing, "bats") {
			if out, err := s.Run("mise", "install", "node"); err != nil {
				return p, fmt.Errorf("mise install node (required by bats): %w\n%s", err, strings.TrimSpace(string(out)))
			}
			if out, err := s.Run("mise", "install", "--force", "npm:bats"); err != nil {
				return p, fmt.Errorf("mise install npm:bats: %w\n%s", err, strings.TrimSpace(string(out)))
			}
		}
		if out, err := s.Run("mise", "install"); err != nil {
			return p, fmt.Errorf("mise install: %w\n%s", err, strings.TrimSpace(string(out)))
		}
		// The config just written may have turned an exact pin into
		// `latest`, which mise install leaves at the installed version; ask
		// again so this run upgrades it instead of the next one.
		if p.Outdated, err = s.outdated(nil); err != nil {
			return p, err
		}
	}
	if len(p.Outdated) > 0 {
		// upgrade also schedules the version it replaced for removal (mise
		// 2026.10.3 lists it as "pruned in 1d"), so an agent does not pile up
		// a copy per release.
		if out, err := s.Run("mise", append([]string{"upgrade"}, p.Outdated...)...); err != nil {
			return p, fmt.Errorf("mise upgrade: %w\n%s", err, strings.TrimSpace(string(out)))
		}
	}
	if len(p.Orphans) > 0 {
		// After the install and the upgrade, so nothing goes before what
		// replaces it is in place.
		if out, err := s.Run("mise", append([]string{"prune", "--yes", "--tools"}, orphanTools(p.Orphans)...)...); err != nil {
			return p, fmt.Errorf("mise prune: %w\n%s", err, strings.TrimSpace(string(out)))
		}
	}
	after, unverified, err := s.plan(tools)
	if err != nil {
		return p, err
	}
	if len(after.Orphans) > 0 {
		return p, fmt.Errorf("after mise prune, still installed: %s", strings.Join(after.Orphans, ", "))
	}
	if len(after.Missing) > 0 {
		return p, fmt.Errorf("after mise install, not running at their pin: %s", strings.Join(after.Missing, ", "))
	}
	if len(after.Outdated) > 0 {
		return p, fmt.Errorf("after mise upgrade, still behind their newest release: %s", strings.Join(after.Outdated, ", "))
	}
	if len(after.MissingPackages) == 0 {
		return p, nil
	}
	// mise installs the packages only into a Python it installs; a Python
	// already at its pin when a package was declared gets it here.
	if err := s.pipInstall(after.MissingPackages); err != nil {
		return p, err
	}
	if after, _, err = s.plan(tools); err != nil {
		return p, err
	}
	if len(after.MissingPackages) > 0 {
		return p, fmt.Errorf("after pip install, python packages not importable at their pin: %s", strings.Join(after.MissingPackages, ", "))
	}
	return p, nil
}

// PyDistVersion prints the installed version of the distribution named by
// its first argument, and fails when it is not installed.
const PyDistVersion = "import importlib.metadata as m, sys; print(m.version(sys.argv[1]))"

// missingPythonPackages names the declared packages mise's Python does not
// carry at or above their pin. With no Python resolved, every one is missing.
func (s MiseSync) missingPythonPackages() []string {
	if len(s.PythonPackages) == 0 {
		return nil
	}
	python := s.which("python")
	var missing []string
	for _, pkg := range s.PythonPackages {
		if python == "" {
			missing = append(missing, pkg.Name)
			continue
		}
		out, err := s.Run(python, "-c", PyDistVersion, pkg.Name)
		v := semverRE.Find(out)
		if err != nil || v == nil || !atLeast(string(v), pkg.Version) {
			missing = append(missing, pkg.Name)
		}
	}
	return missing
}

// pipInstall installs the named declared packages at their pin into mise's
// Python.
func (s MiseSync) pipInstall(missing []string) error {
	python := s.which("python")
	if python == "" {
		return errors.New("mise resolves no python to install the python packages into")
	}
	args := []string{"-m", "pip", "install", "--quiet", "--disable-pip-version-check"}
	for _, pkg := range s.PythonPackages {
		for _, name := range missing {
			if pkg.Name == name {
				args = append(args, pkg.Name+"=="+pkg.Version)
			}
		}
	}
	if out, err := s.Run(python, args...); err != nil {
		return fmt.Errorf("pip install into mise's python: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// which is the path mise resolves for an executable, or "".
func (s MiseSync) which(name string) string {
	run := s.Stdout
	if run == nil {
		run = s.Run
	}
	out, err := run("mise", "which", name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// versionArgs names the pinned CLIs that do not answer `--version` with
// their version, and what they answer instead. Without a row, such a tool
// installs and never counts as running at its pin: every sync fails naming it
// and doctor stays red. Measured on darwin-arm64, all seven reject the flag
// as unknown. A tool must not be marked in versions.conf before a released
// dotf carries its row; tests/versions-conf.bats holds that order.
var versionArgs = map[string][]string{
	"argocd":      {"version", "--client"},
	"helm":        {"version"},
	"hcloud":      {"version"},
	"k9s":         {"version", "--short"},
	"kubeconform": {"-v"},
	"kubectl":     {"version", "--client"},
	"kustomize":   {"version"},
}

// runsAtPin resolves the tool through mise and runs it: an install is real
// only when the binary executes and reports at least the pin (lesson 337).
func (s MiseSync) runsAtPin(t MiseTool) bool {
	name := t.Name
	if runtime.GOOS == "windows" && name == "bats" {
		name = "bats.cmd"
	}
	path := s.which(name)
	if path == "" {
		return false
	}
	args, ok := versionArgs[t.Name]
	if !ok {
		args = []string{"--version"}
	}
	v := s.probeVersion(path, args...)
	return v != "" && atLeast(v, t.Version)
}

// probeVersion is ProbeRunner's rule over the sync's two runners: stdout
// first, through Stdout, so a warning on stderr is never read before the
// tool's own banner; a failed run has no other version, whatever its stderr
// names (BUG-142, the BUG-141 class); the merged output counts only for a run
// that exited 0 and printed no version on stdout, the one case that runs the
// tool twice.
func (s MiseSync) probeVersion(path string, args ...string) string {
	run := s.Stdout
	if run == nil {
		run = s.Run
	}
	out, err := run(path, args...)
	if m := semverRE.Find(out); m != nil {
		return string(m)
	}
	if err != nil {
		return ""
	}
	return ProbeVersionArgs(path, s.Run, args...)
}

// writeFileAtomic writes data to path through a temp file in the same
// directory, so a reader never sees a half-written config.
func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".dotfiles-*.toml")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil { //nolint:gosec // a tool list, world-readable by design
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// HomeRunners returns the runners a machine-level sync uses, both run from
// home so a project mise.toml in the caller's directory is not installed as a
// side effect: run merges stdout and stderr (`mise install`), stdout returns
// stdout alone (`mise which`, whose output is a path a stderr warning must not
// corrupt, and a tool's `--version`, read stdout first).
func HomeRunners(home string) (run, stdout Runner) {
	bashBin := ""
	if runtime.GOOS == "windows" {
		out, err := exec.Command("git", "--exec-path").Output()
		if err == nil {
			gitRoot := filepath.Dir(filepath.Dir(filepath.Dir(strings.TrimSpace(string(out)))))
			candidate := filepath.Join(gitRoot, "bin", "bash.exe")
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				bashBin = filepath.Dir(candidate)
			}
		}
	}
	command := func(name string, args ...string) *exec.Cmd {
		c := exec.Command(name, args...) //nolint:gosec // mise, or a tool path mise resolved
		c.Dir = home
		installerEnv := miseInstallEnv(name, args...)
		if bashBin != "" || installerEnv != "" {
			c.Env = os.Environ()
		}
		if bashBin != "" {
			c.Env = append(c.Env, "PATH="+bashBin+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
		if installerEnv != "" {
			c.Env = append(c.Env, installerEnv)
		}
		return c
	}
	run = func(name string, args ...string) ([]byte, error) { return command(name, args...).CombinedOutput() }
	stdout = func(name string, args ...string) ([]byte, error) { return command(name, args...).Output() }
	return run, stdout
}

func miseInstallEnv(name string, args ...string) string {
	if runtime.GOOS == "windows" && name == "mise" && len(args) == 3 &&
		args[0] == "install" && args[1] == "--force" && args[2] == "npm:bats" {
		return "MISE_NPM_PACKAGE_MANAGER=npm"
	}
	return ""
}
