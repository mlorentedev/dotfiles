package initrepo

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every stack gets lint.yml, none included (#2965): a scaffolded repository
// holds workflows from its first commit, whatever its language.
func TestWriteLintForEveryStack(t *testing.T) {
	for _, stack := range []string{"go", "python", "node", "ts", "none"} {
		root := t.TempDir()
		report, err := Init(InitOptions{Root: root, Stack: stack, Date: "2026-11-02", SkipGithub: true})
		if err != nil {
			t.Fatalf("Init(%s): %v", stack, err)
		}
		if s := stepStatus(report, "lint"); s != "ok" {
			t.Errorf("stack %s: lint step status = %q, want ok", stack, s)
		}
		got := readFile(t, filepath.Join(root, ".github", "workflows", "lint.yml"))
		for _, want := range []string{"actionlint", "-shellcheck=shellcheck", "command -v shellcheck", "contents: read", "pull_request:"} {
			if !strings.Contains(got, want) {
				t.Errorf("stack %s: lint.yml missing %q", stack, want)
			}
		}
	}
}

func TestInitDryRunListsLint(t *testing.T) {
	root := t.TempDir()
	report, err := Init(InitOptions{Root: root, Stack: "none", Date: "2026-11-02", DryRun: true, SkipGithub: true})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !strings.Contains(stepDetail(report, "lint"), "lint.yml created") {
		t.Errorf("dry-run lint detail = %q, want it to list lint.yml", stepDetail(report, "lint"))
	}
	if _, err := os.Stat(filepath.Join(root, ".github")); err == nil {
		t.Error("dry-run wrote .github")
	}
}

func TestWriteLintFollowsForgeAndDefaultBranch(t *testing.T) {
	root := repoOn(t, "master", "ssh://git@git.example.lan:2222/owner/repo.git")
	if err := os.MkdirAll(filepath.Join(root, ".gitea"), 0o755); err != nil {
		t.Fatal(err)
	}
	if action, err := WriteLintOpts(root, false); err != nil || action != "created" {
		t.Fatalf("WriteLintOpts = %q, %v; want created", action, err)
	}
	got := readFile(t, filepath.Join(root, ".gitea", "workflows", "lint.yml"))
	if !strings.Contains(got, "branches: [\"master\"]") || strings.Contains(got, "branches: [main]") {
		t.Errorf("lint.yml should trigger on master only:\n%s", got)
	}
	// actionlint defaults to .github/workflows; a Gitea repository names its own.
	if !strings.Contains(got, "-shellcheck=shellcheck .gitea/workflows/*.y*ml") || strings.Contains(got, ".github/workflows/*") {
		t.Errorf("lint.yml should lint .gitea/workflows:\n%s", got)
	}
}

func TestWriteLintSkipsAnExistingLintInAnyForgeDir(t *testing.T) {
	for _, d := range []string{".github", ".gitea", ".forgejo"} {
		root := t.TempDir()
		dest := filepath.Join(root, d, "workflows", "lint.yml")
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, []byte("name: own\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if action, _ := WriteLintOpts(root, false); action != "skipped" {
			t.Errorf("%s: WriteLintOpts = %q, want skipped", d, action)
		}
		if got := readFile(t, dest); got != "name: own\n" {
			t.Errorf("%s: existing lint.yml was rewritten:\n%s", d, got)
		}
	}
}

// The template's pins live in one place: actionlint's version is versions.conf's,
// and actions/checkout is the commit this repository's own workflows use.
func TestLintTemplatePinsMatchTheRepository(t *testing.T) {
	raw, err := ReadTemplate("lint.yml")
	if err != nil {
		t.Fatal(err)
	}
	tmpl := string(raw)
	repo := filepath.Join("..", "..", "..")

	conf := readFile(t, filepath.Join(repo, "versions.conf"))
	want := regexp.MustCompile(`(?m)^ACTIONLINT_VERSION=(\S+)$`).FindStringSubmatch(conf)
	got := regexp.MustCompile(`ACTIONLINT_VERSION: (\S+)`).FindStringSubmatch(tmpl)
	if want == nil || got == nil || got[1] != want[1] {
		t.Errorf("lint.yml ACTIONLINT_VERSION = %v, versions.conf = %v: bump both, and ACTIONLINT_SHA256 with them", got, want)
	}

	checkout := regexp.MustCompile(`actions/checkout@([0-9a-f]{40})`)
	ci := readFile(t, filepath.Join(repo, ".github", "workflows", "ci.yml"))
	if w, g := checkout.FindStringSubmatch(ci), checkout.FindStringSubmatch(tmpl); w == nil || g == nil || g[1] != w[1] {
		t.Errorf("lint.yml checkout pin %v differs from .github/workflows/ci.yml %v", g, w)
	}

	// Every action is a full commit SHA with its tag as a comment, the form
	// scripts/check-actions-pinned.sh accepts.
	pinned := regexp.MustCompile(`^\s*-?\s*uses:\s*[\w.-]+/[\w./-]+@[0-9a-f]{40} # v\d`)
	for _, line := range strings.Split(tmpl, "\n") {
		if strings.Contains(line, "uses:") && !pinned.MatchString(line) {
			t.Errorf("unpinned action in lint.yml: %s", line)
		}
	}
}

func stepDetail(r InitReport, name string) string {
	for _, s := range r.Steps {
		if s.Name == name {
			return s.Detail
		}
	}
	return ""
}
