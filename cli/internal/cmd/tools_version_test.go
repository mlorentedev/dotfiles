package cmd

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// The two banners that broke the setup scripts' token parse, plus the absent
// case: a shell caller tests stdout and the exit status, both must be right.
func TestToolsVersionCmd(t *testing.T) {
	pinToolsVersion(t, "linux", tools.Catalog{})
	toolsVersionRunner = func(name string, _ ...string) ([]byte, error) {
		switch name {
		case "hive":
			return []byte("hive-vault 3.0.0\n"), nil
		case "opencode":
			return []byte("OpenCode locked.\n1.16.2\n"), nil
		}
		return nil, errors.New("not found")
	}

	cases := []struct {
		name    string
		tool    string
		want    string
		wantErr bool
	}{
		{name: "prefixed banner", tool: "hive", want: "3.0.0"},
		{name: "banner line before the number", tool: "opencode", want: "1.16.2"},
		{name: "absent tool: empty stdout, exit error", tool: "nope", want: "", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newToolsVersionCmd()
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{tc.tool})
			err := c.Execute()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got := strings.TrimSpace(out.String()); got != tc.want {
				t.Errorf("stdout = %q, want %q", got, tc.want)
			}
		})
	}
}

// `dotf tools version` probes through tools.ProbeRunner, so a tool that crashes
// is not read at a version its warnings name (BUG-141).
func TestToolsVersion_DropsAFailedRunsStderr(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not on PATH")
	}
	out, err := toolsVersionRunner("sh", "-c", "echo 'ansible-core 2.21.5 supports Python up to 3.14' >&2; exit 3")
	if err == nil || len(out) != 0 {
		t.Errorf("a crash read as output %q (err %v), want none", out, err)
	}
}

// pinToolsVersion fixes the OS and catalog `dotf tools version` answers for,
// and restores the runner, so no checkout's packages.json decides a test.
func pinToolsVersion(t *testing.T, goos string, cat tools.Catalog) {
	t.Helper()
	runner, goos0, load := toolsVersionRunner, toolsVersionGOOS, toolsVersionCatalog
	t.Cleanup(func() { toolsVersionRunner, toolsVersionGOOS, toolsVersionCatalog = runner, goos0, load })
	toolsVersionGOOS = goos
	toolsVersionCatalog = func() (tools.Catalog, error) { return cat, nil }
}

// On Windows, hive installs itself, so `dotf tools version hive` reads the
// launcher the catalog declares, as `dotf tools install` does: a uv copy on
// PATH must not answer for it (#2991). The setup scripts gate the daemon on it.
func TestToolsVersion_ReadsASelfInstalledToolAtItsLauncher(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\L`)
	hive := tools.Tool{Name: "hive", Version: "4.4.2", Source: tools.Source{Type: "uv-tool", Package: "hive-vault",
		SelfInstall: &tools.SelfInstall{Platforms: []string{"windows"}, Args: []string{"self-upgrade", "{version}"}, Launcher: `%LOCALAPPDATA%\hive\bin\hive.cmd`}}}
	pinToolsVersion(t, "windows", tools.Catalog{Tools: []tools.Tool{hive}})
	toolsVersionRunner = func(name string, _ ...string) ([]byte, error) {
		switch name {
		case `C:\L\hive\bin\hive.cmd`:
			return []byte("hive-vault 4.4.2\n"), nil
		case "hive":
			return []byte("hive-vault 4.3.0\n"), nil
		}
		return nil, errors.New("not found")
	}
	c := newToolsVersionCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&bytes.Buffer{})
	c.SetArgs([]string{"hive"})
	if err := c.Execute(); err != nil || strings.TrimSpace(out.String()) != "4.4.2" {
		t.Errorf("tools version hive = %q, %v; want the launcher's 4.4.2", out.String(), err)
	}

	// Absent, the error names the launcher it read, not PATH it never read.
	toolsVersionRunner = func(string, ...string) ([]byte, error) { return nil, errors.New("not found") }
	c = newToolsVersionCmd()
	c.SetOut(&bytes.Buffer{})
	c.SetErr(&bytes.Buffer{})
	c.SetArgs([]string{"hive"})
	if err := c.Execute(); err == nil || !strings.Contains(err.Error(), "launcher") || strings.Contains(err.Error(), "PATH") {
		t.Errorf("absent hive: err = %v, want the launcher named and not PATH", err)
	}
}
