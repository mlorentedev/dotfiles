package cmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// fakeWriter is one object playing BWReader + BWWriter + BWCreator so `set` runs with
// no unlocked Bitwarden. cur holds existing item/field values; notFound items report
// ErrBWItemNotFound (absent); locked makes every read fail with a generic (locked)
// error; sets/created record writes so tests assert exactly what was written.
type fakeWriter struct {
	cur       map[string]string
	notFound  map[string]bool
	locked    bool
	sets      map[string]string
	created   map[string]string
	createdIn map[string]string   // item -> folder name resolved at creation (OPS-028)
	folders   map[string]string   // folder name -> id; absent name -> ResolveFolder creates "new-<name>"
	tamper    func(string) string // optional: corrupt the STORED value so a read-back differs (parity tests)
}

// store reflects a write into cur so a subsequent Field read-back sees it (migrate's
// parity gate); tamper, when set, mangles the stored value to force a mismatch.
func (f *fakeWriter) store(item, field, value string) {
	if f.tamper != nil {
		value = f.tamper(value)
	}
	f.cur[item+"/"+field] = value
}

func newFakeWriter() *fakeWriter {
	return &fakeWriter{
		cur:       map[string]string{},
		notFound:  map[string]bool{},
		sets:      map[string]string{},
		created:   map[string]string{},
		createdIn: map[string]string{},
		folders:   map[string]string{},
	}
}

func (f *fakeWriter) Field(item, field string) (string, error) {
	if f.locked {
		return "", fmt.Errorf("bw get item %q: Vault is locked.", item)
	}
	if f.notFound[item] {
		return "", fmt.Errorf("%w: %q", secrets.ErrBWItemNotFound, item)
	}
	if v, ok := f.cur[item+"/"+field]; ok {
		return v, nil
	}
	return "", fmt.Errorf("%w: %q", secrets.ErrBWFieldNotFound, field)
}

func (f *fakeWriter) SetField(item, field, value string) error {
	f.sets[item+"/"+field] = value
	f.store(item, field, value)
	return nil
}

func (f *fakeWriter) CreateItem(item, field, value, folderID string) error {
	f.created[item+"/"+field] = value
	f.createdIn[item] = folderID
	delete(f.notFound, item) // the item now exists → subsequent reads resolve
	f.store(item, field, value)
	return nil
}

// ResolveFolder mimics BWPut.ResolveFolder's name→id + create-if-absent contract:
// empty name is a no-op, an unseen name is "created" (idempotent — a repeat call
// finds it in folders and returns the same id, never a duplicate).
func (f *fakeWriter) ResolveFolder(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	if id, ok := f.folders[name]; ok {
		return id, nil
	}
	id := "new-" + name
	f.folders[name] = id
	return id, nil
}

func useBwWriter(t *testing.T, w bwWriteClient) {
	t.Helper()
	old := bwWriter
	bwWriter = w
	t.Cleanup(func() { bwWriter = old })
}

// forceNonInteractive pins stdinIsTerminal to false so tests exercise the piped path
// (value from cmd stdin) deterministically — the hidden TTY prompt is live/manual.
func forceNonInteractive(t *testing.T) {
	t.Helper()
	old := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdinIsTerminal = old })
}

const setRegistry = `
version: 1
secrets:
  - {id: bw-token, plane: app, backend: bw, bw: {item: openai, field: api-key}, expose: {env: OPENAI_API_KEY}}
  - {id: bw-multi, plane: app, backend: bw, bw: {item: x-twitter}, expose: {env: {X_API_KEY: {field: api-key}, X_SECRET: {field: api-secret}}}}
  - {id: bw-file, plane: infra, backend: bw, bw: {item: kube, field: notes}, expose: {file: {var: KUBECONFIG, path: "~/.kube/c"}}}
  - {id: bw-foldered, plane: app, backend: bw, bw: {item: foldered-item, field: api-key, folder: apps}, expose: {env: FOLDERED_KEY}}
`

func runSetOut(t *testing.T, fw *fakeWriter, stdin string, args ...string) (string, error) {
	t.Helper()
	useTempRegistry(t, setRegistry)
	useBwReader(t, fw)
	useBwWriter(t, fw)
	forceNonInteractive(t)
	var out bytes.Buffer
	cmd := newSecretsSetCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestSecretsSet_IdempotentUnchanged(t *testing.T) {
	fw := newFakeWriter()
	fw.cur["openai/api-key"] = "current-value"
	out, err := runSetOut(t, fw, "current-value", "bw-token")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unchanged") {
		t.Errorf("want 'unchanged', got %q", out)
	}
	if len(fw.sets) != 0 || len(fw.created) != 0 {
		t.Errorf("idempotent run must write nothing: sets=%v created=%v", fw.sets, fw.created)
	}
}

func TestSecretsSet_UpdateOnChange_EnvTrimmed(t *testing.T) {
	fw := newFakeWriter()
	fw.cur["openai/api-key"] = "old"
	out, err := runSetOut(t, fw, "new-value\n", "bw-token") // trailing newline must be trimmed
	if err != nil {
		t.Fatal(err)
	}
	if fw.sets["openai/api-key"] != "new-value" {
		t.Errorf("env value stored = %q, want %q (trailing newline trimmed)", fw.sets["openai/api-key"], "new-value")
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("want 'updated', got %q", out)
	}
}

