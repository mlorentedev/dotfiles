package fleetlint

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// repo lays files out under a temporary repository root.
func repo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func fixture(t *testing.T, rule, kind string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", rule, kind+".yml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Each bad fixture trips its own rule once and no other; each good fixture,
// which holds the near misses the exemptions exist for, trips nothing.
func TestRuleFixtures(t *testing.T) {
	for _, rule := range []string{RuleTimeout, RuleSHAPin, RulePersistCredentials, RuleNoSecretsInherit,
		RuleCancelInProgress, RuleOSOnPullRequest, RuleSkipCache, RuleBranchFilter} {
		name := strings.TrimPrefix(rule, "fleet/")
		t.Run(name, func(t *testing.T) {
			bad, err := lintFiles(t, map[string]string{".github/workflows/ci.yml": fixture(t, name, "bad")})
			if err != nil {
				t.Fatal(err)
			}
			if len(bad) != 1 || bad[0].Rule != rule || bad[0].Path != ".github/workflows/ci.yml" || bad[0].Line == 0 {
				t.Errorf("bad: got %v, want one %s finding with a line", bad, rule)
			}
			good, err := lintFiles(t, map[string]string{".gitea/workflows/ci.yaml": fixture(t, name, "good")})
			if err != nil {
				t.Fatal(err)
			}
			if len(good) != 0 {
				t.Errorf("good: got %v, want none", good)
			}
		})
	}
}

const twoFindings = `on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: owner/setup@v4
`

func TestExceptionSilencesOnlyItsRuleAndPath(t *testing.T) {
	got, err := lintFiles(t, map[string]string{
		".github/workflows/ci.yml":    twoFindings,
		".github/workflows/other.yml": twoFindings,
		ExceptionsPath: `exceptions:
  - rule: fleet/timeout
    path: .github/workflows/ci.yml
    reason: the job is a one-shot
    issue: "#1"
`,
	})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, f := range got {
		lines = append(lines, f.Rule+" "+f.Path)
	}
	want := "fleet/sha-pin .github/workflows/ci.yml|fleet/timeout .github/workflows/other.yml|fleet/sha-pin .github/workflows/other.yml"
	if strings.Join(lines, "|") != want {
		t.Errorf("got %q\nwant %q", strings.Join(lines, "|"), want)
	}
}

func TestIncompleteExceptionIsAFindingAndSilencesNothing(t *testing.T) {
	got, err := lintFiles(t, map[string]string{
		".github/workflows/ci.yml": twoFindings,
		ExceptionsPath: `exceptions:
  - rule: fleet/timeout
    path: .github/workflows/ci.yml
    issue: "#1"
  - rule: fleet/sha-pin
    path: .github/workflows/ci.yml
    reason: "  "
`,
	})
	if err != nil {
		t.Fatal(err)
	}
	var s []string
	for _, f := range got {
		s = append(s, f.String())
	}
	want := []string{
		".github/lint-exceptions.yml:2: fleet/exception: exception without reason is not honoured",
		".github/lint-exceptions.yml:5: fleet/exception: exception without reason, issue is not honoured",
		".github/workflows/ci.yml:3: fleet/timeout: job build sets no timeout-minutes",
		".github/workflows/ci.yml:6: fleet/sha-pin: owner/setup@v4 is not pinned by a full commit SHA",
	}
	if strings.Join(s, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(s, "\n"), strings.Join(want, "\n"))
	}
}

func TestUnreadableInputIsAnError(t *testing.T) {
	if _, err := lintFiles(t, map[string]string{".github/workflows/ci.yml": "jobs: [unclosed"}); err == nil {
		t.Error("a workflow that does not parse: want an error")
	}
	if _, err := lintFiles(t, map[string]string{ExceptionsPath: "exceptions: {"}); err == nil {
		t.Error("an exceptions file that does not parse: want an error")
	}
	got, err := Lint(t.TempDir(), "main")
	if err != nil || len(got) != 0 {
		t.Errorf("no workflows: got %v, %v, want none and no error", got, err)
	}
}

