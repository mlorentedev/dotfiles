package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The prune arm of checkSecrets (CLI-036, #802). The mirror is copy-only, so a
// blob removed from the repo survives on every machine until doctor --fix
// removes it. #802 decided the semantic: doctor --fix prunes, setup only copies.

const pruneRegistry = "version: 1\nsecrets:\n" +
	"  - {id: ssh, plane: floor, backend: age-offline, age: id_ed25519, expose: {env: SSH_KEY}}\n" +
	"  - {id: openai, plane: app, backend: bw, bw: {item: openai-api-key, field: api-key}, expose: {env: OPENAI_API_KEY}}\n"

// pruneFixture builds a checkout that passes the prune gate (registry + DR
// escrow, no legacy blobs) and a mirror holding the claimed blob, one legacy
// blob, one shell-era partial write and the machine-local files that must
// survive every prune.
func pruneFixture(t *testing.T) (repo, mirror string) {
	t.Helper()
	repo = t.TempDir()
	mirror = t.TempDir()
	writeFile(t, filepath.Join(repo, "secrets", "registry.yaml"), pruneRegistry)
	writeFile(t, filepath.Join(repo, "sensitive", "id_ed25519.secret.age"), "x")
	writeFile(t, filepath.Join(repo, "sensitive", "dr", "bitwarden-export.age"), "x")

	writeFile(t, filepath.Join(mirror, "secrets", "registry.yaml"), pruneRegistry)
	for _, f := range []string{
		"id_ed25519.secret.age",
		"chatgpt.api-key.secret.age",
		"openrouter.api.key.secret.age.tmp.699081",
		"env-mapping.conf",
		"README.md",
		filepath.Join("dr", "bitwarden-export.age"),
	} {
		writeFile(t, filepath.Join(mirror, "sensitive", f), "x")
	}
	return repo, mirror
}

func runCheckSecrets(t *testing.T, repo, mirror string, fix bool) (string, *Report) {
	t.Helper()
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(map[string]string{"HOME": mirror, "DOTFILES_REPO_DIR": repo}, nil, nil)
	checkSecrets(sys, &Config{DotfilesDir: mirror}, rep, fix)
	return buf.String(), rep
}

func mirrorHas(mirror, name string) bool {
	_, err := os.Stat(filepath.Join(mirror, "sensitive", name))
	return err == nil
}

func TestCheckSecrets_FixPrunesUnclaimedMirrorBlob(t *testing.T) {
	repo, mirror := pruneFixture(t)

	out, rep := runCheckSecrets(t, repo, mirror, false)
	if rep.Failures() != 2 {
		t.Fatalf("without --fix: failures = %d, want 2 (the blob and the partial write)\n%s", rep.Failures(), out)
	}
	if !strings.Contains(out, "dotf doctor --fix") {
		t.Errorf("an orphan the checkout lacks must name the --fix remedy\n%s", out)
	}
	if !mirrorHas(mirror, "chatgpt.api-key.secret.age") {
		t.Fatal("a run without --fix must not touch disk")
	}

	out, rep = runCheckSecrets(t, repo, mirror, true)
	if rep.Failures() != 0 {
		t.Fatalf("with --fix: failures = %d, want 0\n%s", rep.Failures(), out)
	}
	if got := strings.Count(out, "[FIX ]"); got != 2 {
		t.Errorf("want one [FIX ] line per pruned file, got %d\n%s", got, out)
	}
	for _, gone := range []string{"chatgpt.api-key.secret.age", "openrouter.api.key.secret.age.tmp.699081"} {
		if mirrorHas(mirror, gone) {
			t.Errorf("%s survived --fix", gone)
		}
	}
	for _, kept := range []string{"id_ed25519.secret.age", "env-mapping.conf", "README.md", filepath.Join("dr", "bitwarden-export.age")} {
		if !mirrorHas(mirror, kept) {
			t.Errorf("%s was pruned; only unclaimed *.secret.age files may be", kept)
		}
	}

	out, _ = runCheckSecrets(t, repo, mirror, true)
	if strings.Contains(out, "[FIX ]") {
		t.Errorf("a second --fix must find nothing to prune\n%s", out)
	}
}

func TestCheckSecrets_FixRefuses(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(t *testing.T, repo, mirror string) (repoArg string)
	}{
		{"blob still in the checkout", func(t *testing.T, repo, _ string) string {
			writeFile(t, filepath.Join(repo, "sensitive", "chatgpt.api-key.secret.age"), "x")
			return repo
		}},
		{"checkout without the DR escrow", func(t *testing.T, repo, _ string) string {
			if err := os.Remove(filepath.Join(repo, "sensitive", "dr", "bitwarden-export.age")); err != nil {
				t.Fatal(err)
			}
			return repo
		}},
		{"checkout without a registry", func(t *testing.T, repo, _ string) string {
			if err := os.Remove(filepath.Join(repo, "secrets", "registry.yaml")); err != nil {
				t.Fatal(err)
			}
			return repo
		}},
		{"mirror is the checkout", func(_ *testing.T, _, mirror string) string {
			return mirror
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mirror := pruneFixture(t)
			repoArg := tc.mutate(t, repo, mirror)

			out, rep := runCheckSecrets(t, repoArg, mirror, true)
			// Only the blob under test is asserted: in the "still in the checkout"
			// row the partial write is absent from the checkout and is rightly pruned.
			if strings.Contains(out, "pruned orphan secret blob: sensitive/chatgpt.api-key.secret.age") {
				t.Errorf("--fix pruned where it must refuse\n%s", out)
			}
			if !mirrorHas(mirror, "chatgpt.api-key.secret.age") {
				t.Error("the blob was removed")
			}
			if !strings.Contains(out, "orphan: chatgpt.api-key.secret.age") || rep.Failures() == 0 {
				t.Errorf("a refused prune must still FAIL, naming the blob\n%s", out)
			}
		})
	}
}

