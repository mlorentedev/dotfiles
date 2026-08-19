package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EscrowFileName is the fixed name of the age-encrypted full-vault DR escrow under
// sensitive/dr/ (ADR-028 §5). A single rolling file — git history is the version trail,
// so a retention policy is a separate concern.
const EscrowFileName = "bitwarden-export.age"

// BWExporter produces the entire Bitwarden vault as raw export bytes (JSON), in memory.
// The seam that keeps Backup unit-testable with no Bitwarden vault and no unlock — tests
// inject a fake; production is BWExport (a `bw sync` + `bw export` shell-out). The
// full-vault analog of BWReader/BWGet.
type BWExporter interface {
	Export() ([]byte, error)
}

// BWExport is the production BWExporter: `bw sync` then `bw export --format json --raw`,
// both `--nointeraction` so a locked vault or a missing BW_SESSION fails fast (bw's stderr
// surfaced) instead of blocking on a prompt. `--raw` keeps the export on stdout — it never
// touches a file. Thin I/O, verified by a live smoke, not in CI (the escrow analog of
// BWGet). The raw JSON is returned in memory; Backup pipes it straight into the encryptor.
type BWExport struct {
	Bin string // bw binary name/path; "" → "bw"
}

// Export refreshes the local cache then returns the full vault export. A failed sync
// (e.g. offline) aborts rather than silently escrowing a stale vault.
func (e BWExport) Export() ([]byte, error) {
	bin := e.Bin
	if bin == "" {
		bin = "bw"
	}
	if _, err := bwRun(bin, "sync"); err != nil {
		return nil, fmt.Errorf("bw sync: %w", exportLockHint(err))
	}
	out, err := bwRun(bin, "export", "--format", "json", "--raw")
	if err != nil {
		return nil, fmt.Errorf("bw export: %w", exportLockHint(err))
	}
	return out, nil
}

// exportLockHint explains the one asymmetry BUG-084 could not remove.
//
// `set`, `rotate` and `migrate` moved to the bw serve daemon, so their locked-vault
// errors now say "run `dotf secrets unlock`". Export cannot follow them: `bw serve`
// exposes NO export route — verified by enumerating the shipped @bitwarden/cli 2026.5.0
// router table, where the sole "/export" string is an *organization* export against the
// cloud API, not personal-vault export. So the escrow is permanently CLI-backed and
// permanently needs that binary's own session.
//
// Telling an operator to run `dotf secrets unlock` here would be actively wrong: it
// unlocks the daemon, the escrow still fails, and the message has taught them a fix that
// cannot work. This is why the DR escrow silently never existed until 2026-08-15 (#997):
// the failure was real, but nothing it printed pointed anywhere useful.
func exportLockHint(err error) error {
	if !isVaultLocked(err) {
		return err
	}
	return fmt.Errorf("%w: the escrow cannot go through the bw serve daemon — `bw serve` "+
		"exposes no export endpoint, so this is the one `dotf secrets` operation that needs "+
		"the bw CLI's own session. `dotf secrets unlock` will NOT fix it. Run:\n"+
		"    BW_SESSION=\"$(bw unlock --raw)\" dotf secrets backup\n"+
		"which confines the session to that one process rather than exporting it: %w",
		ErrBWVaultLocked, err)
}

// bwRun executes `bw <args...> --nointeraction`, returning stdout or an error carrying
// bw's stderr (parity with BWGet/BWPut — never a bare "exit status 1"). --nointeraction
// guarantees no prompt blocks a non-interactive escrow run.
func bwRun(bin string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, append(args, "--nointeraction")...) //nolint:gosec // args are fixed escrow subcommands
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return stdout.Bytes(), nil
}

// Encryptor encrypts plaintext to the given age recipient, returning ciphertext. The seam
// that keeps Backup testable with no age binary and no key; AgeEncrypt is the production
// default — the inverse of Decryptor/AgeDecrypt.
type Encryptor func(plaintext []byte, recipient string) ([]byte, error)

// AgeEncrypt shells out to `age -r <recipient>`, feeding plaintext on stdin and returning
// the binary ciphertext on stdout. The plaintext never touches disk (stdin pipe) — the
// mirror of AgeDecrypt's in-memory plaintext. age's stderr is surfaced on failure.
func AgeEncrypt(plaintext []byte, recipient string) ([]byte, error) {
	cmd := exec.Command("age", "-r", recipient)
	cmd.Stdin = bytes.NewReader(plaintext)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("age encrypt: %s", msg)
	}
	return stdout.Bytes(), nil
}

// RecipientFn derives the age recipient (public key) for the identity at keyPath. The seam
// Backup uses so a test needn't run age-keygen; AgeRecipient is production.
type RecipientFn func(keyPath string) (string, error)

