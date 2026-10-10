package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
// version (empty = absent) and fetcher, writing into a temp dest dir. The staged
// binary's post-checksum probe answers with the sops pin, so a happy-path test
// models a release binary that executes; the probe tests override Probe.
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
		Probe:          func(string, ...string) ([]byte, error) { return []byte("sops 3.13.1 (latest)\n"), nil },
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
	if left, _ := os.ReadDir(in.Dest); len(left) != 1 {
		t.Errorf("Dest holds %d entries, want only the placed binary (the stage dir must be removed)", len(left))
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

// A github-release tool with no asset for this OS/arch is not installable here,
// which Plan reports as "unsupported". Install agrees: a skip that says so, not
// a failure, so `dotf tools install` and `--dry-run` cannot disagree, and a
// catalog entry for one platform never fails the others.
func TestInstall_NoAssetForThisPlatformIsSkippedNotFailed(t *testing.T) {
	in := newTestInstaller(t, "", fakeRelease(t, "x", []byte("x"), false))
	in.GOOS = "plan9"
	var out strings.Builder
	in.Out = &out

	res, err := in.Install(sopsTool())
	if err != nil || res != Skipped {
		t.Fatalf("want a skip without an error, got %v, %v", res, err)
	}
	if !strings.Contains(out.String(), "no release asset for plan9/amd64") {
		t.Errorf("the skip must say why: %q", out.String())
	}
	if entries, _ := os.ReadDir(in.Dest); len(entries) != 0 {
		t.Errorf("a skipped tool placed files: %v", entries)
	}
}

// Plan and Install read the same tool the same way: no asset for the platform
// is "unsupported" to the plan and a skip to the install, never an install
// that then fails or a plan that promises one.
func TestPlanAndInstallAgreeOnAPlatformWithNoAsset(t *testing.T) {
	in := newTestInstaller(t, "", fakeRelease(t, "x", []byte("x"), false))
	in.GOOS = "plan9"

	if p := in.Plan(sopsTool()); p.Action != PlanUnsupported {
		t.Errorf("plan: want %q, got %q", PlanUnsupported, p.Action)
	}
	if res, err := in.Install(sopsTool()); err != nil || res != Skipped {
		t.Errorf("install: want a skip without an error, got %v, %v", res, err)
	}
}

func TestExpectedChecksum_AcceptsSha256sumNameForms(t *testing.T) {
	sums := filepath.Join(t.TempDir(), "SHASUMS256.txt")
	content := "aaa  ./mise-v1-linux-x64\nbbb *mise-v1-macos-arm64\nccc  plain-asset\n"
	if err := os.WriteFile(sums, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for asset, want := range map[string]string{"mise-v1-linux-x64": "aaa", "mise-v1-macos-arm64": "bbb", "plain-asset": "ccc"} {
		got, err := expectedChecksum(sums, asset)
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", asset, got, err, want)
		}
	}
	if _, err := expectedChecksum(sums, "linux-x64"); err == nil {
		t.Error("a name must match whole, not as a suffix of a listed path")
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
// "" = absent). Dest's parent is the npm prefix on Linux and macOS. A successful Run
// installs the version its argv names, as npm and uv do, so the post-install
// probe sees the tool on PATH; the probe tests override CurrentVersion.
func newNpmInstaller(current string, rec *[]string, runErr error) *Installer {
	return &Installer{
		GOOS:           "linux",
		GOARCH:         "amd64",
		Dest:           "/home/u/.local/bin",
		Out:            io.Discard,
		CurrentVersion: func(string) string { return current },
		Run: func(name string, args ...string) error {
			*rec = append(*rec, name+" "+strings.Join(args, " "))
			if runErr == nil {
				spec := args[len(args)-1]
				current = spec[strings.LastIndexAny(spec, "@=")+1:]
			}
			return runErr
		},
		HasCommand: func(string) bool { return true },
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
	// filepath, not a literal: the test also runs on a Windows host, where
	// filepath.Dir turns /home/u/.local/bin into \home\u\.local.
	want := "npm install -g --prefix " + filepath.FromSlash("/home/u/.local") + " @bitwarden/cli@2026.5.0"
	if len(rec) != 1 || rec[0] != want {
		t.Errorf("Run calls = %v, want exactly [%q]", rec, want)
	}
}

// Windows keeps npm's default prefix, %APPDATA%\npm: it is user-owned and on
// PATH there, and --prefix would place the shims at the prefix root instead.
func TestInstallNpm_WindowsKeepsTheDefaultPrefix(t *testing.T) {
	var rec []string
	in := newNpmInstaller("", &rec, nil)
	in.GOOS = "windows"
	if _, err := in.Install(bwTool()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if want := "npm install -g @bitwarden/cli@2026.5.0"; len(rec) != 1 || rec[0] != want {
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
		{"no build for this platform, but installed: the probe still runs", "windows", "3.13.1", release, PlanUnsupported},
		{"a source type Install skips with a warning", "linux", "", Tool{Name: "x", Version: "1.0.0", Source: Source{Type: "homebrew"}}, PlanSkip},
		{"npm below the pin", "linux", "2026.1.0", npm, PlanUpgrade},
		{"uv-tool below the pin", "linux", "4.1.0", hiveTool(), PlanUpgrade},
		{"uv-tool at the pin", "darwin", "4.2.2", hiveTool(), PlanSkip},
		{"a platform the tool does not list", "windows", "", hiveTool(), PlanUnsupported},
		// #1892: what Install refuses, the plan refuses, never install/upgrade/skip.
		{"a release with no checksums file", "linux", "", noSums(release), PlanRefused},
		{"a release with no checksums file, already at the pin", "linux", "3.13.1", noSums(release), PlanRefused},
		{"a release with no checksums file and no build here is only unsupported", "windows", "", noSums(release), PlanUnsupported},
		{"an npm source with no package", "linux", "", noPackage(npm), PlanRefused},
		{"a uv-tool source with no package", "linux", "", noPackage(hiveTool()), PlanRefused},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &Installer{
				GOOS: tc.goos, GOARCH: "amd64", Dest: t.TempDir(),
				CurrentVersion: func(string) string { return tc.installed },
				HasCommand:     func(string) bool { return true },
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

func noSums(t Tool) Tool {
	t.Source.Checksums = ""
	return t
}

func noPackage(t Tool) Tool {
	t.Source.Package = ""
	return t
}

// Plan and Install run one refusal check (#1892): every entry Plan refuses,
// Install fails on with the same reason, before it fetches or runs anything.
func TestPlanAndInstallRefuseTheSameEntries(t *testing.T) {
	release := Tool{Name: "sops", Version: "3.13.1", Source: Source{
		Type: "github-release", Repo: "getsops/sops",
		Asset: map[string]string{"linux": "sops-v{version}.linux.{goarch}"},
	}}
	for _, tool := range []Tool{release, {Name: "bw", Version: "2026.9.0", Source: Source{Type: "npm"}}, noPackage(hiveTool())} {
		t.Run(tool.Name, func(t *testing.T) {
			in := &Installer{
				GOOS: "linux", GOARCH: "amd64", Dest: t.TempDir(), Out: io.Discard,
				CurrentVersion: func(string) string { return "" },
				HasCommand:     func(string) bool { return true },
				Fetch:          func(string, string) error { t.Fatal("a refused entry must not download"); return nil },
				Run:            func(string, ...string) error { t.Fatal("a refused entry must not run a package manager"); return nil },
			}
			p := in.Plan(tool)
			_, err := in.Install(tool)
			if p.Action != PlanRefused || err == nil {
				t.Fatalf("Plan = %+v, Install err = %v: want both to refuse", p, err)
			}
			if p.Note == "" || !strings.HasSuffix(err.Error(), p.Note) {
				t.Errorf("Plan's note %q must be Install's reason %q", p.Note, err)
			}
		})
	}
}

// An unsupported row says which of its two causes it is.
func TestPlanSaysWhyAToolIsUnsupported(t *testing.T) {
	release := Tool{Name: "sops", Version: "3.13.1", Source: Source{
		Type: "github-release", Asset: map[string]string{"linux": "sops"}, Checksums: "sums",
	}}
	in := &Installer{GOOS: "windows", GOARCH: "arm64", Dest: t.TempDir(), CurrentVersion: func(string) string { return "" }}
	if got := in.Plan(release).Note; got != "no release asset for windows/arm64" {
		t.Errorf("no-asset note = %q", got)
	}
	if got := in.Plan(hiveTool()).Note; got != "not installed on windows by this catalog" {
		t.Errorf("unlisted-platform note = %q", got)
	}
}

// TestInstall_DefaultProbeKeepsAVersionPrintedBeforeAFailingExit pins the
// installer to ProbeVersion's rule. A tool that prints its version and then
// exits non-zero is at the pin: `dotf tools version` and the doctor already see
// it that way, and an installer that saw nothing would reinstall on every run.
func TestInstall_DefaultProbeKeepsAVersionPrintedBeforeAFailingExit(t *testing.T) {
	var probed []string
	probe := func(tool Tool) Runner {
		return func(name string, args ...string) ([]byte, error) {
			probed = append(probed, name)
			return []byte(tool.Name + " " + tool.Version + "\nwarning: unrelated\n"), errors.New("exit status 1")
		}
	}

	t.Run("npm, probed on PATH", func(t *testing.T) {
		var rec []string
		in := newNpmInstaller("", &rec, nil)
		in.CurrentVersion = nil
		in.Probe = probe(bwTool())
		res, err := in.Install(bwTool())
		if err != nil || res != Skipped || len(rec) != 0 {
			t.Errorf("Install = %v, %v, npm calls %v; want Skipped with no npm call", res, err, rec)
		}
	})

	// A uv tool lives on PATH like an npm global, not in Dest. Probing Dest would
	// read it as absent and reinstall it on every run.
	t.Run("uv-tool, probed on PATH", func(t *testing.T) {
		var rec []string
		in := newNpmInstaller("", &rec, nil)
		in.CurrentVersion = nil
		in.Probe = probe(hiveTool())
		probed = nil
		res, err := in.Install(hiveTool())
		if err != nil || res != Skipped || len(rec) != 0 {
			t.Errorf("Install = %v, %v, uv calls %v; want Skipped with no uv call", res, err, rec)
		}
		if len(probed) != 1 || probed[0] != "hive" {
			t.Errorf("probed %v, want hive on PATH", probed)
		}
	})

	t.Run("github-release, probed in Dest", func(t *testing.T) {
		in := newTestInstaller(t, "", func(url, _ string) error { return fmt.Errorf("unexpected download %s", url) })
		in.CurrentVersion = nil
		in.Probe = probe(sopsTool())
		bin := filepath.Join(in.Dest, "sops")
		if err := os.WriteFile(bin, []byte("placed"), 0o600); err != nil {
			t.Fatal(err)
		}
		probed = nil
		res, err := in.Install(sopsTool())
		if err != nil || res != Skipped {
			t.Errorf("Install = %v, %v; want Skipped", res, err)
		}
		if len(probed) != 1 || probed[0] != bin {
			t.Errorf("probed %v, want the binary in Dest (%s)", probed, bin)
		}
	})
}

func hiveTool() Tool {
	return Tool{Name: "hive", Version: "4.2.2", Profile: "full", Source: Source{
		Type: "uv-tool", Package: "hive-vault", Platforms: []string{"linux", "darwin"},
	}}
}

// uv replaces a different installed version with the pinned one, in either
// direction, and exits 0 when the pin is already installed (measured with uv
// 0.9.29), so one argv covers install and upgrade. --force lets it replace an
// entry point another installer left in ~/.local/bin, which uv otherwise
// refuses with exit 2; decideAction runs it only below the pin or when absent.
func TestInstallUvTool(t *testing.T) {
	const want = "uv tool install --force hive-vault==4.2.2"
	cases := []struct {
		name, current string
		want          Result
		calls         int
	}{
		{"absent", "", Installed, 1},
		{"below the pin", "4.1.0", Upgraded, 1},
		{"at the pin", "4.2.2", Skipped, 0},
		{"above the pin is never downgraded", "4.3.0", Skipped, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec []string
			res, err := newNpmInstaller(tc.current, &rec, nil).Install(hiveTool())
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			if res != tc.want || len(rec) != tc.calls {
				t.Fatalf("Install = %v with calls %v, want %v with %d call(s)", res, rec, tc.want, tc.calls)
			}
			if tc.calls == 1 && rec[0] != want {
				t.Errorf("Run = %q, want %q", rec[0], want)
			}
		})
	}
}

func TestInstallUvTool_Failures(t *testing.T) {
	var rec []string
	_, err := newNpmInstaller("", &rec, errors.New("exit status 2")).Install(hiveTool())
	if err == nil || !strings.Contains(err.Error(), "uv tool install --force hive-vault==4.2.2") {
		t.Errorf("a failed uv run returned %v, want the argv named", err)
	}
	noPkg := hiveTool()
	noPkg.Source.Package = ""
	rec = nil
	if _, err := newNpmInstaller("", &rec, nil).Install(noPkg); err == nil || len(rec) != 0 {
		t.Errorf("a uv-tool without a package returned %v and ran %v, want an error and no run", err, rec)
	}
}

// A tool whose platforms exclude this OS is not a failure: hive is a uv tool on
// POSIX only, and Windows installs it through hive's own layout (hive ADR-019).
// Counting it as failed would turn every Windows `dotf tools install` red.
func TestInstall_PlatformsGateIsASkipNotAFailure(t *testing.T) {
	var rec []string
	in := newNpmInstaller("", &rec, nil)
	in.GOOS = "windows"
	res, err := in.Install(hiveTool())
	if err != nil || res != Skipped || len(rec) != 0 {
		t.Errorf("Install on windows = %v, %v, calls %v; want Skipped, nil, none", res, err, rec)
	}
	in.GOOS = "darwin"
	if res, err := in.Install(hiveTool()); err != nil || res != Installed {
		t.Errorf("Install on darwin = %v, %v; want Installed", res, err)
	}
}

// The first catalog pass on a fresh machine finds no uv (mise installs it in
// the sync that follows) and no npm (nothing installs node yet). Each is a named
// skip, not a failure, and the plan says what the apply does, so a converge
// probe does not read the wait as an unfinished install.
func TestInstall_AMissingManagerIsANamedSkip(t *testing.T) {
	bw := Tool{Name: "bw", Version: "2026.9.0", Source: Source{Type: "npm", Package: "@bitwarden/cli"}}
	for _, tc := range []struct {
		tool    Tool
		manager string
	}{{hiveTool(), "uv"}, {bw, "npm"}} {
		t.Run(tc.manager, func(t *testing.T) {
			var rec []string
			var out strings.Builder
			in := newNpmInstaller("", &rec, nil)
			in.Out = &out
			in.HasCommand = func(name string) bool { return name != tc.manager }
			res, err := in.Install(tc.tool)
			if err != nil || res != Skipped || len(rec) != 0 {
				t.Fatalf("Install without %s = %v, %v, calls %v; want Skipped, nil, none", tc.manager, res, err, rec)
			}
			if !strings.Contains(out.String(), tc.manager+" is not on PATH") {
				t.Errorf("the skip does not name %s:\n%s", tc.manager, out.String())
			}
			if p := in.Plan(tc.tool); p.Action != PlanMissingManager || p.Note != "waits on "+tc.manager {
				t.Errorf("Plan without %s = %+v, want %q naming it", tc.manager, p, PlanMissingManager)
			}
			in.CurrentVersion = func(string) string { return tc.tool.Version }
			if got := in.Plan(tc.tool).Action; got != PlanSkip {
				t.Errorf("Plan at the pin without %s = %q, want %q: an installed tool needs no manager", tc.manager, got, PlanSkip)
			}
		})
	}
}

// PLAT-001a W1: an install is a success only once the tool executes. Setup used
// to log SUCCESS for a linux-amd64 binary placed on darwin/arm64 (an ELF that
// cannot exec), which then shadowed a working copy on PATH. The staged binary is
// probed after the checksum gate and before it is placed, so a binary that does
// not run, or runs an older version than the pin, never reaches Dest.
func TestInstall_StagedBinaryMustExecuteAtThePin(t *testing.T) {
	cases := []struct {
		name   string
		probe  Runner
		reason string
	}{
		{"exec format error", func(string, ...string) ([]byte, error) {
			return nil, errors.New("fork/exec: exec format error")
		}, "does not run"},
		{"runs but prints no version", func(string, ...string) ([]byte, error) {
			return []byte("usage: sops [options]\n"), nil
		}, "does not run"},
		{"runs below the pin", func(string, ...string) ([]byte, error) {
			return []byte("sops 3.12.0\n"), nil
		}, "below the pin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := newTestInstaller(t, "", fakeRelease(t, "sops-v3.13.1.linux.amd64", []byte("elf"), false))
			var staged string
			in.Probe = func(name string, args ...string) ([]byte, error) {
				staged = name
				return tc.probe(name, args...)
			}
			res, err := in.Install(sopsTool())
			if err == nil || res != Skipped {
				t.Fatalf("Install = %v, %v; want Skipped and an error", res, err)
			}
			if !strings.Contains(err.Error(), "sops") || !strings.Contains(err.Error(), tc.reason) {
				t.Errorf("error %q does not name the tool and %q", err, tc.reason)
			}
			if _, err := os.Stat(filepath.Join(in.Dest, "sops")); !os.IsNotExist(err) {
				t.Error("a binary that failed its probe must not be placed in Dest")
			}
			// Staged on Dest's own mount (a noexec /tmp must not refuse a binary
			// that runs from Dest), never at the final path, and cleaned up.
			if filepath.Dir(filepath.Dir(staged)) != in.Dest || staged == filepath.Join(in.Dest, "sops") {
				t.Errorf("probed %q, want a staged copy in a hidden dir inside Dest", staged)
			}
			if left, _ := os.ReadDir(in.Dest); len(left) != 0 {
				t.Errorf("Dest holds %d leftover entries after a refused install", len(left))
			}
		})
	}
}

// npm and uv exit 0 and still leave nothing runnable when their global bin dir
// is not on PATH (an nvm/prefix mismatch), or when an older copy shadows the new
// one. Both used to read as "installed".
func TestInstall_PackageManagerToolMustRunOnPathAfterward(t *testing.T) {
	cases := []struct {
		name   string
		tool   Tool
		after  string
		reason string
	}{
		{"npm, not on PATH", bwTool(), "", "does not run on PATH"},
		{"npm, shadowed by an older copy", bwTool(), "2026.1.0", "below the pin"},
		{"uv-tool, not on PATH", hiveTool(), "", "does not run on PATH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec []string
			in := newNpmInstaller("", &rec, nil)
			in.CurrentVersion = func(string) string {
				if len(rec) == 0 {
					return ""
				}
				return tc.after
			}
			res, err := in.Install(tc.tool)
			if len(rec) != 1 {
				t.Fatalf("package manager calls = %v, want one", rec)
			}
			if err == nil || res != Skipped {
				t.Fatalf("Install = %v, %v; want Skipped and an error", res, err)
			}
			if !strings.Contains(err.Error(), tc.tool.Name) || !strings.Contains(err.Error(), tc.reason) {
				t.Errorf("error %q does not name %s and %q", err, tc.tool.Name, tc.reason)
			}
		})
	}
}
