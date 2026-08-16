package secrets

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sampleExport stands in for `bw export --format json` — a small but valid JSON document.
const sampleExport = `{"encrypted":false,"items":[{"name":"x","login":{"password":"p"}}]}`

// fakeExporter is the BWExporter seam fake: a canned export or a canned error.
type fakeExporter struct {
	data []byte
	err  error
}

func (f fakeExporter) Export() ([]byte, error) { return f.data, f.err }

// b64Encrypt / b64DecryptFile are a reversible stand-in for age, so Backup's round-trip is
// exercised with no age binary and no key. Ciphertext is base64 — it does not contain the
// raw plaintext, so "no plaintext on disk" is a real structural assertion.
func b64Encrypt(pt []byte, _ string) ([]byte, error) {
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(pt)))
	base64.StdEncoding.Encode(enc, pt)
	return enc, nil
}

func b64DecryptFile(path, _ string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(dec, data)
	if err != nil {
		return nil, err
	}
	return dec[:n], nil
}

// okConfig wires a fully-faked, passing Backup over a fresh temp dest dir.
func okConfig(t *testing.T, exp BWExporter) BackupConfig {
	t.Helper()
	return BackupConfig{
		Exporter:  exp,
		Recipient: func(string) (string, error) { return "age1fakerecipient", nil },
		Encrypt:   b64Encrypt,
		Decrypt:   b64DecryptFile,
		KeyPath:   filepath.Join(t.TempDir(), "key.txt"),
		DestDir:   filepath.Join(t.TempDir(), "dr"),
	}
}

func TestBackup_WritesCiphertext_0600(t *testing.T) {
	cfg := okConfig(t, fakeExporter{data: []byte(sampleExport)})
	path, err := Backup(cfg)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if filepath.Base(path) != EscrowFileName {
		t.Errorf("escrow name = %q, want %q", filepath.Base(path), EscrowFileName)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := b64Encrypt([]byte(sampleExport), "")
	if !bytes.Equal(got, want) {
		t.Errorf("on-disk bytes are not the ciphertext the encryptor produced")
	}
	if runtime.GOOS != "windows" { // POSIX perm bits are only meaningful off Windows
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("escrow mode = %v, want 0600", fi.Mode().Perm())
		}
	}
}

func TestBackup_NoPlaintextOnDisk(t *testing.T) {
	cfg := okConfig(t, fakeExporter{data: []byte(sampleExport)})
	path, err := Backup(cfg)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("dest dir holds %d files, want exactly the .age escrow", len(entries))
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			t.Errorf("a plaintext json export was written to disk: %s", e.Name())
		}
	}
	got, _ := os.ReadFile(path)
	if bytes.Contains(got, []byte(sampleExport)) {
		t.Errorf("the escrow file contains the raw plaintext export")
	}
}

func TestBackup_RecipientThreadedToEncryptor(t *testing.T) {
	var gotRecipient string
	cfg := okConfig(t, fakeExporter{data: []byte(sampleExport)})
	cfg.Recipient = func(string) (string, error) { return "age1specific", nil }
	cfg.Encrypt = func(pt []byte, r string) ([]byte, error) {
		gotRecipient = r
		return b64Encrypt(pt, r)
	}
	if _, err := Backup(cfg); err != nil {
		t.Fatal(err)
	}
	if gotRecipient != "age1specific" {
		t.Errorf("recipient passed to encryptor = %q, want age1specific", gotRecipient)
	}
}

func TestBackup_RoundTripVerifyPasses(t *testing.T) {
	cfg := okConfig(t, fakeExporter{data: []byte(sampleExport)})
	if _, err := Backup(cfg); err != nil {
		t.Fatalf("a clean round-trip must verify: %v", err)
	}
}

