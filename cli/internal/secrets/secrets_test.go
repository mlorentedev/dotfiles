package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMapping(t *testing.T) {
	home := "/home/u"
	in := strings.NewReader(`
# a comment
GITHUB_PERSONAL_ACCESS_TOKEN=github.token

  OPENAI_API_KEY = chatgpt.api-key
@KUBECONFIG=kubelab.kubeconfig>~/.kube/kubelab.config
@SSH_KEY=id_ed25519>~/.ssh/id_ed25519
not-a-mapping-line
`)
	entries, err := ParseMapping(in, home)
	if err != nil {
		t.Fatalf("ParseMapping: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4: %+v", len(entries), entries)
	}

	want := map[string]Entry{
		"GITHUB_PERSONAL_ACCESS_TOKEN": {Var: "GITHUB_PERSONAL_ACCESS_TOKEN", File: "github.token"},
		"OPENAI_API_KEY":               {Var: "OPENAI_API_KEY", File: "chatgpt.api-key"},
		"KUBECONFIG":                   {Var: "KUBECONFIG", File: "kubelab.kubeconfig", IsFile: true, Dest: home + "/.kube/kubelab.config"},
		"SSH_KEY":                      {Var: "SSH_KEY", File: "id_ed25519", IsFile: true, Dest: home + "/.ssh/id_ed25519"},
	}
	for _, e := range entries {
		w, ok := want[e.Var]
		if !ok {
			t.Errorf("unexpected entry %+v", e)
			continue
		}
		if e != w {
			t.Errorf("entry %s = %+v, want %+v", e.Var, e, w)
		}
	}
}

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
