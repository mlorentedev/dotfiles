package secrets

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// usbFixture is a repo escrow encrypted to a real age identity, a mounted
// "stick" (an empty directory) and this machine's key. Real age, not the seams:
// the claim under test is that the stick's identity opens the escrow, and only
// a real decryption can say that.
type usbFixture struct {
	cfg    USBConfig
	export []byte
}

func requireAge(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"age", "age-keygen"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not on PATH", bin)
		}
	}
}

func newIdentity(t *testing.T, dir string) string {
	t.Helper()
	key := filepath.Join(dir, "key.txt")
	if out, err := exec.Command("age-keygen", "-o", key).CombinedOutput(); err != nil {
		t.Fatalf("age-keygen: %v: %s", err, out)
	}
	return key
}

// writeEscrow encrypts export to key's recipient at dir/bitwarden-export.age and
// writes the manifest derived from it beside it.
func writeEscrow(t *testing.T, dir, key string, export []byte) {
	t.Helper()
	recipient, err := AgeRecipient(key)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("age", "-r", recipient, "-o", filepath.Join(dir, EscrowFileName))
	cmd.Stdin = strings.NewReader(string(export))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("age encrypt: %v: %s", err, out)
	}
	m, err := ManifestFrom(export)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func exportOf(ids ...string) []byte {
	var items []string
	for _, id := range ids {
		items = append(items, `{"id":"`+id+`","revisionDate":"2026-10-01T00:00:00Z","login":{"password":"hunter2"}}`)
	}
	return []byte(`{"items":[` + strings.Join(items, ",") + `]}`)
}

func newUSBFixture(t *testing.T) usbFixture {
	t.Helper()
	requireAge(t)
	home, repoDR, mount := t.TempDir(), t.TempDir(), t.TempDir()
	key := newIdentity(t, home)
	export := exportOf("a", "b", "c")
	writeEscrow(t, repoDR, key, export)
	now := time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	return usbFixture{
		cfg:    USBConfig{Mount: mount, KeyPath: key, RepoDR: repoDR, Host: "macmini", Now: func() time.Time { return now }},
		export: export,
	}
}

func readRecord(t *testing.T, dir string) DrillRecord {
	t.Helper()
	rec, err := ReadDrillRecord(dir)
	if err != nil {
		t.Fatalf("drill record: %v", err)
	}
	return rec
}