func TestSecretsSet_FileShape_BytesExact(t *testing.T) {
	fw := newFakeWriter()
	fw.cur["kube/notes"] = "old"
	if _, err := runSetOut(t, fw, "line1\nline2\n", "bw-file"); err != nil {
		t.Fatal(err)
	}
	if fw.sets["kube/notes"] != "line1\nline2\n" {
		t.Errorf("file value stored = %q, want byte-exact multi-line", fw.sets["kube/notes"])
	}
}

func TestSecretsSet_Disambiguation(t *testing.T) {
	fw := newFakeWriter()
	_, err := runSetOut(t, fw, "v", "bw-multi") // no [var] on a multi-var secret
	if err == nil || !strings.Contains(err.Error(), "X_API_KEY") || !strings.Contains(err.Error(), "X_SECRET") {
		t.Errorf("multi-var with no [var] must error listing vars, got %v", err)
	}

	fw2 := newFakeWriter() // x-twitter/api-secret absent -> append the field
	if _, err := runSetOut(t, fw2, "secret-v", "bw-multi", "X_SECRET"); err != nil {
		t.Fatal(err)
	}
	if fw2.sets["x-twitter/api-secret"] != "secret-v" {
		t.Errorf("disambiguated write = %v, want x-twitter/api-secret=secret-v", fw2.sets)
	}
}

func TestSecretsSet_CreateAbsent_Gated(t *testing.T) {
	// --yes -> CreateItem
	fw := newFakeWriter()
	fw.notFound["openai"] = true
	out, err := runSetOut(t, fw, "fresh", "bw-token", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if fw.created["openai/api-key"] != "fresh" {
		t.Errorf("create recorded = %v, want openai/api-key=fresh", fw.created)
	}
	if !strings.Contains(out, "created") {
		t.Errorf("want 'created', got %q", out)
	}

	// no --yes, non-interactive -> error mentioning --yes, no create
	fw2 := newFakeWriter()
	fw2.notFound["openai"] = true
	_, err = runSetOut(t, fw2, "fresh", "bw-token")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Errorf("non-interactive create without --yes must error mentioning --yes, got %v", err)
	}
	if len(fw2.created) != 0 {
		t.Errorf("must not create without --yes: %v", fw2.created)
	}

	// locked vault -> fail loud, never create, even with --yes (the security crux)
	fw3 := newFakeWriter()
	fw3.locked = true
	_, err = runSetOut(t, fw3, "fresh", "bw-token", "--yes")
	if err == nil {
		t.Error("locked vault must fail loud")
	}
	if len(fw3.created) != 0 {
		t.Errorf("locked vault must NEVER create (could duplicate a hidden item): %v", fw3.created)
	}
}

// TestSecretsSet_CreateAbsent_UsesDeclaredFolder proves createAbsent resolves the
// registry's bw.folder and threads the resolved id into CreateItem — OPS-028 AC2, the
// end-to-end counterpart of TestNewItemBody_Folder's pure JSON-body check.
func TestSecretsSet_CreateAbsent_UsesDeclaredFolder(t *testing.T) {
	fw := newFakeWriter()
	fw.notFound["foldered-item"] = true
	if _, err := runSetOut(t, fw, "fresh", "bw-foldered", "--yes"); err != nil {
		t.Fatal(err)
	}
	if fw.createdIn["foldered-item"] != "new-apps" {
		t.Errorf("createdIn = %v, want foldered-item resolved via apps", fw.createdIn)
	}
}

// TestSecretsSet_CreateAbsent_NoFolderDeclared proves the unfoldered case is unchanged
// — an empty bw.folder resolves to an empty folderID (OPS-028 AC5, no regression).
func TestSecretsSet_CreateAbsent_NoFolderDeclared(t *testing.T) {
	fw := newFakeWriter()
	fw.notFound["openai"] = true
	if _, err := runSetOut(t, fw, "fresh", "bw-token", "--yes"); err != nil {
		t.Fatal(err)
	}
	if got, ok := fw.createdIn["openai"]; !ok || got != "" {
		t.Errorf("createdIn[openai] = %q, want empty (no folder declared)", got)
	}
}

// TestSecretsSet_DryRun_NeverResolvesFolder proves --dry-run never triggers folder
// resolution — ResolveFolder can CREATE a Bitwarden folder as a side effect, so a
// dry-run declaring intent must not touch the vault at all (OPS-028).
func TestSecretsSet_DryRun_NeverResolvesFolder(t *testing.T) {
	fw := newFakeWriter()
	fw.notFound["foldered-item"] = true
	if _, err := runSetOut(t, fw, "fresh", "bw-foldered", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if len(fw.folders) != 0 {
		t.Errorf("dry-run must never resolve/create a folder: %v", fw.folders)
	}
}

func TestSecretsSet_EmptyRefused(t *testing.T) {
	fw := newFakeWriter()
	fw.cur["openai/api-key"] = "x"
	_, err := runSetOut(t, fw, "\n", "bw-token") // trims to empty
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("empty value must be refused, got %v", err)
	}
	if len(fw.sets) != 0 {
		t.Error("an empty value must not write")
	}
}

func TestSecretsSet_DryRunInert(t *testing.T) {
	fw := newFakeWriter()
	fw.cur["openai/api-key"] = "old"
	out, err := runSetOut(t, fw, "new", "bw-token", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "would update") {
		t.Errorf("want 'would update', got %q", out)
	}
	if len(fw.sets) != 0 {
		t.Error("--dry-run must not write")
	}

	fw2 := newFakeWriter()
	fw2.notFound["openai"] = true
	out, err = runSetOut(t, fw2, "new", "bw-token", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "would create") {
		t.Errorf("want 'would create', got %q", out)
	}
	if len(fw2.created) != 0 {
		t.Error("--dry-run must not create")
	}
}