func TestMultiDocumentFileIsReadWhole(t *testing.T) {
	got, err := lintFiles(t, map[string]string{".github/workflows/ci.yml": "on: push\njobs: {}\n---\n" + twoFindings})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("got %v, want the second document's two findings", got)
	}
}

// An explicit true is the same leak as leaving the key out.
func TestPersistCredentialsTrueIsAFinding(t *testing.T) {
	wf := strings.Replace(fixture(t, "persist-credentials", "bad"), "fetch-depth: 0", "persist-credentials: true", 1)
	got, err := lintFiles(t, map[string]string{".github/workflows/ci.yml": wf})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Rule != RulePersistCredentials {
		t.Errorf("got %v, want one %s finding", got, RulePersistCredentials)
	}
}

// lintFiles lints a repository holding files, whose default branch is main.
func lintFiles(t *testing.T, files map[string]string) ([]Finding, error) {
	t.Helper()
	return Lint(repo(t, files), "main")
}

// A branch filter with no default branch to check it against is an error,
// never a pass; with no filter the default branch is not needed.
func TestBranchFilterNeedsADefaultBranch(t *testing.T) {
	_, err := Lint(repo(t, map[string]string{".github/workflows/ci.yml": fixture(t, "branch-filter", "bad")}), "")
	if !errors.Is(err, ErrNoDefaultBranch) {
		t.Errorf("got %v, want ErrNoDefaultBranch", err)
	}
	if _, err := Lint(repo(t, map[string]string{".github/workflows/ci.yml": twoFindings}), ""); err != nil {
		t.Errorf("no filter: %v", err)
	}
}

func TestDefaultBranchReadsOriginHEAD(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", "https://example.invalid/r.git"},
		{"symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if got := DefaultBranch(dir); got != "trunk" {
		t.Errorf("got %q, want trunk", got)
	}
	if got := DefaultBranch(t.TempDir()); got != "" {
		t.Errorf("not a checkout: got %q, want empty", got)
	}
}

// A literal Windows label is a finding on a pull request and not on a push.
func TestOSOnPullRequestReadsLiteralLabels(t *testing.T) {
	job := "jobs:\n  w:\n    runs-on: [self-hosted, Windows]\n    timeout-minutes: 5\n    steps:\n      - run: x\n"
	pr := "on: [pull_request]\nconcurrency:\n  group: g\n  cancel-in-progress: true\n" + job
	got, err := lintFiles(t, map[string]string{".github/workflows/ci.yml": pr})
	if err != nil || len(got) != 1 || got[0].Rule != RuleOSOnPullRequest {
		t.Errorf("pull request: got %v, %v", got, err)
	}
	got, err = lintFiles(t, map[string]string{".github/workflows/ci.yml": "on: push\n" + job})
	if err != nil || len(got) != 0 {
		t.Errorf("push: got %v, %v", got, err)
	}
}

func TestMatchesBranch(t *testing.T) {
	cases := []struct {
		patterns []string
		branch   string
		want     bool
	}{
		{[]string{"main"}, "main", true},
		{[]string{"master"}, "main", false},
		{[]string{"*", "!main"}, "main", false},
		{[]string{"!main", "*"}, "main", true},
		{[]string{"**"}, "release/v1", true},
		{[]string{"*"}, "release/v1", false},
		{[]string{"rele?ase/*"}, "relase/v1", true},
		{[]string{"rel?ase/*"}, "release/v1", false},
		{[]string{"main?"}, "main", true},
		{[]string{"v1+"}, "v111", true},
		{[]string{"**", "!ma[a-z]n"}, "main", false},
		{[]string{"ma[0-9]n"}, "main", false},
		{[]string{`ma\*n`}, "ma*n", true},
		{[]string{`ma\*n`}, "main", false},
		{[]string{"ma.n"}, "main", false},
		{[]string{"+main"}, "+main", true},
	}
	for _, c := range cases {
		if got := matchesBranch(c.patterns, c.branch); got != c.want {
			t.Errorf("matchesBranch(%q, %q) = %v, want %v", c.patterns, c.branch, got, c.want)
		}
	}
}

