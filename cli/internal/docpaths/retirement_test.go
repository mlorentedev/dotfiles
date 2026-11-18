package docpaths

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// preCommitHook is the part of a pre-commit hook the cutover is read from.
type preCommitHook struct {
	ID            string   `yaml:"id"`
	Entry         string   `yaml:"entry"`
	PassFilenames *bool    `yaml:"pass_filenames"`
	Stages        []string `yaml:"stages"`
}

// docPathHooks parses a pre-commit configuration and returns the hooks whose
// entry runs the doc-path check, old or new. A commented-out hook is not one.
func docPathHooks(t *testing.T, raw string) []preCommitHook {
	t.Helper()
	var config struct {
		Repos []struct {
			Hooks []preCommitHook `yaml:"hooks"`
		} `yaml:"repos"`
	}
	if err := yaml.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatalf("parse pre-commit configuration: %v", err)
	}
	var out []preCommitHook
	for _, repo := range config.Repos {
		for _, hook := range repo.Hooks {
			if strings.Contains(hook.Entry, "doc-paths") {
				out = append(out, hook)
			}
		}
	}
	return out
}

// cutOver reports what keeps hooks from being the one hook the cutover left:
// `dotf lint doc-paths`, over the whole tree, at pre-commit.
func cutOver(hooks []preCommitHook) string {
	if len(hooks) != 1 {
		return "want one doc-path hook"
	}
	h := hooks[0]
	switch {
	case h.Entry != "dotf lint doc-paths":
		return "entry is " + h.Entry
	case h.PassFilenames == nil || *h.PassFilenames:
		return "pass_filenames is not false: the lint discovers the governed files itself"
	case !reflect.DeepEqual(h.Stages, []string{"pre-commit"}):
		return "stages are not [pre-commit]"
	}
	return ""
}

func TestShellGuardIsRetiredAfterCutover(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	if why := cutOver(docPathHooks(t, readText(t, filepath.Join(repo, ".pre-commit-config.yaml")))); why != "" {
		t.Errorf("pre-commit has not cut over to dotf lint doc-paths: %s", why)
	}
	ci := readText(t, filepath.Join(repo, ".github", "workflows", "ci.yml"))
	if strings.Contains(ci, "scripts/check-doc-paths.sh") {
		t.Errorf("ci.yml still invokes the retired shell guard")
	}
	for _, rel := range []string{
		"scripts/check-doc-paths.sh",
		"tests/check-doc-paths.bats",
		"tests/check-doc-paths-real.bats",
	} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("%s still exists after cutover", rel)
		}
	}
}

// The guard reads the hook, not the file's text: each of these configurations
// holds the substring the guard used to look for (CLI-141).
func TestCutOverReadsTheHookNotTheText(t *testing.T) {
	const head = "repos:\n  - repo: local\n    hooks:\n"
	hook := func(lines ...string) string { return head + "      - " + strings.Join(lines, "\n        ") + "\n" }
	good := []string{"id: check-doc-paths", "entry: dotf lint doc-paths", "language: system", "pass_filenames: false", "stages: [pre-commit]"}
	if why := cutOver(docPathHooks(t, hook(good...))); why != "" {
		t.Fatalf("the hook as committed is refused: %s", why)
	}
	for name, config := range map[string]string{
		"commented out":        head + "      - id: other\n        entry: other\n      # - id: check-doc-paths\n      #   entry: dotf lint doc-paths\n",
		"passes filenames":     hook("id: check-doc-paths", "entry: dotf lint doc-paths", "pass_filenames: true", "stages: [pre-commit]"),
		"filenames by default": hook("id: check-doc-paths", "entry: dotf lint doc-paths", "stages: [pre-commit]"),
		"another stage":        hook("id: check-doc-paths", "entry: dotf lint doc-paths", "pass_filenames: false", "stages: [manual]"),
		"the retired script":   hook("id: check-doc-paths", "entry: scripts/check-doc-paths.sh", "pass_filenames: false", "stages: [pre-commit]"),
		"both hooks":           hook(good...) + "      - id: old\n        entry: scripts/check-doc-paths.sh\n",
	} {
		if cutOver(docPathHooks(t, config)) == "" {
			t.Errorf("%s: accepted as cut over", name)
		}
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
