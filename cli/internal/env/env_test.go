package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testContract() *contract {
	return &contract{EnvVars: []contractVar{
		{Name: "HOME", Default: nil}, // OS-provided, no default -> must be skipped
		{Name: "DOTFILES_REPO_DIR", Default: map[string]string{
			"linux": "$HOME/Projects/dotfiles", "windows": "$env:USERPROFILE\\Projects\\dotfiles"}},
		{Name: "VAULT_PATH", Default: map[string]string{
			"linux": "$HOME/Projects/knowledge", "windows": "$env:USERPROFILE\\Projects\\knowledge"}},
	}}
}

func resolved(got []ResolvedVar) map[string]string {
	m := map[string]string{}
	for _, rv := range got {
		m[rv.Name] = rv.Value
	}
	return m
}

func TestResolveOverrideBeatsDefaultAndSkipsNoDefault(t *testing.T) {
	c := testContract()
	m := &machine{Paths: map[string]string{"VAULT_PATH": "/data/vault"}}

	got := Resolve(c, m, "linux", "/home/me")

	if _, ok := resolved(got)["HOME"]; ok {
		t.Error("HOME has no default — must not be generated")
	}
	want := map[string]string{
		"DOTFILES_REPO_DIR": "/home/me/Projects/dotfiles", // default, home expanded
		"VAULT_PATH":        "/data/vault",                // machine.json override wins
	}
	for k, v := range want {
		if resolved(got)[k] != v {
			t.Errorf("%s = %q, want %q", k, resolved(got)[k], v)
		}
	}
	if len(got) != 2 {
		t.Errorf("got %d vars, want 2: %+v", len(got), got)
	}
}

func TestResolveDarwinFallsBackToLinuxDefault(t *testing.T) {
	c := &contract{EnvVars: []contractVar{
		{Name: "VAULT_PATH", Default: map[string]string{"linux": "$HOME/Projects/knowledge"}},
	}}
	got := Resolve(c, &machine{}, "darwin", "/Users/me")
	if len(got) != 1 || got[0].Value != "/Users/me/Projects/knowledge" {
		t.Fatalf("darwin should fall back to linux default: %+v", got)
	}
}

func TestResolveWindowsExpandsUserprofile(t *testing.T) {
	got := resolved(Resolve(testContract(), &machine{}, "windows", `C:\Users\me`))
	if got["VAULT_PATH"] != `C:\Users\me\Projects\knowledge` {
		t.Errorf("windows VAULT_PATH = %q", got["VAULT_PATH"])
	}
}

func TestRenderShKeepsAlreadySetValue(t *testing.T) {
	out := Render(FormatSh, []ResolvedVar{{Name: "VAULT_PATH", Value: "/data/vault"}})
	if !strings.Contains(out, `export VAULT_PATH="${VAULT_PATH:-/data/vault}"`) {
		t.Errorf("sh render missing guarded export:\n%s", out)
	}
}

func TestRenderPs1OnlyAssignsWhenUnset(t *testing.T) {
	out := Render(FormatPs1, []ResolvedVar{{Name: "VAULT_PATH", Value: `C:\data\vault`}})
	if !strings.Contains(out, `if (-not $env:VAULT_PATH) { $env:VAULT_PATH = 'C:\data\vault' }`) {
		t.Errorf("ps1 render wrong:\n%s", out)
	}
}

func TestGenerateWriteIdempotentAndCheck(t *testing.T) {
	dir := t.TempDir()
	contractPath := filepath.Join(dir, "env-contract.json")
	if err := os.WriteFile(contractPath,
		[]byte(`{"env_vars":[{"name":"VAULT_PATH","default":{"linux":"$HOME/Projects/knowledge"}}]}`),
		0o644); err != nil {
		t.Fatal(err)
	}
	machinePath := filepath.Join(dir, "machine.json") // absent -> contract defaults
	out := filepath.Join(dir, "paths.sh")
	opts := Options{ContractPath: contractPath, MachinePath: machinePath, GOOS: "linux", Home: "/home/me", Output: out}

	res, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Wrote || !strings.Contains(res.Content, "/home/me/Projects/knowledge") {
		t.Fatalf("first generate: wrote=%v content=%q", res.Wrote, res.Content)
	}

	res2, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Wrote {
		t.Error("identical content must not be rewritten")
	}

	if err := os.WriteFile(out, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chk := opts
	chk.Check = true
	res3, err := Generate(chk)
	if err != nil {
		t.Fatal(err)
	}
	if !res3.Drifted {
		t.Error("expected drift after manual edit")
	}
}
