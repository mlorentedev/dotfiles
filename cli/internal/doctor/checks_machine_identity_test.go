package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

// #1236 AC3 / #1547: a machine with no declared identity is reported before a
// dispatch has to discover it, and the row names the command that fixes it.
func TestCheckMachineIdentity_ByState(t *testing.T) {
	cases := []struct {
		name   string
		body   string // empty: no machine.json at all
		want   Status
		needle string
	}{
		{name: "no machine.json → WARN naming the verb", want: StatusWarn, needle: "dotf env identity set <ID>"},
		{name: "paths only → WARN naming the verb", body: `{"paths": {"DOTFILES_REPO_DIR": "/r"}}`, want: StatusWarn, needle: "dotf env identity set <ID>"},
		{name: "an id of only whitespace → WARN", body: `{"machine": {"id": "  "}}`, want: StatusWarn, needle: "dotf env identity set <ID>"},
		{name: "declared → PASS naming the id", body: `{"machine": {"id": "macmini"}}`, want: StatusPass, needle: "macmini"},
		{name: "declared with a deny list → PASS naming what it denies", body: `{"machine": {"id": "work"}, "pools": {"deny": ["nan", "claude"]}}`, want: StatusPass, needle: "denies nan, claude"},
		{name: "malformed → WARN with the parse error", body: `{not json`, want: StatusWarn, needle: "parse machine.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			config := filepath.Join(home, ".config")
			t.Setenv("XDG_CONFIG_HOME", config)
			if tc.body != "" {
				path := envpkg.MachinePath(home)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			s := newSys(map[string]string{"HOME": home, "XDG_CONFIG_HOME": config}, nil, nil)
			var buf bytes.Buffer
			checkMachineIdentity(s, capture(&buf))
			if got := statusOfLine(buf.String(), tc.needle); got != tc.want {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.want), buf.String())
			}
		})
	}
}
