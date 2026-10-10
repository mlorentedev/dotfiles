package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

const testCatalog = `{"tools":[{"name":"sops","version":"3.13.1","profile":"full",` +
	`"source":{"type":"github-release","repo":"getsops/sops","asset":{` +
	`"linux":"sops-v{version}.linux.{goarch}","darwin":"sops-v{version}.darwin.{goarch}",` +
	`"windows":"sops-v{version}.{goarch}.exe"},"checksums":"sops-v{version}.checksums.txt"}}]}`

func TestToolsList(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "packages.json"), []byte(testCatalog), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_DIR", dir)
	outsideCheckout(t)

	stdout, _, err := execute(t, "tools", "list")
	if err != nil {
		t.Fatalf("tools list: %v", err)
	}
	for _, want := range []string{"sops", "3.13.1", "full"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q\n%s", want, stdout)
		}
	}
}

func TestToolsList_MissingCatalog(t *testing.T) {
	t.Setenv("DOTFILES_DIR", t.TempDir()) // empty dir → no packages.json
	outsideCheckout(t)
	if _, _, err := execute(t, "tools", "list"); err == nil {
		t.Fatal("expected an error when packages.json is absent")
	}
}

func TestToolsInstall_MissingCatalog(t *testing.T) {
	t.Setenv("DOTFILES_DIR", t.TempDir())
	outsideCheckout(t)
	if _, _, err := execute(t, "tools", "install"); err == nil {
		t.Fatal("expected an error when packages.json is absent")
	}
}

