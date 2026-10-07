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

// Source declares how to fetch a tool. Three kinds:
//   - "github-release": a pinned per-OS/arch release binary, verified against the
//     release checksums by the installer (CLI-029 PR-B), mirroring the
//     deterministic age/install-dotf pattern rather than relying on winget/apt.
//   - "npm": a globally-installed npm package (Package), pinned by Version. Used
//     for tools whose first-class distribution is npm and that do not ship a
//     raw, checksum-manifested github-release binary — e.g. the Bitwarden CLI
//     (@bitwarden/cli), whose releases are zip archives under a cli-v{date} tag
//     with no sha256 manifest (#577, ADR-028 Phase 0).
//   - "uv-tool": a Python package installed with `uv tool install`, pinned by
//     Version. Package is the PyPI name; the tool's Name is the binary it puts
//     on PATH (hive-vault installs hive, #1993).
type Source struct {
	Type string `json:"type"`
	Repo string `json:"repo"`
	// Package is the npm package name for source.type "npm" (e.g.
	// "@bitwarden/cli") and the PyPI name for "uv-tool" (e.g. "hive-vault").
	// Unused by github-release sources.
	Package string `json:"package,omitempty"`
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
	seen := make(map[string]struct{}, len(c.Tools))
	for _, t := range c.Tools {
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
	}
	return c, nil
}

// SupportsOS reports whether the tool installs on goos: true when Platforms is
// empty, otherwise only for a listed GOOS.
func (t Tool) SupportsOS(goos string) bool {
	return platform.Supports(t.Source.Platforms, goos)
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
