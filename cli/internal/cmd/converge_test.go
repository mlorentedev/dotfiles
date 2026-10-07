package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// convergeFixture is a checkout with a harness tree and one manifest target,
// and an empty HOME: a machine before its first converge.
func convergeFixture(t *testing.T) (repo, home string) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	for rel, content := range map[string]string{
		"harness/manifest.json": `{"targets":[{"file":"AGENTS.md"}]}`,
		"AGENTS.md":             "# AGENTS\n",
	} {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DOTFILES_DIR", "")
	return repo, home
}

func TestConvergePlan_ListsApplicableReconcilersAndTouchesNothing(t *testing.T) {
	repo, home := convergeFixture(t)

	stdout, _, err := execute(t, "converge", "--plan", "--repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"converge plan", "records-mirror", "[CHANGE]", "2 to write", "1 to change"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("plan output lacks %q:\n%s", want, stdout)
		}
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("--plan wrote into HOME: %v", entries)
	}
}

func TestConverge_AppliesThenASecondRunChangesNothing(t *testing.T) {
	repo, home := convergeFixture(t)

	if _, _, err := execute(t, "converge", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".dotfiles", "AGENTS.md")); err != nil {
		t.Fatalf("apply did not mirror the records: %v", err)
	}
	stdout, _, err := execute(t, "converge", "--repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "0 changed") || !strings.Contains(stdout, "[ OK ]") {
		t.Errorf("second run should change nothing:\n%s", stdout)
	}
}
