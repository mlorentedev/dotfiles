package doctor

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

const persistContract = `{"env_vars":[
  {"name":"DOTFILES_REPO_DIR","required":false,"default":{"linux":"$HOME/Projects/dotfiles","windows":"$env:USERPROFILE\\Projects\\dotfiles"},"validation":"path_exists"},
  {"name":"VAULT_PATH","required":false,"default":{"linux":"$HOME/Projects/knowledge","windows":"$env:USERPROFILE\\Projects\\knowledge"},"validation":"path_exists"}
]}`

// The persisted-scope check (CLI-058, #1324) compares the resolved contract
// against the per-user persistent environment through the seam: all present
// and equal → PASS; missing or different → WARN naming them and the remedy;
// unreadable → WARN; no seam (an OS without the scope) → no section at all.
func TestCheckPersistedEnv_ByStatus(t *testing.T) {
	cases := []struct {
		name   string
		stored map[string]string // "*" = exactly the resolved value
		getErr error
		want   Status
		needle string
	}{
		{"all persisted → PASS", map[string]string{"DOTFILES_REPO_DIR": "*", "VAULT_PATH": "*"}, nil, StatusPass, "persisted at user scope"},
		{"one missing → WARN naming it", map[string]string{"DOTFILES_REPO_DIR": "*"}, nil, StatusWarn, "VAULT_PATH"},
		{"one different → WARN naming it", map[string]string{"DOTFILES_REPO_DIR": "*", "VAULT_PATH": `C:\elsewhere`}, nil, StatusWarn, "VAULT_PATH"},
		{"registry unreadable → WARN", nil, errors.New("access denied"), StatusWarn, "unreadable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			mirror := filepath.Join(home, ".dotfiles")
			contract := filepath.Join(mirror, "env-contract.json")
			writeFile(t, contract, persistContract)
			resolved, err := envpkg.ResolveVars(contract, envpkg.MachinePath(home), "windows", home)
			if err != nil {
				t.Fatal(err)
			}
			sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home}, nil, nil)
			sys.GOOS = "windows"
			sys.UserEnv = func(name string) (string, bool, error) {
				if tc.getErr != nil {
					return "", false, tc.getErr
				}
				v, ok := tc.stored[name]
				if !ok {
					return "", false, nil
				}
				if v == "*" {
					for _, rv := range resolved {
						if rv.Name == name {
							return rv.Value, true, nil
						}
					}
				}
				return v, true, nil
			}
			var buf bytes.Buffer
			rep := capture(&buf)

			checkPersistedEnv(sys, &Config{DotfilesDir: mirror}, rep)

			if got := statusOfLine(buf.String(), tc.needle); got != tc.want {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.want), buf.String())
			}
		})
	}

	t.Run("no seam → no section", func(t *testing.T) {
		sys := newSys(nil, nil, nil)
		sys.UserEnv = nil
		var buf bytes.Buffer
		checkPersistedEnv(sys, &Config{DotfilesDir: t.TempDir()}, capture(&buf))
		if buf.Len() != 0 {
			t.Fatalf("an OS without a per-user scope must print nothing, got\n%s", buf.String())
		}
	})
}
