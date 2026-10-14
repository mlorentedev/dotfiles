package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// Doctor's vault checks resolve the vault through the ADR-025 cascade, never a
// path guessed in the check (#2013 debt ledger, D13): VAULT_PATH wins, then
// machine.json and the contract default, and nothing resolving means "no
// vault", reported as such rather than probed at ~/Projects/knowledge.
func TestVaultChecksResolveThroughTheContract(t *testing.T) {
	triggersCfg := func(t *testing.T) *Config {
		t.Helper()
		dir := t.TempDir()
		p := filepath.Join(dir, filepath.FromSlash(harness.TriggersFile))
		mkdirAll(t, filepath.Dir(p))
		if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return &Config{DotfilesDir: dir}
	}
	checks := []struct {
		name string
		run  func(t *testing.T, sys *System, rep *Report)
		want string // what the check says when no vault resolves
	}{
		{"vault presence", func(_ *testing.T, sys *System, rep *Report) { checkVault(sys, rep) }, "no vault path resolves"},
		{"vault hooks", func(_ *testing.T, sys *System, rep *Report) { checkVaultHooks(sys, rep, false) }, "no vault resolves"},
		{"auto-memory link", func(t *testing.T, sys *System, rep *Report) { checkAutoMemoryLink(sys, t.TempDir(), rep, false) }, "no vault resolves"},
		{"trigger targets", func(t *testing.T, sys *System, rep *Report) { checkTriggerTargets(sys, triggersCfg(t), rep) }, "no vault resolves"},
	}

	for _, c := range checks {
		t.Run(c.name+": nothing resolves → says so, probes no default", func(t *testing.T) {
			home := t.TempDir()
			// A vault at the old literal default must not be picked up.
			mkdirAll(t, filepath.Join(home, "Projects", "knowledge", "00_meta"))
			sys := newSys(map[string]string{"HOME": home}, nil, nil)
			sys.ResolvePath = func(string) string { return "" }
			var buf bytes.Buffer
			c.run(t, sys, capture(&buf))
			out := buf.String()
			if !strings.Contains(out, c.want) || strings.Contains(out, filepath.Join("Projects", "knowledge")) {
				t.Fatalf("want %q and no probe of the literal default\n%s", c.want, out)
			}
		})
		t.Run(c.name+": machine.json resolution is the one used", func(t *testing.T) {
			resolved := filepath.Join(t.TempDir(), "elsewhere")
			sys := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
			sys.ResolvePath = func(k string) string {
				if k == "VAULT_PATH" {
					return resolved
				}
				return ""
			}
			var buf bytes.Buffer
			c.run(t, sys, capture(&buf))
			if out := buf.String(); strings.Contains(out, c.want) {
				t.Fatalf("a resolved vault must not read as none\n%s", out)
			}
			if got := sys.vaultPath(); got != resolved {
				t.Fatalf("vaultPath() = %q, want %q", got, resolved)
			}
		})
	}

	t.Run("VAULT_PATH in doctor's environment wins over the resolver", func(t *testing.T) {
		sys := newSys(map[string]string{"VAULT_PATH": "/from/env"}, nil, nil)
		sys.ResolvePath = func(string) string { return "/from/machine" }
		if got := sys.vaultPath(); got != "/from/env" {
			t.Fatalf("vaultPath() = %q, want /from/env", got)
		}
	})
}
