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
		file        string
		content     string
		wantErr     bool
		wantRemoved bool
	}{
		{"legacy shim is retired and the install converges", "hive.cmd", legacy, false, true},
		{"a hive.cmd that is not the legacy launcher is kept", "hive.cmd", "@echo off\r\nmy-own-hive %*\r\n", true, false},
		{"a differently named shadow is kept", "hive.bat", legacy, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shim := filepath.Join(t.TempDir(), tc.file)
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
				return "4.4.2"
			}
			in.PathCopies = func(string) []string {
				if _, err := os.Stat(shim); err == nil {
					return []string{shim, `C:\uv\hive.exe`}
				}
				return []string{`C:\uv\hive.exe`}
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
			if removed := os.IsNotExist(statErr); removed != tc.wantRemoved {
				t.Errorf("shim removed = %v, want %v", removed, tc.wantRemoved)
			}
		})
	}
}
