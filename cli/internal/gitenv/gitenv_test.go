package gitenv

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestWithoutRepository_DropsTheLocationVariablesAndKeepsEverythingElse(t *testing.T) {
	got := WithoutRepository([]string{
		"GIT_DIR=/x", "git_work_tree=/y", "GIT_SHALLOW_FILE=/s",
		"GIT_CONFIG_GLOBAL=/g", "GIT_CONFIG_COUNT=1", "HOME=/h", "NOEQUALS",
	})
	want := []string{"GIT_CONFIG_GLOBAL=/g", "GIT_CONFIG_COUNT=1", "HOME=/h", "NOEQUALS"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// git names its own repository-local variables. Each one that says where the
// repository is must be in the list; the GIT_CONFIG* ones carry configuration
// a caller passes on purpose (`git -c`), not a location, so they stay.
func TestRepositoryVars_CoverWhatGitCallsLocal(t *testing.T) {
	out, err := exec.Command("git", "rev-parse", "--local-env-vars").Output()
	if err != nil {
		t.Fatalf("git rev-parse --local-env-vars: %v", err)
	}
	for _, name := range strings.Fields(string(out)) {
		if strings.HasPrefix(name, "GIT_CONFIG") {
			continue
		}
		if !slices.Contains(RepositoryVars, name) {
			t.Errorf("git calls %s repository-local, and the list does not drop it", name)
		}
	}
}

func TestRepositoryVars_AreUpperCaseAndSorted(t *testing.T) {
	for _, name := range RepositoryVars {
		if name != strings.ToUpper(name) {
			t.Errorf("%s: WithoutRepository compares upper-cased keys", name)
		}
	}
	if !slices.IsSorted(RepositoryVars) {
		t.Errorf("RepositoryVars is not sorted: %v", RepositoryVars)
	}
}
