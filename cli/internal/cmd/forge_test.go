package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forgeSchemaPath is resolved at package init, while the working directory is
// still the package's: makeRepo chdirs into a temp repo before seedForge runs.
var forgeSchemaPath, _ = filepath.Abs(filepath.Join("..", "..", "..", "forge", "branch-protection.schema.json"))

// seedForge lays down a minimal declaration and its schema (copied from the
// real one, so the command validates exactly what the repo ships).
func seedForge(t *testing.T, root, repos string) {
	t.Helper()
	stubForgeFresh(t, func(string) error { return nil })
	schema, err := os.ReadFile(forgeSchemaPath)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "forge")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `{"policy":{"required_approving_review_count":0,"rationale":"single maintainer: self-approval is impossible and enforce_admins is on"},"repos":{` + repos + `}}`
	for name, body := range map[string][]byte{"branch-protection.schema.json": schema, "branch-protection.json": []byte(doc)} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func stubForgeGH(t *testing.T, fn func(args ...string) (string, string, error)) {
	t.Helper()
	orig := forgeRun
	t.Cleanup(func() { forgeRun = orig })
	forgeRun = fn
}

// stubForgeFresh replaces the freshness gate; seedForge passes it, since
// makeRepo's repository has no origin. internal/forge tests the real one
// against real git histories.
func stubForgeFresh(t *testing.T, fn func(root string) error) {
	t.Helper()
	orig := forgeFresh
	t.Cleanup(func() { forgeFresh = orig })
	forgeFresh = fn
}

const unprotectedRepo = `"o/r":{"branch":"main","state":"unprotected","reason":"not protected on purpose"}`

func TestForgeProtectionCheckExitClean(t *testing.T) {
	root := makeRepo(t)
	seedForge(t, root, unprotectedRepo)
	stubForgeGH(t, func(...string) (string, string, error) {
		return "", "gh: Branch not protected (HTTP 404)\n", errors.New("exit status 1")
	})
	out, _, err := execute(t, "forge", "protection", "check")
	if err != nil {
		t.Fatalf("a declaration live agrees with must exit 0: %v\n%s", err, out)
	}
	if !strings.Contains(out, "o/r") || !strings.Contains(out, "not protected on purpose") {
		t.Errorf("a declared state is shown with its reason:\n%s", out)
	}
}

func TestForgeProtectionCheckExitOnDrift(t *testing.T) {
	root := makeRepo(t)
	seedForge(t, root, unprotectedRepo)
	stubForgeGH(t, func(...string) (string, string, error) {
		return `{"enforce_admins":{"enabled":true}}`, "", nil // protected live, declared unprotected
	})
	out, _, err := execute(t, "forge", "protection", "check")
	if err == nil || !strings.Contains(out, "[DRIFT] o/r") || !strings.Contains(out, "protection") {
		t.Fatalf("drift must exit non-zero and name repo and field: err=%v\n%s", err, out)
	}
}

func TestForgeProtectionCheckExitWhenUnanswerable(t *testing.T) {
	root := makeRepo(t)
	seedForge(t, root, unprotectedRepo)
	stubForgeGH(t, func(...string) (string, string, error) {
		return "", "error connecting to api.github.com\n", errors.New("exit status 1")
	})
	out, _, err := execute(t, "forge", "protection", "check")
	if err == nil || !strings.Contains(out, "[UNANSWERABLE] o/r") {
		t.Fatalf("an unanswerable check must never exit 0: err=%v\n%s", err, out)
	}
}

func TestForgeProtectionCheckRepoFilter(t *testing.T) {
	root := makeRepo(t)
	seedForge(t, root, unprotectedRepo+`,"o/s":{"branch":"main","state":"unavailable","reason":"private repository"}`)
	stubForgeGH(t, func(...string) (string, string, error) {
		return "", "gh: Branch not protected (HTTP 404)\n", errors.New("exit status 1")
	})
	out, _, err := execute(t, "forge", "protection", "check", "--repo", "o/s")
	if err != nil || strings.Contains(out, "o/r") || !strings.Contains(out, "o/s") {
		t.Fatalf("--repo must restrict the check to one repository: err=%v\n%s", err, out)
	}
	if _, _, err := execute(t, "forge", "protection", "check", "--repo", "o/nope"); err == nil {
		t.Fatal("--repo naming an undeclared repository must be refused")
	}
}

