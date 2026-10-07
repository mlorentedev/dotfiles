package cmd

import (
	"os"
	"os/exec"
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
	out := stdout + stderr
	if !strings.Contains(out, "work") || !strings.Contains(out, "project") {
		t.Errorf("`dotf vault` should print help listing the work and project subcommands:\n%s", out)
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

	ctx := filepath.Join(vaultDir, "50_work", "45-development", "acme-sensors", "edge-fw", "context.md")
	if _, err := os.Stat(ctx); err != nil {
		t.Errorf("context.md not written: %v", err)
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
	if strings.Contains(stdout, "skipped  context.md") {
		t.Errorf("--force should regenerate, not skip:\n%s", stdout)
	}
}

func TestVaultProjectScaffolds(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("VAULT_PATH", vaultDir)
	t.Setenv("HOME", t.TempDir()) // isolate the linkMemory symlink off the real ~/.claude
	pinClock(t)
	repo := filepath.Join(t.TempDir(), "myrepo")

	stdout, _, err := execute(t, "vault", "project", repo, "--stack", "go")
	if err != nil {
		t.Fatalf("vault project: %v", err)
	}
	if !strings.Contains(stdout, "[OK] Project vault entry") {
		t.Errorf("missing success line:\n%s", stdout)
	}

	ctx := filepath.Join(vaultDir, "10_projects", "myrepo", "context.md")
	if _, err := os.Stat(ctx); err != nil {
		t.Errorf("context.md not written: %v", err)
	}
	b, _ := os.ReadFile(ctx)
	if !strings.Contains(string(b), "myrepo") || !strings.Contains(string(b), `created: "2026-06-13"`) {
		t.Errorf("tokens not substituted (repo/date):\n%s", string(b))
	}
}

func TestVaultProjectErrorsWhenVaultAbsent(t *testing.T) {
	t.Setenv("VAULT_PATH", filepath.Join(t.TempDir(), "nope"))
	_, _, err := execute(t, "vault", "project", t.TempDir())
	if err == nil {
		t.Fatal("expected error when the vault is absent (vault-only command)")
	}
}

func TestVaultProjectForceFlagPlumbed(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("VAULT_PATH", vaultDir)
	t.Setenv("HOME", t.TempDir())
	pinClock(t)
	repo := filepath.Join(t.TempDir(), "myrepo")

	if _, _, err := execute(t, "vault", "project", repo); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// Second run without --force skips; with --force regenerates.
	stdout, _, err := execute(t, "vault", "project", repo)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if !strings.Contains(stdout, "skipped") {
		t.Errorf("re-run without --force should skip:\n%s", stdout)
	}
	stdout, _, err = execute(t, "vault", "project", repo, "--force")
	if err != nil {
		t.Fatalf("force run: %v", err)
	}
	if strings.Contains(stdout, "skipped  context.md") {
		t.Errorf("--force should regenerate, not skip:\n%s", stdout)
	}
}

// TestVaultHealthWiring covers what the golden corpus in
// cli/internal/vault/health_golden_test.go cannot, because it calls RunHealth
// directly: $VAULT_DIR resolution, the --vault override, the --verbose flag,
// and RunHealth's code reaching the process exit status. Obsidian is made
// absent (PATH holds nothing), which is the documented exit-1 abort, so the
// run touches no GUI and needs no stub.
func TestVaultHealthWiring(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("VAULT_DIR", vaultDir)
	t.Setenv("VAULT_NAME", "")
	t.Setenv("PATH", t.TempDir())

	cases := []struct {
		args     []string
		wantName string
	}{
		{[]string{"vault", "health"}, "knowledge"},
		{[]string{"vault", "health", "--vault", "other", "--verbose"}, "other"},
		{[]string{"vault", "health", "-v"}, "knowledge"},
	}
	for _, c := range cases {
		stdout, _, err := execute(t, c.args...)
		if got := ExitCode(err); got != 1 {
			t.Errorf("%v: exit %d (err %v), want 1: obsidian absent from PATH", c.args, got, err)
		}
		if want := "Vault: " + c.wantName + " (" + vaultDir + ")"; !strings.Contains(stdout, want) {
			t.Errorf("%v: report lacks %q:\n%s", c.args, want, stdout)
		}
		// Exit 1 has other causes; this line names the abort the test is about.
		if want := "Obsidian CLI not found in PATH"; !strings.Contains(stdout, want) {
			t.Errorf("%v: exit 1 was not the obsidian-absent abort, report lacks %q:\n%s", c.args, want, stdout)
		}
	}
}

// #2115: a repository the vault tracks under another slug is reported, not
// scaffolded a second time under its directory name.
func TestVaultProjectReportsARepositoryTrackedUnderAnotherSlug(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("VAULT_PATH", vaultDir)
	t.Setenv("HOME", t.TempDir())
	pinClock(t)
	ctxDir := filepath.Join(vaultDir, "10_projects", "homelab")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "context.md"), []byte("---\nrepo_url: \"https://github.com/owner/kubelab\"\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(t.TempDir(), "checkout")
	for _, args := range [][]string{{"init", "-q", repo}, {"-C", repo, "remote", "add", "origin", "git@github.com:owner/kubelab.git"}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	stdout, _, err := execute(t, "vault", "project", repo)
	if err != nil {
		t.Fatalf("vault project: %v", err)
	}
	if !strings.Contains(stdout, "[SKIP]") || !strings.Contains(stdout, "10_projects/homelab") || strings.Contains(stdout, "[OK]") {
		t.Errorf("want a [SKIP] naming 10_projects/homelab:\n%s", stdout)
	}
}
