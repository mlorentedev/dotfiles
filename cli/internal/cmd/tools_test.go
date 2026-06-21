package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	if _, _, err := execute(t, "tools", "list"); err == nil {
		t.Fatal("expected an error when packages.json is absent")
	}
}

func TestToolsInstall_MissingCatalog(t *testing.T) {
	t.Setenv("DOTFILES_DIR", t.TempDir())
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
