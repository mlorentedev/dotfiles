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
		{"all persisted, record current → PASS", map[string]string{"DOTFILES_REPO_DIR": "*", "VAULT_PATH": "*", envpkg.ManagedMarker: "DOTFILES_REPO_DIR;VAULT_PATH"}, nil, StatusPass, "persisted at user scope"},
		// CLI-065: every variable in place but no ownership record yet (a box
		// that persisted before the record existed) — one run writes it.
		{"all persisted, no record yet → WARN naming the record", map[string]string{"DOTFILES_REPO_DIR": "*", "VAULT_PATH": "*"}, nil, StatusWarn, envpkg.ManagedMarker},
		{"one missing → WARN naming it", map[string]string{"DOTFILES_REPO_DIR": "*"}, nil, StatusWarn, "VAULT_PATH"},
		{"one different → WARN naming it", map[string]string{"DOTFILES_REPO_DIR": "*", "VAULT_PATH": `C:\elsewhere`}, nil, StatusWarn, "VAULT_PATH"},
		{"registry unreadable → WARN", nil, errors.New("access denied"), StatusWarn, "unreadable"},
		// CLI-065 (#1363): the marker still lists a name the contract retired.
		{"retired name still persisted → WARN naming it", map[string]string{"DOTFILES_REPO_DIR": "*", "VAULT_PATH": "*", "OLD_NAME": "x", envpkg.ManagedMarker: "DOTFILES_REPO_DIR;OLD_NAME;VAULT_PATH"}, nil, StatusWarn, "OLD_NAME"},
		// Review finding: drift and a retired name at once — both are named.
		{"drift and retired together → WARN naming the drift", map[string]string{"DOTFILES_REPO_DIR": "*", "OLD_NAME": "x", envpkg.ManagedMarker: "DOTFILES_REPO_DIR;OLD_NAME;VAULT_PATH"}, nil, StatusWarn, "VAULT_PATH"},
		{"drift and retired together → WARN naming the retired", map[string]string{"DOTFILES_REPO_DIR": "*", "OLD_NAME": "x", envpkg.ManagedMarker: "DOTFILES_REPO_DIR;OLD_NAME;VAULT_PATH"}, nil, StatusWarn, "OLD_NAME"},
		{"marker in sync → PASS", map[string]string{"DOTFILES_REPO_DIR": "*", "VAULT_PATH": "*", envpkg.ManagedMarker: "DOTFILES_REPO_DIR;VAULT_PATH"}, nil, StatusPass, "persisted at user scope"},
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

	// Review finding F2: a contract persist would refuse is reported, not
	// compared — doctor agrees with persist and --check.
	t.Run("reserved contract name → WARN naming the refusal", func(t *testing.T) {
		home := t.TempDir()
		mirror := filepath.Join(home, ".dotfiles")
		writeFile(t, filepath.Join(mirror, "env-contract.json"),
			`{"env_vars":[{"name":"`+envpkg.ManagedMarker+`","required":false,"default":{"linux":"x","windows":"x"}}]}`)
		sys := newSys(map[string]string{"HOME": home, "USERPROFILE": home}, nil, nil)
		sys.GOOS = "windows"
		sys.UserEnv = func(string) (string, bool, error) { return "", false, nil }
		var buf bytes.Buffer
		checkPersistedEnv(sys, &Config{DotfilesDir: mirror}, capture(&buf))
		if got := statusOfLine(buf.String(), "reserved"); got != StatusWarn {
			t.Fatalf("want a WARN naming the reserved name, got %q\n%s", tagOf(got), buf.String())
		}
	})

	// #2013 S3: on macOS the scope is the launchd session, which logout clears.
	// `dotf env persist` alone would restore it until the next logout; the
	// converge step also loads the login agent, so that is the remedy named.
	t.Run("darwin drift → WARN naming the converge step and the GUI apps", func(t *testing.T) {
		home := t.TempDir()
		mirror := filepath.Join(home, ".dotfiles")
		writeFile(t, filepath.Join(mirror, "env-contract.json"), persistContract)
		sys := newSys(map[string]string{"HOME": home}, nil, nil)
		sys.GOOS = "darwin"
		sys.UserEnv = func(string) (string, bool, error) { return "", false, nil }
		var buf bytes.Buffer
		checkPersistedEnv(sys, &Config{DotfilesDir: mirror}, capture(&buf))
		out := buf.String()
		if got := statusOfLine(out, "dotf converge --only env-persist"); got != StatusWarn {
			t.Fatalf("want a WARN naming the converge step, got %q\n%s", tagOf(got), out)
		}
		if statusOfLine(out, "launched from the Dock") != StatusWarn {
			t.Errorf("the WARN must say who reads the scope on macOS\n%s", out)
		}
	})

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
