package vault

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeRepoURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/o/r":                 "github.com/o/r",
		"https://github.com/o/r.git":             "github.com/o/r",
		"https://github.com/o/r/":                "github.com/o/r",
		"git@github.com:o/r.git":                 "github.com/o/r",
		"ssh://git@github.com/o/r":               "github.com/o/r",
		"ssh://git@git.example.lan:2222/o/r.git": "git.example.lan/o/r",
		"https://user@GitHub.com/o/r":            "github.com/o/r",
		"https://github.com/Owner/Repo":          "github.com/Owner/Repo",
		"":                                       "",
		"/srv/git/r.git":                         "",
		"git.example.lan:homelab/notes.git":      "git.example.lan/homelab/notes",
		"C:/src/repo":                            "",
		"./rel/repo:x":                           "",
	}
	for in, want := range cases {
		if got := normalizeRepoURL(in); got != want {
			t.Errorf("normalizeRepoURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// repoWithOrigin is a repository directory named name whose origin is url.
func repoWithOrigin(t *testing.T, name, url string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	for _, args := range [][]string{{"init", "-q", root}, {"-C", root, "remote", "add", "origin", url}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func trackProject(t *testing.T, vault, slug, repoURL string) {
	t.Helper()
	dir := filepath.Join(vault, "10_projects", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: " + slug + "\nrepo_url: \"" + repoURL + "\"  # canonical\n---\n\n# " + slug + "\n"
	if err := os.WriteFile(filepath.Join(dir, "context.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWriteProjectEntrySkipsARepositoryTrackedUnderAnotherSlug(t *testing.T) {
	vault := t.TempDir()
	trackProject(t, vault, "homelab", "https://github.com/owner/kubelab")
	root := repoWithOrigin(t, "kubelab-checkout", "git@github.com:owner/kubelab.git")

	res, err := WriteProjectEntry(ProjectEntryOptions{VaultPath: vault, RepoRoot: root, Stack: "go", Date: "2026-10-07"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "skipped" || !strings.Contains(res.Reason, "10_projects/homelab") {
		t.Errorf("got %s %q, want skipped naming 10_projects/homelab", res.Action, res.Reason)
	}
	if _, err := os.Stat(filepath.Join(vault, "10_projects", "kubelab-checkout")); err == nil {
		t.Error("a duplicate entry was created for a repository the vault already tracks")
	}
}

func TestWriteProjectEntryWritesWhenTheMatchIsAmbiguousOrItsOwn(t *testing.T) {
	t.Run("two entries claim the repository", func(t *testing.T) {
		vault := t.TempDir()
		trackProject(t, vault, "a", "https://github.com/owner/tool")
		trackProject(t, vault, "b", "git@github.com:owner/tool.git")
		root := repoWithOrigin(t, "tool", "https://github.com/owner/tool.git")
		res, err := WriteProjectEntry(ProjectEntryOptions{VaultPath: vault, RepoRoot: root, Date: "2026-10-07"})
		if err != nil || res.Action != "written" {
			t.Errorf("got %s %q, %v; want written: an ambiguity is not resolved by guessing", res.Action, res.Reason, err)
		}
	})
	t.Run("the matching entry is this repository's own slug", func(t *testing.T) {
		vault := t.TempDir()
		trackProject(t, vault, "tool", "https://github.com/owner/tool")
		root := repoWithOrigin(t, "tool", "https://github.com/owner/tool")
		res, err := WriteProjectEntry(ProjectEntryOptions{VaultPath: vault, RepoRoot: root, Date: "2026-10-07"})
		if err != nil || res.Action != "written" {
			t.Errorf("got %s %q, %v; want written (skip-if-present per file)", res.Action, res.Reason, err)
		}
		if len(res.Skipped) == 0 || res.Skipped[0] != "context.md" {
			t.Errorf("the existing context.md should be left alone, skipped = %v", res.Skipped)
		}
	})
}

func TestWriteProjectEntryRecordsTheOriginAsRepoURL(t *testing.T) {
	vault := t.TempDir()
	root := repoWithOrigin(t, "tool", "git@github.com:owner/tool.git")
	if _, err := WriteProjectEntry(ProjectEntryOptions{VaultPath: vault, RepoRoot: root, Date: "2026-10-07"}); err != nil {
		t.Fatal(err)
	}
	ctx := filepath.Join(vault, "10_projects", "tool", "context.md")
	if got := frontmatterValue(ctx, "repo_url"); got != "git@github.com:owner/tool.git" {
		t.Errorf("repo_url = %q, want the origin URL, so the next lookup can find it", got)
	}
}

// fakeToken stands in for a token embedded in a remote URL. The URLs are
// assembled at run time so the source holds no user:password@host literal for
// a secret scanner to report.
const fakeToken = "s3cret"

func TestWithoutCredentials(t *testing.T) {
	cases := map[string]string{
		"https://oauth2:" + fakeToken + "@git.example.lan/o/r.git": "https://git.example.lan/o/r.git",
		"https://token@github.com/o/r":                             "https://github.com/o/r",
		"ssh://git:" + fakeToken + "@host/o/r":                     "ssh://host/o/r",
		"ssh://git@host:2222/o/r":                                  "ssh://git@host:2222/o/r",
		"git@github.com:o/r.git":                                   "git@github.com:o/r.git",
		"https://github.com/o/r":                                   "https://github.com/o/r",
	}
	for in, want := range cases {
		if got := withoutCredentials(in); got != want {
			t.Errorf("withoutCredentials(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteProjectEntryNeverRecordsOrReportsACredential(t *testing.T) {
	vault := t.TempDir()
	root := repoWithOrigin(t, "tool", "https://oauth2:"+fakeToken+"@git.example.lan/owner/tool.git")
	if _, err := WriteProjectEntry(ProjectEntryOptions{VaultPath: vault, RepoRoot: root, Date: "2026-10-07"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(vault, "10_projects", "tool", "context.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), fakeToken) || !strings.Contains(string(b), "https://git.example.lan/owner/tool.git") {
		t.Errorf("context.md should record the origin without its token:\n%s", b)
	}

	// A vault of its own, so the one entry matches and the skip is reached:
	// in the vault above, two entries would make the match ambiguous.
	tracked := t.TempDir()
	trackProject(t, tracked, "renamed", "https://git.example.lan/owner/tool")
	other := repoWithOrigin(t, "checkout", "https://oauth2:"+fakeToken+"@git.example.lan/owner/tool.git")
	res, err := WriteProjectEntry(ProjectEntryOptions{VaultPath: tracked, RepoRoot: other, Date: "2026-10-07"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "skipped" || !strings.Contains(res.Reason, "10_projects/renamed") || strings.Contains(res.Reason, fakeToken) {
		t.Errorf("want a skip naming 10_projects/renamed without the credential, got %q: %q", res.Action, res.Reason)
	}
}

func TestOriginURLIsEmptyInsideAnotherRepository(t *testing.T) {
	root := repoWithOrigin(t, "parent", "https://github.com/o/parent")
	nested := filepath.Join(root, "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := originURL(root); got != "https://github.com/o/parent" {
		t.Errorf("originURL(root) = %q", got)
	}
	if got := originURL(nested); got != "" {
		t.Errorf("originURL(nested) = %q, want \"\": the directory is not the repository", got)
	}
}
