// Package tools reads the declarative cross-OS package catalog (packages.json):
// the tool/install list as data, consumed by dotf rather than expressed as
// per-OS imperative blocks in setup-linux.sh / setup-windows.ps1 (CLI-029,
// piloting the ADR-021 / CLI-028 convergence idea). The catalog is dotf-only —
// no shell or jq parsing — so JSON (matching env-contract.json) is the format.
package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/platform"
)

// Catalog is the parsed packages.json.
type Catalog struct {
	Tools []Tool `json:"tools"`
}

// Tool is one catalog entry: a pinned, cross-OS-installable package.
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Profile string `json:"profile"`
	Source  Source `json:"source"`
}

// Source declares how to fetch a tool. Four kinds:
//   - "github-release": a pinned per-OS/arch release binary, verified against the
//     release checksums by the installer (CLI-029 PR-B), mirroring the
//     deterministic age/install.sh pattern rather than relying on winget/apt.
//   - "npm": a globally-installed npm package (Package), pinned by Version. Used
//     for tools whose first-class distribution is npm and that do not ship a
//     raw, checksum-manifested github-release binary — e.g. the Bitwarden CLI
//     (@bitwarden/cli), whose releases are zip archives under a cli-v{date} tag
//     with no sha256 manifest (#577, ADR-028 Phase 0).
//   - "uv-tool": a Python package installed with `uv tool install`, pinned by
//     Version. Package is the PyPI name; the tool's Name is the binary it puts
//     on PATH (hive-vault installs hive, #1993).
//   - "system": a package the OS package manager owns (git, gh, tmux, a library,
//     a cask), for what has no cross-OS channel. Apt, Brew and Winget name it per
//     manager; linux uses apt, darwin brew, windows winget. A manager with no
//     name skips the entry on that OS, as a platforms miss does. It is not
//     pinned, so Tool.Version must be absent and presence is the convergence
//     rule: the entry is satisfied when its Command is on PATH, or else when the
//     manager lists the package (dpkg-query, brew list --versions, winget list),
//     and it is never upgraded. A catalog that carries one is only read by a
//     dotf that knows the type; an older one skips it with a warning.
type Source struct {
	Type string `json:"type"`
	Repo string `json:"repo"`
	// Package is the npm package name for source.type "npm" (e.g.
	// "@bitwarden/cli") and the PyPI name for "uv-tool" (e.g. "hive-vault").
	// Unused by github-release sources.
	Package string `json:"package,omitempty"`
	// Apt, Brew and Winget are the package names for source.type "system", one
	// per OS manager: an apt package, a Homebrew formula, a winget id.
	Apt    string `json:"apt,omitempty"`
	Brew   string `json:"brew,omitempty"`
	Winget string `json:"winget,omitempty"`
	// Cask is a Homebrew cask token, the darwin package of a GUI app or font.
	// It is its own key, not a Brew value, because brew lists and installs
	// casks in a separate namespace (`--cask`): queried as a formula, an
	// installed cask reads as absent. An entry names a formula or a cask.
	Cask string `json:"cask,omitempty"`
	// Command is the executable a "system" entry puts on PATH. When it declares
	// one, finding it there counts as installed whichever channel put it there,
	// so a copy from another channel is never installed over (and installing
	// needs no privilege). Absent, the manager's own record decides, which is the
	// only answer for a library or a GUI app.
	Command string `json:"command,omitempty"`
	// Platforms lists the GOOS values the tool installs on; absent means all of
	// them. A github-release tool gets this from its Asset map. Other sources
	// need it to name a tool that belongs to one OS family: hive is a uv tool on
	// POSIX, while on Windows hive owns its install layout (hive ADR-019).
	Platforms []string `json:"platforms,omitempty"`
	// Asset maps GOOS -> a release-asset filename template. A per-OS map (not a
	// single template) is required because release naming is irregular across OSes
	// — e.g. sops is "sops-v{version}.linux.{goarch}" but "sops-v{version}.{goarch}.exe"
	// on Windows (no OS token, arch before .exe). Templates expand {version} and
	// {goarch}. A key may also be "GOOS/GOARCH" (e.g. "darwin/arm64"), which wins
	// over the GOOS key, for releases whose tokens {goarch} cannot spell (mise:
	// "linux-x64", "macos-arm64"). A platform with no key is not supported: the
	// installer skips it, and Load rejects a key that names no known platform.
	Asset map[string]string `json:"asset"`
	// Checksums is the release's sha256 manifest filename — a SINGLE file covering
	// every OS asset (so no per-OS map). The name is per-repo: dotf ships
	// "checksums.txt" but sops ships "sops-v{version}.checksums.txt". Templates
	// expand {version}. The installer (CLI-029 PR-B) downloads it and verifies the
	// fetched asset against its entry.
	Checksums string `json:"checksums"`
}

