package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The offline DR copy (ADR-033 D2, #1770) is a whole-disk VeraCrypt volume that
// holds exactly three files, named relative to wherever the volume is mounted.
const (
	USBKeyFile      = "key.txt"
	usbEscrowSubdir = "secrets/dr"
)

// DrillRecordFileName is the committed evidence of the last verification, beside
// the escrow and its manifest. Exact name: the directory's .gitignore is
// deny-by-default and un-ignores this file by name, never by glob.
const DrillRecordFileName = "drill-record.json"

// USBPayload is the volume's declared content, relative to its mount.
func USBPayload() []string {
	return []string{
		USBKeyFile,
		filepath.ToSlash(filepath.Join(usbEscrowSubdir, EscrowFileName)),
		filepath.ToSlash(filepath.Join(usbEscrowSubdir, ManifestFileName)),
	}
}

// USBConfig wires a verification or a refresh of the offline copy. Every seam is
// overridable so tests run with a temporary tree; production leaves them nil.
type USBConfig struct {
	Mount     string      // where the volume is mounted
	KeyPath   string      // this machine's age identity
	RepoDR    string      // the checkout's sensitive/dr: the escrow, its manifest, the record
	Host      string      // written into the record
	Recipient RecipientFn // nil → AgeRecipient
	Decrypt   Decryptor   // nil → AgeDecrypt
	Now       func() time.Time
	// IsMount reports whether Mount is a mounted volume rather than a directory
	// of the disk it sits on. nil → isMountPoint.
	IsMount func(dir string) (bool, error)
}

// USBBackupOptions are the two writes BackupUSB never does by default.
type USBBackupOptions struct {
	// InitIdentity installs this machine's identity on a stick that has none,
	// and only on an empty volume: a new stick, set up once.
	InitIdentity bool
	// Prune removes the files outside the declared payload.
	Prune bool
}

// USBReport is what a verification established. It carries the public
// recipient and counts, never key material and never plaintext.
type USBReport struct {
	Recipient string
	// RepoItems is the repo's escrow decrypted with the stick's identity: RECOVER
	// steps 1-3, the account-independent recovery floor.
	RepoItems int
	// USBItems is the stick's own escrow copy, decrypted with its identity: the
	// recovery that needs no repo clone at all.
	USBItems int
	// Current says the stick's escrow and manifest are the repo's, byte for byte.
	Current bool
	// Extra lists files on the volume outside the declared payload, OS metadata
	// aside.
	Extra []string
}

// DrillRecord is the committed evidence of a passing verification, read by
// `dotf doctor` on every machine. Nothing in it is a secret: a date, a host, the
// public recipient, a count and the manifest's digest.
type DrillRecord struct {
	VerifiedAt   string `json:"verified_at"` // RFC 3339, UTC
	Host         string `json:"host"`
	Recipient    string `json:"recipient"`
	Items        int    `json:"items"`
	EscrowDigest string `json:"escrow_digest"` // the manifest digest the stick held
	// RehearsedAt is the last full RECOVER rehearsal (import into a throwaway
	// account, resync, verify). A verification never sets it, and keeps it.
	RehearsedAt string `json:"rehearsed_at,omitempty"`
}

func (c USBConfig) withDefaults() USBConfig {
	if c.Recipient == nil {
		c.Recipient = AgeRecipient
	}
	if c.Decrypt == nil {
		c.Decrypt = AgeDecrypt
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.IsMount == nil {
		c.IsMount = isMountPoint
	}
	return c
}

func (c USBConfig) usbPath(rel string) string {
	return filepath.Join(c.Mount, filepath.FromSlash(rel))
}

// VerifyUSB checks the offline copy by consequence and, when every check
// passes, writes the drill record. It decrypts in memory only, prints nothing,
// and returns every failed check at once, so one run names everything to fix.
//
//  1. The stick's identity has this machine's recipient.
//  2. The repo's escrow decrypts with the stick's identity to the repo manifest.
//  3. The stick's escrow decrypts with the stick's identity to the stick's manifest.
//  4. The stick's escrow and manifest are the repo's (a stale copy fails).
//  5. Nothing else is on the volume.
func VerifyUSB(cfg USBConfig) (USBReport, error) {
	cfg = cfg.withDefaults()
	var rep USBReport
	var problems []string
	fail := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }

	if info, err := os.Stat(cfg.Mount); err != nil || !info.IsDir() {
		return rep, fmt.Errorf("usb: no mounted volume at %s", cfg.Mount)
	}
	usbKey := cfg.usbPath(USBKeyFile)
	usbRecipient, err := cfg.Recipient(usbKey)
	switch {
	case err != nil:
		fail("the stick's identity: %v", err)
	default:
		localRecipient, lerr := cfg.Recipient(cfg.KeyPath)
		switch {
		case lerr != nil:
			fail("this machine's identity: %v", lerr)
		case usbRecipient == "" || usbRecipient != localRecipient:
			// Two failed reads must not compare equal, so an empty one never matches.
			fail("the stick's identity is not this machine's: recipient %q, here %q", usbRecipient, localRecipient)
		default:
			rep.Recipient = usbRecipient
		}
	}

	if rep.Recipient != "" {
		n, err := escrowMatchesManifest(cfg.Decrypt, filepath.Join(cfg.RepoDR, EscrowFileName), filepath.Join(cfg.RepoDR, ManifestFileName), usbKey)
		if err != nil {
			fail("the repo's escrow with the stick's identity: %v", err)
		}
		rep.RepoItems = n
		n, err = escrowMatchesManifest(cfg.Decrypt, cfg.usbPath(USBPayload()[1]), cfg.usbPath(USBPayload()[2]), usbKey)
		if err != nil {
			fail("the stick's escrow: %v", err)
		}
		rep.USBItems = n
	}

	rep.Current = true
	for i, name := range []string{EscrowFileName, ManifestFileName} {
		same, err := sameFile(filepath.Join(cfg.RepoDR, name), cfg.usbPath(USBPayload()[i+1]))
		if err != nil || !same {
			rep.Current = false
		}
	}
	if !rep.Current {
		fail("the stick's escrow is not the repo's: refresh it with `dotf secrets usb backup --mount %s`", cfg.Mount)
	}

	if rep.Extra, err = usbExtra(cfg.Mount); err != nil {
		fail("list the volume: %v", err)
	} else if len(rep.Extra) > 0 {
		fail("files outside the declared payload: %s (remove them with `dotf secrets usb backup --mount %s --prune`)",
			strings.Join(rep.Extra, ", "), cfg.Mount)
	}

	if len(problems) > 0 {
		return rep, errors.New("usb verify failed:\n  - " + strings.Join(problems, "\n  - "))
	}
	return rep, writeDrillRecord(cfg, rep)
}

// escrowMatchesManifest decrypts escrow with key, in memory, and requires the
// manifest derived from the plaintext to equal the one stored beside it. It
// returns the item count.
func escrowMatchesManifest(decrypt Decryptor, escrow, manifest, key string) (int, error) {
	stored, err := readManifest(manifest)
	if err != nil {
		return 0, err
	}
	plaintext, err := decrypt(escrow, key)
	if err != nil {
		return 0, fmt.Errorf("decrypt %s: %w", filepath.Base(escrow), err)
	}
	defer zero(plaintext)
	got, err := ManifestFrom(plaintext)
	if err != nil {
		return 0, err
	}
	if got != stored {
		return got.Count, fmt.Errorf("decrypts to %d items (digest %.12s), but its manifest says %d (digest %.12s)",
			got.Count, got.Digest, stored.Count, stored.Digest)
	}
	return got.Count, nil
}

func readManifest(path string) (EscrowManifest, error) {
	var m EscrowManifest
	b, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("%s: %w", path, err)
	}
	return m, nil
}

