package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// fileTool is a pin that ships a file, not an executable: a zsh plugin. mise
// has no registry entry for it, so it installs through mise's http backend
// (`http:<name>`), which unpacks a tarball into the same install directory on
// macOS and Linux, and refuses a download whose digest is not the pinned one.
// The probes that ask an executable for `--version` cannot ask a file, so the
// sync checks the file itself (DX-007, #2654).
type fileTool struct {
	// URL is the https archive of a version; {{ version }} is the pin.
	URL string
	// File is the file the tool exists to provide, relative to its install
	// directory. A tool counts as installed when it is there.
	File string
	// SHA256 is the digest of each version's archive. A version with no entry
	// does not parse: an archive nothing vouches for is never fetched.
	SHA256 map[string]string
}

// fileTools is keyed by the name the pin derives from its versions.conf
// variable (ZSH_AUTOSUGGESTIONS_VERSION is zsh-autosuggestions). A released
// dotf that does not know the name renders it as a registry tool and fails the
// install, so a pin is marked in versions.conf only once DOTF_VERSION is a
// release carrying its row here; tests/versions-conf.bats holds that order.
var fileTools = map[string]fileTool{
	"zsh-autosuggestions": {
		URL:  "https://github.com/zsh-users/zsh-autosuggestions/archive/refs/tags/v{{ version }}.tar.gz",
		File: "zsh-autosuggestions.zsh",
		SHA256: map[string]string{
			"0.7.1": "0df7affff21cd87ed298e6a3970ed08a1dd66a6efa676454ee5b091ad503badf",
		},
	},
}

const httpBackend = "http:"

func isFileTool(name string) bool {
	_, ok := fileTools[name]
	return ok
}

// skipsHere reports a file tool on a platform it is not installed on: zsh
// plugins have no Windows shell to load them, so the sync renders, installs,
// probes and prunes nothing for them there.
func skipsHere(name string) bool {
	return runtime.GOOS == "windows" && isFileTool(name)
}

// BinaryPins is tools without the file tools, for a caller that asks mise
// about executables (`mise bin-paths`), which has no answer for a plugin.
func BinaryPins(tools []MiseTool) []MiseTool {
	var out []MiseTool
	for _, t := range tools {
		if !isFileTool(t.Name) {
			out = append(out, t)
		}
	}
	return out
}

// miseSpec is the tool as mise lists it: `http:<name>` for a file tool.
func miseSpec(name string) string {
	if isFileTool(name) {
		return httpBackend + name
	}
	return name
}

// pinName is the pin name behind a name `mise ls` reports: the inverse of
// miseSpec, so an installed plugin is matched to the pin that asks for it and
// not pruned as an orphan on every run.
func pinName(listed string) string {
	if name, ok := strings.CutPrefix(listed, httpBackend); ok && isFileTool(name) {
		return name
	}
	return listed
}

// validateFileTools rejects a pin the table cannot install safely.
func validateFileTools(tools []MiseTool, latest []string) error {
	for _, t := range tools {
		ft, ok := fileTools[t.Name]
		if !ok {
			continue
		}
		if contains(latest, t.Name) {
			return fmt.Errorf("versions.conf: %s is a file tool pinned by digest and cannot track latest", t.Name)
		}
		if ft.SHA256[t.Version] == "" {
			known := make([]string, 0, len(ft.SHA256))
			for v := range ft.SHA256 {
				known = append(known, v)
			}
			sort.Strings(known)
			return fmt.Errorf("versions.conf: %s %s has no pinned sha256 in fileTools (known: %s); add its digest before bumping the pin", t.Name, t.Version, strings.Join(known, ", "))
		}
	}
	return nil
}

// renderFileTool is the [tools] line for a file tool.
func renderFileTool(t MiseTool) string {
	ft := fileTools[t.Name]
	url := strings.ReplaceAll(ft.URL, "{{ version }}", t.Version)
	return fmt.Sprintf("%q = { version = %q, url = %q, checksum = %q }\n", miseSpec(t.Name), t.Version, url, "sha256:"+ft.SHA256[t.Version])
}

// reinstallEmptyFileTools repairs a file tool whose install directory is there
// without its file (a deleted file, an interrupted unpack): `mise install`
// counts the directory as the install and does nothing, so the sync would fail
// on every run until someone removed it by hand. --force unpacks the
// digest-checked archive again.
func (s MiseSync) reinstallEmptyFileTools(tools []MiseTool) error {
	for _, t := range tools {
		if !isFileTool(t.Name) || skipsHere(t.Name) || s.fileToolInstalled(t) {
			continue
		}
		if out, err := s.Run("mise", "install", "--force", miseSpec(t.Name)+"@"+t.Version); err != nil {
			return fmt.Errorf("mise install --force %s: %w\n%s", t.Name, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// fileToolInstalled asks mise where the pinned version is installed and looks
// for the file there, and for the same file through `latest`, which is the
// path .zshrc sources. `mise which` finds executables only, and an install
// directory that exists proves nothing about what a half-finished unpack left.
func (s MiseSync) fileToolInstalled(t MiseTool) bool {
	run := s.Stdout
	if run == nil {
		run = s.Run
	}
	out, err := run("mise", "where", miseSpec(t.Name))
	if err != nil {
		return false
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" || filepath.Base(dir) != t.Version {
		return false
	}
	file := fileTools[t.Name].File
	if fi, err := os.Stat(filepath.Join(dir, file)); err != nil || !fi.Mode().IsRegular() {
		return false
	}
	// The shell sources <install root>/latest/<file> (no `mise` process at
	// startup), so that path must resolve too: a missing or dangling `latest`
	// would leave the sync satisfied and the plugin never loaded.
	fi, err := os.Stat(filepath.Join(filepath.Dir(dir), "latest", file))
	return err == nil && fi.Mode().IsRegular()
}
