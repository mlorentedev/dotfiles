package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runLintFleetCmd(t *testing.T, args ...string) (string, int) {
	t.Helper()
	root := New("test", "")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"lint", "fleet"}, args...))
	code := ExitCode(root.Execute())
	return out.String(), code
}

func writeWorkflow(t *testing.T, dir, body string) {
	t.Helper()
	wd := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wd, "ci.yml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLintFleetExitCodes(t *testing.T) {
	clean := t.TempDir()
	writeWorkflow(t, clean, "on: push\njobs:\n  a:\n    runs-on: x\n    timeout-minutes: 5\n    steps:\n      - run: true\n")
	if out, code := runLintFleetCmd(t, "--dir", clean); code != 0 || !strings.Contains(out, "[OK]") {
		t.Errorf("clean: exit %d, %q", code, out)
	}

	dirty := t.TempDir()
	writeWorkflow(t, dirty, "on: push\njobs:\n  a:\n    runs-on: x\n    steps:\n      - run: true\n")
	summary := filepath.Join(t.TempDir(), "summary.md")
	out, code := runLintFleetCmd(t, "--dir", dirty, "--summary", summary)
	if code != 1 || !strings.Contains(out, ".github/workflows/ci.yml:3: fleet/timeout: job a sets no timeout-minutes") {
		t.Errorf("dirty: exit %d, %q", code, out)
	}
	table, err := os.ReadFile(summary)
	if err != nil || !strings.Contains(string(table), "| `fleet/timeout` | `.github/workflows/ci.yml:3` |") {
		t.Errorf("summary: %v, %q", err, table)
	}

	broken := t.TempDir()
	writeWorkflow(t, broken, "jobs: [unclosed")
	if out, code := runLintFleetCmd(t, "--dir", broken); code != 2 || !strings.Contains(out, "[ERROR]") {
		t.Errorf("broken: exit %d, %q", code, out)
	}
	if _, code := runLintFleetCmd(t, "--dir", clean, "extra"); code != 2 {
		t.Errorf("stray argument: exit %d, want 2", code)
	}
}

func TestLintFleetDefaultBranch(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "on:\n  push:\n    branches: [master]\njobs: {}\n")
	out, code := runLintFleetCmd(t, "--dir", dir)
	if code != 2 || !strings.Contains(out, "--default-branch") {
		t.Errorf("unknown default branch: exit %d, %q", code, out)
	}
	if out, code := runLintFleetCmd(t, "--dir", dir, "--default-branch", "master"); code != 0 {
		t.Errorf("filter names it: exit %d, %q", code, out)
	}
	if out, code := runLintFleetCmd(t, "--dir", dir, "--default-branch", "main"); code != 1 || !strings.Contains(out, "fleet/branch-filter") {
		t.Errorf("filter misses it: exit %d, %q", code, out)
	}
}

// The default branch comes from the injected lookup when no flag names it,
// and an unknown one is exit 2, not a pass: no git process runs here.
func TestLintFleetAsksForTheDefaultBranch(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "on:\n  push:\n    branches: [master]\njobs: {}\n")
	for branch, want := range map[string]int{"master": 0, "main": 1, "": 2} {
		var out bytes.Buffer
		var asked string
		deps := lintFleetDeps{branchOf: func(d string) string { asked = d; return branch }}
		if code := ExitCode(runLintFleet(&out, &out, deps, dir, "", "")); code != want || asked != dir {
			t.Errorf("origin/HEAD %q: exit %d (asked %q), want %d: %s", branch, code, asked, want, out.String())
		}
	}
	deps := lintFleetDeps{branchOf: func(string) string { t.Error("a flag was given; the lookup ran"); return "" }}
	if code := ExitCode(runLintFleet(&bytes.Buffer{}, &bytes.Buffer{}, deps, dir, "master", "")); code != 0 {
		t.Errorf("--default-branch master: exit %d", code)
	}
}
