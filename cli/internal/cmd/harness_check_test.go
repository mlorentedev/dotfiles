package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// againstVaultFixture is a checkout whose one skill record is stamped from the
// returned vault, and that passes the offline check.
func againstVaultFixture(t *testing.T) (repo, vault string) {
	t.Helper()
	repo, vault = t.TempDir(), t.TempDir()
	src := "---\nname: demo\n---\nbody\n"
	sum := sha256.Sum256([]byte(src))
	write := func(p, s string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(repo, "harness", "manifest.json"), `{"enforced":[],"targets":[]}`)
	write(filepath.Join(vault, "00_meta", "skills", "demo", "SKILL.md"), src)
	write(filepath.Join(repo, "harness", "skills", "demo", "SKILL.md"),
		"---\ngenerated: true\ngenerated_from: 00_meta/skills/demo/SKILL.md\ngenerated_sha: "+hex.EncodeToString(sum[:])[:16]+"\nname: demo\n---\nbody\n")
	return repo, vault
}

func runCheck(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := newHarnessCheckCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// AC6 (#1545): a vault skill edited and never refreshed fails the check, and
// only when --against-vault asks for the vault.
func TestHarnessCheck_AgainstVaultFlagsAnUnrefreshedVaultEdit(t *testing.T) {
	repo, vault := againstVaultFixture(t)
	if out, err := runCheck(t, "--repo-root", repo, "--against-vault", "--vault", vault); err != nil {
		t.Fatalf("a refreshed record should pass: %v\n%s", err, out)
	}

	src := filepath.Join(vault, "00_meta", "skills", "demo", "SKILL.md")
	if err := os.WriteFile(src, []byte("---\nname: demo\n---\nbody, edited in the vault\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := runCheck(t, "--repo-root", repo, "--against-vault", "--vault", vault)
	if !errors.Is(err, errHarnessDrift) || !strings.Contains(out, "[STALE] harness/skills/demo") {
		t.Fatalf("want drift naming skills/demo, got %v\n%s", err, out)
	}
	if out, err := runCheck(t, "--repo-root", repo); err != nil {
		t.Fatalf("the offline check never reads the vault: %v\n%s", err, out)
	}
}

func TestHarnessCheck_AgainstVaultRefusesWithoutAVault(t *testing.T) {
	repo, _ := againstVaultFixture(t)
	_, err := runCheck(t, "--repo-root", repo, "--against-vault", "--vault", filepath.Join(t.TempDir(), "absent"))
	if err == nil || errors.Is(err, errHarnessDrift) || !strings.Contains(err.Error(), "no vault") {
		t.Fatalf("want a refusal naming the missing vault, got %v", err)
	}
}
