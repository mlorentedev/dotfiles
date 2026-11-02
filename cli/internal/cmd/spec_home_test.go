package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// withOrigin turns makeRepo's placeholder .git into a real repository whose
// origin is url, so the spec commands read it the way they do in a checkout.
func withOrigin(t *testing.T, root, url string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", url}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// recordSpecAuditGH stubs the gh runner like stubSpecAuditGH and returns every
// call's path argument, so a test can assert what was never asked.
func recordSpecAuditGH(t *testing.T, states map[string]string) func() []string {
	t.Helper()
	stubSpecAuditGH(t, states)
	inner := specAuditRun
	var mu sync.Mutex
	var calls []string
	specAuditRun = func(args ...string) (string, string, error) {
		mu.Lock()
		calls = append(calls, args[1])
		mu.Unlock()
		return inner(args...)
	}
	return func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), calls...) }
}

func writeBitacoraRepo(t *testing.T, root, slug string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "specs", ".bitacora-repo"), []byte(slug+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

const giteaOrigin = "https://deploy:s3cr3t@git.example.com/teledyne/projects-toolkit.git"

func TestSpecAuditGiteaOriginFailsClosed(t *testing.T) {
	for _, origin := range []string{giteaOrigin, "git@git.example.com:teledyne/projects-toolkit.git"} {
		t.Run(origin, func(t *testing.T) {
			root := makeRepo(t)
			withOrigin(t, root, origin)
			seedAuditSpec(t, root, "A-001-one", "#1")
			calls := recordSpecAuditGH(t, nil)

			_, stderr, err := execute(t, "spec", "audit")
			if err == nil {
				t.Fatal("want a refusal")
			}
			for _, w := range []string{"git.example.com", "--repo", ".bitacora-repo"} {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error missing %q: %v", w, err)
				}
			}
			if strings.Contains(err.Error()+stderr, "s3cr3t") {
				t.Errorf("the origin URL leaked: %v", err)
			}
			if got := calls(); len(got) != 0 {
				t.Errorf("the forge was asked before the refusal: %v", got)
			}
		})
	}
}

func TestSpecAuditGiteaOriginWithOverride(t *testing.T) {
	root := makeRepo(t)
	withOrigin(t, root, giteaOrigin)
	writeBitacoraRepo(t, root, "mlorentedev/knowledge")
	seedAuditSpec(t, root, "A-001-one", "#1")
	seedAuditSpec(t, root, "B-002-two", "#2")
	calls := recordSpecAuditGH(t, map[string]string{
		"mlorentedev/knowledge/1": "open\tfalse", "mlorentedev/knowledge/2": "closed\tfalse",
	})

	stdout, _, err := execute(t, "spec", "audit")
	if err == nil {
		t.Fatalf("the zombie must keep its FAIL:\n%s", stdout)
	}
	for _, w := range []string{"[FAIL] B-002-two", "mlorentedev/knowledge#2 is CLOSED", "could not be checked"} {
		if !strings.Contains(stdout, w) {
			t.Errorf("output missing %q:\n%s", w, stdout)
		}
	}
	for _, c := range calls() {
		// No archive call at all: the specs are not on github.com, so there is
		// no default branch there to ask (not even `repos//contents/…`).
		if strings.Contains(c, "projects-toolkit") || strings.Contains(c, "/contents/") {
			t.Errorf("github.com was asked about the Gitea repository's archive: %s", c)
		}
	}
}

func TestSpecAuditArchiveUsesOrigin(t *testing.T) {
	root := makeRepo(t)
	withOrigin(t, root, "https://github.com/mlorentedev/kubelab.git")
	seedAuditSpec(t, root, "B-002-two", "#2")
	calls := recordSpecAuditGH(t, map[string]string{
		"mlorentedev/knowledge/2":                              "closed\tfalse",
		"mlorentedev/kubelab/contents/specs/archive/B-002-two": "proposal.md",
	})

	stdout, _, err := execute(t, "spec", "audit", "--repo", "mlorentedev/knowledge")
	if err != nil {
		t.Fatalf("an archived zombie is a WARN: %v\n%s", err, stdout)
	}
	for _, c := range calls() {
		if strings.Contains(c, "knowledge/contents") {
			t.Errorf("the archive was asked of the issue home, not the origin: %s", c)
		}
	}
}

// spec init and spec audit resolve one issue home for one repository: the
// init gate writes the frontmatter the audit later reads.
func TestSpecInitAndAuditAgreeOnTheIssueHome(t *testing.T) {
	root := makeRepo(t)
	withOrigin(t, root, "git@github.com:mlorentedev/kubelab.git")
	writeBitacoraRepo(t, root, "mlorentedev/knowledge")
	t.Setenv("DOTF_BITACORA_REPO", "mlorentedev/elsewhere")
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	pinClock(t)
	stubGhCmd(t, "OPEN\tGate")
	t.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+filepath.Dir(git))

	stdout, _, err := execute(t, "spec", "init", "CLI-007-x", "--issue", "7")
	if err != nil {
		t.Fatalf("spec init: %v", err)
	}
	if !strings.Contains(stdout, "Work-gate OK: mlorentedev/knowledge#7") {
		t.Fatalf("init resolved another home:\n%s", stdout)
	}

	calls := recordSpecAuditGH(t, map[string]string{"mlorentedev/knowledge/7": "open\tfalse"})
	if out, _, err := execute(t, "spec", "audit"); err != nil {
		t.Fatalf("audit: %v\n%s", err, out)
	}
	if got := calls(); len(got) != 1 || got[0] != "repos/mlorentedev/knowledge/issues/7" {
		t.Errorf("audit asked %v, want the issue init gated on", got)
	}
}

func TestSpecInitGiteaOriginFailsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the gh stub is a POSIX shell script")
	}
	for _, origin := range []string{giteaOrigin, "git@git.example.com:teledyne/projects-toolkit.git"} {
		t.Run(origin, func(t *testing.T) {
			root := makeRepo(t)
			withOrigin(t, root, origin)
			// A gh that leaves a mark: the refusal must come before any forge call.
			bin, mark := t.TempDir(), filepath.Join(t.TempDir(), "gh-ran")
			if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\ntouch '"+mark+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			_, _, err := execute(t, "spec", "init", "CLI-007-x", "--issue", "7")
			if err == nil || !strings.Contains(err.Error(), "--bitacora-repo") || !strings.Contains(err.Error(), "git.example.com") {
				t.Fatalf("want a refusal naming the host and --bitacora-repo, got %v", err)
			}
			if _, statErr := os.Stat(mark); statErr == nil {
				t.Error("gh ran before the refusal")
			}
			if _, statErr := os.Stat(filepath.Join(root, "specs", "CLI-007-x")); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("no spec may be scaffolded when the gate cannot resolve")
			}
		})
	}
}