// A blob the checkout still holds is either committed (git rm it, in a PR) or an
// untracked leftover (delete it). #1793: both were reported as "still committed
// … git rm it there", and `git rm` fails on an untracked path.
func TestCheckSecrets_OrphanInCheckoutNamesItsGitState(t *testing.T) {
	const blob = "chatgpt.api-key.secret.age"
	cases := []struct {
		name     string
		lsFiles  func() (string, error)
		want     string
		mustMiss string
	}{
		{"committed", func() (string, error) { return "sensitive/" + blob + "\n", nil },
			"still committed in", ""},
		{"untracked", func() (string, error) { return "", nil },
			"untracked copy", "git rm"},
		{"git state unknown", func() (string, error) { return "", errors.New("not a git repository") },
			"git rm it if tracked, delete it if not", "still committed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mirror := pruneFixture(t)
			writeFile(t, filepath.Join(repo, "sensitive", blob), "x")
			var buf bytes.Buffer
			rep := capture(&buf)
			sys := newSys(map[string]string{"HOME": mirror, "DOTFILES_REPO_DIR": repo}, nil, nil)
			sys.CommandOutputDir = func(dir, name string, args ...string) (string, error) {
				if dir == repo && name == "git" && strings.Join(args, " ") == "ls-files -- sensitive/"+blob {
					return tc.lsFiles()
				}
				return "", errors.New("unexpected command")
			}
			checkSecrets(sys, &Config{DotfilesDir: mirror}, rep, true)
			out := buf.String()
			line := ""
			for _, l := range strings.Split(out, "\n") {
				if strings.Contains(l, "orphan: "+blob) {
					line = l
				}
			}
			if line == "" {
				t.Fatalf("no FAIL line for the blob\n%s", out)
			}
			if !strings.Contains(line, tc.want) {
				t.Errorf("want %q in %q", tc.want, line)
			}
			if tc.mustMiss != "" && strings.Contains(line, tc.mustMiss) {
				t.Errorf("%q must not appear in %q", tc.mustMiss, line)
			}
			if !mirrorHas(mirror, blob) {
				t.Error("a blob the checkout still holds must not be pruned from the mirror")
			}
		})
	}
}