// loadTestCatalog parses the in-test catalog JSON for the run-loop tests.
func loadTestCatalog(t *testing.T) tools.Catalog {
	t.Helper()
	path := filepath.Join(t.TempDir(), "packages.json")
	if err := os.WriteFile(path, []byte(testCatalog), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := tools.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// fakeInstaller wires an Installer whose seams avoid the network: every fetch
// writes a fixed payload, the checksums file lists its real hash, the tool
// reports as absent so Install always proceeds to place it, and the staged
// binary's post-checksum probe answers with the pin (the payload cannot exec).
func fakeInstaller(t *testing.T, fail bool) *tools.Installer {
	t.Helper()
	const payload = "payload"
	sum := sha256.Sum256([]byte(payload))
	checksums := fmt.Sprintf("%s  sops-v3.13.1.linux.amd64\n", hex.EncodeToString(sum[:]))
	fetch := func(url, dest string) error {
		if fail {
			return fmt.Errorf("simulated download failure")
		}
		if strings.Contains(url, "checksums") {
			return os.WriteFile(dest, []byte(checksums), 0o600)
		}
		return os.WriteFile(dest, []byte(payload), 0o600)
	}
	return &tools.Installer{
		GOOS: "linux", GOARCH: "amd64",
		Dest:           t.TempDir(),
		BaseURL:        "https://example.test",
		Fetch:          fetch,
		Out:            io.Discard,
		CurrentVersion: func(string) string { return "" }, // always absent → install
		Probe:          func(string, ...string) ([]byte, error) { return []byte("sops 3.13.1\n"), nil },
	}
}

func TestRunToolsInstall_UnknownTool(t *testing.T) {
	cat := loadTestCatalog(t)
	if err := runToolsInstall(fakeInstaller(t, false), cat, "bogus", io.Discard); err == nil {
		t.Fatal("expected error for a tool not in the catalog")
	}
}

func TestRunToolsInstall_All(t *testing.T) {
	cat := loadTestCatalog(t)
	if err := runToolsInstall(fakeInstaller(t, false), cat, "", io.Discard); err != nil {
		t.Fatalf("install all: %v", err)
	}
}

func TestRunToolsInstall_AggregatesFailure(t *testing.T) {
	cat := loadTestCatalog(t)
	var errs strings.Builder
	err := runToolsInstall(fakeInstaller(t, true), cat, "", &errs)
	if err == nil {
		t.Fatal("expected an aggregated error when a tool fails")
	}
	if !strings.Contains(errs.String(), "warning:") {
		t.Errorf("expected a per-tool warning on stderr, got %q", errs.String())
	}
}

// outsideCheckout runs the test from a directory with no checkout above it, so
// the catalog resolves from DOTFILES_DIR alone. Without it the resolver walks up
// from the package directory and finds this repository's packages.json.
func outsideCheckout(t *testing.T) {
	t.Helper()
	t.Setenv("DOTFILES_REPO_DIR", "")
	t.Chdir(t.TempDir())
}

// writeCatalog writes a one-tool catalog naming tool into dir/packages.json.
func writeCatalog(t *testing.T, dir, tool string) {
	t.Helper()
	cat := strings.Replace(testCatalog, `"name":"sops"`, `"name":"`+tool+`"`, 1)
	if err := os.WriteFile(filepath.Join(dir, "packages.json"), []byte(cat), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The checkout's catalog wins over a stale deploy mirror, the order doctor
// reads (#1381).
func TestToolsList_CheckoutCatalogWins(t *testing.T) {
	checkout, mirror := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(checkout, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCatalog(t, checkout, "fromcheckout")
	writeCatalog(t, mirror, "frommirror")
	t.Setenv("DOTFILES_REPO_DIR", "")
	t.Setenv("DOTFILES_DIR", mirror)
	t.Chdir(checkout)

	stdout, _, err := execute(t, "tools", "list")
	if err != nil {
		t.Fatalf("tools list: %v", err)
	}
	if !strings.Contains(stdout, "fromcheckout") || strings.Contains(stdout, "frommirror") {
		t.Errorf("want the checkout catalog, got\n%s", stdout)
	}
}

func TestToolsList_MirrorWithoutCheckout(t *testing.T) {
	mirror := t.TempDir()
	writeCatalog(t, mirror, "frommirror")
	t.Setenv("DOTFILES_DIR", mirror)
	outsideCheckout(t)

	stdout, _, err := execute(t, "tools", "list")
	if err != nil {
		t.Fatalf("tools list: %v", err)
	}
	if !strings.Contains(stdout, "frommirror") {
		t.Errorf("want the mirror catalog, got\n%s", stdout)
	}
}

// --dry-run reports the plan and changes nothing: the tool is absent, so the
// plan says install, and Dest stays empty.
func TestToolsInstall_DryRun(t *testing.T) {
	mirror, home := t.TempDir(), t.TempDir()
	writeCatalog(t, mirror, "sops")
	t.Setenv("DOTFILES_DIR", mirror)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	outsideCheckout(t)

	stdout, _, err := execute(t, "tools", "install", "--dry-run")
	if err != nil {
		t.Fatalf("tools install --dry-run: %v", err)
	}
	// The fixture declares a build for linux, darwin and windows, so the action
	// is install on every CI leg. Match the whole row: a wrong or empty action
	// cell must fail, not only a missing name.
	row := regexp.MustCompile(`(?m)^sops\s+absent\s+3\.13\.1\s+install\s*$`)
	if !row.MatchString(stdout) {
		t.Errorf("dry-run output has no row `sops absent 3.13.1 install`\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "bin")); !os.IsNotExist(err) {
		t.Errorf("dry-run created or touched ~/.local/bin (stat err %v)", err)
	}
}

func TestListAsset(t *testing.T) {
	hive := tools.Tool{Name: "hive", Version: "4.2.2", Source: tools.Source{
		Type: "uv-tool", Package: "hive-vault", Platforms: []string{"linux", "darwin"}}}
	sops := tools.Tool{Name: "sops", Version: "3.13.1", Source: tools.Source{
		Type: "github-release", Asset: map[string]string{"linux": "sops-v{version}.linux.{goarch}"}}}
	bw := tools.Tool{Name: "bw", Source: tools.Source{Type: "npm", Package: "@bitwarden/cli"}}
	gh := tools.Tool{Name: "gh", Source: tools.Source{Type: "system", Apt: "gh", Winget: "GitHub.cli"}}
	cases := []struct {
		tool       tools.Tool
		goos, want string
	}{
		{hive, "linux", "uv:hive-vault"},
		{hive, "windows", "(not in the catalog on this platform)"},
		{sops, "linux", "sops-v3.13.1.linux.amd64"},
		{sops, "windows", "(no build for this platform)"},
		{bw, "windows", "npm:@bitwarden/cli"},
		{gh, "linux", "apt:gh"},
		{gh, "windows", "winget:GitHub.cli"},
		{gh, "darwin", "(not in the catalog on this platform)"}, // no brew name
	}
	for _, tc := range cases {
		if got := listAsset(tc.tool, tc.goos, "amd64"); got != tc.want {
			t.Errorf("listAsset(%s, %s) = %q, want %q", tc.tool.Name, tc.goos, got, tc.want)
		}
	}
}

// A system package has no pin: the dry-run row shows a dash rather than an empty
// cell that would shift the action column, and says absent or present.
func TestPlanToolsInstall_SystemRows(t *testing.T) {
	cat := tools.Catalog{Tools: []tools.Tool{
		{Name: "gh", Source: tools.Source{Type: "system", Apt: "gh", Brew: "gh"}},
		{Name: "tmux", Source: tools.Source{Type: "system", Apt: "tmux"}},
		{Name: "future", Version: "1.0.0", Source: tools.Source{Type: "flatpak"}},
	}}
	in := &tools.Installer{
		GOOS: "linux", GOARCH: "amd64", Dest: t.TempDir(), Out: io.Discard,
		HasCommand: func(string) bool { return true },
		IsRoot:     func() bool { return true },
		Query: func(_ string, args ...string) ([]byte, error) {
			if args[len(args)-1] == "tmux" {
				return []byte("install ok installed"), nil
			}
			return nil, fmt.Errorf("not installed")
		},
		Run: func(string, ...string) error {
			t.Fatal("a dry run must not run a package manager")
			return nil
		},
	}
	var out strings.Builder
	if err := planToolsInstall(in, cat, "", &out); err != nil {
		t.Fatal(err)
	}
	for _, row := range []string{`(?m)^gh\s+absent\s+-\s+install\s*$`, `(?m)^tmux\s+present\s+-\s+skip\s*$`,
		`(?m)^future\s+absent\s+1\.0\.0\s+skip \(source type "flatpak" is not known to this dotf\)$`} {
		if !regexp.MustCompile(row).MatchString(out.String()) {
			t.Errorf("no row matching %s in\n%s", row, out.String())
		}
	}
}

// One entry of a type this dotf does not know must not turn the whole run red or
// stop the entries after it.
func TestInstallAll_UnknownTypeDoesNotFailTheRun(t *testing.T) {
	var ran []string
	in := &tools.Installer{
		GOOS: "linux", GOARCH: "amd64", Dest: t.TempDir(), Out: io.Discard,
		HasCommand:     func(string) bool { return true },
		CurrentVersion: func(string) string { return "" },
		Probe:          func(string, ...string) ([]byte, error) { return []byte("bw 2026.5.0"), nil },
		Run: func(name string, args ...string) error {
			ran = append(ran, name+" "+strings.Join(args, " "))
			return nil
		},
	}
	// CurrentVersion answers absent until npm "ran", then at the pin.
	in.CurrentVersion = func(string) string {
		if len(ran) > 0 {
			return "2026.5.0"
		}
		return ""
	}
	selected := []tools.Tool{
		{Name: "future", Version: "1.0.0", Source: tools.Source{Type: "flatpak"}},
		{Name: "bw", Version: "2026.5.0", Source: tools.Source{Type: "npm", Package: "@bitwarden/cli"}},
	}
	if err := installAll(in, selected, io.Discard); err != nil {
		t.Fatalf("installAll: %v", err)
	}
	if len(ran) != 1 || !strings.HasPrefix(ran[0], "npm install -g") || !strings.HasSuffix(ran[0], " @bitwarden/cli@2026.5.0") {
		t.Errorf("the tool after the unknown type did not install: ran %v", ran)
	}
}

// A package that needs a sudo password is reported and does not stop, or fail,
// the tools after it.
func TestInstallAll_NeedsSudoDoesNotFailTheRun(t *testing.T) {
	var out strings.Builder
	var ran [][]string
	in := &tools.Installer{
		GOOS: "linux", GOARCH: "amd64", Dest: t.TempDir(), Out: &out,
		HasCommand: func(string) bool { return true },
		IsRoot:     func() bool { return false },
		Query:      func(string, ...string) ([]byte, error) { return nil, fmt.Errorf("a password is required") },
		Run: func(name string, args ...string) error {
			ran = append(ran, append([]string{name}, args...))
			return fmt.Errorf("exit status 1")
		},
	}
	selected := []tools.Tool{
		{Name: "gh", Source: tools.Source{Type: "system", Apt: "gh"}},
		{Name: "tmux", Source: tools.Source{Type: "system", Apt: "tmux"}},
	}
	if err := installAll(in, selected, io.Discard); err != nil {
		t.Fatalf("installAll: %v", err)
	}
	if len(ran) != 2 || ran[0][1] != "-n" || ran[1][1] != "-n" {
		t.Errorf("want both tools attempted with sudo -n, ran %v", ran)
	}
	for _, want := range []string{"gh: needs sudo; run: sudo apt-get install -y --no-remove gh", "tmux: needs sudo; run: sudo apt-get install -y --no-remove tmux"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
	// And one command for both at the end, so a fresh machine is one paste (#2308).
	if want := "\n  sudo apt-get install -y --no-remove gh tmux\n"; !strings.Contains(out.String(), want) {
		t.Errorf("output lacks the combined command %q:\n%s", want, out.String())
	}
}

// A run where sudo was never in the way prints no combined command.
func TestInstallAll_NoSudoNoCombinedCommand(t *testing.T) {
	var out strings.Builder
	in := &tools.Installer{
		GOOS: "linux", GOARCH: "amd64", Dest: t.TempDir(), Out: &out,
		HasCommand: func(string) bool { return true },
		IsRoot:     func() bool { return true },
		Query:      func(string, ...string) ([]byte, error) { return []byte("install ok installed"), nil },
		Run:        func(string, ...string) error { return nil },
	}
	selected := []tools.Tool{{Name: "gh", Source: tools.Source{Type: "system", Apt: "gh"}}}
	if err := installAll(in, selected, io.Discard); err != nil {
		t.Fatalf("installAll: %v", err)
	}
	if strings.Contains(out.String(), "one command") {
		t.Errorf("no package needed sudo, yet:\n%s", out.String())
	}
}

// dryRunCatalog writes a catalog of the given tool objects and points the
// command at it.
func dryRunCatalog(t *testing.T, toolsJSON ...string) {
	t.Helper()
	mirror, home := t.TempDir(), t.TempDir()
	cat := `{"tools":[` + strings.Join(toolsJSON, ",") + `]}`
	if err := os.WriteFile(filepath.Join(mirror, "packages.json"), []byte(cat), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_DIR", mirror)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	outsideCheckout(t)
}

// otherOS is a platform this test is not running on.
func otherOS() string {
	if runtime.GOOS == "linux" {
		return "darwin"
	}
	return "linux"
}

// An unsupported row is not a failure: it exits 0 and says why. The single-name
// form plans that one tool only (#1892).
func TestToolsInstall_DryRunUnsupportedRowForOneNamedTool(t *testing.T) {
	dryRunCatalog(t,
		`{"name":"elsewhere","version":"1.0.0","source":{"type":"npm","package":"x","platforms":["`+otherOS()+`"]}}`,
		`{"name":"other","version":"1.0.0","source":{"type":"npm","package":"y"}}`)

	stdout, _, err := execute(t, "tools", "install", "--dry-run", "elsewhere")
	if err != nil {
		t.Fatalf("an unsupported row must exit 0: %v", err)
	}
	row := regexp.MustCompile(`(?m)^elsewhere\s+absent\s+1\.0\.0\s+unsupported \(not installed on ` + runtime.GOOS + ` by this catalog\)\s*$`)
	if !row.MatchString(stdout) {
		t.Errorf("want an unsupported row saying why\n%s", stdout)
	}
	if strings.Contains(stdout, "other") {
		t.Errorf("the single-name form planned a tool it was not asked about\n%s", stdout)
	}
}

// What install refuses, the dry run refuses: the row says why, and the command
// fails after printing every row, the way install would.
func TestToolsInstall_DryRunFailsOnAnEntryInstallRefuses(t *testing.T) {
	dryRunCatalog(t,
		`{"name":"nopkg","version":"1.0.0","source":{"type":"npm"}}`,
		`{"name":"fine","version":"1.0.0","source":{"type":"npm","package":"y"}}`)

	stdout, _, err := execute(t, "tools", "install", "--dry-run")
	if err == nil || !strings.Contains(err.Error(), "install would refuse: nopkg") {
		t.Fatalf("want the dry run to refuse nopkg, got %v", err)
	}
	if !regexp.MustCompile(`(?m)^nopkg\s+absent\s+1\.0\.0\s+refused \(npm source declares no package\)\s*$`).MatchString(stdout) {
		t.Errorf("want a refused row saying why\n%s", stdout)
	}
	if !strings.Contains(stdout, "fine") {
		t.Errorf("a refusal must not hide the rows after it\n%s", stdout)
	}
}
