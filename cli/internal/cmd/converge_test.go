package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// convergeFixture is a checkout with a harness tree and one manifest target,
// and an empty HOME: a machine before its first converge.
func convergeFixture(t *testing.T) (repo, home string) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	for rel, content := range map[string]string{
		"harness/manifest.json": `{"targets":[{"file":"AGENTS.md"}]}`,
		"AGENTS.md":             "# AGENTS\n",
		"ai/deploy.json":        `{"version": 3, "configs": []}`,
	} {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The real deploy runs scripts/compile-harness.sh, which this fixture does
	// not have; a test that needs it opts in with its own runner (lesson 335).
	saved := convergeOptions
	convergeOptions = func() converge.Options {
		return converge.Options{
			RunHarnessDeploy: func(converge.Env) error { return nil },
			RenderConfigs:    func(string) error { return nil },
			ResolvePath:      func(string) string { return "" },
		}
	}
	t.Cleanup(func() { convergeOptions = saved })
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

// An apply persists what it did under the user state directory, and a second
// run on a converged machine records zero changes. A plan persists nothing.
func TestConverge_SecondRunIsANoOpAndPersistsTheReport(t *testing.T) {
	repo, home := convergeFixture(t)
	state := filepath.Join(home, "state")
	t.Setenv("XDG_STATE_HOME", state)
	report := filepath.Join(state, "dotfiles", "converge", "last.json")

	if _, _, err := execute(t, "converge", "--plan", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(report); !os.IsNotExist(err) {
		t.Fatalf("a plan persisted a report: %v", err)
	}
	for range 2 {
		if _, _, err := execute(t, "converge", "--repo", repo); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("no report after an apply: %v", err)
	}
	var got struct {
		Result  string `json:"result"`
		Changed int    `json:"changed"`
		Entries []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("report is not JSON: %v\n%s", err, raw)
	}
	if got.Result != "ok" || got.Changed != 0 || len(got.Entries) == 0 || got.Entries[0].Status != "ok" {
		t.Errorf("second run's report should record a converged machine:\n%s", raw)
	}
}