// Load reads and parses the catalog at path.
func Load(path string) (Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, fmt.Errorf("read package catalog %q: %w", path, err)
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return Catalog{}, fmt.Errorf("parse package catalog %q: %w", path, err)
	}
	// A name listed twice installs twice and reports twice, and every reader
	// that looks a tool up by name silently takes the first (a duplicated
	// copilot entry shipped in a PR and read as "already installed; skipping"
	// on the second line, AI-038/#1321). Refuse it at the one place every
	// consumer goes through.
	rawKeys, err := sourceKeys(b)
	if err != nil {
		return Catalog{}, fmt.Errorf("parse package catalog %q: %w", path, err)
	}
	seen := make(map[string]struct{}, len(c.Tools))
	for i, t := range c.Tools {
		if _, dup := seen[t.Name]; dup {
			return Catalog{}, fmt.Errorf("parse package catalog %q: tool %q is listed more than once", path, t.Name)
		}
		seen[t.Name] = struct{}{}
		// A misspelt platform makes the tool unsupported everywhere, and the
		// installer reports an unsupported tool as a skip, so nothing else
		// would ever say so.
		// A missing asset is a skip, so a release tool with no asset map (or
		// one misspelt "assets", which the decoder drops) would install
		// nothing anywhere and say only "no release asset".
		if t.Source.Type == "github-release" && len(t.Source.Asset) == 0 {
			return Catalog{}, fmt.Errorf("parse package catalog %q: github-release tool %q declares no asset", path, t.Name)
		}
		// Likewise a misspelt key.
		for key := range t.Source.Asset {
			if !platform.ValidKey(key) {
				return Catalog{}, fmt.Errorf("parse package catalog %q: tool %q has asset key %q (want a GOOS, or GOOS/GOARCH with amd64 or arm64)", path, t.Name, key)
			}
		}
		if p := platform.Unknown(t.Source.Platforms); p != "" {
			return Catalog{}, fmt.Errorf("parse package catalog %q: tool %q lists unknown platform %q (want linux, darwin or windows)", path, t.Name, p)
		}
		if t.Source.Type == "system" {
			if err := validateSystem(t, rawKeys[i]); err != nil {
				return Catalog{}, fmt.Errorf("parse package catalog %q: %w", path, err)
			}
		}
	}
	return c, nil
}

// KnownSourceTypes are the source.type values this dotf reads. Install skips any
// other with a warning, so a guard (not the reader) is what catches a misspelt
// type in the catalog that ships.
var KnownSourceTypes = []string{"github-release", "npm", "uv-tool", "system"}

// systemKeys are the source keys a "system" entry may carry. The typed decoder
// drops any other key without a word, so a misspelt or unsupported manager
// (pacman, dnf) would read as "no name for this OS" and skip in silence.
var systemKeys = map[string]bool{"type": true, "apt": true, "brew": true, "cask": true, "winget": true, "command": true, "platforms": true}

