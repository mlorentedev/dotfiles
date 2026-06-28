package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeDecryptor returns deterministic plaintext per age file (with a trailing
// newline, as `age -d` does) so the loader's newline-stripping is exercised.
func fakeDecryptor(t *testing.T) Decryptor {
	t.Helper()
	return func(ageFile, _ string) ([]byte, error) {
		base := filepath.Base(ageFile)
		return []byte("secret-of-" + base + "\n"), nil
	}
}

func loaderFor(t *testing.T) *Loader {
	t.Helper()
	return &Loader{
		SecretsDir: t.TempDir(),
		KeyPath:    "/unused/key.txt",
		Decrypt:    fakeDecryptor(t),
	}
}

func TestEnvFor_EnvSecrets_NewlineStripped(t *testing.T) {
	l := loaderFor(t)
	entries := []Entry{
		{Var: "OPENAI_API_KEY", File: "chatgpt.api-key"},
		{Var: "PYPI_TOKEN", File: "pypi.token"},
	}
	env, err := l.EnvFor(entries, nil)
	if err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	got := strings.Join(env, "\n")
	if !strings.Contains(got, "OPENAI_API_KEY=secret-of-chatgpt.api-key.secret.age") {
		t.Errorf("missing/!stripped OPENAI_API_KEY in %q", got)
	}
	if strings.Contains(got, "\\n") || strings.Contains(env[0], "\n") {
		t.Errorf("env value not newline-stripped: %q", env[0])
	}
}

func TestEnvFor_OnlyFilter(t *testing.T) {
	l := loaderFor(t)
	entries := []Entry{
		{Var: "A", File: "a"},
		{Var: "B", File: "b"},
		{Var: "C", File: "c"},
	}
	env, err := l.EnvFor(entries, map[string]bool{"B": true})
	if err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	if len(env) != 1 || !strings.HasPrefix(env[0], "B=") {
		t.Fatalf("--only B should yield exactly B, got %v", env)
	}
}

func TestEnvFor_FileSecret_MaterializedDest(t *testing.T) {
	l := loaderFor(t)
	dest := filepath.Join(t.TempDir(), "sub", "kubeconfig")
	entries := []Entry{{Var: "KUBECONFIG", File: "kubelab.kubeconfig", IsFile: true, Dest: dest}}

	env, err := l.EnvFor(entries, nil)
	if err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	want := "KUBECONFIG=" + dest
	if len(env) != 1 || env[0] != want {
		t.Fatalf("env = %v, want [%q]", env, want)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("file secret not materialized: %v", err)
	}
	// File secrets keep their content verbatim (newline included).
	if string(data) != "secret-of-kubelab.kubeconfig.secret.age\n" {
		t.Errorf("materialized content = %q", data)
	}
	if info, _ := os.Stat(dest); info != nil && info.Mode().Perm() != 0o600 {
		// POSIX hosts only — NTFS drops the mode bits.
		if !strings.Contains(fmt.Sprint(info.Mode()), "Irregular") {
			t.Logf("note: mode = %v (0600 expected on POSIX)", info.Mode())
		}
	}
}

func TestEnvFor_DecryptError_FailsFast(t *testing.T) {
	l := loaderFor(t)
	l.Decrypt = func(string, string) ([]byte, error) { return nil, fmt.Errorf("age: no identity") }
	if _, err := l.EnvFor([]Entry{{Var: "X", File: "x"}}, nil); err == nil {
		t.Fatal("expected EnvFor to fail when decryption fails")
	}
}

// statMode reports the materialized file's POSIX permission bits, or skips on a
// filesystem that drops mode bits (NTFS surfaces an "Irregular" mode) so the
// permission assertions stay POSIX-only — matching the existing materialize test.
func statMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	if runtime.GOOS == "windows" {
		// Windows has no POSIX permission bits — os.Chmod only toggles read-only and
		// Stat reports 0666/0444 regardless of the requested mode. The Mode feature
		// degrades to best-effort there (NTFS uses ACLs), so the bit-exact assertion
		// is POSIX-only. The materialization itself is still covered on Windows by the
		// content/overwrite tests.
		t.Skip("Windows does not honor POSIX permission bits — assertion is POSIX-only")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if strings.Contains(fmt.Sprint(info.Mode()), "Irregular") {
		t.Skip("filesystem drops mode bits — permission assertion is POSIX-only")
	}
	return info.Mode().Perm()
}

// TestEnvFor_FileSecret_CustomMode is the #612 B2 guard: a file secret's registry
// mode is applied at materialization, not silently forced to 0600.
func TestEnvFor_FileSecret_CustomMode(t *testing.T) {
	l := loaderFor(t)
	dest := filepath.Join(t.TempDir(), "kubeconfig")
	entries := []Entry{{Var: "KUBECONFIG", File: "k", IsFile: true, Dest: dest, Mode: 0o640}}

	if _, err := l.EnvFor(entries, nil); err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	if got := statMode(t, dest); got != 0o640 {
		t.Errorf("materialized mode = %o, want 0640", got)
	}
}

// TestEnvFor_FileSecret_DefaultMode pins that a zero Mode (the registry omitted it)
// falls back to 0600 — the secret-file default must never widen on omission.
func TestEnvFor_FileSecret_DefaultMode(t *testing.T) {
	l := loaderFor(t)
	dest := filepath.Join(t.TempDir(), "kubeconfig")
	entries := []Entry{{Var: "KUBECONFIG", File: "k", IsFile: true, Dest: dest, Mode: 0}}

	if _, err := l.EnvFor(entries, nil); err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	if got := statMode(t, dest); got != 0o600 {
		t.Errorf("materialized mode = %o, want 0600 (default)", got)
	}
}

// TestEnvFor_FileSecret_AtomicOverwrite is the #612 B4 guard: materializing over an
// existing file fully replaces it (atomic rename), leaving no stale bytes from a
// longer previous content — the failure mode a truncating WriteFile risks on a
// partial write.
func TestEnvFor_FileSecret_AtomicOverwrite(t *testing.T) {
	l := loaderFor(t)
	dest := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(dest, []byte("OLD-AND-LONGER-PREVIOUS-CONTENT"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{Var: "KUBECONFIG", File: "k", IsFile: true, Dest: dest}}

	if _, err := l.EnvFor(entries, nil); err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "secret-of-k.secret.age\n" {
		t.Errorf("overwrite left stale content: %q", data)
	}
}
