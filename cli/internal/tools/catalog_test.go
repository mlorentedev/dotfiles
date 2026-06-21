package tools

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleCatalog = `{
  "tools": [
    {
      "name": "sops",
      "version": "3.9.0",
      "profile": "full",
      "source": {
        "type": "github-release",
        "repo": "getsops/sops",
        "asset": {
          "linux": "sops-v{version}.linux.{goarch}",
          "darwin": "sops-v{version}.darwin.{goarch}",
          "windows": "sops-v{version}.exe"
        }
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
	if got.Name != "sops" || got.Version != "3.9.0" || got.Profile != "full" {
		t.Fatalf("unexpected tool: %+v", got)
	}
	if got.Source.Type != "github-release" || got.Source.Repo != "getsops/sops" {
		t.Fatalf("unexpected source: %+v", got.Source)
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
		Version: "3.9.0",
		Source: Source{Asset: map[string]string{
			"linux":   "sops-v{version}.linux.{goarch}",
			"darwin":  "sops-v{version}.darwin.{goarch}",
			"windows": "sops-v{version}.exe",
		}},
	}
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "sops-v3.9.0.linux.amd64"},
		{"linux", "arm64", "sops-v3.9.0.linux.arm64"},
		{"darwin", "arm64", "sops-v3.9.0.darwin.arm64"},
		{"windows", "amd64", "sops-v3.9.0.exe"}, // irregular: no goos/goarch in the name
		{"plan9", "amd64", ""},                  // unsupported OS → empty
	}
	for _, tc := range cases {
		if got := tool.AssetName(tc.goos, tc.goarch); got != tc.want {
			t.Errorf("AssetName(%q,%q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}
