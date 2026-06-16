package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultListedInRoot(t *testing.T) {
	stdout, stderr, err := execute(t, "--help")
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	if !strings.Contains(stdout+stderr, "vault") {
		t.Errorf("root help should list the vault command:\n%s", stdout+stderr)
	}
}

func TestVaultParentPrintsHelp(t *testing.T) {
	// The parent is runnable (RunE: cmd.Help) so `dotf vault` is a first-class
	// command, not a demoted help-topic.
	stdout, stderr, err := execute(t, "vault")
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	if !strings.Contains(stdout+stderr, "work") {
		t.Errorf("`dotf vault` should print help listing the work subcommand:\n%s", stdout+stderr)
	}
}

func TestVaultWorkArgValidation(t *testing.T) {
	for _, args := range [][]string{
		{"vault", "work"},
		{"vault", "work", "only-one"},
		{"vault", "work", "a", "b", "c"},
	} {
		if _, _, err := execute(t, args...); err == nil {
			t.Errorf("expected arg-count error for %v", args)
		}
	}
}

func TestVaultWorkScaffolds(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("VAULT_PATH", vaultDir)
	pinClock(t)

	stdout, _, err := execute(t, "vault", "work", "acme-sensors", "edge-fw")
	if err != nil {
		t.Fatalf("vault work: %v", err)
	}
	if !strings.Contains(stdout, "[OK] Work-SDK vault entry") {
		t.Errorf("missing success line:\n%s", stdout)
	}

	ctx := filepath.Join(vaultDir, "50_work", "45-development", "acme-sensors", "edge-fw", "00-context.md")
	if _, err := os.Stat(ctx); err != nil {
		t.Errorf("00-context.md not written: %v", err)
	}
	b, _ := os.ReadFile(ctx)
	if !strings.Contains(string(b), `created: "2026-06-13"`) {
		t.Errorf("date token not substituted from pinned clock:\n%s", string(b))
	}
}

func TestVaultWorkErrorsWhenVaultAbsent(t *testing.T) {
	t.Setenv("VAULT_PATH", filepath.Join(t.TempDir(), "nope"))
	_, _, err := execute(t, "vault", "work", "acme", "edge")
	if err == nil {
		t.Fatal("expected error when the vault is absent (vault-only command)")
	}
}

func TestVaultWorkForceFlagPlumbed(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("VAULT_PATH", vaultDir)
	pinClock(t)

	if _, _, err := execute(t, "vault", "work", "acme", "edge"); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// Second run without --force skips; with --force regenerates.
	stdout, _, err := execute(t, "vault", "work", "acme", "edge")
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if !strings.Contains(stdout, "skipped") {
		t.Errorf("re-run without --force should skip:\n%s", stdout)
	}
	stdout, _, err = execute(t, "vault", "work", "acme", "edge", "--force")
	if err != nil {
		t.Fatalf("force run: %v", err)
	}
	if strings.Contains(stdout, "skipped  00-context.md") {
		t.Errorf("--force should regenerate, not skip:\n%s", stdout)
	}
}
