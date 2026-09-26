// Package pi reconciles pi's installed packages with the manifest this
// repository declares (HARNESS-139, #1628).
//
// It replaces a shell twin pair in setup-linux.sh and setup-windows.ps1 that
// only ever installed the difference, so removing an entry from the manifest
// left the package installed on every machine that already had it. Here the
// manifest is the whole declaration: what it does not name is removed, and a
// path it retires is moved to an archive, never deleted.
//
// pi owns ~/.pi/agent/settings.json and rewrites it at runtime (#754), so this
// package never writes that file. Every change goes through pi's own CLI.
package pi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ManifestFile is the declaration, relative to the repository root.
const ManifestFile = "ai/pi/packages.json"

// Package is one declared pi package.
type Package struct {
	// Source is the argument to `pi install`, including its version.
	Source string `json:"source"`
	Why    string `json:"why"`
	// Requires names the executables the package needs on PATH. A package
	// whose requirement does not resolve is a feature that reads as present
	// and does not work, which is what pi-memory was without qmd.
	Requires []string `json:"requires,omitempty"`
}

// Retired is a path under ~/.pi/agent that the manifest retires: it is moved
// to ~/.pi/agent/archive/ and never deleted.
type Retired struct {
	Path string `json:"path"`
	Why  string `json:"why"`
}

// Manifest is ai/pi/packages.json.
type Manifest struct {
	Version  int       `json:"version"`
	Packages []Package `json:"packages"`
	Retire   []Retired `json:"retire,omitempty"`
}

// archiveDir is where retired paths go, under the agent dir.
const archiveDir = "archive"

// LoadManifest reads and validates the manifest under repoRoot. Anything it
// cannot read is an error, never an empty declaration: an empty want-list would
// plan the removal of every live package.
func LoadManifest(repoRoot string) (Manifest, error) {
	var m Manifest
	raw, err := os.ReadFile(filepath.Join(repoRoot, ManifestFile)) // #nosec G304 -- the repository's own manifest
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, fmt.Errorf("%s: %w", ManifestFile, err)
	}
	return m, m.validate()
}

func (m Manifest) validate() error {
	if len(m.Packages) == 0 {
		return fmt.Errorf("%s declares no packages; refusing to read that as \"remove everything\"", ManifestFile)
	}
	seen := map[string]bool{}
	for _, p := range m.Packages {
		if strings.TrimSpace(p.Source) == "" {
			return fmt.Errorf("%s: a package has no source", ManifestFile)
		}
		id := Identity(p.Source)
		if seen[id] {
			return fmt.Errorf("%s: %s is declared twice", ManifestFile, id)
		}
		seen[id] = true
	}
	for _, r := range m.Retire {
		clean := filepath.Clean(r.Path)
		if r.Path == "" || filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, "..") || clean == archiveDir {
			return fmt.Errorf("%s: retire path %q must name something under ~/.pi/agent other than %s/", ManifestFile, r.Path, archiveDir)
		}
	}
	return nil
}

// LiveSources reads the packages pi has recorded in its settings file, in both
// entry forms: a string, and upstream's object form carrying `source`. An
// absent file is an empty live set; one that exists and does not parse is an
// error, because removals planned against it would be guesses.
func LiveSources(settingsPath string) ([]string, error) {
	raw, err := os.ReadFile(settingsPath) // #nosec G304 -- pi's own settings file
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s struct {
		Packages []json.RawMessage `json:"packages"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", settingsPath, err)
	}
	var out []string
	for _, entry := range s.Packages {
		var str string
		if json.Unmarshal(entry, &str) == nil {
			out = append(out, str)
			continue
		}
		var obj struct {
			Source string `json:"source"`
		}
		if json.Unmarshal(entry, &obj) == nil && obj.Source != "" {
			out = append(out, obj.Source)
		}
	}
	return out, nil
}

// Identity is a source without its version: `npm:@scope/name@1.2.3` and
// `npm:@scope/name@2.0.0` are the same package. pi install replaces the entry
// of the same package (measured on pi 0.87.1), so identity decides removal.
func Identity(source string) string {
	prefix, rest, ok := strings.Cut(source, ":")
	if !ok || strings.HasPrefix(rest, "/") || strings.HasPrefix(source, ".") {
		return source
	}
	// A leading @ is a scope, not a version separator.
	if i := strings.LastIndex(rest, "@"); i > 0 {
		rest = rest[:i]
	}
	return prefix + ":" + rest
}

// Plan is what an apply would do, in the order it does it.
type Plan struct {
	Remove   []string
	Install  []string
	Retire   []Retired
	agentDir string
}

// Empty reports that live already matches the declaration.
func (p Plan) Empty() bool { return len(p.Remove)+len(p.Install)+len(p.Retire) == 0 }

// NewPlan diffs the manifest against the live sources: remove what the
// manifest does not declare at any version, install what is not live at
// exactly its declared source, and retire the declared paths that exist.
func NewPlan(m Manifest, live []string, agentDir string) Plan {
	p := Plan{agentDir: agentDir}
	declared, present := map[string]bool{}, map[string]bool{}
	for _, pkg := range m.Packages {
		declared[Identity(pkg.Source)] = true
	}
	for _, s := range live {
		present[s] = true
		if !declared[Identity(s)] {
			p.Remove = append(p.Remove, s)
		}
	}
	for _, pkg := range m.Packages {
		if !present[pkg.Source] {
			p.Install = append(p.Install, pkg.Source)
		}
	}
	for _, r := range m.Retire {
		if _, err := os.Lstat(filepath.Join(agentDir, r.Path)); err == nil {
			p.Retire = append(p.Retire, r)
		}
	}
	return p
}