// sourceKeys lists, per tool, the keys its source object carries, for the checks
// the typed decode cannot make.
func sourceKeys(b []byte) ([]map[string]json.RawMessage, error) {
	var raw struct {
		Tools []struct {
			Source map[string]json.RawMessage `json:"source"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	keys := make([]map[string]json.RawMessage, len(raw.Tools))
	for i, t := range raw.Tools {
		keys[i] = t.Source
	}
	return keys, nil
}

// validateSystem refuses a "system" entry that could not converge or that asks
// for something it does not do, each naming the entry.
func validateSystem(t Tool, keys map[string]json.RawMessage) error {
	for key := range keys {
		if !systemKeys[key] {
			return fmt.Errorf("tool %q: system source has unknown key %q (want apt, brew, cask, winget, command or platforms)", t.Name, key)
		}
	}
	if t.Source.Apt == "" && t.Source.Brew == "" && t.Source.Cask == "" && t.Source.Winget == "" {
		return fmt.Errorf("tool %q: system source names no package manager (want apt, brew, cask or winget)", t.Name)
	}
	if t.Source.Brew != "" && t.Source.Cask != "" {
		return fmt.Errorf("tool %q: system source names both a brew formula and a cask; darwin installs one package per entry", t.Name)
	}
	// platforms narrows the OSes an entry names a package for; when it leaves
	// none, the entry loads and then skips everywhere without a word.
	if len(t.Source.Platforms) > 0 {
		reachable := false
		for _, goos := range t.Source.Platforms {
			if _, pkg := t.Source.SystemPackage(goos); pkg != "" {
				reachable = true
			}
		}
		if !reachable {
			return fmt.Errorf("tool %q: system source lists platforms %v but names no package for any of them", t.Name, t.Source.Platforms)
		}
	}
	if t.Version != "" {
		return fmt.Errorf("tool %q: system packages are not pinned, so version %q has no effect; remove it", t.Name, t.Version)
	}
	for manager, pkg := range map[string]string{"apt": t.Source.Apt, "brew": t.Source.Brew, "cask": t.Source.Cask, "winget": t.Source.Winget, "command": t.Source.Command} {
		// The name becomes one argument of the manager's command line: a flag
		// or a second word would change what that command does.
		if strings.HasPrefix(pkg, "-") || strings.ContainsAny(pkg, " \t\n") {
			return fmt.Errorf("tool %q: %s name %q is not a package name", t.Name, manager, pkg)
		}
	}
	return nil
}

// SupportsOS reports whether the tool installs on goos: true when Platforms is
// empty, otherwise only for a listed GOOS.
func (t Tool) SupportsOS(goos string) bool {
	if t.Source.Type == "system" {
		// No name for this OS's manager is the same answer as a platforms miss,
		// so Install, Plan and `tools list` skip it without a branch each.
		if _, pkg := t.Source.SystemPackage(goos); pkg == "" {
			return false
		}
	}
	return platform.Supports(t.Source.Platforms, goos)
}

// SystemPackage is the manager and package name a "system" entry gives goos:
// apt on linux, brew on darwin (brew-cask for a cask), winget on windows. The package is "" when the
// entry names none there, or goos has no manager.
func (s Source) SystemPackage(goos string) (manager, pkg string) {
	switch goos {
	case "linux":
		return "apt", s.Apt
	case "darwin":
		if s.Cask != "" {
			return "brew-cask", s.Cask
		}
		return "brew", s.Brew
	case "windows":
		return "winget", s.Winget
	}
	return "", ""
}

// AssetName resolves the release-asset filename for the given OS/arch, or "" when
// the catalog declares no asset for that OS (a tool unavailable on this platform).
//
// A "goos/goarch" key names the asset for exactly one platform and wins over the
// "goos" key, for releases whose arch or OS tokens {goarch} cannot spell (mise:
// linux-x64, macos-arm64).
func (t Tool) AssetName(goos, goarch string) string {
	tmpl, ok := t.Source.Asset[goos+"/"+goarch]
	if !ok {
		tmpl, ok = t.Source.Asset[goos]
	}
	if !ok {
		return ""
	}
	return t.expand(tmpl, goarch)
}

// ChecksumsName resolves the sha256-manifest filename, or "" when the catalog
// declares none. {goarch} is accepted for symmetry but checksum manifests are
// arch-agnostic, so it is rarely present.
func (t Tool) ChecksumsName(goarch string) string {
	if t.Source.Checksums == "" {
		return ""
	}
	return t.expand(t.Source.Checksums, goarch)
}

// expand fills the {version}/{goarch} placeholders shared by the asset and
// checksum templates.
func (t Tool) expand(tmpl, goarch string) string {
	return strings.NewReplacer("{version}", t.Version, "{goarch}", goarch).Replace(tmpl)
}
