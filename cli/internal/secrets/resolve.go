package secrets

import (
	"bytes"
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
// quiet; a present-but-undecryptable file surfaces age's own stderr (the actual
// cause — "no identity matched any of the recipients", a malformed header, …) and
// not a bare "exit status 1" (#612 B3, parity with BWGet's stderr handling).
func AgeDecrypt(ageFile, keyPath string) ([]byte, error) {
	if _, err := os.Stat(ageFile); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrSecretAbsent, filepath.Base(ageFile))
	}
	cmd := exec.Command("age", "--decrypt", "--identity", keyPath, ageFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		// .Output() stashes stderr inside *ExitError, but %w renders it as the
		// opaque "exit status 1"; capture and surface the message itself instead.
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("age decrypt %s: %s", filepath.Base(ageFile), msg)
	}
	return stdout.Bytes(), nil
}

// Resolver turns one Entry into its plaintext secret bytes. One implementation per
// backend; EnvFor dispatches on Entry.Backend through a resolver map, so adding a
// backend (bws, Vault) is a new Resolver + one map entry — no edit to the resolution
// loop, no change to run/show/render (Open/Closed). ADR-028 §2.
type Resolver interface {
	Resolve(e Entry) ([]byte, error)
}

// NotMaterialized marks a backend that must never reach a child process, whatever
// its health check does. It is separate from Verifier on purpose: "answers its own
// health question" and "must not be handed out" are two different properties that
// happen to coincide in the one backend that has either today, and EnvFor used the
// first as a proxy for the second until the round-2 review of OPS-026 pointed out
// that a future backend implementing Verifier for any other reason would vanish
// from the secret set in silence. That is this repository's recurring defect —
// a signal consulted for a question it was not asked — inside the change that
// exists to fix an instance of it.
//
// The reason string is what the refusal says, so the two consumers cannot drift.
type NotMaterialized interface {
	NotMaterializedReason(e Entry) string
}

// Verifier is the optional half of Resolver, for a backend whose health question
// is not "did it resolve". Loader.Verify prefers it when a resolver implements it
// and falls back to Resolve otherwise, so the ordinary backends are untouched.
//
// It exists because one backend's answer cannot be produced by resolving: the age
// root has no source to resolve FROM, and asking `run` to hand it out is a
// widening of blast radius rather than a health check (#937).
type Verifier interface {
	VerifyEntry(e Entry) error
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
		BackendDefault:       age,
		BackendAge:           age,
		BackendAgeOffline:    age,
		BackendBW:            bwResolver{reader: l.BW},
		BackendFileAuthority: fileAuthorityResolver{},
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
		// A backend that answers its own health question is one nothing resolves —
		// today, the age root. Skipping it when resolving EVERYTHING is the point:
		// `run` asks for the secret set, not for the key those secrets are
		// decrypted with, and nobody asked for this one.
		//
		// The skip is deliberately conditional on `only == nil`. Naming it
		// explicitly (`--only AGE_KEY_PERSONAL`) falls through to Resolve, which
		// refuses out loud. Skipping there too would answer an explicit request
		// with silence, which is the failure mode this repository keeps finding.
		if _, never := r.(NotMaterialized); never && only == nil {
			continue
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

// RawResolve resolves one entry's backend-native plaintext with no transformation —
// no newline stripping, no file materialization. migrate uses it instead of EnvFor so
// a multi-line secret (e.g. BEEHIIV_DNS_RECORDS) survives the age→bw cutover intact;
// EnvFor's stripNewlines is an env-injection convention (single-line child-process
// tokens), not a property of the secret itself, and applying it during migration was
// silently flattening multi-line values to one line (#612 B6).
func (l *Loader) RawResolve(e Entry) ([]byte, error) {
	r, ok := l.resolvers()[e.Backend]
	if !ok {
		return nil, fmt.Errorf("secret %q: unknown backend %q", e.Var, e.Backend)
	}
	return r.Resolve(e)
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
	// A backend may answer the health question itself. Checked before Resolve,
	// because for the one backend that does, Resolve is defined to refuse.
	if v, isVerifier := r.(Verifier); isVerifier {
		return v.VerifyEntry(e)
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

// fileAuthorityResolver is the age root's backend: a secret whose authority is the
// plaintext file on this disk. There is no ciphertext and no remote, so there is
// nothing to resolve — and handing the key that decrypts every other secret to a
// child process through the same facade as those secrets widens the blast radius
// instead of narrowing it, which is the opposite of what `run` is for.
//
// So Resolve refuses, deliberately and with a reason, and the health question is
// answered by VerifyEntry instead: present, and 0600.
//
// What VerifyEntry does NOT yet check is that the file still matches the copy held
// off this machine. That comparison is the point of #1000 — a drifted copy is
// indistinguishable from a good one until the day it is needed — and it needs a
// live Bitwarden session, so it is a second slice rather than a silent omission.
// Until it lands, this check answers a narrower question than the one that matters,
// and says so here rather than letting a reader assume otherwise.
type fileAuthorityResolver struct{}

func (fileAuthorityResolver) NotMaterializedReason(e Entry) string {
	return fmt.Sprintf("%s is the age root: it is not materialized through this facade, "+
		"because every other secret is decrypted with it", e.Var)
}

func (r fileAuthorityResolver) Resolve(e Entry) ([]byte, error) {
	return nil, errors.New(r.NotMaterializedReason(e))
}

// VerifyEntry reports the root's health without reading its contents. A missing
// file is ErrSecretAbsent (MISSING, not FAILED) — the same tolerance every other
// backend gets for "not provisioned on this machine", since a fresh checkout
// legitimately has no key yet. A wrong mode is FAILED: the file is present and
// readable by someone it should not be, which is a real defect and not an absence.
func (fileAuthorityResolver) VerifyEntry(e Entry) error {
	if e.Dest == "" {
		return fmt.Errorf("no path declared for the file-authority secret")
	}
	fi, err := os.Stat(e.Dest)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrSecretAbsent, e.Dest)
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", e.Dest, err)
	}
	// Not just directories: a FIFO, socket or device node at this path can carry
	// mode 0600 and pass every other check here, while `age --decrypt` fails on it.
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file (%s)", e.Dest, fi.Mode().Type())
	}
	if fi.Size() == 0 {
		return fmt.Errorf("%s is empty", e.Dest)
	}
	// Permission bits only, and only where they exist — see checkKeyMode's two
	// build-tagged halves. setuid/sticky and the type bits are not what was
	// declared, and a mask mismatch there would report a confusing cause.
	return checkKeyMode(fi, e.Dest, e.Mode)
}
