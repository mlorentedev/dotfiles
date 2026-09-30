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

// bwTool mirrors the packages.json npm entry under test.
func bwTool() Tool {
	return Tool{
		Name:    "bw",
		Version: "2026.5.0",
		Profile: "full",
		Source:  Source{Type: "npm", Package: "@bitwarden/cli"},
	}
}

// newNpmInstaller wires an Installer whose npm Run is recorded into rec and whose
// PATH version probe is faked via CurrentVersion (current = installed version,
// "" = absent). Dest is irrelevant — npm globals never touch it.
func newNpmInstaller(current string, rec *[]string, runErr error) *Installer {
	return &Installer{
		GOOS:           "linux",
		GOARCH:         "amd64",
		Dest:           "/unused",
		Out:            io.Discard,
		CurrentVersion: func(string) string { return current },
		Run: func(name string, args ...string) error {
			*rec = append(*rec, name+" "+strings.Join(args, " "))
			return runErr
		},
	}
}

func TestInstallNpm_Fresh(t *testing.T) {
	var rec []string
	in := newNpmInstaller("", &rec, nil)
	res, err := in.Install(bwTool())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res != Installed {
		t.Errorf("Result = %v, want Installed", res)
	}
	want := "npm install -g @bitwarden/cli@2026.5.0"
	if len(rec) != 1 || rec[0] != want {
		t.Errorf("Run calls = %v, want exactly [%q]", rec, want)
	}
}

func TestInstallNpm_UpgradeWhenBelowPin(t *testing.T) {
	var rec []string
	in := newNpmInstaller("2026.4.0", &rec, nil)
	res, err := in.Install(bwTool())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res != Upgraded {
		t.Errorf("Result = %v, want Upgraded", res)
	}
	if len(rec) != 1 {
		t.Errorf("expected one npm install call, got %v", rec)
	}
}

func TestInstallNpm_SkipWhenAtOrAbovePin(t *testing.T) {
	for _, current := range []string{"2026.5.0", "2026.6.0", "2027.0.0"} {
		t.Run(current, func(t *testing.T) {
			var rec []string
			in := newNpmInstaller(current, &rec, nil)
			res, err := in.Install(bwTool())
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			if res != Skipped {
				t.Errorf("Result = %v, want Skipped", res)
			}
			if len(rec) != 0 {
				t.Errorf("must not run npm when already at/above pin, got %v", rec)
			}
		})
	}
}

func TestInstallNpm_RunFailure(t *testing.T) {
	var rec []string
	in := newNpmInstaller("", &rec, fmt.Errorf("npm: command not found"))
	if _, err := in.Install(bwTool()); err == nil {
		t.Fatal("expected error when npm install fails")
	}
}

func TestInstallNpm_MissingPackage(t *testing.T) {
	var rec []string
	in := newNpmInstaller("", &rec, nil)
	tool := bwTool()
	tool.Source.Package = ""
	if _, err := in.Install(tool); err == nil {
		t.Fatal("expected error when npm source declares no package")
	}
	if len(rec) != 0 {
		t.Errorf("must not run npm with no package, got %v", rec)
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

// Plan runs the reconcile decision Install runs, and touches nothing: the fetch
// and npm seams fail the test if the plan reaches them.
func TestInstallerPlan(t *testing.T) {
	release := Tool{Name: "sops", Version: "3.13.1", Source: Source{
		Type: "github-release", Repo: "getsops/sops",
		Asset:     map[string]string{"linux": "sops-v{version}.linux.{goarch}"},
		Checksums: "sops-v{version}.checksums.txt",
	}}
	npm := Tool{Name: "bw", Version: "2026.9.0", Source: Source{Type: "npm", Package: "@bitwarden/cli"}}
	cases := []struct {
		name, goos, installed string
		tool                  Tool
		want                  PlanAction
	}{
		{"absent", "linux", "", release, PlanInstall},
		{"below the pin", "linux", "3.12.0", release, PlanUpgrade},
		{"at the pin", "linux", "3.13.1", release, PlanSkip},
		{"above the pin is never downgraded", "linux", "3.14.0", release, PlanSkip},
		{"no build for this platform", "windows", "", release, PlanUnsupported},
		{"npm below the pin", "linux", "2026.1.0", npm, PlanUpgrade},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &Installer{
				GOOS: tc.goos, GOARCH: "amd64", Dest: t.TempDir(),
				CurrentVersion: func(string) string { return tc.installed },
				Fetch: func(string, string) error {
					t.Fatal("a plan must not download")
					return nil
				},
				Run: func(string, ...string) error {
					t.Fatal("a plan must not run a package manager")
					return nil
				},
			}
			got := in.Plan(tc.tool)
			if got.Action != tc.want {
				t.Errorf("Plan(%s).Action = %q, want %q", tc.tool.Name, got.Action, tc.want)
			}
			if got.Installed != tc.installed || got.Pin != tc.tool.Version {
				t.Errorf("Plan(%s) = %+v, want installed %q pin %q", tc.tool.Name, got, tc.installed, tc.tool.Version)
			}
		})
	}
}