func sameFile(a, b string) (bool, error) {
	x, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	y, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(x, y), nil
}

// usbExtra lists the files on the volume outside the declared payload.
func usbExtra(mount string) ([]string, error) {
	content, err := usbContent(mount)
	if err != nil {
		return nil, err
	}
	payload := USBPayload()
	var extra []string
	for _, rel := range content {
		if !slices.Contains(payload, rel) {
			extra = append(extra, rel)
		}
	}
	return extra, nil
}

// usbContent lists every file on the volume, relative to its mount. The
// metadata an OS writes on any volume it mounts is not the owner's content and
// is skipped: AppleDouble and Finder files, Spotlight and fsevents indexes, the
// trash, and Windows' recycle bin and volume information.
func usbContent(mount string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(mount, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == mount {
			return nil
		}
		if osMetadata(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(mount, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, err
}

func osMetadata(name string) bool {
	switch name {
	case ".DS_Store", ".Trashes", ".fseventsd", ".Spotlight-V100", ".TemporaryItems", "System Volume Information", "$RECYCLE.BIN":
		return true
	}
	return strings.HasPrefix(name, "._")
}

// writeDrillRecord writes the record of a passing verification, keeping the
// last rehearsal date of the record it replaces.
func writeDrillRecord(cfg USBConfig, rep USBReport) error {
	path := filepath.Join(cfg.RepoDR, DrillRecordFileName)
	stored, err := readManifest(filepath.Join(cfg.RepoDR, ManifestFileName))
	if err != nil {
		return err
	}
	rec := DrillRecord{
		VerifiedAt:   cfg.Now().UTC().Format(time.RFC3339),
		Host:         cfg.Host,
		Recipient:    rep.Recipient,
		Items:        rep.USBItems,
		EscrowDigest: stored.Digest,
	}
	if prev, err := ReadDrillRecord(cfg.RepoDR); err == nil {
		rec.RehearsedAt = prev.RehearsedAt
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWrite(path, append(b, '\n'))
}

// ReadDrillRecord reads the committed drill record from dir.
func ReadDrillRecord(dir string) (DrillRecord, error) {
	var rec DrillRecord
	b, err := os.ReadFile(filepath.Join(dir, DrillRecordFileName))
	if err != nil {
		return rec, err
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		return rec, fmt.Errorf("%s: %w", DrillRecordFileName, err)
	}
	return rec, nil
}

// BackupUSB refreshes the offline copy from the repo and then verifies it.
//
// It writes the private identity, so where it writes is the first thing it
// checks: the target must be a mounted volume, never a directory on the disk
// it sits on. A mount point that was created and never mounted (Linux's
// /media/secrets), or a typo one level up, would otherwise receive the age key
// in the clear. The identity is installed only when asked (InitIdentity) and
// only on a volume with nothing on it; a stick that holds a different identity
// is refused before anything is written, because it may hold the only copy of
// a key some other escrow still needs.
func BackupUSB(cfg USBConfig, opts USBBackupOptions) (USBReport, error) {
	cfg = cfg.withDefaults()
	if info, err := os.Stat(cfg.Mount); err != nil || !info.IsDir() {
		return USBReport{}, fmt.Errorf("usb: no mounted volume at %s", cfg.Mount)
	}
	switch mounted, err := cfg.IsMount(cfg.Mount); {
	case err != nil:
		return USBReport{}, fmt.Errorf("usb: is %s a mounted volume: %w", cfg.Mount, err)
	case !mounted:
		return USBReport{}, fmt.Errorf("usb: %s is not a mounted volume, it is a directory of the disk it sits on: mount the stick there first; nothing was written", cfg.Mount)
	}
	usbKey := cfg.usbPath(USBKeyFile)
	switch _, err := os.Stat(usbKey); {
	case err == nil:
		usbRecipient, err := cfg.Recipient(usbKey)
		if err != nil {
			return USBReport{}, fmt.Errorf("usb: the stick's identity: %w", err)
		}
		localRecipient, err := cfg.Recipient(cfg.KeyPath)
		if err != nil {
			return USBReport{}, fmt.Errorf("usb: this machine's identity: %w", err)
		}
		if usbRecipient == "" || usbRecipient != localRecipient {
			return USBReport{}, fmt.Errorf("usb: the stick holds another identity (recipient %q, here %q); nothing was written", usbRecipient, localRecipient)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return USBReport{}, fmt.Errorf("usb: %w", err)
	case !opts.InitIdentity:
		return USBReport{}, fmt.Errorf("usb: the stick at %s has no %s; to set up a new stick, re-run with --init-identity; nothing was written", cfg.Mount, USBKeyFile)
	default:
		content, err := usbContent(cfg.Mount)
		if err != nil {
			return USBReport{}, fmt.Errorf("usb: list the volume: %w", err)
		}
		if len(content) > 0 {
			return USBReport{}, fmt.Errorf("usb: --init-identity sets up an empty stick, and %s holds %s; nothing was written", cfg.Mount, strings.Join(content, ", "))
		}
		if err := copyFile(cfg.KeyPath, usbKey); err != nil {
			return USBReport{}, fmt.Errorf("usb: install the identity: %w", err)
		}
	}

	for i, name := range []string{EscrowFileName, ManifestFileName} {
		if err := copyFile(filepath.Join(cfg.RepoDR, name), cfg.usbPath(USBPayload()[i+1])); err != nil {
			return USBReport{}, fmt.Errorf("usb: copy %s: %w", name, err)
		}
	}

	if opts.Prune {
		extra, err := usbExtra(cfg.Mount)
		if err != nil {
			return USBReport{}, fmt.Errorf("usb: list the volume: %w", err)
		}
		for _, rel := range extra {
			if err := os.Remove(cfg.usbPath(rel)); err != nil {
				return USBReport{}, fmt.Errorf("usb: prune %s: %w", rel, err)
			}
		}
	}
	return VerifyUSB(cfg)
}

// copyFile copies src over dst through a temp file and a rename, so an
// interrupted copy never leaves half a file on the stick. It does not chmod:
// os.CreateTemp already creates 0600 where the filesystem keeps modes, and on
// the exFAT the runbook recommends, which keeps none, Linux refuses a chmod
// with EPERM. The volume's access control is its VeraCrypt password.
func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	defer zero(b)
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op once the rename succeeds
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
