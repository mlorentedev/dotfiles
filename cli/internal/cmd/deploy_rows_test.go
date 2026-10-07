package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// row is how a `dotf deploy` report line starts for verb and name.
func row(verb, name string) string {
	return fmt.Sprintf("%-*s %-*s ", deployVerbWidth, verb, deployNameWidth, name)
}

// reportColumn is where the third column of every report line starts. It is a
// literal on purpose: widening a column is a decision this test makes visible.
const reportColumn = 32

// #1664 (6): `would fix mode` and `mode fixed` padded the name to a different
// width than `in sync` and `deployed`, so the columns moved line by line. Every
// line of a bare deploy, from every step, starts its third column at one place.
func TestDeployCmd_EveryReportLineSharesItsColumns(t *testing.T) {
	other := "windows"
	if runtime.GOOS == "windows" {
		other = "linux"
	}
	repo, home := t.TempDir(), t.TempDir()
	writeMirrorFixture(t, filepath.Join(repo, "ai", "deploy.json"), `{
  "version": 4,
  "configs": [
    {"name": "fresh", "src": "ai/a.json", "dst": "{HOME}/.fresh/a.json"},
    {"name": "copilot-settings", "src": "ai/a.json", "dst": "{HOME}/.long/a.json"},
    {"name": "adopted", "src": "ai/a.json", "dst": "{HOME}/.adopted/a.json"},
    {"name": "sec", "src": "ai/a.json", "dst": "{HOME}/.sec/a.json", "mode": "0600"},
    {"name": "gated", "src": "ai/a.json", "dst": "{HOME}/.gated/a.json", "requires": "gatedtool"},
    {"name": "elsewhere", "src": "ai/a.json", "dst": "{HOME}/.else/a.json", "platforms": ["`+other+`"]}
  ]
}`)
	writeMirrorFixture(t, filepath.Join(repo, "ai", "a.json"), `{"a":true}`)
	writeMirrorFixture(t, filepath.Join(home, ".long", "a.json"), `{"a":true}`)    // in sync
	writeMirrorFixture(t, filepath.Join(home, ".adopted", "a.json"), `{"mine":1}`) // backed up
	writeMirrorFixture(t, filepath.Join(home, ".sec", "a.json"), `{"a":true}`)     // mode drift
	if err := os.Chmod(filepath.Join(home, ".sec", "a.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := deployCommandAvailable
	t.Cleanup(func() { deployCommandAvailable = orig })
	deployCommandAvailable = func(string) bool { return false }

	dry, err := runDeploy(t, repo, home, []string{"--dry-run"})
	if err != nil {
		t.Fatalf("%v\n%s", err, dry)
	}
	out, err := runDeploy(t, repo, home, nil)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	report := dry + out
	for _, want := range []string{"would deploy", "would fix mode", "in sync", "deployed", "mode fixed", "kept the previous file", "skipped", "orca-hooks", "claude-mcp"} {
		if !strings.Contains(report, want) {
			t.Fatalf("the fixture must produce a %q line:\n%s", want, report)
		}
	}
	for _, line := range strings.Split(strings.TrimRight(report, "\n"), "\n") {
		if len(line) <= reportColumn || line[reportColumn-1] != ' ' || line[reportColumn] == ' ' {
			t.Errorf("third column not at %d:\n%q\nin:\n%s", reportColumn, line, report)
		}
	}
}

// The name column is as wide as the longest name a deploy reports. A longer
// manifest entry would push its own lines out of the column, so adding one
// fails here, where widening deployNameWidth is one edit.
func TestDeployRow_EveryShippedNameFits(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "ai", "deploy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Configs []struct {
			Name string `json:"name"`
		} `json:"configs"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	names := []string{"orca-hooks", "claude-mcp", "claude-plugins"}
	for _, c := range m.Configs {
		names = append(names, c.Name)
	}
	for _, n := range names {
		if len(n) > deployNameWidth {
			t.Errorf("%q is %d characters; the report's name column holds %d", n, len(n), deployNameWidth)
		}
	}
}
