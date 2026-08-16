package secrets

import (
	"fmt"
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

// A mapped var whose secret hits a real decrypt error is left intact (render is
// non-fatal by default) but reported as Unresolved WITH its specific error (no
// longer swallowed) — the placeholder falls through to the runtime resolver.
func TestRender_UnresolvedDecryptError_LeftIntactWithError(t *testing.T) {
	reg := parseRenderReg(t, renderRegistry)
	path := renderFixture(t, `x={env:NAN_API_KEY}`)
	l := loaderFor(t)
	l.Decrypt = func(string, string) ([]byte, error) { return nil, fmt.Errorf("age: no identity matched") }

	res, err := Render(path, reg, l, "/h")
	if err != nil {
		t.Fatalf("Render must not fail on a single undecryptable secret: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != `x={env:NAN_API_KEY}` {
		t.Errorf("undecryptable placeholder should be left intact, got %q", got)
	}
	if len(res.Unresolved) != 1 || res.Unresolved[0].Var != "NAN_API_KEY" {
		t.Fatalf("Unresolved = %v, want one entry for NAN_API_KEY", res.Unresolved)
	}
	if res.Unresolved[0].Err == nil || !strings.Contains(res.Unresolved[0].Err.Error(), "no identity matched") {
		t.Errorf("Unresolved error not surfaced: %v", res.Unresolved[0].Err)
	}
	if len(res.Missing) != 0 {
		t.Errorf("a decrypt error is not 'absent'; Missing = %v", res.Missing)
	}
}

// A genuinely-absent secret (ErrSecretAbsent) is the quiet, non-fatal case: left
// intact, classified Missing (not Unresolved), no error.
func TestRender_AbsentSecret_QuietMissing(t *testing.T) {
	reg := parseRenderReg(t, renderRegistry)
	path := renderFixture(t, `x={env:NAN_API_KEY}`)
	l := loaderFor(t)
	l.Decrypt = func(string, string) ([]byte, error) { return nil, fmt.Errorf("%w: nan.api-key", ErrSecretAbsent) }

	res, err := Render(path, reg, l, "/h")
	if err != nil {
		t.Fatalf("Render must not fail on an absent secret: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != `x={env:NAN_API_KEY}` {
		t.Errorf("absent placeholder should be left intact, got %q", got)
	}
	if len(res.Missing) != 1 || res.Missing[0] != "NAN_API_KEY" {
		t.Errorf("Missing = %v, want [NAN_API_KEY]", res.Missing)
	}
	if len(res.Unresolved) != 0 {
		t.Errorf("absent must not be Unresolved; Unresolved = %v", res.Unresolved)
	}
}

// An empty resolved value is a real failure now (EnvFor rejects it) → Unresolved,
// not a silent substitution of "".
func TestRender_EmptyValue_Unresolved(t *testing.T) {
	reg := parseRenderReg(t, renderRegistry)
	path := renderFixture(t, `x={env:NAN_API_KEY}`)
	l := loaderFor(t)
	l.Decrypt = func(string, string) ([]byte, error) { return []byte("\n"), nil } // strips to ""

	res, err := Render(path, reg, l, "/h")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != `x={env:NAN_API_KEY}` {
		t.Errorf("empty-valued placeholder should be left intact, got %q", got)
	}
	if len(res.Unresolved) != 1 || res.Unresolved[0].Var != "NAN_API_KEY" {
		t.Fatalf("Unresolved = %v, want NAN_API_KEY (empty value)", res.Unresolved)
	}
	if !strings.Contains(res.Unresolved[0].Err.Error(), "empty value") {
		t.Errorf("expected an empty-value error, got %v", res.Unresolved[0].Err)
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

// A var exposed by two secrets is now rejected at PARSE (#612 B1), centralizing for
// all backends what render's envSourceMap caught age-only. render keeps that dedup as
// defense-in-depth for a Registry hand-built past validate, but the normal path fails
// earlier — at ParseRegistry — which this pins (the shell twin silently took the first
// mapping line; we refuse the ambiguity up front).
func TestParseRegistry_DuplicateVar_Rejected(t *testing.T) {
	const dup = `
version: 1
secrets:
  - { id: a, plane: app, backend: age, age: a.src, expose: { env: SHARED } }
  - { id: b, plane: app, backend: age, age: b.src, expose: { env: SHARED } }
`
	if _, err := ParseRegistry([]byte(dup)); err == nil {
		t.Fatal("expected ParseRegistry to reject a var exposed by two secrets")
	}
}

// The claim the whole BUG-081b argument rests on: render substitutes {env:VAR}
// and nothing else, so a config written in pi's own `${VAR}` syntax passes
// through byte-identical.
//
// If this were false, flipping ai/pi/models.json to pi's syntax would require
// changing the deploy blocks in BOTH setup scripts — the growth the CLI
// convergence epic exists to reverse. It is asserted rather than assumed
// because the entire "no setup edits" scope decision depends on it.
func TestRender_LeavesShellStyleVariablesUntouched(t *testing.T) {
	const src = `{"providers":{"nan":{"apiKey":"${NAN_API_KEY}","alt":"$NAN_API_KEY"}}}`
	path := renderFixture(t, src)
	reg := parseRenderReg(t, renderRegistry)

	res, err := Render(path, reg, loaderFor(t), "/h")
	if err != nil {
		t.Fatalf("render must not fail on a config it has nothing to do in: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != src {
		t.Errorf("render rewrote a config it should have passed through:\nwant %s\ngot  %s", src, got)
	}
	if res.Substituted != 0 {
		t.Errorf("render reported %d substitution(s) in a file with no {env:} placeholders", res.Substituted)
	}
}