// AgeRecipient runs `age-keygen -y <keyPath>` to print the public recipient of the private
// identity — the repo's established self-encrypt convention (load-secrets, the runbooks,
// and ci.yml all derive the recipient this way). The private key stays the single source
// of truth; no recipient is stored separately. A missing/unreadable key fails fast.
func AgeRecipient(keyPath string) (string, error) {
	cmd := exec.Command("age-keygen", "-y", keyPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("age-keygen -y %s: %s", keyPath, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// BackupConfig wires the DR-escrow run: the seams (all overridable for tests) plus the
// identity key and the destination dir. ADR-028 §5.
type BackupConfig struct {
	Exporter  BWExporter  // required: full-vault export (BWExport in prod)
	Recipient RecipientFn // nil → AgeRecipient
	Encrypt   Encryptor   // nil → AgeEncrypt
	Decrypt   Decryptor   // nil → AgeDecrypt (round-trip verify)
	KeyPath   string      // age identity — derives the recipient and verifies the artifact
	DestDir   string      // dir to write into (must be the checkout: env.RepoSensitiveDir + "dr")
	DestName  string      // "" → EscrowFileName
}

// Backup runs the disaster-recovery escrow: it exports the entire Bitwarden vault,
// encrypts it to the operator's own age recipient, writes the ciphertext atomically
// (0600) under DestDir, then verifies the on-disk artifact round-trips to the exact
// export. The vault plaintext is piped exporter→encryptor in memory and is NEVER written
// to disk; only the .age ciphertext lands. A verify failure removes the file. Returns the
// written path.
//
// Order matters for the no-partial-file guarantee: recipient, export, and encrypt all run
// BEFORE any write, so a locked vault or an absent key fails fast with nothing on disk.
func Backup(cfg BackupConfig) (string, error) {
	if cfg.Exporter == nil {
		return "", errors.New("backup: no exporter configured")
	}
	recipientFn := cfg.Recipient
	if recipientFn == nil {
		recipientFn = AgeRecipient
	}
	encrypt := cfg.Encrypt
	if encrypt == nil {
		encrypt = AgeEncrypt
	}
	name := cfg.DestName
	if name == "" {
		name = EscrowFileName
	}

	// 1. Recipient from the identity — fail fast if the key is absent/unreadable.
	recipient, err := recipientFn(cfg.KeyPath)
	if err != nil {
		return "", fmt.Errorf("backup: age identity required at %s to encrypt the escrow: %w", cfg.KeyPath, err)
	}
	if strings.TrimSpace(recipient) == "" {
		return "", fmt.Errorf("backup: empty age recipient derived from %s", cfg.KeyPath)
	}

	// 2. Export the full vault — fail fast if bw is locked/unreachable.
	plaintext, err := cfg.Exporter.Export()
	if err != nil {
		return "", fmt.Errorf("backup: %w", err)
	}
	if len(plaintext) == 0 {
		return "", errors.New("backup: bw export produced an empty vault — refusing to escrow nothing")
	}
	defer zero(plaintext) // best-effort scrub of the whole-vault plaintext after use

	// 3. Encrypt in memory; the plaintext never reaches disk.
	ciphertext, err := encrypt(plaintext, recipient)
	if err != nil {
		return "", fmt.Errorf("backup: %w", err)
	}

	// 4. Atomic 0600 write of the ciphertext only.
	if err := os.MkdirAll(cfg.DestDir, 0o700); err != nil {
		return "", fmt.Errorf("backup: create %s: %w", cfg.DestDir, err)
	}
	path := filepath.Join(cfg.DestDir, name)
	if err := AtomicWrite(path, ciphertext); err != nil {
		return "", fmt.Errorf("backup: %w", err)
	}

	// 5. Verify the on-disk artifact round-trips to the exact export.
	if err := verifyEscrow(path, plaintext, cfg); err != nil {
		_ = os.Remove(path) // never leave a corrupt/empty escrow behind
		return "", fmt.Errorf("backup: round-trip verify failed (removed %s): %w", name, err)
	}

	// 6. Describe what was escrowed, so a later check can ask whether the escrow
	// still matches the vault rather than only how old it is (#1077). Derived from
	// the plaintext already in memory — no extra call, no session of its own.
	//
	// Written AFTER the verify, and a failure here does NOT remove the escrow. The
	// escrow is the artifact that recovers the account; the manifest is bookkeeping
	// about it. Deleting a verified escrow over a bookkeeping file would invert the
	// priorities exactly when they matter most. The state a failure leaves behind —
	// escrow present, manifest absent — is the same one every escrow written before
	// this feature is in, and the freshness check treats it as SKIP-with-remedy
	// rather than as a fault.
	if err := writeManifest(filepath.Dir(path), plaintext); err != nil {
		return path, fmt.Errorf("backup: escrow written and verified at %s, but its manifest could not be: %w\n"+
			"re-run `dotf secrets backup` to mint one; the escrow itself is good", path, err)
	}
	return path, nil
}

// verifyEscrow decrypts the just-written escrow and asserts it round-trips to the exact
// exported plaintext and parses as a non-empty JSON document — the "verified round-trip"
// ADR-028 §3 gates age-file retirement (C6) on. Decrypting the FILE (not the in-memory
// ciphertext) also confirms the bytes that actually landed on disk are recoverable.
func verifyEscrow(path string, want []byte, cfg BackupConfig) error {
	decrypt := cfg.Decrypt
	if decrypt == nil {
		decrypt = AgeDecrypt
	}
	got, err := decrypt(path, cfg.KeyPath)
	if err != nil {
		return fmt.Errorf("decrypt back: %w", err)
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("decrypted escrow does not match the export (%d vs %d bytes)", len(got), len(want))
	}
	if !json.Valid(got) {
		return errors.New("escrow plaintext is not valid JSON (bw export format drift?)")
	}
	return nil
}

// zero overwrites b in place — a best-effort scrub of plaintext from process memory once
// the escrow is encrypted (defense in depth; Go's GC may still hold copies).
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// writeManifest derives the escrow's description and writes it beside the artifact.
// 0600 like everything else here, though it holds no secret: the directory is
// deny-by-default and a permissive file in it invites the assumption that the next
// one may be permissive too.
func writeManifest(destDir string, export []byte) error {
	m, err := ManifestFrom(export)
	if err != nil {
		return err
	}
	blob, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	return AtomicWrite(filepath.Join(destDir, ManifestFileName), append(blob, '\n'))
}
