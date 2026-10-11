package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
	"github.com/mlorentedev/dotfiles/cli/internal/identity"
	"github.com/mlorentedev/dotfiles/cli/internal/tools"
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
		"env-contract.json":     `{"env_vars": []}`,
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
	// The identity epilogue reads bw, gh and the vault of whoever runs the
	// test; the fixture is a machine with none of them and no terminal.
	stubIdentity(t, identity.Facts{}, nil)
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
	pathFile := filepath.Base(env.DefaultOutput(runtime.GOOS, home)) // paths.ps1 on Windows
	for _, want := range []string{"converge plan", "records-mirror", "env-generate", "[CHANGE]", "3 to write", pathFile + " to write", "2 to change"} {
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
	records := ""
	for _, e := range got.Entries {
		if e.Name == "records-mirror" {
			records = e.Status
		}
	}
	if got.Result != "ok" || got.Changed != 0 || records != "ok" {
		t.Errorf("second run's report should record a converged machine:\n%s", raw)
	}
}

// A machine from zero: no checkout anywhere, run from outside any repository.
// The plan resolves the default checkout path, plans the clone, and holds the
// steps that read the checkout back instead of failing on its absence.
func TestConvergePlan_FromZeroPlansTheCloneFirst(t *testing.T) {
	_, home := convergeFixture(t)
	t.Setenv("DOTFILES_REPO_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Chdir(t.TempDir())
	saved := convergeOptions
	convergeOptions = func() converge.Options {
		o := saved()
		o.GitRun = gitconfig.ExecRunner
		o.CloneURL = "https://example.invalid/dotfiles.git"
		return o
	}

	stdout, _, err := execute(t, "converge", "--plan")
	if err != nil {
		t.Fatalf("a plan from zero must not fail: %v\n%s", err, stdout)
	}
	want := filepath.Join(home, "Projects", "dotfiles")
	for _, s := range []string{"checkout", "clone https://example.invalid/dotfiles.git into " + want, "waits for checkout"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("plan lacks %q:\n%s", s, stdout)
		}
	}
	if _, err := os.Stat(want); err == nil {
		t.Error("a plan must not clone")
	}
}

// Run from inside another project, converge plans against the declared
// checkout instead of refusing the project it happens to stand in.
func TestConvergePlan_FromAnotherProjectUsesTheDeclaredCheckout(t *testing.T) {
	_, home := convergeFixture(t)
	t.Setenv("DOTFILES_REPO_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	other := t.TempDir()
	if err := os.Mkdir(filepath.Join(other, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(other)

	if got, want := convergeCheckout(home), env.DefaultCheckoutDir(home); got != want {
		t.Errorf("convergeCheckout() = %q, want the default checkout %q", got, want)
	}
}

// The tools step puts on PATH the dirs the production wiring names, so they
// must be where the catalog installs and where mise keeps its shims: a dir
// that drifts from the installer's Dest leaves what it installs unreachable.
func TestConvergeOptions_ToolsBinDirsAreWhereTheToolLayerPlaces(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MISE_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "")
	o := convergeOptions()
	in, ok := o.ToolsCatalog.(*tools.Installer)
	if !ok {
		t.Fatalf("ToolsCatalog is %T, want *tools.Installer", o.ToolsCatalog)
	}
	want := []string{tools.MiseShimsDir(home, runtime.GOOS, os.Getenv), in.Dest}
	if strings.Join(o.ToolsBinDirs, "|") != strings.Join(want, "|") {
		t.Errorf("ToolsBinDirs = %v, want %v", o.ToolsBinDirs, want)
	}
}
