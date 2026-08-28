package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleCatalog = `{
  "tools": [
    {
      "name": "sops",
      "version": "3.13.1",
      "profile": "full",
      "source": {
        "type": "github-release",
        "repo": "getsops/sops",
        "asset": {
          "linux": "sops-v{version}.linux.{goarch}",
          "darwin": "sops-v{version}.darwin.{goarch}",
          "windows": "sops-v{version}.{goarch}.exe"
        },
        "checksums": "sops-v{version}.checksums.txt"
      }
    }
  ]
}`

func writeCatalog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "packages.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	cat, err := Load(writeCatalog(t, sampleCatalog))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cat.Tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(cat.Tools))
	}
	got := cat.Tools[0]
	if got.Name != "sops" || got.Version != "3.13.1" || got.Profile != "full" {
		t.Fatalf("unexpected tool: %+v", got)
	}
	if got.Source.Type != "github-release" || got.Source.Repo != "getsops/sops" {
		t.Fatalf("unexpected source: %+v", got.Source)
	}
	if got.Source.Checksums != "sops-v{version}.checksums.txt" {
		t.Fatalf("unexpected checksums template: %q", got.Source.Checksums)
	}
}

func TestChecksumsName(t *testing.T) {
	tool := Tool{Version: "3.13.1", Source: Source{Checksums: "sops-v{version}.checksums.txt"}}
	if got, want := tool.ChecksumsName("amd64"), "sops-v3.13.1.checksums.txt"; got != want {
		t.Errorf("ChecksumsName = %q, want %q", got, want)
	}
	if got := (Tool{Version: "1.0.0"}).ChecksumsName("amd64"); got != "" {
		t.Errorf("ChecksumsName with no template = %q, want empty", got)
	}
}

// A tool listed twice is a catalog error, not two installs: the duplicate
// copilot entry of AI-038 (#1321) installed once and then reported "already
// installed; skipping" for its twin, which read as idempotence.
func TestLoad_RejectsDuplicateToolNames(t *testing.T) {
	dup := `{"tools":[{"name":"copilot","version":"1.0.81","profile":"full","source":{"type":"npm","package":"@github/copilot"}},{"name":"copilot","version":"1.0.81","profile":"full","source":{"type":"npm","package":"@github/copilot"}}]}`
	_, err := Load(writeCatalog(t, dup))
	if err == nil {
		t.Fatal("a duplicated tool name must be a load error")
	}
	if got := err.Error(); !strings.Contains(got, `tool "copilot" is listed more than once`) {
		t.Fatalf("error must name the tool, got: %v", err)
	}
}

func TestLoad_Errors(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected error for missing catalog")
	}
	if _, err := Load(writeCatalog(t, "{ not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAssetName(t *testing.T) {
	tool := Tool{
		Version: "3.13.1",
		Source: Source{Asset: map[string]string{
			"linux":   "sops-v{version}.linux.{goarch}",
			"darwin":  "sops-v{version}.darwin.{goarch}",
			"windows": "sops-v{version}.{goarch}.exe",
		}},
	}
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "sops-v3.13.1.linux.amd64"},
		{"linux", "arm64", "sops-v3.13.1.linux.arm64"},
		{"darwin", "arm64", "sops-v3.13.1.darwin.arm64"},
		{"windows", "amd64", "sops-v3.13.1.amd64.exe"}, // irregular: goarch before .exe, no OS token
		{"plan9", "amd64", ""},                         // unsupported OS → empty
	}
	for _, tc := range cases {
		if got := tool.AssetName(tc.goos, tc.goarch); got != tc.want {
			t.Errorf("AssetName(%q,%q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}
