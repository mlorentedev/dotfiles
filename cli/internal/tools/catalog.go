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

// Source declares how to fetch a tool. The pilot supports only "github-release":
// a pinned per-OS/arch release binary (verified against the release checksums by
// the installer — CLI-029 PR-B), mirroring the deterministic age/install-dotf
// pattern rather than relying on winget/apt availability.
type Source struct {
	Type string `json:"type"`
	Repo string `json:"repo"`
	// Asset maps GOOS -> a release-asset filename template. A per-OS map (not a
	// single template) is required because release naming is irregular across OSes
	// — e.g. sops is "sops-v{version}.linux.{goarch}" but "sops-v{version}.{goarch}.exe"
	// on Windows (no OS token, arch before .exe). Templates expand {version} and
	// {goarch}.
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
	return c, nil
}

// AssetName resolves the release-asset filename for the given OS/arch, or "" when
// the catalog declares no asset for that OS (a tool unavailable on this platform).
func (t Tool) AssetName(goos, goarch string) string {
	tmpl, ok := t.Source.Asset[goos]
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