func TestBackup_TamperedCiphertext_RemovesFileAndErrors(t *testing.T) {
	cfg := okConfig(t, fakeExporter{data: []byte(sampleExport)})
	// Decrypt returns bytes that don't match the export → round-trip mismatch.
	cfg.Decrypt = func(string, string) ([]byte, error) { return []byte(`{"items":[]}`), nil }
	if _, err := Backup(cfg); err == nil {
		t.Fatal("expected a round-trip verify failure")
	}
	if _, err := os.Stat(filepath.Join(cfg.DestDir, EscrowFileName)); !os.IsNotExist(err) {
		t.Errorf("a corrupt escrow must be removed on verify failure")
	}
}

func TestBackup_NonJSONExport_FailsVerify(t *testing.T) {
	cfg := okConfig(t, fakeExporter{data: []byte("this is not json")})
	if _, err := Backup(cfg); err == nil {
		t.Fatal("expected a non-JSON export to fail the verify (format drift guard)")
	}
	if _, err := os.Stat(filepath.Join(cfg.DestDir, EscrowFileName)); !os.IsNotExist(err) {
		t.Errorf("the unverifiable escrow must be removed")
	}
}

func TestBackup_ExporterError_NoFile(t *testing.T) {
	cfg := okConfig(t, fakeExporter{err: fmt.Errorf("bw export: vault is locked")})
	if _, err := Backup(cfg); err == nil {
		t.Fatal("expected the exporter error to surface")
	}
	if _, err := os.Stat(filepath.Join(cfg.DestDir, EscrowFileName)); !os.IsNotExist(err) {
		t.Errorf("no escrow may be written when the export fails")
	}
}

func TestBackup_RecipientError_NoFile(t *testing.T) {
	cfg := okConfig(t, fakeExporter{data: []byte(sampleExport)})
	cfg.Recipient = func(string) (string, error) { return "", fmt.Errorf("age key not found") }
	if _, err := Backup(cfg); err == nil {
		t.Fatal("expected the recipient error to surface")
	}
	if _, err := os.Stat(filepath.Join(cfg.DestDir, EscrowFileName)); !os.IsNotExist(err) {
		t.Errorf("no escrow may be written when the recipient cannot be derived")
	}
}

func TestBackup_EmptyExport_Refused(t *testing.T) {
	cfg := okConfig(t, fakeExporter{data: []byte{}})
	if _, err := Backup(cfg); err == nil {
		t.Fatal("expected a refusal to escrow an empty vault")
	}
}

func TestBackup_NilExporter_Errors(t *testing.T) {
	if _, err := Backup(BackupConfig{}); err == nil {
		t.Fatal("expected an error when no exporter is configured")
	}
}

// TestExportLockHint_NamesTheOnlyInvocationThatWorks is AC4 of BUG-084.
//
// Export is the one write-side operation with no daemon equivalent, so its locked-vault
// error must NOT send the operator to `dotf secrets unlock` (which unlocks the daemon and
// leaves the escrow failing). It must name the BW_SESSION form, and say why.
func TestExportLockHint_NamesTheOnlyInvocationThatWorks(t *testing.T) {
	got := exportLockHint(errors.New("Vault is locked."))
	if !errors.Is(got, ErrBWVaultLocked) {
		t.Fatalf("must wrap ErrBWVaultLocked, got %v", got)
	}
	msg := got.Error()
	for _, want := range []string{
		`BW_SESSION="$(bw unlock --raw)" dotf secrets backup`,
		"no export endpoint",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("export lock error must contain %q, got: %s", want, msg)
		}
	}
	// The critical negative: it must not teach the fix that cannot work here.
	if strings.Contains(msg, "run `dotf secrets unlock`") {
		t.Fatalf("export error must not prescribe `dotf secrets unlock` — it does not fix export: %s", msg)
	}
}

// TestExportLockHint_PassesThroughUnrelatedErrors: an offline sync failure must not be
// reported as a lock problem.
func TestExportLockHint_PassesThroughUnrelatedErrors(t *testing.T) {
	orig := errors.New("Failed to fetch: getaddrinfo ENOTFOUND")
	if got := exportLockHint(orig); got != orig { //nolint:errorlint // identity is the assertion
		t.Fatalf("unrelated error was rewritten: %v", got)
	}
}
