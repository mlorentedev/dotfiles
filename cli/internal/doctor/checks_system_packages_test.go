package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const catalogWithAptEntries = `{"tools":[
  {"name":"gh","profile":"full","source":{"type":"system","apt":"gh","brew":"gh"}},
  {"name":"docker","profile":"full","source":{"type":"system","apt":"docker.io","brew":"docker"}},
  {"name":"tmux","profile":"full","source":{"type":"system","apt":"tmux","brew":"tmux"}},
  {"name":"sops","version":"3.13.1","profile":"full","source":{"type":"github-release","repo":"getsops/sops","asset":{"linux":"x"},"checksums":"c"}}
]}`

// systemPackagesReport runs the check on a linux box where tmux is installed,
// gh and docker.io are not, and `sudo -n true` succeeds only when sudo is set.
func systemPackagesReport(t *testing.T, sudo bool) string {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("as root apt needs no sudo, so nothing waits on it")
	}
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithAptEntries)
	cmds := map[string]string{"dpkg-query -W -f=${Status} tmux": "install ok installed"}
	if sudo {
		cmds["sudo -n true"] = ""
	}
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, []string{"apt-get", "sudo", "dpkg-query"}, cmds)
	sys.GOOS = "linux"
	var buf bytes.Buffer
	checkSystemPackages(sys, &Config{DotfilesDir: t.TempDir()}, capture(&buf))
	return buf.String()
}

// Every package that waits on a sudo password is named, with ONE command that
// installs them all — the command `dotf tools install` prints (#2308).
func TestCheckSystemPackages_WarnsWithOneCommandForEveryPackage(t *testing.T) {
	out := systemPackagesReport(t, false)
	if !strings.Contains(out, "[WARN]") || !strings.Contains(out, "2 system package(s) wait on sudo") || !strings.Contains(out, "(gh, docker)") {
		t.Fatalf("want a WARN naming gh and docker\n%s", out)
	}
	if !strings.Contains(out, "`sudo apt-get install -y --no-remove gh docker.io`") {
		t.Errorf("want the one combined command\n%s", out)
	}
	if strings.Contains(out, "tmux") || strings.Contains(out, "sops") {
		t.Errorf("an installed package and a non-system entry are not sudo's to install\n%s", out)
	}
}

// When sudo runs without a password, dotf installs the packages itself, so
// nothing waits on the person.
func TestCheckSystemPackages_PassesWhenSudoDoesNotAsk(t *testing.T) {
	out := systemPackagesReport(t, true)
	if !strings.Contains(out, "[ OK ] no system package waits on sudo") {
		t.Errorf("want PASS\n%s", out)
	}
}

// Off Linux nothing escalates, so the check asks no package manager anything:
// a winget query costs seconds per entry, for an answer that cannot be a wait.
func TestCheckSystemPackages_AsksNoManagerOffLinux(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithAptEntries)
	for _, goos := range []string{"darwin", "windows"} {
		sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, []string{"brew", "winget", "sudo"}, nil)
		sys.GOOS = goos
		var asked []string
		sys.CommandOutput = func(name string, args ...string) (string, error) {
			asked = append(asked, name)
			return "", nil
		}
		var buf bytes.Buffer
		checkSystemPackages(sys, &Config{DotfilesDir: t.TempDir()}, capture(&buf))
		if len(asked) != 0 || strings.Contains(buf.String(), "[WARN]") {
			t.Errorf("%s: asked %v\n%s", goos, asked, buf.String())
		}
	}
}

// A Linux box without apt (Fedora, Arch) is a skip, not an OK: nothing there
// could be planned, so "none waits on sudo" would be a false answer.
func TestCheckSystemPackages_SkipsALinuxWithoutApt(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "packages.json"), catalogWithAptEntries)
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": repo}, []string{"sudo"}, nil)
	sys.GOOS = "linux"
	var buf bytes.Buffer
	checkSystemPackages(sys, &Config{DotfilesDir: t.TempDir()}, capture(&buf))
	if !strings.Contains(buf.String(), "apt-get is not on PATH") || strings.Contains(buf.String(), "[ OK ]") {
		t.Errorf("want a skip naming apt-get\n%s", buf.String())
	}
}
