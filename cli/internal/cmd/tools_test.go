package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCatalog = `{"tools":[{"name":"sops","version":"3.13.1","profile":"full",` +
	`"source":{"type":"github-release","repo":"getsops/sops","asset":{` +
	`"linux":"sops-v{version}.linux.{goarch}","darwin":"sops-v{version}.darwin.{goarch}",` +
	`"windows":"sops-v{version}.{goarch}.exe"}}}]}`

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
