package tools

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The asset, member and checksums names install.sh and install.ps1 download;
// a drift here fetches a file the release does not have.
func TestDotfRelease(t *testing.T) {
	r := DotfRelease("0.80.0")
	for _, c := range []struct{ goos, goarch, asset, member string }{
		{"linux", "amd64", "dotf_0.80.0_linux_amd64.tar.gz", "dotf"},
		{"darwin", "arm64", "dotf_0.80.0_darwin_arm64.tar.gz", "dotf"},
		{"windows", "amd64", "dotf_0.80.0_windows_amd64.zip", "dotf.exe"},
		{"windows", "arm64", "dotf_0.80.0_windows_arm64.zip", "dotf.exe"},
	} {
		if got := r.AssetName(c.goos, c.goarch); got != c.asset {
			t.Errorf("%s/%s asset = %q, want %q", c.goos, c.goarch, got, c.asset)
		}
		if got := r.MemberName(c.goos, c.goarch); got != c.member {
			t.Errorf("%s/%s member = %q, want %q", c.goos, c.goarch, got, c.member)
		}
	}
	if got := r.ChecksumsName("amd64"); got != "checksums.txt" {
		t.Errorf("checksums = %q, want checksums.txt", got)
	}
	if r.Name != "dotf" || r.Source.Repo != "mlorentedev/dotfiles" {
		t.Errorf("release = %s from %s", r.Name, r.Source.Repo)
	}
}

// selfInstaller serves a dotf 0.80.0 release for goos/amd64 whose payload is
// body, with checksums.txt listing sum for the asset ("" = the real sum,
// "absent" = no entry), and a probe answering probe.
func selfInstaller(t *testing.T, goos string, body []byte, sum, probe string) *Installer {
	t.Helper()
	r := DotfRelease("0.80.0")
	asset, member := r.AssetName(goos, "amd64"), r.MemberName(goos, "amd64")
	entries := []archiveEntry{{name: member, mode: 0o755, body: body}}
	archive := tarGzArchive(t, entries)
	if goos == "windows" {
		archive = zipArchive(t, entries)
	}
	switch sum {
	case "":
		sum = fmt.Sprintf("%s  %s\n", sha256hex(archive), asset)
	case "absent":
		sum = fmt.Sprintf("%s  dotf_0.80.0_plan9_amd64.tar.gz\n", sha256hex(archive))
	default:
		sum = fmt.Sprintf("%s  %s\n", sum, asset)
	}
	files := map[string][]byte{asset: archive, "checksums.txt": []byte(sum)}
	in := newTestInstaller(t, "", func(url, dest string) error {
		if !strings.Contains(url, "/mlorentedev/dotfiles/releases/download/v0.80.0/") {
			return fmt.Errorf("unexpected URL %s", url)
		}
		data, ok := files[url[strings.LastIndex(url, "/")+1:]]
		if !ok {
			return fmt.Errorf("404 %s", url)
		}
		return os.WriteFile(dest, data, 0o600)
	})
	in.GOOS = goos
	in.Probe = func(string, ...string) ([]byte, error) {
		if probe == "" {
			return nil, errors.New("exec format error")
		}
		return []byte(probe), nil
	}
	return in
}

// target writes the "running" dotf at dir/<name> and returns its path.
func target(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("running 0.79.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// CLI-133 AC3: a verified release replaces the binary at the target path, on
// each OS's archive format.
func TestInstallSelf(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			dst := target(t, binFilename("dotf", goos))
			in := selfInstaller(t, goos, []byte("dotf 0.80.0 payload"), "", "dotf version 0.80.0")
			if err := in.InstallSelf(DotfRelease("0.80.0"), dst); err != nil {
				t.Fatalf("InstallSelf: %v", err)
			}
			if got, _ := os.ReadFile(dst); !bytes.Equal(got, []byte("dotf 0.80.0 payload")) { //nolint:gosec // test path
				t.Errorf("target holds %q, want the release payload", got)
			}
		})
	}
}

// CLI-133 AC3: every way the release can fail verification leaves the running
// binary byte-identical.
func TestInstallSelf_RefusalsLeaveTheTargetUntouched(t *testing.T) {
	for _, c := range []struct {
		name, file, sum, probe, want string
	}{
		{"checksum mismatch", "dotf", strings.Repeat("0", 64), "dotf version 0.80.0", "checksum mismatch"},
		{"asset not in checksums.txt", "dotf", "absent", "dotf version 0.80.0", "not listed"},
		{"staged binary does not run", "dotf", "", "", "does not run"},
		{"staged binary is another version", "dotf", "", "dotf version 0.79.0", "below the pin"},
		{"target is not named dotf", "dotf-dev", "", "dotf version 0.80.0", "not named dotf"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dst := target(t, c.file)
			in := selfInstaller(t, "linux", []byte("dotf 0.80.0 payload"), c.sum, c.probe)
			err := in.InstallSelf(DotfRelease("0.80.0"), dst)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to contain %q", err, c.want)
			}
			if got, _ := os.ReadFile(dst); !bytes.Equal(got, []byte("running 0.79.0")) { //nolint:gosec // test path
				t.Errorf("target changed to %q", got)
			}
			entries, _ := os.ReadDir(filepath.Dir(dst))
			if len(entries) != 1 {
				t.Errorf("leftovers beside the target: %v", entries)
			}
		})
	}
}
