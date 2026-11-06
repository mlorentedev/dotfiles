package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type archiveEntry struct {
	name string
	body []byte
	mode os.FileMode
	link string
}

func githubArchiveTool(asset, member string) Tool {
	return Tool{
		Name:    "ts-bridge",
		Version: "1.20.0",
		Profile: "full",
		Source: Source{
			Type:      "github-archive",
			Repo:      "mlorentedev/ts-bridge",
			Asset:     map[string]string{"windows": asset, "linux": asset},
			Checksums: "checksums.txt",
			Member:    map[string]string{"windows": member, "linux": member},
		},
	}
}

func TestInstallGitHubArchiveZip(t *testing.T) {
	const asset = "ts-bridge-v1.20.0-windows-amd64.zip"
	payload := []byte("windows-binary")
	in := archiveInstaller(t, "windows", asset, zipArchive(t, []archiveEntry{{name: "ts-bridge.exe", body: payload, mode: 0o755}}))

	if res, err := in.Install(githubArchiveTool(asset, "ts-bridge.exe")); err != nil || res != Installed {
		t.Fatalf("Install() = %v, %v", res, err)
	}
	if got, _ := os.ReadFile(filepath.Join(in.Dest, "ts-bridge.exe")); !bytes.Equal(got, payload) {
		t.Fatalf("placed bytes = %q", got)
	}
}

func TestInstallGitHubArchiveTarGz(t *testing.T) {
	const asset = "ts-bridge-v1.20.0-linux-amd64.tar.gz"
	payload := []byte("linux-binary")
	in := archiveInstaller(t, "linux", asset, tarGzArchive(t, []archiveEntry{{name: "ts-bridge", body: payload, mode: 0o755}}))

	if _, err := in.Install(githubArchiveTool(asset, "ts-bridge")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(in.Dest, "ts-bridge")); !bytes.Equal(got, payload) {
		t.Fatalf("placed bytes = %q", got)
	}
}

func TestInstallGitHubArchiveRejectsInvalidMembersAndKeepsInstalledBinary(t *testing.T) {
	cases := []struct {
		name, asset, member string
		body                []byte
	}{
		{"missing", "tool.zip", "ts-bridge.exe", zipArchive(t, []archiveEntry{{name: "other.exe", body: []byte("x"), mode: 0o755}})},
		{"duplicate", "tool.zip", "ts-bridge.exe", zipArchive(t, []archiveEntry{
			{name: "ts-bridge.exe", body: []byte("one"), mode: 0o755},
			{name: "ts-bridge.exe", body: []byte("two"), mode: 0o755},
		})},
		{"directory", "tool.zip", "ts-bridge.exe", zipArchive(t, []archiveEntry{{name: "ts-bridge.exe", mode: os.ModeDir | 0o755}})},
		{"symlink", "tool.tar.gz", "ts-bridge", tarGzArchive(t, []archiveEntry{{name: "ts-bridge", link: "other", mode: os.ModeSymlink | 0o777}})},
		{"unsupported", "tool.rar", "ts-bridge.exe", []byte("not-an-archive")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := archiveInstaller(t, "windows", tc.asset, tc.body)
			dst := filepath.Join(in.Dest, "ts-bridge.exe")
			if err := os.WriteFile(dst, []byte("working-old"), 0o755); err != nil {
				t.Fatal(err)
			}
			in.CurrentVersion = func(string) string { return "1.19.0" }

			if plan := in.Plan(githubArchiveTool(tc.asset, tc.member)); tc.name == "unsupported" &&
				plan.Action != PlanRefused {
				t.Fatalf("Plan() = %+v, want refused", plan)
			}
			if _, err := in.Install(githubArchiveTool(tc.asset, tc.member)); err == nil {
				t.Fatal("Install() accepted an invalid archive member")
			}
			if got, _ := os.ReadFile(dst); string(got) != "working-old" {
				t.Fatalf("failed install replaced destination with %q", got)
			}
		})
	}
}

func TestInstallGitHubArchiveRejectsChecksumBeforeExtraction(t *testing.T) {
	const asset = "tool.zip"
	body := []byte("not-a-zip")
	files := map[string][]byte{
		asset:           body,
		"checksums.txt": []byte(strings.Repeat("0", 64) + "  " + asset + "\n"),
	}
	in := newTestInstaller(t, "", func(url, dest string) error {
		name := url[strings.LastIndex(url, "/")+1:]
		return os.WriteFile(dest, files[name], 0o600)
	})
	in.GOOS = "windows"

	_, err := in.Install(githubArchiveTool(asset, "ts-bridge.exe"))
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") ||
		strings.Contains(err.Error(), "open zip") {
		t.Fatalf("Install() error = %v", err)
	}
}

func TestPlaceBinaryAtomicallyReplacesContentAndExecutableMode(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	dstDir := t.TempDir()
	dst := filepath.Join(dstDir, "tool")
	if err := os.WriteFile(src, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := placeBinary(src, dst); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "new" {
		t.Fatalf("destination = %q", got)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Fatalf("destination mode = %04o, want 0755", info.Mode().Perm())
		}
	}
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "tool" {
		t.Fatalf("staging files survived: %v", entries)
	}
}

func archiveInstaller(t *testing.T, goos, asset string, body []byte) *Installer {
	t.Helper()
	checksums := fmt.Sprintf("%s  %s\n", sha256hex(body), asset)
	files := map[string][]byte{asset: body, "checksums.txt": []byte(checksums)}
	in := newTestInstaller(t, "", func(url, dest string) error {
		name := url[strings.LastIndex(url, "/")+1:]
		data, ok := files[name]
		if !ok {
			return fmt.Errorf("404 %s", name)
		}
		return os.WriteFile(dest, data, 0o600)
	})
	in.GOOS = goos
	in.Probe = func(string, ...string) ([]byte, error) { return []byte("ts-bridge v1.20.0"), nil }
	return in
}

func zipArchive(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, entry := range entries {
		h := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		h.SetMode(entry.mode)
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func tarGzArchive(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		h := &tar.Header{Name: entry.name, Mode: int64(entry.mode.Perm()), Size: int64(len(entry.body))}
		switch {
		case entry.mode&os.ModeSymlink != 0:
			h.Typeflag, h.Linkname, h.Size = tar.TypeSymlink, entry.link, 0
		case entry.mode.IsDir():
			h.Typeflag, h.Size = tar.TypeDir, 0
		default:
			h.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(tw, bytes.NewReader(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
