package cmd

import (
	"bytes"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// fakeSetter records uploads ("repo|name" -> value) so sync tests assert with no gh.
type fakeSetter map[string]string

func (f fakeSetter) SetSecret(repo, name, value string) error {
	f[repo+"|"+name] = value
	return nil
}

// useGHSecretSetter points the ghSecretSetter seam at a fake for the test's duration.
func useGHSecretSetter(t *testing.T, s secrets.GitHubSecretSetter) {
	t.Helper()
	old := ghSecretSetter
	ghSecretSetter = s
	t.Cleanup(func() { ghSecretSetter = old })
}

// useOriginResolver overrides the --repo default resolver (bypasses git).
func useOriginResolver(t *testing.T, fn func() (string, error)) {
	t.Helper()
	old := repoOriginResolver
	repoOriginResolver = fn
	t.Cleanup(func() { repoOriginResolver = old })
}

// syncCiExec runs `secrets sync ci` against a temp registry with fake age/bw/gh seams,
// returning (stdout, error). ageVals maps an age source base name to its plaintext.
func syncCiExec(t *testing.T, registry string, setter fakeSetter, bw fakeBW, ageVals map[string]string, args ...string) (string, error) {
	t.Helper()
	useTempRegistry(t, registry)
	useBwReader(t, bw)
	useGHSecretSetter(t, setter)
	old := ageDecryptor
	ageDecryptor = func(ageFile, _ string) ([]byte, error) {
		base := strings.TrimSuffix(filepath.Base(ageFile), ".secret.age")
		if v, ok := ageVals[base]; ok {
			return []byte(v), nil
		}
		return []byte("age:" + base), nil
	}
	t.Cleanup(func() { ageDecryptor = old })

	var out bytes.Buffer
	cmd := newSecretsSyncCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(append([]string{"ci"}, args...))
	return func() (string, error) { err := cmd.Execute(); return out.String(), err }()
}

const ciRegistry = `
version: 1
secrets:
  - {id: rel, plane: app, backend: age, age: rel.src, expose: {env: RELEASE_TOKEN}, consumers: ["ci:mlorentedev/dotfiles"]}
  - {id: bw1, plane: app, backend: bw, bw: {item: it, field: f}, expose: {env: BW_TOKEN}, consumers: ["ci:mlorentedev/dotfiles"]}
  - {id: other, plane: app, backend: age, age: other.src, expose: {env: OTHER_TOKEN}, consumers: ["ci:mlorentedev/payments"]}
`

// AC1 — selects + uploads exactly this repo's CI set; other-repo entries excluded.
func TestSecretsSyncCi_SelectsRepoSet(t *testing.T) {
	setter := fakeSetter{}
	out, err := syncCiExec(t, ciRegistry, setter, fakeBW{"it/f": "bw-val"},
		map[string]string{"rel.src": "rel-val"}, "--repo", "mlorentedev/dotfiles")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for k := range setter {
		got = append(got, k)
	}
	slices.Sort(got)
	want := []string{"mlorentedev/dotfiles|BW_TOKEN", "mlorentedev/dotfiles|RELEASE_TOKEN"}
	if !slices.Equal(got, want) {
		t.Errorf("uploads = %v, want %v (other-repo excluded)", got, want)
	}
	if !strings.Contains(out, "set RELEASE_TOKEN") {
		t.Errorf("missing upload line:\n%s", out)
	}
}

// AC2 — backend-agnostic: the age and bw entries upload identically (the unblock).
func TestSecretsSyncCi_AgeAndBwUploadIdentically(t *testing.T) {
	setter := fakeSetter{}
	_, err := syncCiExec(t, ciRegistry, setter, fakeBW{"it/f": "bw-val"},
		map[string]string{"rel.src": "rel-val"}, "--repo", "mlorentedev/dotfiles")
	if err != nil {
		t.Fatal(err)
	}
	if setter["mlorentedev/dotfiles|RELEASE_TOKEN"] != "rel-val" {
		t.Errorf("age value = %q, want rel-val", setter["mlorentedev/dotfiles|RELEASE_TOKEN"])
	}
	if setter["mlorentedev/dotfiles|BW_TOKEN"] != "bw-val" {
		t.Errorf("bw value = %q, want bw-val", setter["mlorentedev/dotfiles|BW_TOKEN"])
	}
}

// AC3 — exclusions: file / floor|offline / GITHUB_* skipped with a reason, no upload.
func TestSecretsSyncCi_Exclusions(t *testing.T) {
	const reg = `
version: 1
secrets:
  - {id: kc, plane: infra, backend: age, age: kc.src, bw: {item: kc, field: notes}, expose: {file: {var: KUBECONFIG, path: "~/.k"}}, consumers: ["ci:mlorentedev/dotfiles"]}
  - {id: floor, plane: floor, backend: age-offline, age: floor.src, expose: {env: FLOOR_TOKEN}, consumers: ["ci:mlorentedev/dotfiles"]}
  - {id: gh, plane: app, backend: age, age: gh.src, expose: {env: GITHUB_PAT}, consumers: ["ci:mlorentedev/dotfiles"]}
`
	setter := fakeSetter{}
	out, err := syncCiExec(t, reg, setter, fakeBW{}, nil, "--repo", "mlorentedev/dotfiles")
	if err != nil {
		t.Fatal(err)
	}
	if len(setter) != 0 {
		t.Errorf("exclusions must upload nothing, got %v", setter)
	}
	for _, want := range []string{"skip kc:", "skip floor:", "skip GITHUB_PAT:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing skip report %q in:\n%s", want, out)
		}
	}
}

// AC4 — --dry-run reports VAR->repo with byte lengths, never a value, and uploads nothing.
func TestSecretsSyncCi_DryRunInert(t *testing.T) {
	setter := fakeSetter{}
	out, err := syncCiExec(t, ciRegistry, setter, fakeBW{"it/f": "bw-secret"},
		map[string]string{"rel.src": "rel-secret"}, "--repo", "mlorentedev/dotfiles", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if len(setter) != 0 {
		t.Errorf("--dry-run must upload nothing, got %v", setter)
	}
	if !strings.Contains(out, "would set RELEASE_TOKEN") || !strings.Contains(out, "bytes)") {
		t.Errorf("dry-run output missing 'would set' + byte length:\n%s", out)
	}
	if strings.Contains(out, "rel-secret") || strings.Contains(out, "bw-secret") {
		t.Errorf("--dry-run leaked a secret value:\n%s", out)
	}
}

// AC5 — --repo defaults to the origin slug; an invalid slug errors.
func TestSecretsSyncCi_RepoResolution(t *testing.T) {
	t.Run("origin default", func(t *testing.T) {
		useOriginResolver(t, func() (string, error) { return "mlorentedev/dotfiles", nil })
		setter := fakeSetter{}
		_, err := syncCiExec(t, ciRegistry, setter, fakeBW{"it/f": "bw-val"},
			map[string]string{"rel.src": "rel-val"})
		if err != nil {
			t.Fatal(err)
		}
		if setter["mlorentedev/dotfiles|RELEASE_TOKEN"] != "rel-val" {
			t.Errorf("origin default did not target the resolved repo: %v", setter)
		}
	})
	t.Run("bad slug", func(t *testing.T) {
		setter := fakeSetter{}
		_, err := syncCiExec(t, ciRegistry, setter, fakeBW{}, nil, "--repo", "notaslug")
		if err == nil || !strings.Contains(err.Error(), "invalid --repo") {
			t.Fatalf("want invalid-repo error, got %v", err)
		}
		if len(setter) != 0 {
			t.Error("a bad slug must upload nothing")
		}
	})
}
