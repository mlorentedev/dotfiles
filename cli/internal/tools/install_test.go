package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sopsTool is the catalog entry under test (mirrors packages.json).
func sopsTool() Tool {
	return Tool{
		Name:    "sops",
		Version: "3.13.1",
		Profile: "full",
		Source: Source{
			Type: "github-release",
			Repo: "getsops/sops",
			Asset: map[string]string{
				"linux":   "sops-v{version}.linux.{goarch}",
				"darwin":  "sops-v{version}.darwin.{goarch}",
				"windows": "sops-v{version}.{goarch}.exe",
			},
			Checksums: "sops-v{version}.checksums.txt",
		},
	}
}

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// fakeRelease builds a Fetcher serving the given files (filename -> bytes) and a
// checksums.txt computed over them, with `corrupt` letting a test inject a wrong
// hash for one asset. A URL whose basename is unknown returns an error — the
// offline/404 case.
func fakeRelease(t *testing.T, asset string, binary []byte, corrupt bool) Fetcher {
	t.Helper()
	hash := sha256hex(binary)
	if corrupt {
		hash = strings.Repeat("0", 64)
	}
	checksums := fmt.Sprintf("%s  %s\n%s  some-other-asset\n", hash, asset, sha256hex([]byte("x")))
	files := map[string][]byte{
		asset:                        binary,
		"sops-v3.13.1.checksums.txt": []byte(checksums),
	}
	return func(url, dest string) error {
		base := url[strings.LastIndex(url, "/")+1:]
		content, ok := files[base]
		if !ok {
			return fmt.Errorf("404 fetching %s", url)
		}
		return os.WriteFile(dest, content, 0o600)
	}
}

// newTestInstaller wires an Installer for linux/amd64 with the given current
// version (empty = absent) and fetcher, writing into a temp dest dir.
func newTestInstaller(t *testing.T, current string, fetch Fetcher) *Installer {
	t.Helper()
	return &Installer{
		GOOS:           "linux",
		GOARCH:         "amd64",
		Dest:           t.TempDir(),
		BaseURL:        "https://example.test",
		Fetch:          fetch,
		Out:            io.Discard,
		CurrentVersion: func(string) string { return current },
	}
}

func TestInstall_Fresh(t *testing.T) {
	tool := sopsTool()
	binary := []byte("fake-sops-3.13.1-elf")
	in := newTestInstaller(t, "", fakeRelease(t, "sops-v3.13.1.linux.amd64", binary, false))

	res, err := in.Install(tool)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res != Installed {
		t.Errorf("Result = %v, want Installed", res)
	}
	placed := filepath.Join(in.Dest, "sops")
	got, err := os.ReadFile(placed)
	if err != nil {
		t.Fatalf("read placed binary: %v", err)
	}
	if string(got) != string(binary) {
		t.Errorf("placed content = %q, want %q", got, binary)
	}
	info, err := os.Stat(placed)
	if err != nil {
		t.Fatal(err)
	}
	// The exec bit is only representable on a POSIX host filesystem; NTFS drops
	// it, so assert it only when the test host itself is non-Windows.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
		t.Errorf("binary not executable: mode %v", info.Mode())
	}
}

func TestInstall_ChecksumMismatch(t *testing.T) {
	in := newTestInstaller(t, "", fakeRelease(t, "sops-v3.13.1.linux.amd64", []byte("payload"), true))
	if _, err := in.Install(sopsTool()); err == nil {
		t.Fatal("expected checksum-mismatch error")
	}
	if _, err := os.Stat(filepath.Join(in.Dest, "sops")); !os.IsNotExist(err) {
		t.Error("binary should not be placed on checksum mismatch")
	}
}

func TestInstall_MissingChecksumEntry(t *testing.T) {
	// Asset that the checksums file does not list.
	tool := sopsTool()
	tool.Source.Asset["linux"] = "sops-v{version}.linux.unlisted"
	in := newTestInstaller(t, "", fakeRelease(t, "sops-v3.13.1.linux.amd64", []byte("payload"), false))
	if _, err := in.Install(tool); err == nil {
		t.Fatal("expected error when asset is absent from checksums.txt")
	}
}

func TestInstall_DownloadFailure(t *testing.T) {
	// Fetcher that always 404s — the offline / missing-release case.
	fetch := func(url, dest string) error { return fmt.Errorf("offline: %s", url) }
	in := newTestInstaller(t, "", fetch)
	if _, err := in.Install(sopsTool()); err == nil {
		t.Fatal("expected error on download failure")
	}
}

func TestInstall_UnsupportedOS(t *testing.T) {
	in := newTestInstaller(t, "", fakeRelease(t, "x", []byte("x"), false))
	in.GOOS = "plan9"
	if _, err := in.Install(sopsTool()); err == nil {
		t.Fatal("expected error for an OS with no declared asset")
	}
}

func TestInstall_SkipWhenAtOrAbovePin(t *testing.T) {
	for _, current := range []string{"3.13.1", "3.14.0", "4.0.0"} {
		t.Run(current, func(t *testing.T) {
			fetched := false
			fetch := func(url, dest string) error { fetched = true; return fmt.Errorf("should not fetch") }
			in := newTestInstaller(t, current, fetch)
			res, err := in.Install(sopsTool())
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			if res != Skipped {
				t.Errorf("Result = %v, want Skipped", res)
			}
			if fetched {
				t.Error("must not download when already at/above pin")
			}
		})
	}
}

func TestInstall_UpgradeWhenBelowPin(t *testing.T) {
	binary := []byte("fake-sops-3.13.1")
	in := newTestInstaller(t, "3.12.0", fakeRelease(t, "sops-v3.13.1.linux.amd64", binary, false))
	res, err := in.Install(sopsTool())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res != Upgraded {
		t.Errorf("Result = %v, want Upgraded", res)
	}
	got, _ := os.ReadFile(filepath.Join(in.Dest, "sops"))
	if string(got) != string(binary) {
		t.Errorf("upgraded content = %q, want %q", got, binary)
	}
}

func TestInstall_WindowsBinaryName(t *testing.T) {
	binary := []byte("fake-sops.exe")
	in := newTestInstaller(t, "", fakeRelease(t, "sops-v3.13.1.amd64.exe", binary, false))
	in.GOOS = "windows"
	if _, err := in.Install(sopsTool()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(in.Dest, "sops.exe")); err != nil {
		t.Errorf("windows binary should be placed as sops.exe: %v", err)
	}
}

func TestDecideAction(t *testing.T) {
	cases := []struct {
		installed, pin string
		want           action
	}{
		{"", "3.13.1", actionInstall},        // absent
		{"3.12.0", "3.13.1", actionUpgrade},  // below pin
		{"3.13.1", "3.13.1", actionSkip},     // exact match
		{"3.14.0", "3.13.1", actionSkip},     // newer — never downgrade
		{"3.13", "3.13.1", actionUpgrade},    // 3.13.0 < 3.13.1
		{"garbage", "3.13.1", actionUpgrade}, // unparseable parses to 0 → below pin
	}
	for _, tc := range cases {
		if got := decideAction(tc.installed, tc.pin); got != tc.want {
			t.Errorf("decideAction(%q,%q) = %v, want %v", tc.installed, tc.pin, got, tc.want)
		}
	}
}
