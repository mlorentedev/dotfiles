package secrets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Decryptor decrypts the age file at ageFile using the identity at keyPath,
// returning the plaintext. The seam that keeps EnvFor unit-testable with no age
// binary and no real key; AgeDecrypt is the production default.
type Decryptor func(ageFile, keyPath string) ([]byte, error)

// AgeDecrypt shells out to `age --decrypt --identity <key> <file>` — the same
// tool and key load-secrets.sh uses (Phase 0 provisions age). Plaintext is
// returned in memory; it is never written to disk for env secrets.
func AgeDecrypt(ageFile, keyPath string) ([]byte, error) {
	out, err := exec.Command("age", "--decrypt", "--identity", keyPath, ageFile).Output()
	if err != nil {
		return nil, fmt.Errorf("age decrypt %s: %w", filepath.Base(ageFile), err)
	}
	return out, nil
}

// Loader resolves entries to child-process environment, decrypting on demand.
type Loader struct {
	SecretsDir string    // dir holding <file>.secret.age (…/sensitive)
	KeyPath    string    // age identity key path
	Decrypt    Decryptor // nil → AgeDecrypt
}

// EnvFor decrypts the selected entries and returns "KEY=VALUE" strings for the
// child environment. When only is non-nil, only entries whose Var is in it are
// resolved (smaller blast radius). Env secrets are returned as VAR=<value> with
// newlines stripped (parity with load-secrets' `tr -d '\n'`); file secrets are
// written to Dest (0600, parent dirs created) and returned as VAR=<dest>.
//
// A decryption failure is returned immediately (fail-fast) so the child is never
// launched with a partially-populated secret set.
func (l *Loader) EnvFor(entries []Entry, only map[string]bool) ([]string, error) {
	decrypt := l.Decrypt
	if decrypt == nil {
		decrypt = AgeDecrypt
	}

	var env []string
	for _, e := range entries {
		if only != nil && !only[e.Var] {
			continue
		}
		ageFile := filepath.Join(l.SecretsDir, e.File+".secret.age")
		plaintext, err := decrypt(ageFile, l.KeyPath)
		if err != nil {
			return nil, err
		}

		if e.IsFile {
			if err := materialize(e.Dest, plaintext); err != nil {
				return nil, err
			}
			env = append(env, e.Var+"="+e.Dest)
			continue
		}
		env = append(env, e.Var+"="+stripNewlines(string(plaintext)))
	}
	return env, nil
}

// materialize writes a file secret to dest with 0600, creating parent dirs.
func materialize(dest string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("create dir for %s: %w", dest, err)
	}
	if err := os.WriteFile(dest, content, 0o600); err != nil {
		return fmt.Errorf("write file secret %s: %w", dest, err)
	}
	return nil
}

// stripNewlines removes CR/LF from an env value — env tokens are single-line, and
// `age -d` appends a trailing newline that must not become part of the value.
func stripNewlines(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}
