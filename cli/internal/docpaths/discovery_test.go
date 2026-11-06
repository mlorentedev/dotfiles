package docpaths

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
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
	setIncompleteGitConfig(t)

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

func TestRepositoryRootIgnoresGitStderrOnSuccess(t *testing.T) {
	repo := gitRepo(t)
	t.Setenv("GIT_TRACE", "1")

	got, err := RepositoryRoot(repo)
	if err != nil {
		t.Fatal(err)
	}
	assertSameFile(t, got, repo)
}

func TestRepositoryLocationReturnsWorkingDirectoryPrefix(t *testing.T) {
	repo := gitRepo(t)
	nested := filepath.Join(repo, "ai", "nan")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	root, prefix, err := RepositoryLocation(nested)
	if err != nil {
		t.Fatal(err)
	}
	assertSameFile(t, root, repo)
	if prefix != "ai/nan/" {
		t.Fatalf("RepositoryLocation() prefix = %q, want %q", prefix, "ai/nan/")
	}
}

func TestDiscoverIncludesGitFailure(t *testing.T) {
	repo := t.TempDir()
	want := gitFailureDetail(t, repo, "ls-files", "-z", "--", "*.md")

	_, err := Discover(repo, Policy{Version: 1, GovernedBasenames: []string{"AGENTS.md"}})
	if err == nil {
		t.Fatal("Discover() succeeded outside a Git checkout")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Discover() error = %q, want Git stderr %q", err, want)
	}
}

func TestDiscoverFromLinkedWorktree(t *testing.T) {
	repo := gitRepo(t)
	writeRepoFile(t, repo, "AGENTS.md", "# fixture\n")
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
		"commit", "-qm", "fixture")
	worktree := filepath.Join(t.TempDir(), "linked worktree")
	gitRun(t, repo, "worktree", "add", "-q", "-b", "fixture-linked", worktree)
	setIncompleteGitConfig(t)

	policy := Policy{
		Version:           1,
		GovernedBasenames: []string{"AGENTS.md"},
		RequiredMembers:   []string{"AGENTS.md"},
	}
	got, err := Discover(worktree, policy)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"AGENTS.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Discover() = %v, want %v", got, want)
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

func gitFailureDetail(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = cleanGitConfigEnv(os.Environ())
	_, err := cmd.Output()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("git %v error = %v, want exit error", args, err)
	}
	return strings.TrimSpace(string(exit.Stderr))
}

func setIncompleteGitConfig(t *testing.T) {
	t.Helper()
	for _, name := range []string{"GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0"} {
		value, ok := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if ok {
				_ = os.Setenv(name, value)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
}

func assertSameFile(t *testing.T, left, right string) {
	t.Helper()
	leftInfo, err := os.Stat(left)
	if err != nil {
		t.Fatal(err)
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(leftInfo, rightInfo) {
		t.Fatalf("%q and %q are not the same directory", left, right)
	}
}
