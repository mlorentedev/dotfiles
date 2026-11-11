package fleetlint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	for _, rule := range []string{RuleTimeout, RuleSHAPin, RulePersistCredentials, RuleNoSecretsInherit} {
		name := strings.TrimPrefix(rule, "fleet/")
		t.Run(name, func(t *testing.T) {
			bad, err := Lint(repo(t, map[string]string{".github/workflows/ci.yml": fixture(t, name, "bad")}))
			if err != nil {
				t.Fatal(err)
			}
			if len(bad) != 1 || bad[0].Rule != rule || bad[0].Path != ".github/workflows/ci.yml" || bad[0].Line == 0 {
				t.Errorf("bad: got %v, want one %s finding with a line", bad, rule)
			}
			good, err := Lint(repo(t, map[string]string{".gitea/workflows/ci.yaml": fixture(t, name, "good")}))
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
	got, err := Lint(repo(t, map[string]string{
		".github/workflows/ci.yml":    twoFindings,
		".github/workflows/other.yml": twoFindings,
		ExceptionsPath: `exceptions:
  - rule: fleet/timeout
    path: .github/workflows/ci.yml
    reason: the job is a one-shot
    issue: "#1"
`,
	}))
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
	got, err := Lint(repo(t, map[string]string{
		".github/workflows/ci.yml": twoFindings,
		ExceptionsPath: `exceptions:
  - rule: fleet/timeout
    path: .github/workflows/ci.yml
    issue: "#1"
  - rule: fleet/sha-pin
    path: .github/workflows/ci.yml
    reason: "  "
`,
	}))
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
	if _, err := Lint(repo(t, map[string]string{".github/workflows/ci.yml": "jobs: [unclosed"})); err == nil {
		t.Error("a workflow that does not parse: want an error")
	}
	if _, err := Lint(repo(t, map[string]string{ExceptionsPath: "exceptions: {"})); err == nil {
		t.Error("an exceptions file that does not parse: want an error")
	}
	got, err := Lint(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Errorf("no workflows: got %v, %v, want none and no error", got, err)
	}
}

func TestMultiDocumentFileIsReadWhole(t *testing.T) {
	got, err := Lint(repo(t, map[string]string{".github/workflows/ci.yml": "on: push\njobs: {}\n---\n" + twoFindings}))
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
	got, err := Lint(repo(t, map[string]string{".github/workflows/ci.yml": wf}))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Rule != RulePersistCredentials {
		t.Errorf("got %v, want one %s finding", got, RulePersistCredentials)
	}
}
