package docpaths

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverGovernedFiles(t *testing.T) {
	repo := gitRepo(t)
	for _, rel := range []string{
		"AGENTS.md",
		"cli/README.md",
		"docs/README.md",
		"harness/generated/AGENTS.md",
		"src/other.md",
	} {
		writeRepoFile(t, repo, rel, "# fixture\n")
	}
	gitRun(t, repo, "add", ".")
	writeRepoFile(t, repo, "ai/untracked/AGENTS.md", "# untracked\n")
	writeRepoFile(t, repo, "ai/probe/AGENTS.md", "# intent to add\n")
	gitRun(t, repo, "add", "-N", "ai/probe/AGENTS.md")

	policy := Policy{
		Version:           1,
		GovernedBasenames: []string{"AGENTS.md", "README.md"},
		ExcludePrefixes:   []string{"harness/", "specs/", "docs/"},
		RequiredMembers:   []string{"AGENTS.md", "cli/README.md"},
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.hooksPath")

	got, err := Discover(repo, policy)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"AGENTS.md", "ai/probe/AGENTS.md", "cli/README.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Discover() = %v, want %v", got, want)
	}
}

func TestDiscoverFailsWhenRequiredMemberIsMissing(t *testing.T) {
	repo := gitRepo(t)
	writeRepoFile(t, repo, "AGENTS.md", "# fixture\n")
	gitRun(t, repo, "add", ".")
	policy := Policy{
		Version:           1,
		GovernedBasenames: []string{"AGENTS.md"},
		RequiredMembers:   []string{"AGENTS.md", "ai/agy/AGY.md"},
	}
	if _, err := Discover(repo, policy); err == nil {
		t.Fatal("Discover() succeeded without a required governed member")
	}
}

func gitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitRun(t, repo, "init", "-q")
	return repo
}

func gitRun(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = cleanGitConfigEnv(os.Environ())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func assertInside(t *testing.T, root, path string) {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || filepath.IsAbs(rel) {
		t.Fatalf("%s is outside %s", path, root)
	}
}
