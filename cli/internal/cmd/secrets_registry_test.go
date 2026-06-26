package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// fakeBW is a map-backed secrets.BWReader ("item/field" → value) so command tests
// resolve bw secrets with no unlocked Bitwarden.
type fakeBW map[string]string

func (f fakeBW) Field(item, field string) (string, error) {
	if v, ok := f[item+"/"+field]; ok {
		return v, nil
	}
	return "", fmt.Errorf("fake bw: no %s/%s", item, field)
}

// useBwReader points the bwReader seam at a fake for the duration of a test.
func useBwReader(t *testing.T, r secrets.BWReader) {
	t.Helper()
	old := bwReader
	bwReader = r
	t.Cleanup(func() { bwReader = old })
}

const testRegistry = `
version: 1
secrets:
  - {id: nan-api-key, plane: app, backend: age, age: nan.api-key, expose: {env: NAN_API_KEY}}
  - {id: x-twitter, plane: app, backend: age, expose: {env: {X_API_KEY: {age: x.api-key}, X_BEARER_TOKEN: {age: x.bearer-token}}}}
  - {id: kubelab-kubeconfig, plane: infra, backend: age, age: kubelab.kubeconfig, expose: {file: {var: KUBECONFIG, path: "~/.kube/c", mode: "0600"}}}
`

// useTempRegistry points registryPath at a fixture for the duration of a test.
func useTempRegistry(t *testing.T, body string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "registry.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	old := registryPath
	registryPath = func() string { return p }
	t.Cleanup(func() { registryPath = old })
}

func TestSecretsLs_ListsIdsAndVars_NoValues(t *testing.T) {
	useTempRegistry(t, testRegistry)
	var out bytes.Buffer
	cmd := newSecretsLsCmd()
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"nan-api-key", "x-twitter", "kubelab-kubeconfig", "NAN_API_KEY", "KUBECONFIG"} {
		if !strings.Contains(got, want) {
			t.Errorf("ls output missing %q:\n%s", want, got)
		}
	}
}

func TestSecretsLs_Pairs_EnvOnly(t *testing.T) {
	useTempRegistry(t, testRegistry)
	var out bytes.Buffer
	cmd := newSecretsLsCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--pairs"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	// VAR<TAB>age-source, one per env var; file secrets (KUBECONFIG) excluded.
	for _, want := range []string{"NAN_API_KEY\tnan.api-key", "X_API_KEY\tx.api-key", "X_BEARER_TOKEN\tx.bearer-token"} {
		if !strings.Contains(got, want) {
			t.Errorf("--pairs missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "KUBECONFIG") {
		t.Errorf("--pairs must exclude file secrets, got:\n%s", got)
	}
}

func TestSecretsShow_SingleEnv_ScrubbedNoTrailingNewline(t *testing.T) {
	useTempRegistry(t, testRegistry)
	old := ageDecryptor
	ageDecryptor = func(_, _ string) ([]byte, error) { return []byte("the-value\n"), nil }
	t.Cleanup(func() { ageDecryptor = old })

	var out bytes.Buffer
	cmd := newSecretsShowCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"nan-api-key"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "the-value" {
		t.Errorf("show = %q, want %q (no trailing newline)", out.String(), "the-value")
	}
}

func TestSecretsShow_RejectsFileAndMultiAndUnknown(t *testing.T) {
	useTempRegistry(t, testRegistry)
	for _, id := range []string{"x-twitter", "kubelab-kubeconfig", "nope"} {
		cmd := newSecretsShowCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{id})
		if err := cmd.Execute(); err == nil {
			t.Errorf("show %q: expected an error (must point at run / unknown)", id)
		}
	}
}

func TestSecretsShow_ResolvesBwBackend(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-one, plane: app, backend: bw, bw: {item: bw-item, field: password}, expose: {env: B_KEY}}\n")
	useBwReader(t, fakeBW{"bw-item/password": "bw-secret"})
	var out bytes.Buffer
	cmd := newSecretsShowCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"bw-one"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("show bw-one: %v", err)
	}
	if out.String() != "bw-secret" {
		t.Errorf("show bw = %q, want %q (resolved from Bitwarden, no trailing newline)", out.String(), "bw-secret")
	}
}

