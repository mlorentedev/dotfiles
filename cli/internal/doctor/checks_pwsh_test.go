package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

const (
	pwshVersionsConf = "# pwsh-module: PSScriptAnalyzer\nPSSCRIPTANALYZER_VERSION=1.25.0\n# pwsh-module: Pester\nPESTER_VERSION=6.2.0\n"
	pwshCatalog      = `{"tools": [{"name": "powershell", "profile": "full", "source": {"type": "system", "brew": "powershell", "winget": "Microsoft.PowerShell", "command": "pwsh"}}]}`
)

// pwshFixture is a machine on goos whose checkout pins the two modules and
// whose catalog installs pwsh through brew and winget. missing is what the
// import check answers; onPath says whether pwsh resolves.
func pwshFixture(t *testing.T, goos string, onPath bool, missing string) (*System, *Config) {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "versions.conf"), pwshVersionsConf)
	writeFile(t, filepath.Join(dir, "packages.json"), pwshCatalog)
	writeFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main\n")
	var path []string
	if onPath {
		path = []string{"pwsh"}
	}
	s := newSys(map[string]string{"HOME": dir, "DOTFILES_REPO_DIR": dir}, path, map[string]string{"pwsh --version": "PowerShell 7.6.7\n"})
	s.GOOS = goos
	s.CommandOutput = func(name string, args ...string) (string, error) {
		if name == "pwsh" && len(args) == 1 && args[0] == "--version" {
			return "PowerShell 7.6.7\n", nil
		}
		if name == "pwsh" && len(args) == 4 && strings.Contains(args[3], "Import-Module") {
			return missing, nil
		}
		return "", nil
	}
	return s, &Config{VersionsPath: filepath.Join(dir, "versions.conf"), DotfilesDir: dir}
}

func runCheckPowerShell(s *System, cfg *Config) (*Report, string) {
	var b bytes.Buffer
	rep := capture(&b)
	checkPowerShell(s, cfg, rep)
	return rep, b.String()
}

func TestCheckPowerShell(t *testing.T) {
	cases := []struct {
		name     string
		goos     string
		onPath   bool
		missing  string
		fails    int
		warns    int
		contains string
	}{
		{name: "pwsh and both modules at their pin", goos: "darwin", onPath: true, contains: "2 module(s) at their pin"},
		{name: "a module below its pin warns with the remedy", goos: "windows", onPath: true, missing: "MISSING Pester\n", warns: 1, contains: "dotf converge --only pwsh-modules"},
		{name: "pwsh missing where the catalog installs it fails", goos: "darwin", fails: 1, contains: "dotf converge --only tools"},
		{name: "pwsh missing where nothing installs it skips", goos: "linux", contains: "lint-powershell job is the gate"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, cfg := pwshFixture(t, c.goos, c.onPath, c.missing)
			rep, out := runCheckPowerShell(s, cfg)
			if rep.Failures() != c.fails || rep.Warnings() != c.warns || !strings.Contains(out, c.contains) {
				t.Errorf("fails=%d warns=%d, want %d/%d and %q\n%s", rep.Failures(), rep.Warnings(), c.fails, c.warns, c.contains, out)
			}
		})
	}
}

func TestCheckPowerShell_AMalformedMarkerFails(t *testing.T) {
	s, cfg := pwshFixture(t, "darwin", true, "")
	writeFile(t, cfg.VersionsPath, "# pwsh-module: Pester\n")
	if rep, out := runCheckPowerShell(s, cfg); rep.Failures() != 1 {
		t.Errorf("want one failure\n%s", out)
	}
}
