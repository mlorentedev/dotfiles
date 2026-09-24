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
