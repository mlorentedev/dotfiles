package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// A migratable secret: single env var, backend age, has a bw: target, local-only
// consumer, unique age source. Block form so SetBackendBW can rewrite it.
const migratableRegistry = `
version: 1
secrets:
  - id: NAN_API_KEY
    plane: app
    backend: age
    age: nan.api-key
    bw: { item: nan-api-key, field: api-key }
    expose: { env: NAN_API_KEY }
    consumers: [local]
`

// migrateExec runs migrate against a temp registry + fake bw + fake age, returning
// (stdout, error).
func migrateExec(t *testing.T, registry string, fw *fakeWriter, ageVal string, args ...string) (string, error) {
	t.Helper()
	useTempRegistry(t, registry)
	useBwReader(t, fw)
	useBwWriter(t, fw)
	forceNonInteractive(t)
	old := ageDecryptor
	ageDecryptor = func(_, _ string) ([]byte, error) { return []byte(ageVal), nil }
	t.Cleanup(func() { ageDecryptor = old })

	var out bytes.Buffer
	cmd := newSecretsMigrateCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// backendOfID re-reads the (temp) registry and returns the backend recorded for id.
func backendOfID(t *testing.T, id string) string {
	t.Helper()
	data, err := os.ReadFile(registryPath())
	if err != nil {
		t.Fatal(err)
	}
	reg, err := secrets.ParseRegistry(data)
	if err != nil {
		t.Fatalf("registry no longer parses: %v", err)
	}
	s := reg.Lookup(id)
	if s == nil {
		t.Fatalf("id %q vanished from the registry", id)
	}
	return s.Backend
}

func TestSecretsMigrate_EndToEnd(t *testing.T) {
	fw := newFakeWriter()
	fw.notFound["nan-api-key"] = true // target item absent → --yes creates it
	out, err := migrateExec(t, migratableRegistry, fw, "nan-value\n", "NAN_API_KEY", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if fw.created["nan-api-key/api-key"] != "nan-value" {
		t.Errorf("created = %v, want nan-api-key/api-key=nan-value (trailing newline trimmed)", fw.created)
	}
	if got := backendOfID(t, "NAN_API_KEY"); got != "bw" {
		t.Errorf("registry backend = %q, want bw (flipped)", got)
	}
	if !strings.Contains(out, "migrated") {
		t.Errorf("want 'migrated', got %q", out)
	}
}

// TestSecretsMigrate_FileSecret_ByteExact proves migrate now handles file secrets
// (expose: { file: ... }), and does so byte-exact — no trailing-newline trim at all,
// unlike env secrets. The fixture deliberately has NO trailing newline so a
// regression that trims OR one that appends is equally caught (CLI-024-secrets-file-migrate AC1/AC3).
func TestSecretsMigrate_FileSecret_ByteExact(t *testing.T) {
	const fileRegistry = `
version: 1
secrets:
  - id: KUBECONFIG
    plane: infra
    backend: age
    age: kubelab.kubeconfig
    bw: { item: kubelab-kubeconfig, field: notes }
    expose: { file: { var: KUBECONFIG, path: "~/.kube/kubelab.config", mode: "0600" } }
    consumers: [local]
`
	fw := newFakeWriter()
	fw.notFound["kubelab-kubeconfig"] = true // target item absent → --yes creates it
	kubeconfig := "apiVersion: v1\nclusters:\n- cluster:\n    server: https://kubelab\nkind: Config" // no trailing newline
	out, err := migrateExec(t, fileRegistry, fw, kubeconfig, "KUBECONFIG", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if fw.created["kubelab-kubeconfig/notes"] != kubeconfig {
		t.Errorf("created = %q, want %q byte-exact (no trim, nothing appended)", fw.created["kubelab-kubeconfig/notes"], kubeconfig)
	}
	if got := backendOfID(t, "KUBECONFIG"); got != "bw" {
		t.Errorf("registry backend = %q, want bw (flipped)", got)
	}
	if !strings.Contains(out, "migrated") {
		t.Errorf("want 'migrated', got %q", out)
	}
}

// TestSecretsMigrate_PreservesInteriorNewlines proves migrate no longer routes the
// age plaintext through EnvFor's single-line stripNewlines: a multi-line secret
// (BEEHIIV_DNS_RECORDS-shaped) must survive the cutover with its interior newlines
// intact, only the trailing one trimmed — the parity gate is where a regression here
// would actually surface, so this exercises migrateExec end-to-end, not ageValue in
// isolation (#612 B6).
func TestSecretsMigrate_PreservesInteriorNewlines(t *testing.T) {
	const multilineRegistry = `
version: 1
secrets:
  - id: DNS_RECORDS
    plane: app
    backend: age
    age: dns.records
    bw: { item: dns-records, field: notes }
    expose: { env: DNS_RECORDS }
    consumers: [local]
`
	fw := newFakeWriter()
	fw.notFound["dns-records"] = true // target item absent → --yes creates it
	multiline := "TXT record one\nTXT record two\nTXT record three\n"
	out, err := migrateExec(t, multilineRegistry, fw, multiline, "DNS_RECORDS", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	want := "TXT record one\nTXT record two\nTXT record three" // trailing \n trimmed, interior kept
	if fw.created["dns-records/notes"] != want {
		t.Errorf("created = %q, want %q (interior newlines preserved)", fw.created["dns-records/notes"], want)
	}
	if got := backendOfID(t, "DNS_RECORDS"); got != "bw" {
		t.Errorf("registry backend = %q, want bw (flipped)", got)
	}
	if !strings.Contains(out, "migrated") {
		t.Errorf("want 'migrated', got %q", out)
	}
}

// TestSecretsMigrate_EmptyAgeValueRefused proves ageValue's own empty-value guard
// (reimplemented locally now that it bypasses EnvFor) still fails loud rather than
// migrating a blank secret (#612 A1 parity).
func TestSecretsMigrate_EmptyAgeValueRefused(t *testing.T) {
	fw := newFakeWriter()
	_, err := migrateExec(t, migratableRegistry, fw, "\n", "NAN_API_KEY", "--yes")
	if err == nil || !strings.Contains(err.Error(), "empty value") {
		t.Fatalf("want an empty-value error, got %v", err)
	}
	if len(fw.sets) != 0 || len(fw.created) != 0 {
		t.Errorf("must write nothing on an empty age value: sets=%v created=%v", fw.sets, fw.created)
	}
}

// TestSecretsMigrate_UsesDeclaredFolder proves migrate threads the registry's
// bw.folder through to the create-item write, same as `set` — OPS-028 AC2.
func TestSecretsMigrate_UsesDeclaredFolder(t *testing.T) {
	const foldered = `
version: 1
secrets:
  - id: NAN_API_KEY
    plane: app
    backend: age
    age: nan.api-key
    bw: { item: nan-api-key, field: api-key, folder: Dotfiles/apps }
    expose: { env: NAN_API_KEY }
    consumers: [local]
`
	fw := newFakeWriter()
	fw.notFound["nan-api-key"] = true
	if _, err := migrateExec(t, foldered, fw, "nan-value\n", "NAN_API_KEY", "--yes"); err != nil {
		t.Fatal(err)
	}
	if fw.createdIn["nan-api-key"] != "new-Dotfiles/apps" {
		t.Errorf("createdIn = %v, want nan-api-key resolved via Dotfiles/apps", fw.createdIn)
	}
}

func TestSecretsMigrate_ParityMismatchAbortsBeforeFlip(t *testing.T) {
	fw := newFakeWriter()
	fw.tamper = func(v string) string { return v + "X" } // bw stores a different value
	_, err := migrateExec(t, migratableRegistry, fw, "nan-value", "NAN_API_KEY", "--yes")
	if err == nil || !strings.Contains(err.Error(), "parity mismatch") {
		t.Fatalf("want a parity-mismatch error, got %v", err)
	}
	if got := backendOfID(t, "NAN_API_KEY"); got != "age" {
		t.Errorf("registry backend = %q, want age (NOT flipped on parity failure)", got)
	}
}

func TestSecretsMigrate_Idempotent_AlreadyBw(t *testing.T) {
	const alreadyBw = `
version: 1
secrets:
  - id: ALREADY
    plane: app
    backend: bw
    bw: { item: it, field: password }
    expose: { env: ALREADY }
    consumers: [local]
`
	fw := newFakeWriter()
	fw.cur["it/password"] = "v" // resolves → verified
	out, err := migrateExec(t, alreadyBw, fw, "unused", "ALREADY")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "already bw") {
		t.Errorf("want 'already bw', got %q", out)
	}
	if len(fw.sets) != 0 || len(fw.created) != 0 {
		t.Errorf("already-bw migrate must write nothing: sets=%v created=%v", fw.sets, fw.created)
	}
	if got := backendOfID(t, "ALREADY"); got != "bw" {
		t.Errorf("registry backend = %q, want bw (unchanged)", got)
	}
}

// File secrets are no longer scope-guarded here — see TestSecretsMigrate_FileSecret_ByteExact.
func TestSecretsMigrate_ScopeGuards(t *testing.T) {
	cases := []struct {
		name, registry, id, wantErr string
	}{
		{
			name: "missing bw block", id: "NOBW", wantErr: "no bw:",
			registry: "version: 1\nsecrets:\n  - {id: NOBW, plane: app, backend: age, age: x, expose: {env: NOBW}, consumers: [local]}\n",
		},
		{
			name: "floor", id: "SSH_KEY", wantErr: "floor",
			registry: "version: 1\nsecrets:\n  - {id: SSH_KEY, plane: floor, backend: age-offline, age: id_ed25519, expose: {file: {var: SSH_KEY, path: \"~/.ssh/id\"}}, consumers: [local]}\n",
		},
		{
			name: "shared age source", id: "GH_A", wantErr: "shares its age source",
			registry: "version: 1\nsecrets:\n" +
				"  - {id: GH_A, plane: app, backend: age, age: github.token, bw: {item: gh-a, field: api-token}, expose: {env: GH_A}, consumers: [local]}\n" +
				"  - {id: GH_B, plane: app, backend: age, age: github.token, bw: {item: gh-b, field: api-token}, expose: {env: GH_B}, consumers: [local]}\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fw := newFakeWriter()
			_, err := migrateExec(t, c.registry, fw, "v", c.id, "--yes")
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("guard %q: err = %v, want one containing %q", c.name, err, c.wantErr)
			}
			if len(fw.sets) != 0 || len(fw.created) != 0 {
				t.Errorf("guard %q must write nothing: sets=%v created=%v", c.name, fw.sets, fw.created)
			}
		})
	}
}

func TestSecretsMigrate_DryRunInert(t *testing.T) {
	fw := newFakeWriter()
	fw.notFound["nan-api-key"] = true // target absent → "would create"
	out, err := migrateExec(t, migratableRegistry, fw, "nan-value", "NAN_API_KEY", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "would create") || !strings.Contains(out, "would flip registry") {
		t.Errorf("dry-run output = %q, want 'would create' + 'would flip registry'", out)
	}
	if len(fw.sets) != 0 || len(fw.created) != 0 {
		t.Errorf("--dry-run must write nothing: sets=%v created=%v", fw.sets, fw.created)
	}
	if got := backendOfID(t, "NAN_API_KEY"); got != "age" {
		t.Errorf("registry backend = %q, want age (--dry-run does not flip)", got)
	}
}
