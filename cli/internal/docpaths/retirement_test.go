package docpaths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellGuardIsRetiredAfterCutover(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	precommit := readText(t, filepath.Join(repo, ".pre-commit-config.yaml"))
	if !strings.Contains(precommit, "entry: dotf lint doc-paths") ||
		strings.Contains(precommit, "scripts/check-doc-paths.sh") {
		t.Errorf("pre-commit has not cut over to dotf lint doc-paths")
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

func readText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