// TestSecretsRun_ResolvesBwBackend exercises run's wiring: registry → secretLoader →
// EnvFor → bwResolver → the bw reader. --only selects the bw secret by id.
func TestSecretsRun_ResolvesBwBackend(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-foo, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: FOO}}\n")
	useBwReader(t, fakeBW{"it/password": "bw-secret-value"})

	reg, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	sel, err := resolveOnly(reg, "bw-foo") // --only by id → the secret's vars
	if err != nil {
		t.Fatal(err)
	}
	env, err := buildChildEnv(reg, sel)
	if err != nil {
		t.Fatalf("buildChildEnv: %v", err)
	}
	found := false
	for _, kv := range env {
		if kv == "FOO=bw-secret-value" {
			found = true
		}
	}
	if !found {
		t.Errorf("run did not inject the bw secret FOO=bw-secret-value into the child env")
	}
}

func TestStripBackendAuth(t *testing.T) {
	in := []string{"PATH=/bin", "BW_SESSION=tok", "FOO=bar", "OP_SERVICE_ACCOUNT_TOKEN=x", "BW_CLIENTSECRET=y", "EMPTY="}
	got := strings.Join(stripBackendAuth(in), " ")
	for _, banned := range []string{"BW_SESSION", "OP_SERVICE_ACCOUNT_TOKEN", "BW_CLIENTSECRET"} {
		if strings.Contains(got, banned) {
			t.Errorf("%s must be stripped from the child env, got: %s", banned, got)
		}
	}
	for _, kept := range []string{"PATH=/bin", "FOO=bar", "EMPTY="} {
		if !strings.Contains(got, kept) {
			t.Errorf("non-auth var %q must be kept, got: %s", kept, got)
		}
	}
}

// TestBuildChildEnv_StripsBackendAuth proves the resolved secret reaches the child
// while the backend unlock token (BW_SESSION) does not — the child gets what it was
// granted, never the key that opens the whole vault.
func TestBuildChildEnv_StripsBackendAuth(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-foo, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: FOO}}\n")
	useBwReader(t, fakeBW{"it/password": "granted-value"})
	t.Setenv("BW_SESSION", "unlock-token-must-not-leak")

	reg, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	env, err := buildChildEnv(reg, nil)
	if err != nil {
		t.Fatalf("buildChildEnv: %v", err)
	}
	var grantedFound bool
	for _, kv := range env {
		if strings.HasPrefix(kv, "BW_SESSION=") {
			t.Error("BW_SESSION must NOT be passed to the child environment")
		}
		if kv == "FOO=granted-value" {
			grantedFound = true
		}
	}
	if !grantedFound {
		t.Error("the granted secret FOO=granted-value must still reach the child env")
	}
}

