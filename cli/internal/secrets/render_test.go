package secrets

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// renderRegistry exposes one single-var env secret plus a file secret, so render
// can prove it (a) substitutes the env placeholder and (b) never treats the file
// secret's var as a substitution target nor materializes it as a side effect.
const renderRegistry = `
version: 1
secrets:
  - id: nan-api-key
    plane: app
    backend: age
    age: nan.api-key
    expose: { env: NAN_API_KEY }
  - id: kubelab-kubeconfig
    plane: infra
    backend: age
    age: kubelab.kubeconfig
    expose: { file: { var: KUBECONFIG, path: "~/.kube/kubelab.config", mode: "0600" } }
`

func renderFixture(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "opencode.jsonc")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}
	return p
}

func parseRenderReg(t *testing.T, yml string) *Registry {
	t.Helper()
	reg, err := ParseRegistry([]byte(yml))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	return reg
}

// AC1 — a registry-mapped {env:VAR} is replaced with its decrypted value; an
// unmapped placeholder ({env:HOME}) is left intact for the runtime resolver.
func TestRender_SubstitutesMapped_LeavesUnmappedIntact(t *testing.T) {
	reg := parseRenderReg(t, renderRegistry)
	path := renderFixture(t, `{"key":"{env:NAN_API_KEY}","home":"{env:HOME}"}`)

	res, err := Render(path, reg, loaderFor(t), "/h")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, _ := os.ReadFile(path)
	// fakeDecryptor yields "secret-of-<age-basename>\n"; EnvFor strips the newline.
	want := `{"key":"secret-of-nan.api-key.secret.age","home":"{env:HOME}"}`
	if string(got) != want {
		t.Errorf("rendered = %q, want %q", got, want)
	}
	if res.Substituted != 1 {
		t.Errorf("Substituted = %d, want 1", res.Substituted)
	}
	if len(res.Unmapped) != 1 || res.Unmapped[0] != "HOME" {
		t.Errorf("Unmapped = %v, want [HOME]", res.Unmapped)
	}
}

// AC2 — atomic write at 0600 with byte parity: render must NOT append a trailing
// newline the source file did not have.
func TestRender_Mode0600_NoTrailingNewlineDrift(t *testing.T) {
	reg := parseRenderReg(t, renderRegistry)
	path := renderFixture(t, `x={env:NAN_API_KEY}`) // no trailing newline

	if _, err := Render(path, reg, loaderFor(t), "/h"); err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, _ := os.ReadFile(path)
	if strings.HasSuffix(string(got), "\n") {
		t.Errorf("render added a trailing newline: %q", got)
	}
	if string(got) != "x=secret-of-nan.api-key.secret.age" {
		t.Errorf("rendered = %q", got)
	}
	if runtime.GOOS != "windows" { // NTFS drops POSIX mode bits
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("mode = %v (want 0600)", info.Mode().Perm())
		}
	}
}

// A file with no placeholders is left byte-for-byte untouched (and not rewritten).
func TestRender_NoPlaceholders_NoOp(t *testing.T) {
	reg := parseRenderReg(t, renderRegistry)
	const body = `{"plain":"config","n":1}`
	path := renderFixture(t, body)

	res, err := Render(path, reg, loaderFor(t), "/h")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != body {
		t.Errorf("no-op render changed content: %q", got)
	}
	if res.Substituted != 0 {
		t.Errorf("Substituted = %d, want 0", res.Substituted)
	}
}

// Parity with the shell twin: a mapped var whose secret cannot be decrypted is
// left intact and reported as unresolved — render must NOT fail-fast (setup must
// still complete; the placeholder falls through to the runtime resolver).
func TestRender_UnresolvedDecryptError_LeftIntact(t *testing.T) {
	reg := parseRenderReg(t, renderRegistry)
	path := renderFixture(t, `x={env:NAN_API_KEY}`)
	l := loaderFor(t)
	l.Decrypt = func(string, string) ([]byte, error) { return nil, os.ErrNotExist }

	res, err := Render(path, reg, l, "/h")
	if err != nil {
		t.Fatalf("Render must not fail on an undecryptable secret: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != `x={env:NAN_API_KEY}` {
		t.Errorf("undecryptable placeholder should be left intact, got %q", got)
	}
	if len(res.Unresolved) != 1 || res.Unresolved[0] != "NAN_API_KEY" {
		t.Errorf("Unresolved = %v, want [NAN_API_KEY]", res.Unresolved)
	}
}

// A file secret's var is NOT an env-substitution target: {env:KUBECONFIG} is left
// intact and the kube config is NOT materialized to disk as a side effect.
func TestRender_FileSecretVar_LeftIntactNotMaterialized(t *testing.T) {
	reg := parseRenderReg(t, renderRegistry)
	path := renderFixture(t, `k={env:KUBECONFIG}`)

	res, err := Render(path, reg, loaderFor(t), "/h")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != `k={env:KUBECONFIG}` {
		t.Errorf("file-secret var should be left intact, got %q", got)
	}
	if len(res.Unmapped) != 1 || res.Unmapped[0] != "KUBECONFIG" {
		t.Errorf("Unmapped = %v, want [KUBECONFIG]", res.Unmapped)
	}
	if _, err := os.Stat(filepath.Join("/h", ".kube", "kubelab.config")); err == nil {
		t.Error("render materialized a file secret as a side effect")
	}
}

// render needs a deterministic VAR→source map: if the registry exposes one var
// from two distinct secrets, render fails fast (the shell twin silently took the
// first mapping line; render refuses the ambiguity).
func TestRender_DuplicateVar_FailsFast(t *testing.T) {
	const dup = `
version: 1
secrets:
  - { id: a, plane: app, backend: age, age: a.src, expose: { env: SHARED } }
  - { id: b, plane: app, backend: age, age: b.src, expose: { env: SHARED } }
`
	reg := parseRenderReg(t, dup)
	path := renderFixture(t, `v={env:SHARED}`)

	if _, err := Render(path, reg, loaderFor(t), "/h"); err == nil {
		t.Fatal("expected Render to fail when one var is exposed by two secrets")
	}
}