func TestBackupUSB_FillsAnEmptyStickAndRecordsTheDrill(t *testing.T) {
	f := newUSBFixture(t)
	rep, err := BackupUSB(f.cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.RepoItems != 3 || rep.USBItems != 3 || !rep.Current || len(rep.Extra) != 0 {
		t.Fatalf("report = %+v", rep)
	}
	for _, rel := range USBPayload() {
		if _, err := os.Stat(filepath.Join(f.cfg.Mount, filepath.FromSlash(rel))); err != nil {
			t.Errorf("payload %s: %v", rel, err)
		}
	}
	rec := readRecord(t, f.cfg.RepoDR)
	want, _ := ManifestFrom(f.export)
	if rec.Items != 3 || rec.EscrowDigest != want.Digest || rec.Host != "macmini" || rec.VerifiedAt != "2026-10-13T09:00:00Z" || rec.Recipient != rep.Recipient {
		t.Fatalf("record = %+v", rec)
	}
}

// The record is committed and the report is printed: neither may carry key
// material or plaintext.
func TestUSB_RecordAndReportCarryNoSecret(t *testing.T) {
	f := newUSBFixture(t)
	rep, err := BackupUSB(f.cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	record, err := os.ReadFile(filepath.Join(f.cfg.RepoDR, DrillRecordFileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, out := range []string{string(record), (func() string { b, _ := json.Marshal(rep); return string(b) })()} {
		for _, secret := range []string{"AGE-SECRET-KEY", "hunter2"} {
			if strings.Contains(out, secret) {
				t.Errorf("%q leaked into %s", secret, out)
			}
		}
	}
}

// The mutation the drill exists to catch: a manifest that does not describe its
// escrow. Off by one must fail, and must not record a pass.
func TestVerifyUSB_ManifestOffByOneFails(t *testing.T) {
	f := newUSBFixture(t)
	if _, err := BackupUSB(f.cfg, false); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(f.cfg.RepoDR, DrillRecordFileName)); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{f.cfg.RepoDR, filepath.Join(f.cfg.Mount, filepath.FromSlash(usbEscrowSubdir))} {
		path := filepath.Join(dir, ManifestFileName)
		m, err := readManifest(path)
		if err != nil {
			t.Fatal(err)
		}
		m.Count++
		b, _ := json.Marshal(m)
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := VerifyUSB(f.cfg)
	if err == nil || !strings.Contains(err.Error(), "decrypts to 3 items") || !strings.Contains(err.Error(), "manifest says 4") {
		t.Fatalf("err = %v", err)
	}
	if _, err := ReadDrillRecord(f.cfg.RepoDR); !os.IsNotExist(err) {
		t.Fatalf("a failed verification wrote a record: %v", err)
	}
}

func TestVerifyUSB_StaleCopyNamesTheRefresh(t *testing.T) {
	f := newUSBFixture(t)
	if _, err := BackupUSB(f.cfg, false); err != nil {
		t.Fatal(err)
	}
	writeEscrow(t, f.cfg.RepoDR, f.cfg.KeyPath, exportOf("a", "b", "c", "d"))
	rep, err := VerifyUSB(f.cfg)
	if err == nil || rep.Current || !strings.Contains(err.Error(), "dotf secrets usb backup --mount") {
		t.Fatalf("rep = %+v, err = %v", rep, err)
	}
	if rep.RepoItems != 4 || rep.USBItems != 3 {
		t.Fatalf("both escrows should still decrypt: %+v", rep)
	}
}

// A stick that holds another identity may hold the only copy of a key some
// other escrow needs: refuse, and write nothing.
func TestBackupUSB_RefusesAnotherIdentity(t *testing.T) {
	f := newUSBFixture(t)
	other := newIdentity(t, t.TempDir())
	b, err := os.ReadFile(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.cfg.Mount, USBKeyFile), b, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = BackupUSB(f.cfg, false)
	if err == nil || !strings.Contains(err.Error(), "another identity") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.cfg.Mount, filepath.FromSlash(usbEscrowSubdir))); !os.IsNotExist(err) {
		t.Fatalf("the escrow was copied onto a stick that was refused: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(f.cfg.Mount, USBKeyFile))
	if string(got) != string(b) {
		t.Fatal("the stick's identity was overwritten")
	}
}

func TestUSB_ExtraFilesFailUntilPruned(t *testing.T) {
	f := newUSBFixture(t)
	stray := filepath.Join(f.cfg.Mount, "secrets", "github.secret")
	if err := os.MkdirAll(filepath.Dir(stray), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{stray, filepath.Join(f.cfg.Mount, ".DS_Store"), filepath.Join(f.cfg.Mount, "._key.txt"),
		filepath.Join(f.cfg.Mount, ".Spotlight-V100", "store.db")} {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rep, err := BackupUSB(f.cfg, false)
	if err == nil || len(rep.Extra) != 1 || rep.Extra[0] != "secrets/github.secret" {
		t.Fatalf("rep = %+v, err = %v", rep, err)
	}
	if _, err := ReadDrillRecord(f.cfg.RepoDR); !os.IsNotExist(err) {
		t.Fatalf("a stick with extra files recorded a pass: %v", err)
	}
	if _, err := BackupUSB(f.cfg, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Fatalf("prune left %s: %v", stray, err)
	}
	if _, err := os.Stat(filepath.Join(f.cfg.Mount, ".DS_Store")); err != nil {
		t.Fatalf("prune removed OS metadata: %v", err)
	}
}

func TestVerifyUSB_KeepsTheRehearsalDate(t *testing.T) {
	f := newUSBFixture(t)
	if _, err := BackupUSB(f.cfg, false); err != nil {
		t.Fatal(err)
	}
	rec := readRecord(t, f.cfg.RepoDR)
	rec.RehearsedAt = "2026-09-01"
	b, _ := json.Marshal(rec)
	if err := os.WriteFile(filepath.Join(f.cfg.RepoDR, DrillRecordFileName), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyUSB(f.cfg); err != nil {
		t.Fatal(err)
	}
	if got := readRecord(t, f.cfg.RepoDR).RehearsedAt; got != "2026-09-01" {
		t.Fatalf("rehearsed_at = %q", got)
	}
}

// Two failed reads must not compare equal: an empty recipient never matches.
func TestVerifyUSB_EmptyRecipientsNeverMatch(t *testing.T) {
	mount := t.TempDir()
	cfg := USBConfig{
		Mount: mount, KeyPath: "k", RepoDR: t.TempDir(),
		Recipient: func(string) (string, error) { return "", nil },
		Decrypt: func(string, string) ([]byte, error) {
			t.Fatal("decrypted with an unverified identity")
			return nil, nil
		},
	}
	_, err := VerifyUSB(cfg)
	if err == nil || !strings.Contains(err.Error(), "not this machine's") {
		t.Fatalf("err = %v", err)
	}
}

func TestVerifyUSB_NoVolume(t *testing.T) {
	_, err := VerifyUSB(USBConfig{Mount: filepath.Join(t.TempDir(), "absent")})
	if err == nil || !strings.Contains(err.Error(), "no mounted volume") {
		t.Fatalf("err = %v", err)
	}
}
