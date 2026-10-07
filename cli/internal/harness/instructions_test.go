package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func instructionsRepo(t *testing.T) (repo, home string) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"agents":{"presence":[
		{"agent":"claude","file":".claude/CLAUDE.md","source":"ai/claude/CLAUDE.md"},
		{"agent":"copilot","file":".copilot/ci.md","source":"ai/copilot/ci.md","requires_command":"copilot"}
	]}}`)
	writeFile(t, filepath.Join(repo, "ai", "claude", "CLAUDE.md"), "# CLAUDE\n\n"+enforcedV1)
	writeFile(t, filepath.Join(repo, "ai", "copilot", "ci.md"), "# COPILOT\n")
	return repo, home
}

func none(string) bool { return false }

func TestDeployInstructions_WritesTheSourceOnAFreshMachine(t *testing.T) {
	repo, home := instructionsRepo(t)

	out, err := DeployInstructions(repo, home, none, false)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if string(got) != "# CLAUDE\n\n"+enforcedV1 {
		t.Errorf("deployed:\n%s", got)
	}
	if len(out) != 2 || out[0].Status != "deployed" || out[1].Status != "skipped" {
		t.Errorf("outcomes: %+v", out)
	}
}

// The presence roster and the skill catalog are written into the deployed file
// after the copy. Redeploying a changed source keeps them, so the next steps
// replace them in place instead of finding them gone.
func TestDeployInstructions_KeepsTheDeployOnlyRegionsAndIsIdempotent(t *testing.T) {
	repo, home := instructionsRepo(t)
	dst := filepath.Join(home, ".claude", "CLAUDE.md")
	writeFile(t, dst, "# CLAUDE\n\n"+enforcedV1+presence+catalog)
	writeFile(t, filepath.Join(repo, "ai", "claude", "CLAUDE.md"), "# CLAUDE\n\n"+enforcedV2)

	if _, err := DeployInstructions(repo, home, none, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	for _, must := range []string{"rule two", "AGENT-PRESENCE (sha256:p1)", "skill catalog"} {
		if !strings.Contains(string(got), must) {
			t.Errorf("deployed file lacks %q:\n%s", must, got)
		}
	}
	if strings.Contains(string(got), "sha256:aaa") {
		t.Errorf("the stale source region survived:\n%s", got)
	}
	if !DeployedMatchesSource(string(got), "# CLAUDE\n\n"+enforcedV2) {
		t.Error("the deployed file does not match its source")
	}
	again, err := DeployInstructions(repo, home, none, false)
	if err != nil || again[0].Status != "current" {
		t.Errorf("second deploy: want current, got %+v, %v", again, err)
	}
}

func TestDeployInstructions_DryRunWritesNothing(t *testing.T) {
	repo, home := instructionsRepo(t)

	out, err := DeployInstructions(repo, home, none, true)
	if err != nil || out[0].Status != "deployed" {
		t.Fatalf("dry run: %+v, %v", out, err)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("a dry run wrote into HOME: %v", entries)
	}
}

func TestDeployInstructions_ReplacesASymlinkWithACopy(t *testing.T) {
	repo, home := instructionsRepo(t)
	dst := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(repo, "ai", "claude", "CLAUDE.md"), dst); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := DeployInstructions(repo, home, none, false); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(dst); err != nil || fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("the deployed file is still a symlink (BUG-100): %v", err)
	}
}

func TestDeployInstructions_AMissingSourceFailsAfterDeployingTheRest(t *testing.T) {
	repo, home := instructionsRepo(t)
	if err := os.Remove(filepath.Join(repo, "ai", "copilot", "ci.md")); err != nil {
		t.Fatal(err)
	}

	_, err := DeployInstructions(repo, home, func(c string) bool { return c == "copilot" }, false)
	if err == nil || !strings.Contains(err.Error(), "ai/copilot/ci.md") {
		t.Fatalf("want an error naming the missing source, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".claude", "CLAUDE.md")); statErr != nil {
		t.Error("a missing source stopped the other targets from deploying")
	}
}

// copilot-instructions.md reserves an empty GENERATED slot, and
// compile-harness.sh's replace_region writes the skill catalog into it in
// place. Appending the catalog instead would leave the slot behind, and the
// next replace_region would fill both: two catalogs.
func TestDeployInstructions_TheCatalogFillsTheSourceSlotInPlace(t *testing.T) {
	const slot = "<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n"
	has := func(c string) bool { return c == "copilot" }
	for _, tc := range []struct {
		name, deployed, want string
	}{
		{"a filled slot keeps the catalog where it was",
			"# COPILOT\n" + catalog + "\nmore\n" + presence,
			"# COPILOT\n" + catalog + "\nmore v2\n" + presence},
		{"an empty slot is not carried over twice",
			"# COPILOT\n\n" + slot + "\nmore\n" + presence,
			"# COPILOT\n\n" + slot + "\nmore v2\n" + presence},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, home := instructionsRepo(t)
			writeFile(t, filepath.Join(repo, "ai", "copilot", "ci.md"), "# COPILOT\n\n"+slot+"\nmore v2\n")
			dst := filepath.Join(home, ".copilot", "ci.md")
			writeFile(t, dst, tc.deployed)

			if _, err := DeployInstructions(repo, home, has, false); err != nil {
				t.Fatal(err)
			}
			if got, _ := os.ReadFile(dst); string(got) != tc.want {
				t.Errorf("deployed:\n%s\nwant:\n%s", got, tc.want)
			}
			again, err := DeployInstructions(repo, home, has, false)
			if err != nil || again[1].Status != "current" {
				t.Errorf("second deploy: want current, got %+v, %v", again, err)
			}
		})
	}
}