// An explicit cancel-in-progress: false is the same as none.
func TestCancelInProgressFalseIsAFinding(t *testing.T) {
	wf := "on: pull_request\nconcurrency:\n  group: g\n  cancel-in-progress: false\njobs: {}\n"
	got, err := lintFiles(t, map[string]string{".github/workflows/ci.yml": wf})
	if err != nil || len(got) != 1 || got[0].Rule != RuleCancelInProgress || got[0].Line != 2 {
		t.Errorf("got %v, %v", got, err)
	}
}

// A pull-request trigger reruns on a push unless its types: leave out
// synchronize; the default set holds it.
func TestRerunsOnPush(t *testing.T) {
	for on, want := range map[string]bool{
		"pull_request":                                  true,
		"[push, pull_request]":                          true,
		"{pull_request: }":                              true,
		"{pull_request: {branches: [main]}}":            true,
		"{pull_request_target: {types: [synchronize]}}": true,
		"{pull_request: {types: [opened, reopened]}}":   false,
		"{push: {branches: [main]}}":                    false,
	} {
		var n yaml.Node
		if err := yaml.Unmarshal([]byte(on), &n); err != nil {
			t.Fatal(err)
		}
		if got := rerunsOnPush(n.Content[0]); got != want {
			t.Errorf("on: %s: got %v, want %v", on, got, want)
		}
	}
}

// A boolean is read in every spelling YAML and an action's inputs accept:
// False is false, and a quoted "false" is too.
func TestBooleansInAnySpelling(t *testing.T) {
	for v, want := range map[string]bool{"false": true, "False": true, "FALSE": true, `"false"`: true, "true": false, "no": false, "${{ x }}": false} {
		var n yaml.Node
		if err := yaml.Unmarshal([]byte("k: "+v), &n); err != nil {
			t.Fatal(err)
		}
		if got := isBool(mapValue(n.Content[0], "k"), false); got != want {
			t.Errorf("%s: got %v, want %v", v, got, want)
		}
	}
	wf := "on: pull_request\nconcurrency:\n  group: g\n  cancel-in-progress: False\njobs: {}\n"
	if got, err := lintFiles(t, map[string]string{".github/workflows/ci.yml": wf}); err != nil || len(got) != 1 || got[0].Rule != RuleCancelInProgress {
		t.Errorf("cancel-in-progress: False: got %v, %v", got, err)
	}
}

// A literal matrix is read as GitHub expands it: an object-valued axis
// through its property path, and an exclude: on that axis alone removes it.
func TestOSOnPullRequestReadsTheMatrix(t *testing.T) {
	job := func(matrix, runsOn string) string {
		return "on: pull_request\nconcurrency:\n  group: g\n  cancel-in-progress: true\njobs:\n  t:\n    timeout-minutes: 5\n" +
			"    strategy:\n      matrix:\n" + matrix + "    runs-on: " + runsOn + "\n    steps:\n      - run: true\n"
	}
	for name, c := range map[string]struct {
		wf   string
		want int
	}{
		"nested":            {job("        runner: [{os: ubuntu-latest}, {os: windows-latest}]\n", "${{ matrix.runner.os }}"), 1},
		"nested linux only": {job("        runner: [{os: ubuntu-latest}]\n", "${{ matrix.runner.os }}"), 0},
		"excluded":          {job("        os: [ubuntu-latest, windows-latest]\n        exclude: [{os: windows-latest}]\n", "${{ matrix.os }}"), 0},
		"partly excluded":   {job("        os: [ubuntu-latest, windows-latest]\n        go: [a, b]\n        exclude: [{os: windows-latest, go: a}]\n", "${{ matrix.os }}"), 1},
		"included":          {job("        os: [ubuntu-latest]\n        include: [{os: macos-latest}]\n", "${{ matrix.os }}"), 1},
	} {
		got, err := lintFiles(t, map[string]string{".github/workflows/ci.yml": c.wf})
		if err != nil || len(got) != c.want {
			t.Errorf("%s: got %v, %v, want %d finding(s)", name, got, err, c.want)
		}
	}
}
