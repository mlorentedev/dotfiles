package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// syncFixture is a checkout whose versions.conf marks two CLIs for mise, an
// empty mise config dir, and a fake mise that installs whatever is asked of it.
func syncFixture(t *testing.T) (repo, cfgDir string) {
	t.Helper()
	repo, cfgDir = t.TempDir(), t.TempDir()
	conf := "GO_VERSION=1.26.0\n# mise: cli\nAGE_VERSION=1.3.1\n# mise: cli\nJQ_VERSION=1.8.2\n"
	if err := os.WriteFile(filepath.Join(repo, "versions.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	installed := map[string]bool{}
	saved := toolsSyncRunner
	toolsSyncRunner = func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && len(args) == 1 && args[0] == "install":
			installed["age"], installed["jq"] = true, true
			return nil, nil
		case name == "mise" && len(args) == 2 && args[0] == "which" && installed[args[1]]:
			return []byte("/bin/" + args[1]), nil
		case name == "/bin/age":
			return []byte("v1.3.1"), nil
		case name == "/bin/jq":
			return []byte("jq-1.8.2"), nil
		}
		return nil, errors.New("not installed")
	}
	t.Cleanup(func() { toolsSyncRunner = saved })
	t.Setenv("MISE_CONFIG_DIR", cfgDir)
	t.Setenv("DOTFILES_REPO_DIR", repo)
	return repo, cfgDir
}

func TestToolsSync_DryRunListsTheWorkAndWritesNothing(t *testing.T) {
	repo, cfgDir := syncFixture(t)

	stdout, _, err := execute(t, "tools", "sync", "--dry-run", "--versions", filepath.Join(repo, "versions.conf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"conf.d/dotfiles.toml", "would write", "age 1.3.1", "jq 1.8.2", "to install: age, jq"} {
		if !strings.Contains(filepath.ToSlash(stdout), want) {
			t.Errorf("dry run lacks %q:\n%s", want, stdout)
		}
	}
	if entries, _ := os.ReadDir(cfgDir); len(entries) != 0 {
		t.Errorf("a dry run wrote into the mise config dir: %v", entries)
	}
}

func TestToolsSync_InstallsAndASecondRunReportsNothingToDo(t *testing.T) {
	repo, _ := syncFixture(t)
	versions := filepath.Join(repo, "versions.conf")

	if _, _, err := execute(t, "tools", "sync", "--versions", versions); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := execute(t, "tools", "sync", "--versions", versions)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "2 tool(s) at their pin; nothing to do") {
		t.Errorf("second run should be a no-op:\n%s", stdout)
	}
}