const protectedRepo = `"o/r":{"branch":"main","protection":{"required_status_checks":null,"required_pull_request_reviews":null,` +
	`"enforce_admins":true,"required_signatures":false,"required_linear_history":false,"allow_force_pushes":false,` +
	`"allow_deletions":false,"block_creations":false,"required_conversation_resolution":false,"lock_branch":false,"allow_fork_syncing":false}}`

func TestForgeProtectionApplyDryRunPlansAndWritesNothing(t *testing.T) {
	root := makeRepo(t)
	seedForge(t, root, protectedRepo)
	var writes []string
	stubForgeGH(t, func(args ...string) (string, string, error) {
		if len(args) > 1 && args[1] == "-X" {
			writes = append(writes, strings.Join(args, " "))
		}
		return `{"enforce_admins":{"enabled":false}}`, "", nil
	})
	out, _, err := execute(t, "forge", "protection", "apply", "--dry-run")
	if err != nil || !strings.Contains(out, "[PLANNED] o/r") || !strings.Contains(out, "enforce_admins: false -> true") {
		t.Fatalf("a dry run plans the change and exits 0: err=%v\n%s", err, out)
	}
	if !strings.Contains(out, "changed=1") || len(writes) != 0 {
		t.Fatalf("want changed=1 and no write, got writes=%v\n%s", writes, out)
	}
}

func TestForgeProtectionApplyUnchangedExitsClean(t *testing.T) {
	root := makeRepo(t)
	seedForge(t, root, protectedRepo)
	stubForgeGH(t, func(args ...string) (string, string, error) {
		return `{"enforce_admins":{"enabled":true}}`, "", nil
	})
	out, _, err := execute(t, "forge", "protection", "apply")
	if err != nil || !strings.Contains(out, "[UNCHANGED] o/r") || !strings.Contains(out, "changed=0") {
		t.Fatalf("a converged repository reports changed=0 and exits 0: err=%v\n%s", err, out)
	}
}

func TestForgeProtectionApplyExitsNonZeroWhenAReadFails(t *testing.T) {
	root := makeRepo(t)
	seedForge(t, root, protectedRepo)
	stubForgeGH(t, func(...string) (string, string, error) {
		return "", "error connecting to api.github.com\n", errors.New("exit status 1")
	})
	out, _, err := execute(t, "forge", "protection", "apply")
	if err == nil || !strings.Contains(out, "[FAILED] o/r") {
		t.Fatalf("a failed read must never exit 0: err=%v\n%s", err, out)
	}
}

func TestForgeProtectionRefusesAStaleDeclaration(t *testing.T) {
	for _, args := range [][]string{
		{"forge", "protection", "check"},
		{"forge", "protection", "apply"},
		{"forge", "protection", "apply", "--dry-run"},
	} {
		t.Run(strings.Join(args[2:], " "), func(t *testing.T) {
			root := makeRepo(t)
			seedForge(t, root, unprotectedRepo)
			stubForgeFresh(t, func(string) error {
				return errors.New("forge/branch-protection.json differs from origin/main's")
			})
			asked := false
			stubForgeGH(t, func(...string) (string, string, error) { asked = true; return "", "", nil })

			_, errOut, err := execute(t, args...)
			if err == nil {
				t.Fatal("a stale declaration must exit non-zero")
			}
			if !strings.Contains(errOut, "origin/main") {
				t.Errorf("the refusal must say what the declaration differs from:\n%s", errOut)
			}
			if asked {
				t.Error("the forge must not be asked, let alone written, for a stale declaration")
			}
		})
	}
}

func TestForgeProtectionUnmergedSkipsTheFreshnessGate(t *testing.T) {
	root := makeRepo(t)
	seedForge(t, root, unprotectedRepo)
	stubForgeFresh(t, func(string) error { t.Error("--unmerged must not run the freshness gate"); return nil })
	stubForgeGH(t, func(...string) (string, string, error) {
		return "", "gh: Branch not protected (HTTP 404)\n", errors.New("exit status 1")
	})
	if out, _, err := execute(t, "forge", "protection", "apply", "--dry-run", "--unmerged"); err != nil {
		t.Fatalf("--unmerged uses this checkout's file: %v\n%s", err, out)
	}
}
