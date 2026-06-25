package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

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

func TestSecretsShow_RejectsBwBackend(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-one, plane: app, backend: bw, expose: {env: B_KEY}}\n")
	cmd := newSecretsShowCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"bw-one"})
	if err := cmd.Execute(); err == nil {
		t.Error("show on a bw-backed secret must error (bw not supported until Phase 3)")
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
