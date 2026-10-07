package initrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// repoOn is a fresh repository whose unborn branch is branch, with origin set
// to url when url is not empty.
func repoOn(t *testing.T, branch, url string) string {
	t.Helper()
	root := t.TempDir()
	gitIn(t, root, "init", "-q", "-b", branch)
	if url != "" {
		gitIn(t, root, "remote", "add", "origin", url)
	}
	return root
}

// fakeToken stands in for a token embedded in a remote URL. The URL is
// assembled at run time so the source holds no user:password@host literal for
// a secret scanner to report.
const fakeToken = "s3cret"

func credURL(hostPath string) string { return "https://oauth2:" + fakeToken + "@" + hostPath }

func TestClassifyOrigin(t *testing.T) {
	const gitea = "ssh://git@git.example.lan:2222/owner/repo.git"
	cases := []struct {
		name, url, marker string
		want              Forge
	}{
		{"no origin", "", "", ForgeAbsent},
		{"github ssh", "git@github.com:owner/repo.git", "", ForgeGitHub},
		{"github https", "https://github.com/owner/repo", "", ForgeGitHub},
		{"self-hosted, no marker", gitea, "", ForgeOther},
		{"self-hosted with .gitea/", gitea, ".gitea", ForgeGitea},
		{"self-hosted with .forgejo/", gitea, ".forgejo", ForgeGitea},
		// A .gitea/ directory does not make a github.com remote Gitea.
		{"github with .gitea/", "git@github.com:owner/repo.git", ".gitea", ForgeGitHub},
	}
	for _, c := range cases {
		root := repoOn(t, "main", c.url)
		if c.marker != "" {
			if err := os.MkdirAll(filepath.Join(root, c.marker), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if got, _ := ClassifyOrigin(root); got != c.want {
			t.Errorf("%s: ClassifyOrigin = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDefaultBranch(t *testing.T) {
	t.Run("not a repository falls back to main", func(t *testing.T) {
		if got := DefaultBranch(t.TempDir()); got != "main" {
			t.Errorf("got %q, want main", got)
		}
	})
	t.Run("the checkout's unborn branch", func(t *testing.T) {
		if got := DefaultBranch(repoOn(t, "master", "")); got != "master" {
			t.Errorf("got %q, want master", got)
		}
	})
	t.Run("a detached HEAD reads the local branches, not a literal main", func(t *testing.T) {
		root := repoOn(t, "master", "ssh://git@host/o/r.git")
		gitIn(t, root, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "c")
		gitIn(t, root, "checkout", "-q", "--detach")
		if got := DefaultBranch(root); got != "master" {
			t.Errorf("got %q, want master", got)
		}
	})
	t.Run("origin's HEAD wins over the checkout's branch", func(t *testing.T) {
		root := repoOn(t, "feature", "ssh://git@host/o/r.git")
		gitIn(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
		if got := DefaultBranch(root); got != "trunk" {
			t.Errorf("got %q, want trunk", got)
		}
	})
}

func TestWriteCIFollowsForgeAndDefaultBranch(t *testing.T) {
	root := repoOn(t, "master", "ssh://git@git.example.lan:2222/owner/repo.git")
	if err := os.MkdirAll(filepath.Join(root, ".gitea"), 0o755); err != nil {
		t.Fatal(err)
	}
	if action, err := WriteCI(root, "go"); err != nil || action != "created" {
		t.Fatalf("WriteCI = %q, %v; want created", action, err)
	}
	got := readFile(t, filepath.Join(root, ".gitea", "workflows", "ci.yml"))
	if !strings.Contains(got, "branches: [\"master\"]") || strings.Contains(got, "branches: [main]") {
		t.Errorf("ci.yml should trigger on master only:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".github")); err == nil {
		t.Error("a Gitea repository should not get a .github directory")
	}
	// A second run sees the ci.yml where it was written.
	if action, _ := WriteCI(root, "go"); action != "skipped" {
		t.Errorf("second WriteCI = %q, want skipped", action)
	}
}

func TestWriteCISkipsAGitHubCIInAGiteaRepository(t *testing.T) {
	root := repoOn(t, "main", "ssh://git@host/o/r.git")
	for _, d := range []string{".gitea", filepath.Join(".github", "workflows")} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte("name: own\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if action, _ := WriteCI(root, "go"); action != "skipped" {
		t.Errorf("WriteCI = %q, want skipped: the repository already has a ci.yml", action)
	}
}

func TestYAMLBranch(t *testing.T) {
	cases := map[string]string{
		"main":      `"main"`,
		"1.0":       `"1.0"`, // bare, a YAML float
		"on":        `"on"`,  // bare, a YAML 1.1 boolean
		"feat: odd": `"feat: odd"`,
		`say"hi`:    `"say\"hi"`,
	}
	for in, want := range cases {
		if got := yamlBranch(in); got != want {
			t.Errorf("yamlBranch(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestOriginHostNeverEchoesCredentials(t *testing.T) {
	cases := map[string]string{
		credURL("git.example.lan/o/r.git"):       "git.example.lan",
		"ssh://git@git.example.lan:2222/o/r.git": "git.example.lan",
		"git@git.example.lan:o/r.git":            "git.example.lan",
		"/srv/git/r.git":                         "a local path",
	}
	for in, want := range cases {
		if got := originHost(in); got != want {
			t.Errorf("originHost(%q) = %q, want %q", in, got, want)
		}
	}
	root := repoOn(t, "main", credURL("git.example.lan/o/r.git"))
	if _, detail := initGithubWithDryRun(root, true); strings.Contains(detail, fakeToken) {
		t.Errorf("the github step printed the credential: %q", detail)
	}
}

func TestInitGithubStepNamesANonGitHubOrigin(t *testing.T) {
	root := repoOn(t, "master", "ssh://git@git.example.lan:2222/owner/repo.git")
	status, detail := initGithubWithDryRun(root, true)
	if status != "skipped" || !strings.Contains(detail, "not a GitHub remote") || strings.Contains(detail, "no origin remote") {
		t.Errorf("got %s %q; want skipped naming a non-GitHub origin", status, detail)
	}
	if status, detail := initGithubWithDryRun(repoOn(t, "main", ""), true); status != "skipped" || !strings.Contains(detail, "no origin remote yet") {
		t.Errorf("no origin: got %s %q", status, detail)
	}
}

func TestWriteCISkipsACIUnderTheOtherGiteaFamilyDirectory(t *testing.T) {
	// An empty .forgejo/ makes WorkflowDir .forgejo/workflows, while the
	// repository's CI lives under .gitea/ from before a migration.
	root := repoOn(t, "main", "ssh://git@host/o/r.git")
	for _, d := range []string{".forgejo", filepath.Join(".gitea", "workflows")} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".gitea", "workflows", "ci.yml"), []byte("name: own\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if action, _ := WriteCI(root, "go"); action != "skipped" {
		t.Errorf("WriteCI = %q, want skipped: .gitea/workflows/ci.yml exists", action)
	}
}

func TestParseOriginRepoErrorNeverQuotesTheURL(t *testing.T) {
	_, err := ParseOriginRepo(credURL("github.com/only-owner"))
	if err == nil {
		t.Fatal("want an error: the path has no owner/name")
	}
	if strings.Contains(err.Error(), fakeToken) || !strings.Contains(err.Error(), "github.com") {
		t.Errorf("error should name the host and not the token: %v", err)
	}
}
