package doctor

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// branchProtectionRepo lays down forge/ with the real schema and a
// declaration of the given repos.
func branchProtectionRepo(t *testing.T, repos string) string {
	t.Helper()
	root := t.TempDir()
	schema, err := os.ReadFile(filepath.Join("..", "..", "..", "forge", "branch-protection.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "forge"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `{"policy":{"required_approving_review_count":0,"rationale":"single maintainer: self-approval is impossible and enforce_admins is on"},"repos":{` + repos + `}}`
	for name, body := range map[string][]byte{"branch-protection.schema.json": schema, "branch-protection.json": []byte(doc)} {
		if err := os.WriteFile(filepath.Join(root, "forge", name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCheckBranchProtection(t *testing.T) {
	const (
		unprotected = `"o/r":{"branch":"main","state":"unprotected","reason":"not protected on purpose"}`
		unavailable = `"o/p":{"branch":"main","state":"unavailable","reason":"private, free plan"}`
	)
	notProtected := func(time.Duration, string, ...string) (string, string, error) {
		return "", "gh: Branch not protected (HTTP 404)\n", errors.New("exit status 1")
	}
	cases := []struct {
		name               string
		repos              string
		ghOnPath           bool
		gh                 func(time.Duration, string, ...string) (string, string, error)
		wantFail, wantWarn int
		wantSubstr, forbid string
	}{
		{"declared state confirmed", unprotected, true, notProtected, 0, 0, "not protected on purpose", ""},
		{"drift is a FAIL naming repo and field", unprotected, true,
			func(time.Duration, string, ...string) (string, string, error) {
				return `{"enforce_admins":{"enabled":true}}`, "", nil
			}, 1, 0, "o/r: protection: declared none, live required", ""},
		{"unanswerable is a WARN, never a PASS", unprotected, true,
			func(time.Duration, string, ...string) (string, string, error) {
				return "", "error connecting to api.github.com\n", errors.New("exit status 1")
			}, 0, 1, "not verified", "match their declaration"},
		{"unavailable is shown with its reason", unavailable, true, notProtected, 0, 0, "private, free plan", ""},
		{"gh absent skips, never passes", unprotected, false, notProtected, 0, 0, "gh not on PATH", "match their declaration"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := branchProtectionRepo(t, tc.repos)
			onPath := []string{}
			if tc.ghOnPath {
				onPath = append(onPath, "gh")
			}
			sys := newSys(map[string]string{"DOTFILES_REPO_DIR": root}, onPath, nil)
			sys.CommandOutputBounded = tc.gh
			var buf bytes.Buffer
			rep := capture(&buf)
			checkBranchProtection(sys, rep)
			out := buf.String()
			if rep.Failures() != tc.wantFail || rep.Warnings() != tc.wantWarn {
				t.Fatalf("fail=%d warn=%d, want %d/%d\n%s", rep.Failures(), rep.Warnings(), tc.wantFail, tc.wantWarn, out)
			}
			if !strings.Contains(out, "branch-protection") || !strings.Contains(out, tc.wantSubstr) {
				t.Errorf("output missing the section or %q:\n%s", tc.wantSubstr, out)
			}
			if tc.forbid != "" && strings.Contains(out, tc.forbid) {
				t.Errorf("output must not contain %q:\n%s", tc.forbid, out)
			}
		})
	}
}

func TestCheckBranchProtectionWithoutADeclarationSkips(t *testing.T) {
	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": t.TempDir()}, []string{"gh"}, nil)
	var buf bytes.Buffer
	rep := capture(&buf)
	checkBranchProtection(sys, rep)
	if rep.Failures() != 0 || !strings.Contains(buf.String(), "no forge/branch-protection.json") {
		t.Fatalf("a checkout without a declaration has nothing to check: %s", buf.String())
	}
}

// A checkout behind a landed declaration change, whose old declaration still
// matches the forge, would PASS: the false OK of 2026-10-15. The sweep compares
// with origin as last fetched and WARNs.
func TestCheckBranchProtectionWarnsOnADeclarationThatHasNotLanded(t *testing.T) {
	const declared = `"o/r":{"branch":"main","state":"unprotected","reason":"not protected on purpose"}`
	root := branchProtectionRepo(t, declared)
	origin := filepath.Join(t.TempDir(), "origin.git")
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git(root, "init", "-q", "-b", "main")
	git(root, "add", "forge")
	git(root, "commit", "-q", "-m", "declare")
	git(root, "init", "-q", "--bare", "-b", "main", origin)
	git(root, "remote", "add", "origin", origin)
	// main lands a newer declaration; this checkout goes back to the old one.
	decl := filepath.Join(root, "forge", "branch-protection.json")
	body, err := os.ReadFile(decl)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(decl, bytes.Replace(body, []byte("on purpose"), []byte("on purpose, newer"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	git(root, "commit", "-q", "-am", "newer")
	git(root, "push", "-q", "origin", "main")
	git(root, "reset", "-q", "--hard", "HEAD~1")

	sys := newSys(map[string]string{"DOTFILES_REPO_DIR": root}, []string{"gh"}, nil)
	sys.CommandOutputBounded = func(time.Duration, string, ...string) (string, string, error) {
		return "", "gh: Branch not protected (HTTP 404)\n", errors.New("exit status 1")
	}
	var buf bytes.Buffer
	rep := capture(&buf)
	checkBranchProtection(sys, rep)
	if rep.Warnings() != 1 || !strings.Contains(buf.String(), "differs from origin/main's") {
		t.Fatalf("want one WARN naming origin/main, got warn=%d:\n%s", rep.Warnings(), buf.String())
	}
}
