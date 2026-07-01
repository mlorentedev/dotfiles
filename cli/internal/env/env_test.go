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

// TestRealContractRendersAgeKeyDiscovery is the end-to-end regression guard for
// #518: the actual repo env-contract.json must resolve AGE_KEY_PATH and
// SOPS_AGE_KEY_FILE to the deployed age key on both OSes, and both must survive
// into the rendered path files. Discovery is only automatic if the real
// contract carries these vars with a default (a var without a default is skipped
// by Resolve), so this ties the shipped contract to the shipped behavior — not a
// synthetic fixture.
func TestRealContractRendersAgeKeyDiscovery(t *testing.T) {
	// Test runs with cwd = cli/internal/env; the repo root is three up.
	c, err := loadContract(filepath.Join("..", "..", "..", "env-contract.json"))
	if err != nil {
		t.Fatalf("load real env-contract.json: %v", err)
	}

	cases := []struct {
		goos, home, want string
	}{
		{"linux", "/home/me", "/home/me/.config/age/key.txt"},
		{"windows", `C:\Users\me`, `C:\Users\me\.config\age\key.txt`},
	}
	for _, tc := range cases {
		got := resolved(Resolve(c, &machine{}, tc.goos, tc.home))
		for _, name := range []string{"AGE_KEY_PATH", "SOPS_AGE_KEY_FILE"} {
			if got[name] != tc.want {
				t.Errorf("%s[%s] = %q, want %q", name, tc.goos, got[name], tc.want)
			}
		}
	}

	// Both vars must reach the rendered sh path file, guarded so an already-set
	// value wins (the ADR-025 cascade rule #1).
	sh := Render(FormatSh, Resolve(c, &machine{}, "linux", "/home/me"))
	for _, name := range []string{"AGE_KEY_PATH", "SOPS_AGE_KEY_FILE"} {
		if !strings.Contains(sh, "export "+name+"=") {
			t.Errorf("rendered paths.sh missing export for %s:\n%s", name, sh)
		}
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

func TestResolveContractPathPrefersRepoCheckout(t *testing.T) {
	// A valid DOTFILES_REPO_DIR checkout wins over the deployed copy, so a dev
	// machine never generates from a stale ~/.dotfiles/env-contract.json.
	repo := t.TempDir()
	want := filepath.Join(repo, "env-contract.json")
	if err := os.WriteFile(want, []byte(`{"env_vars":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", repo)

	if got := ResolveContractPath(); got != want {
		t.Errorf("ResolveContractPath() = %q, want repo copy %q", got, want)
	}
}

func TestResolveContractPathRepoMissingFallsThrough(t *testing.T) {
	// DOTFILES_REPO_DIR set but carrying no contract (or a stale/non-existent path)
	// must NOT short-circuit — resolution falls through to the deployed copy.
	t.Setenv("DOTFILES_REPO_DIR", t.TempDir()) // empty dir: no env-contract.json
	deployed := t.TempDir()
	want := filepath.Join(deployed, "env-contract.json")
	if err := os.WriteFile(want, []byte(`{"env_vars":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_DIR", deployed)

	if got := ResolveContractPath(); got != want {
		t.Errorf("ResolveContractPath() = %q, want deployed copy %q", got, want)
	}
}

func TestRepoDirPrefersDotfilesRepoDir(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("DOTFILES_REPO_DIR", repo)
	if got := RepoDir(); got != repo {
		t.Errorf("RepoDir() = %q, want %q", got, repo)
	}
}

func TestRepoDirWalksUpForGitWhenNoEnv(t *testing.T) {
	// No DOTFILES_REPO_DIR → RepoDir walks up from cwd for a .git entry.
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "cli", "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", "") // force the walk-up branch
	t.Chdir(sub)

	got, err := filepath.EvalSymlinks(RepoDir()) // tmp dirs may be symlinked (macOS)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("RepoDir() = %q, want repo root %q", got, want)
	}
}

func TestRepoDirNoneFound(t *testing.T) {
	t.Setenv("DOTFILES_REPO_DIR", "")
	t.Chdir(t.TempDir()) // a bare tmp dir has no .git ancestor
	if got := RepoDir(); got != "" {
		t.Errorf("RepoDir() = %q, want empty (no checkout)", got)
	}
}

func TestResolveRegistryPathPrefersRepoCheckout(t *testing.T) {
	// The checkout's registry (the version-controlled SSOT) wins over the deployed
	// copy — the read side of ADR-030 / #635.
	repo := t.TempDir()
	want := filepath.Join(repo, "secrets", "registry.yaml")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", repo)

	if got := ResolveRegistryPath(); got != want {
		t.Errorf("ResolveRegistryPath() = %q, want repo copy %q", got, want)
	}
}

func TestResolveRegistryPathFallsBackToDeployed(t *testing.T) {
	// Checkout present but carrying no registry → fall through to the deployed copy
	// under DOTFILES_DIR rather than returning a non-existent repo path.
	t.Setenv("DOTFILES_REPO_DIR", t.TempDir()) // empty: no secrets/registry.yaml
	deployed := t.TempDir()
	t.Setenv("DOTFILES_DIR", deployed)
	want := filepath.Join(deployed, "secrets", "registry.yaml")

	if got := ResolveRegistryPath(); got != want {
		t.Errorf("ResolveRegistryPath() = %q, want deployed copy %q", got, want)
	}
}

func TestRepoRegistryPathReturnsCheckoutPath(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("DOTFILES_REPO_DIR", repo)
	got, err := RepoRegistryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(repo, "secrets", "registry.yaml"); got != want {
		t.Errorf("RepoRegistryPath() = %q, want %q", got, want)
	}
}

func TestRepoRegistryPathFailsLoudWithoutCheckout(t *testing.T) {
	// The write seam must refuse to fall back to the deployed copy: a migrate written
	// there is reverted on the next redeploy and never reaches git (#635).
	t.Setenv("DOTFILES_REPO_DIR", "")
	t.Chdir(t.TempDir())
	if _, err := RepoRegistryPath(); err == nil {
		t.Fatal("RepoRegistryPath() = nil error, want fail-loud when no checkout is found")
	}
}

func TestResolveSensitiveDirPrefersRepoCheckout(t *testing.T) {
	// The checkout's sensitive/ (where a rotation lands) wins over the deployed copy, so
	// the values track the same source as the registry that maps them (ADR-030).
	repo := t.TempDir()
	want := filepath.Join(repo, "sensitive")
	if err := os.Mkdir(want, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", repo)

	if got := ResolveSensitiveDir(); got != want {
		t.Errorf("ResolveSensitiveDir() = %q, want repo copy %q", got, want)
	}
}

func TestResolveSensitiveDirFallsBackToDeployed(t *testing.T) {
	// Checkout present but with no sensitive/ dir → fall through to the deployed store
	// under DOTFILES_DIR rather than returning a non-existent repo path.
	t.Setenv("DOTFILES_REPO_DIR", t.TempDir()) // empty: no sensitive/
	deployed := t.TempDir()
	t.Setenv("DOTFILES_DIR", deployed)
	want := filepath.Join(deployed, "sensitive")

	if got := ResolveSensitiveDir(); got != want {
		t.Errorf("ResolveSensitiveDir() = %q, want deployed copy %q", got, want)
	}
}

func TestRepoSensitiveDirPrefersRepoCheckout(t *testing.T) {
	// The DR-escrow write seam returns the checkout's sensitive/ so the escrow is
	// committable (the write side of ADR-030 / #635).
	repo := t.TempDir()
	t.Setenv("DOTFILES_REPO_DIR", repo)
	got, err := RepoSensitiveDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(repo, "sensitive"); got != want {
		t.Errorf("RepoSensitiveDir() = %q, want %q", got, want)
	}
}

func TestRepoSensitiveDirNoCheckoutFailsLoud(t *testing.T) {
	// The write seam must refuse the deployed copy: an escrow written there is reverted
	// on the next redeploy and never reaches git (#635) — fail loud instead.
	t.Setenv("DOTFILES_REPO_DIR", "")
	t.Chdir(t.TempDir())
	if _, err := RepoSensitiveDir(); err == nil {
		t.Fatal("RepoSensitiveDir() = nil error, want fail-loud when no checkout is found")
	}
}
