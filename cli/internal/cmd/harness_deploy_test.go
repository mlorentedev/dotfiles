package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deployRepo is a checkout with the committed maps and enforced records, one
// agent record (whose tier the caller picks) and a doctrine target.
func deployRepo(t *testing.T, tier string) (repo, home string) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"harness/model-map.json", "harness/model-map.schema.json",
		"harness/capability-map.json", "harness/capability-map.schema.json", "harness/enforced/no-auto-merge.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", f)) //nolint:gosec // committed inputs
		if err != nil {
			t.Fatal(err)
		}
		write(f, string(raw))
	}
	write("harness/manifest.json", `{"vault_subpath":"00_meta/patterns",
		"agents":{"vault_subpath":"v","record_dir":"harness/agents",
			"deploy":[{"agent":"claude","render":"agent-md","dir":".claude/agents"}]},
		"doctrine":{"inject":["no-auto-merge"],"deploy":[{"agent":"agy","file":".gemini/GEMINI.md","char_cap":12000}]}}`)
	write("harness/agents/demo/AGENT.md", "---\nname: demo\ndescription: d\nkind: invocable\nmodel: "+tier+"\n---\nbody\n")
	return repo, home
}

func TestHarnessDeploy_RunsEverySurfaceThenConverges(t *testing.T) {
	repo, home := deployRepo(t, "mid")
	stdout, stderr, err := execute(t, "harness", "deploy", "--repo-root", repo, "--home", home)
	if err != nil || !strings.HasSuffix(stdout, "[deploy] OK\n") {
		t.Fatalf("err %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	for _, f := range []string{".claude/agents/demo.md", ".gemini/GEMINI.md"} {
		if _, err := os.Stat(filepath.Join(home, f)); err != nil {
			t.Errorf("%s was not deployed", f)
		}
	}
	stdout, _, err = execute(t, "harness", "deploy", "--repo-root", repo, "--home", home)
	if err != nil || strings.Contains(stdout, " -> ") {
		t.Errorf("a second run wrote something (%v):\n%s", err, stdout)
	}
}

// A failed agent record stops presence and doctrine, which carry the roster
// those records feed, as compile-harness.sh stopped them.
func TestHarnessDeploy_AFailedAgentStopsTheRosterSurfaces(t *testing.T) {
	repo, home := deployRepo(t, "nonesuch")
	_, stderr, err := execute(t, "harness", "deploy", "--repo-root", repo, "--home", home)
	if err == nil || !strings.Contains(err.Error(), "nonesuch") {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(stderr, "presence and doctrine NOT written") {
		t.Errorf("stderr does not say what was skipped:\n%s", stderr)
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini", "GEMINI.md")); err == nil {
		t.Error("doctrine was written after an agent record failed")
	}
}
