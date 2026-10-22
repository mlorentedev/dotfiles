package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall_LegacyHiveShimAheadOfTheNewCopy(t *testing.T) {
	const legacy = "@echo off\r\n\"C:\\Users\\u\\AppData\\Local\\hive\\runtime\\current\\Scripts\\hive.exe\" %*\r\n"
	cases := []struct {
		name        string
		dir         string
		content     string
		shimLast    bool   // the shim sits after the working copy on PATH
		afterRemove string // what hive reports once the shim is gone
		wantErr     bool
		wantKept    bool
	}{
		{"legacy shim is retired and the install converges", "hive/bin", legacy, false, "4.4.2", false, false},
		{"a hive.cmd that is not the legacy launcher is kept", "hive/bin", "@echo off\r\nmy-own-hive %*\r\n", false, "4.4.2", true, true},
		{"a launcher outside hive/bin is kept", "tools", legacy, false, "4.4.2", true, true},
		{"a shim behind the working copy is kept", "hive/bin", legacy, true, "4.4.2", true, true},
		{"the shim is restored when the re-probe still fails", "hive/bin", legacy, false, "", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), filepath.FromSlash(tc.dir))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			shim := filepath.Join(dir, "hive.cmd")
			if err := os.WriteFile(shim, []byte(tc.content), 0o755); err != nil {
				t.Fatal(err)
			}
			var rec []string
			in := newNpmInstaller("", &rec, nil)
			in.CurrentVersion = func(string) string {
				if len(rec) == 0 {
					return ""
				}
				if _, err := os.Stat(shim); err == nil {
					return "4.2.1"
				}
				return tc.afterRemove
			}
			in.PathCopies = func(string) []string {
				copies := []string{`C:\uv\hive.exe`}
				if _, err := os.Stat(shim); err != nil {
					return copies
				}
				if tc.shimLast {
					return append(copies, shim)
				}
				return append([]string{shim}, copies...)
			}
			tool := Tool{Name: "hive", Version: "4.4.2", Source: Source{Type: "uv-tool", Package: "hive-vault"}}
			_, err := in.Install(tool)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Install error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "below the pin") {
				t.Errorf("error %q lost the original verdict", err)
			}
			_, statErr := os.Stat(shim)
			if kept := statErr == nil; kept != tc.wantKept {
				t.Errorf("shim kept = %v, want %v", kept, tc.wantKept)
			}
			if _, err := os.Stat(shim + ".retired"); err == nil {
				t.Errorf("backup %s.retired left behind", shim)
			}
		})
	}
}
