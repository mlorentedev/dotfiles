package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
// writes a fixed payload, the checksums file lists its real hash, and the tool
// reports as absent so Install always proceeds to place it.
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
