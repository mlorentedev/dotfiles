package secrets

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stubAge points PATH at a fake `age` binary that writes msg to stderr and exits 1,
// so AgeDecrypt's error path is exercised with no real age and no real key (the same
// POSIX-script fake-binary pattern the cmd tests use for `gh`).
func stubAge(t *testing.T, msg string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("age stub uses a POSIX shell script")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' '" + msg + "' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "age"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

// TestAgeDecryptSurfacesStderr is the #612 B3 guard: a present-but-undecryptable age
// file must fail with age's own message, not a bare "exit status 1". Regression cover
// for the .Output()+%w bug that swallowed the actual cause.
func TestAgeDecryptSurfacesStderr(t *testing.T) {
	const ageErr = "age: error: no identity matched any of the recipients"
	stubAge(t, ageErr)

	// The file must exist so AgeDecrypt gets past the ErrSecretAbsent stat guard and
	// reaches the shell-out — content is irrelevant, the fake age ignores it.
	ageFile := filepath.Join(t.TempDir(), "token.secret.age")
	if err := os.WriteFile(ageFile, []byte("ciphertext"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := AgeDecrypt(ageFile, "/unused/key.txt")
	if err == nil {
		t.Fatal("expected an error when age exits non-zero, got nil")
	}
	if !strings.Contains(err.Error(), ageErr) {
		t.Errorf("error should surface age's stderr.\n got: %v\nwant substring: %q", err, ageErr)
	}
	if strings.Contains(err.Error(), "exit status") {
		t.Errorf("error still leaks the opaque exit-status wrapper instead of age's message: %v", err)
	}
}

// TestAgeDecryptAbsentFile keeps the ErrSecretAbsent classification intact: a missing
// age file is "not provisioned here" (quiet for render), never a decrypt failure.
func TestAgeDecryptAbsentFile(t *testing.T) {
	_, err := AgeDecrypt(filepath.Join(t.TempDir(), "missing.secret.age"), "/unused/key.txt")
	if err == nil {
		t.Fatal("expected ErrSecretAbsent for a missing age file, got nil")
	}
	if !strings.Contains(err.Error(), ErrSecretAbsent.Error()) {
		t.Errorf("a missing age file must wrap ErrSecretAbsent, got: %v", err)
	}
}