func TestSecretsRender_SubstitutesInPlace(t *testing.T) {
	useTempRegistry(t, testRegistry)
	old := ageDecryptor
	ageDecryptor = func(_, _ string) ([]byte, error) { return []byte("rendered-value\n"), nil }
	t.Cleanup(func() { ageDecryptor = old })

	cfg := filepath.Join(t.TempDir(), "opencode.jsonc")
	if err := os.WriteFile(cfg, []byte(`{"k":"{env:NAN_API_KEY}","h":"{env:HOME}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newSecretsRenderCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{cfg})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(cfg)
	// NAN_API_KEY is registry-mapped → substituted; HOME is not → left for runtime.
	if want := `{"k":"rendered-value","h":"{env:HOME}"}`; string(got) != want {
		t.Errorf("render = %q, want %q", got, want)
	}
}

func TestSecretsShow_EmptyValue_Errors(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-empty, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: E_KEY}}\n")
	useBwReader(t, fakeBW{"it/password": ""})
	cmd := newSecretsShowCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"bw-empty"})
	if err := cmd.Execute(); err == nil {
		t.Error("show must error when the secret resolves to an empty value")
	}
}

// render default stays non-fatal on one failing secret (placeholder left, exit 0);
// --strict turns the same real failure into a non-zero exit (#612 A2).
func TestSecretsRender_Strict_NonZeroOnFailure(t *testing.T) {
	old := ageDecryptor
	ageDecryptor = func(_, _ string) ([]byte, error) { return nil, fmt.Errorf("age: vault locked") }
	t.Cleanup(func() { ageDecryptor = old })

	render := func(args ...string) error {
		useTempRegistry(t, testRegistry)
		cfg := filepath.Join(t.TempDir(), "c.jsonc")
		if err := os.WriteFile(cfg, []byte(`{"k":"{env:NAN_API_KEY}"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		c := newSecretsRenderCmd()
		c.SetOut(io.Discard)
		c.SetErr(io.Discard)
		c.SetArgs(append(args, cfg))
		return c.Execute()
	}
	if err := render(); err != nil {
		t.Fatalf("render default must not fail on one unresolved secret: %v", err)
	}
	if err := render("--strict"); err == nil {
		t.Error("render --strict must exit non-zero when a mapped secret fails to resolve")
	}
}

func TestResolveOnly_EmptyTokens_Errors(t *testing.T) {
	reg, err := secrets.ParseRegistry([]byte(testRegistry))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveOnly(reg, ","); err == nil {
		t.Error("--only with only-empty tokens must error (it selects nothing)")
	}
	if _, err := resolveOnly(reg, "  ,  "); err == nil {
		t.Error("--only with whitespace-only tokens must error")
	}
}

func TestResolveOnly_IdSelectsAllVars_NameSelectsOne(t *testing.T) {
	reg, err := secrets.ParseRegistry([]byte(testRegistry))
	if err != nil {
		t.Fatal(err)
	}
	set, err := resolveOnly(reg, "x-twitter") // id → all its vars
	if err != nil || len(set) != 2 || !set["X_API_KEY"] || !set["X_BEARER_TOKEN"] {
		t.Errorf("id selector = %v (err %v), want {X_API_KEY,X_BEARER_TOKEN}", set, err)
	}
	set, _ = resolveOnly(reg, "X_BEARER_TOKEN") // var name → just itself
	if len(set) != 1 || !set["X_BEARER_TOKEN"] {
		t.Errorf("var selector = %v, want {X_BEARER_TOKEN}", set)
	}
	if s, _ := resolveOnly(reg, "  "); s != nil {
		t.Error("empty --only must resolve to nil (= all)")
	}
	if _, err := resolveOnly(reg, "bogus"); err == nil {
		t.Error("unknown --only token must error")
	}
}

func TestSecretsVerify_ReportsStatuses_NoValues(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n"+
		"  - {id: age-ok, plane: app, backend: age, age: a.key, expose: {env: AGE_OK}}\n"+
		"  - {id: age-missing, plane: app, backend: age, age: gone, expose: {env: AGE_MISSING}}\n"+
		"  - {id: bw-fail, plane: app, backend: bw, bw: {item: nope, field: password}, expose: {env: BW_FAIL}}\n")
	old := ageDecryptor
	ageDecryptor = func(ageFile, _ string) ([]byte, error) {
		if strings.Contains(ageFile, "gone") {
			return nil, fmt.Errorf("%w: gone", secrets.ErrSecretAbsent)
		}
		return []byte("the-secret-value\n"), nil
	}
	t.Cleanup(func() { ageDecryptor = old })
	useBwReader(t, fakeBW{}) // every bw lookup fails

	var out bytes.Buffer
	cmd := newSecretsVerifyCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	got := out.String()
	for _, want := range []string{"OK", "AGE_OK", "MISSING", "AGE_MISSING", "FAILED", "BW_FAIL"} {
		if !strings.Contains(got, want) {
			t.Errorf("verify output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "the-secret-value") {
		t.Error("verify must NOT print secret values")
	}
	if err == nil {
		t.Error("verify must exit non-zero when a secret FAILED")
	}
}

func TestSecretsVerify_ScopesById(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n"+
		"  - {id: a, plane: app, backend: age, age: a, expose: {env: AAA}}\n"+
		"  - {id: b, plane: app, backend: age, age: b, expose: {env: BBB}}\n")
	old := ageDecryptor
	ageDecryptor = func(string, string) ([]byte, error) { return []byte("v\n"), nil }
	t.Cleanup(func() { ageDecryptor = old })

	var out bytes.Buffer
	cmd := newSecretsVerifyCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"a"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("verify a: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "AAA") {
		t.Errorf("verify a must check AAA: %s", got)
	}
	if strings.Contains(got, "BBB") {
		t.Errorf("verify a must NOT check BBB: %s", got)
	}
}

func TestSecretsVerify_UnknownId_Errors(t *testing.T) {
	useTempRegistry(t, testRegistry)
	cmd := newSecretsVerifyCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"no-such-id"})
	if err := cmd.Execute(); err == nil {
		t.Error("verify with an unknown id must error")
	}
}
