package initrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

func readLaneFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func TestBootstrapLane_ConfigLoads(t *testing.T) {
	root := t.TempDir()
	res, err := BootstrapLane(root, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Config != "created" {
		t.Fatalf("config action = %q, want created", res.Config)
	}
	cfg, err := lane.LoadConfig(root)
	if err != nil {
		t.Fatalf("lane.LoadConfig rejects the scaffolded file: %v", err)
	}
	if len(cfg.Queue) != 0 {
		t.Errorf("queue = %v, want empty", cfg.Queue)
	}
	if cfg.BranchPrefix != "fix" || cfg.Base != "origin/main" || cfg.Template != ".dotf/lane-prompt.md" {
		t.Errorf("config = %+v, want the documented defaults", cfg)
	}
}

func TestBootstrapLane_PromptRenders(t *testing.T) {
	root := t.TempDir()
	if _, err := BootstrapLane(root, false, false); err != nil {
		t.Fatal(err)
	}
	tmpl := readLaneFile(t, root, lanePromptPath)
	got, err := lane.RenderPrompt(tmpl, lane.PromptData{Number: 7, Title: "a title", Branch: "fix/7-a-title", Remaining: []int{8, 9}})
	if err != nil {
		t.Fatalf("lane.RenderPrompt rejects the scaffolded prompt: %v", err)
	}
	for _, want := range []string{"#7 a title", "fix/7-a-title", "#8 #9"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered prompt lacks %q", want)
		}
	}
}

func TestBootstrapLane_AgentsSection(t *testing.T) {
	t.Run("creates AGENTS.md when absent", func(t *testing.T) {
		root := t.TempDir()
		res, err := BootstrapLane(root, false, false)
		if err != nil {
			t.Fatal(err)
		}
		if res.Agents != "created" {
			t.Fatalf("agents action = %q, want created", res.Agents)
		}
		if !strings.Contains(readLaneFile(t, root, "AGENTS.md"), lanesHeading+"\n") {
			t.Error("AGENTS.md lacks the ## Lanes heading")
		}
	})
	t.Run("appends after prior content", func(t *testing.T) {
		root := t.TempDir()
		prior := "# AGENTS.md\n\nkeep this line\n"
		if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(prior), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := BootstrapLane(root, false, false)
		if err != nil {
			t.Fatal(err)
		}
		if res.Agents != "appended" {
			t.Fatalf("agents action = %q, want appended", res.Agents)
		}
		got := readLaneFile(t, root, "AGENTS.md")
		if !strings.HasPrefix(got, prior) || !strings.Contains(got, lanesHeading+"\n") {
			t.Errorf("prior content not kept, or section missing:\n%s", got)
		}
	})
}

func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		out[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestBootstrapLane_Idempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := BootstrapLane(root, false, false); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	res, err := BootstrapLane(root, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if res != (LaneResult{Config: "unchanged", Prompt: "unchanged", Agents: "unchanged"}) {
		t.Errorf("second run = %+v, want all unchanged", res)
	}
	after := snapshot(t, root)
	for p, b := range before {
		if after[p] != b {
			t.Errorf("%s changed on the second run", p)
		}
	}
	if len(after) != len(before) {
		t.Errorf("second run added files: %d -> %d", len(before), len(after))
	}
}

func TestBootstrapLane_KeepsExisting(t *testing.T) {
	root := t.TempDir()
	dot := filepath.Join(root, ".dotf")
	if err := os.MkdirAll(dot, 0o755); err != nil {
		t.Fatal(err)
	}
	own := map[string]string{
		filepath.Join(dot, "lane.json"):      `{"queue":[5],"branch_prefix":"feat","base":"origin/dev","template":".dotf/lane-prompt.md"}`,
		filepath.Join(dot, "lane-prompt.md"): "my own prompt {{.Number}}\n",
		filepath.Join(root, "AGENTS.md"):     "# AGENTS.md\n\n" + lanesHeading + "\n\nmy own lanes text\n",
	}
	for p, c := range own {
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, err := BootstrapLane(root, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if res != (LaneResult{Config: "unchanged", Prompt: "unchanged", Agents: "unchanged"}) {
		t.Errorf("run over existing files = %+v, want all unchanged", res)
	}
	for p, c := range own {
		if got, _ := os.ReadFile(p); string(got) != c {
			t.Errorf("%s was rewritten without --force", p)
		}
	}
}

func TestBootstrapLane_Force(t *testing.T) {
	root := t.TempDir()
	dot := filepath.Join(root, ".dotf")
	if err := os.MkdirAll(dot, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"queue":[5],"branch_prefix":"feat","base":"origin/dev","template":".dotf/lane-prompt.md"}`
	agents := "# AGENTS.md\n\nbefore\n\n" + lanesHeading + "\n\nstale lanes text\n\n## After\n\nafter stays\n"
	for p, c := range map[string]string{
		filepath.Join(dot, "lane.json"):      cfg,
		filepath.Join(dot, "lane-prompt.md"): "stale prompt\n",
		filepath.Join(root, "AGENTS.md"):     agents,
	} {
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, err := BootstrapLane(root, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if res != (LaneResult{Config: "unchanged", Prompt: "replaced", Agents: "replaced"}) {
		t.Errorf("--force run = %+v", res)
	}
	if got := readLaneFile(t, root, ".dotf/lane.json"); got != cfg {
		t.Errorf("lane.json was touched by --force: %s", got)
	}
	if got := readLaneFile(t, root, ".dotf/lane-prompt.md"); strings.Contains(got, "stale prompt") {
		t.Error("prompt not replaced")
	}
	got := readLaneFile(t, root, "AGENTS.md")
	if strings.Contains(got, "stale lanes text") || !strings.Contains(got, "before") || !strings.Contains(got, "after stays") {
		t.Errorf("section not replaced in place:\n%s", got)
	}
}

func TestBootstrapLane_DryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	res, err := BootstrapLane(root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if res != (LaneResult{Config: "created", Prompt: "created", Agents: "created"}) {
		t.Errorf("dry-run result = %+v, want what would be created", res)
	}
	if got := snapshot(t, root); len(got) != 0 {
		t.Errorf("dry-run wrote %d files", len(got))
	}
}
