package doctor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// roundTripSentinel is the fixed plaintext the age root-of-trust check encrypts
// then decrypts. It proves the *key*, not the *plaintext*, so any constant works;
// it is not a secret.
const roundTripSentinel = "dotf-doctor-age-round-trip"

// checkSecretsTooling verifies the two binaries the ADR-028 secrets-governance
// model depends on are present, plus that the age identity key is present AND
// actually decrypts. It is the governance hook the ADR (§Mitigations) and the
// secrets runbook both name.
//
// Absent bw or age is a FAIL: neither the live SSOT (Bitwarden) nor the DR /
// bootstrap floor (age) can run without them. A missing age key is a WARN, not a
// FAIL — a freshly provisioned machine legitimately has the tools before its key
// is restored from the offline backup (the recover runbook), and FAILing there
// would block an otherwise-healthy setup.
//
// When the key IS present and age/age-keygen are available, the check goes one
// step further than "the file exists": it round-trips a sentinel through the key
// (#518). A present-but-broken key (corrupt, wrong, unreadable) is a FAIL — a red
// check at setup time instead of a silent surprise at recover time.
func checkSecretsTooling(sys *System, rep *Report) {
	rep.Section("Secrets tooling")

	if sys.has("bw") {
		// Says only what PATH presence proves. The old wording ("bw (Bitwarden
		// CLI — live secrets SSOT) found") read as a health verdict on the SSOT
		// itself and stayed green through a 45-day dead session (BUG-074).
		// Reachability is a separate claim, made by checkBitwardenReach.
		rep.Pass("bw (Bitwarden CLI) installed — reachability checked separately")
	} else {
		rep.Fail("bw not in PATH — run 'dotf tools install' (ADR-028 live SSOT)")
	}

	hasAge := sys.has("age")
	if hasAge {
		rep.Pass("age (DR escrow + bootstrap floor) found")
	} else {
		rep.Fail("age not in PATH — re-run setup (ADR-028 DR floor)")
	}

	keyPath := sys.Getenv("AGE_KEY_PATH")
	if keyPath == "" {
		keyPath = filepath.Join(sys.home(), ".config", "age", "key.txt")
	}
	if !pathExists(keyPath) {
		rep.Warn("age identity key missing at " + keyPath + " — restore from offline backup (recover runbook)")
		return
	}
	rep.Pass("age identity key present (" + keyPath + ")")

	// Round-trip guard: proving the key decrypts needs both `age` (encrypt/decrypt)
	// and `age-keygen` (derive the recipient). age absent already FAILed above; if
	// only age-keygen is missing, WARN rather than FAIL — the key may be fine, we
	// just cannot verify it here, and the escrow would surface a real break.
	if !hasAge {
		return
	}
	if !sys.has("age-keygen") {
		rep.Warn("age-keygen not in PATH — cannot verify the age key round-trips (escrow recipient derivation needs it)")
		return
	}
	if err := sys.AgeRoundTrip(keyPath); err != nil {
		rep.Fail("age root-of-trust round-trip FAILED for " + keyPath + ": " + err.Error() +
			" — the DR key cannot decrypt; restore a good key before relying on the escrow")
		return
	}
	rep.Pass("age root-of-trust verified (round-trip)")
}

// ageRoundTrip is the production round-trip: derive the recipient from the key,
// encrypt a sentinel to it, decrypt the ciphertext back, and assert byte
// equality. It reuses the secrets package's age seams (AgeRecipient/AgeEncrypt/
// AgeDecrypt) rather than shelling to age directly, so the doctor package keeps
// no age exec of its own. The plaintext sentinel is not secret; the temp file
// holds only ciphertext and is removed regardless of outcome. Thin I/O — covered
// by a live smoke, not CI (parity with AgeDecrypt/BWGet).
func ageRoundTrip(keyPath string) error {
	recipient, err := secrets.AgeRecipient(keyPath)
	if err != nil {
		return fmt.Errorf("derive recipient: %w", err)
	}
	ciphertext, err := secrets.AgeEncrypt([]byte(roundTripSentinel), recipient)
	if err != nil {
		return fmt.Errorf("encrypt sentinel: %w", err)
	}

	tmp, err := os.CreateTemp("", "dotf-age-roundtrip-*.age")
	if err != nil {
		return fmt.Errorf("create temp ciphertext: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(ciphertext); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write ciphertext: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("flush ciphertext: %w", err)
	}

	got, err := secrets.AgeDecrypt(tmp.Name(), keyPath)
	if err != nil {
		return fmt.Errorf("decrypt sentinel: %w", err)
	}
	if !bytes.Equal(got, []byte(roundTripSentinel)) {
		return errors.New("decrypted bytes differ from the sentinel (key/identity mismatch)")
	}
	return nil
}
