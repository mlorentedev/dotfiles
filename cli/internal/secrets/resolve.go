package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrSecretAbsent marks a secret that is genuinely not provisioned on this machine
// (its age file does not exist), as opposed to a real failure (wrong key, locked
// vault, decrypt error). render treats absent as a quiet, non-fatal case; everything
// else is surfaced loudly (#612 A2).
var ErrSecretAbsent = errors.New("secret not provisioned")

// Decryptor decrypts the age file at ageFile using the identity at keyPath,
// returning the plaintext. The seam that keeps EnvFor unit-testable with no age
// binary and no real key; AgeDecrypt is the production default.
type Decryptor func(ageFile, keyPath string) ([]byte, error)

// AgeDecrypt shells out to `age --decrypt --identity <key> <file>` — the same
// tool and key load-secrets.sh uses (Phase 0 provisions age). Plaintext is
// returned in memory; it is never written to disk for env secrets. A missing age
// file is reported as ErrSecretAbsent (not provisioned here) so render can keep it
// quiet; a present-but-undecryptable file surfaces age's error.
func AgeDecrypt(ageFile, keyPath string) ([]byte, error) {
	if _, err := os.Stat(ageFile); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrSecretAbsent, filepath.Base(ageFile))
	}
	out, err := exec.Command("age", "--decrypt", "--identity", keyPath, ageFile).Output()
	if err != nil {
		return nil, fmt.Errorf("age decrypt %s: %w", filepath.Base(ageFile), err)
	}
	return out, nil
}

// Resolver turns one Entry into its plaintext secret bytes. One implementation per
// backend; EnvFor dispatches on Entry.Backend through a resolver map, so adding a
// backend (bws, Vault) is a new Resolver + one map entry — no edit to the resolution
// loop, no change to run/show/render (Open/Closed). ADR-028 §2.
type Resolver interface {
	Resolve(e Entry) ([]byte, error)
}

// Loader resolves entries to child-process environment, fetching on demand. It holds
// the per-backend seams — Decrypt (age) and BW (Bitwarden) — both injectable so
// EnvFor is unit-testable with no age binary, no age key, and no Bitwarden vault.
type Loader struct {
	SecretsDir string    // dir holding <file>.secret.age (…/sensitive)
	KeyPath    string    // age identity key path
	Decrypt    Decryptor // nil → AgeDecrypt
	BW         BWReader  // bw field reader; nil → bw entries fail with a clear error
}

// resolvers maps a backend name to its Resolver. "" maps to age so a hand-built
// Entry{Var, File} (no Backend set) still resolves — back-compat with pre-bw callers.
func (l *Loader) resolvers() map[string]Resolver {
	age := ageResolver{secretsDir: l.SecretsDir, keyPath: l.KeyPath, decrypt: l.Decrypt}
	return map[string]Resolver{
		"":            age,
		"age":         age,
		"age-offline": age,
		"bw":          bwResolver{reader: l.BW},
	}
}

// EnvFor resolves the selected entries and returns "KEY=VALUE" strings for the child
// environment. When only is non-nil, only entries whose Var is in it are resolved
// (smaller blast radius). Env secrets are returned as VAR=<value> with newlines
// stripped (parity with load-secrets' `tr -d '\n'`); file secrets are written to
// Dest (0600, parent dirs created) and returned as VAR=<dest>.
//
// A resolution failure is returned immediately (fail-fast) so the child is never
// launched with a partially-populated secret set.
func (l *Loader) EnvFor(entries []Entry, only map[string]bool) ([]string, error) {
	resolvers := l.resolvers()

	var env []string
	for _, e := range entries {
		if only != nil && !only[e.Var] {
			continue
		}
		r, ok := resolvers[e.Backend]
		if !ok {
			return nil, fmt.Errorf("secret %q: unknown backend %q", e.Var, e.Backend)
		}
		plaintext, err := r.Resolve(e)
		if err != nil {
			return nil, err
		}

		if e.IsFile {
			if len(plaintext) == 0 {
				return nil, fmt.Errorf("secret %q resolved to empty content (backend %q) — refusing to materialize", e.Var, e.Backend)
			}
			if err := materialize(e.Dest, plaintext, e.Mode); err != nil {
				return nil, err
			}
			env = append(env, e.Var+"="+e.Dest)
			continue
		}
		value := stripNewlines(string(plaintext))
		if value == "" {
			// A blank secret would launch the child unauthenticated with no signal —
			// the silent-empty incident class. Fail loud instead (#612 A1).
			return nil, fmt.Errorf("secret %q resolved to an empty value (backend %q) — refusing to inject", e.Var, e.Backend)
		}
		env = append(env, e.Var+"="+value)
	}
	return env, nil
}

// Verify resolves one entry through its backend resolver as a read-only health check:
// it confirms the secret produces a non-empty value, applying run's empty-value
// rejection, but it never materializes a file secret and never returns the value
// (no leak, no side effect). It returns nil when the secret resolves, ErrSecretAbsent
// (wrapped) when it is genuinely not provisioned here, or the specific failure
// otherwise — so `dotf secrets verify` can classify OK / MISSING / FAILED.
func (l *Loader) Verify(e Entry) error {
	r, ok := l.resolvers()[e.Backend]
	if !ok {
		return fmt.Errorf("unknown backend %q", e.Backend)
	}
	plaintext, err := r.Resolve(e)
	if err != nil {
		return err
	}
	if e.IsFile {
		if len(plaintext) == 0 {
			return fmt.Errorf("resolved to empty content")
		}
		return nil
	}
	if stripNewlines(string(plaintext)) == "" {
		return fmt.Errorf("resolved to an empty value")
	}
	return nil
}

// ageResolver decrypts the age file backing an entry. The Decryptor seam keeps
// resolution testable with no age binary; AgeDecrypt is the production default.
type ageResolver struct {
	secretsDir string
	keyPath    string
	decrypt    Decryptor
}

func (r ageResolver) Resolve(e Entry) ([]byte, error) {
	decrypt := r.decrypt
	if decrypt == nil {
		decrypt = AgeDecrypt
	}
	return decrypt(filepath.Join(r.secretsDir, e.File+".secret.age"), r.keyPath)
}

// bwResolver reads a Bitwarden item field through the BWReader seam. A nil reader
// (Bitwarden locked, bw missing, or not wired) is a clear, actionable error — never
// a panic, never a hang.
type bwResolver struct{ reader BWReader }

func (r bwResolver) Resolve(e Entry) ([]byte, error) {
	if r.reader == nil {
		return nil, fmt.Errorf("secret %q: bw backend unavailable (Bitwarden locked or `bw` missing — run `bw unlock` and export BW_SESSION)", e.Var)
	}
	val, err := r.reader.Field(e.Item, e.Field)
	if err != nil {
		return nil, fmt.Errorf("bw resolve %s/%s: %w", e.Item, e.Field, err)
	}
	return []byte(val), nil
}

// materialize writes a file secret to dest atomically (temp file + rename, never a
// truncated half-write — #612 B4), creating parent dirs. mode sets the file
// permissions; 0 falls back to 0600, the secret-file default (#612 B2).
func materialize(dest string, content []byte, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o600
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("create dir for %s: %w", dest, err)
	}
	if err := AtomicWriteMode(dest, content, mode); err != nil {
		return fmt.Errorf("write file secret %s: %w", dest, err)
	}
	return nil
}

// stripNewlines removes CR/LF from an env value — env tokens are single-line, and
// `age -d` appends a trailing newline that must not become part of the value.
func stripNewlines(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}
