package pwsh

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// ciJob returns the run scripts of one job's steps in
// .github/workflows/ci.yml, parsed rather than cut by line range, so a job that
// is renamed or moved cannot silently widen or empty what the test reads.
func ciJob(t *testing.T, job string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatal(err)
	}
	j, ok := wf.Jobs[job]
	if !ok {
		t.Fatalf("ci.yml has no %s job", job)
	}
	var b strings.Builder
	for _, s := range j.Steps {
		b.WriteString(s.Run)
		b.WriteByte('\n')
	}
	return b.String()
}

// TestLintMatchesCIsLintPowershellJob holds `dotf lint powershell` to CI's
// lint-powershell job (PLAT-003 AC6): the local run is a shift-left of that
// job only while both analyze the same files, under the same settings file,
// at the same severities, with the same pinned analyzer. A change to either
// side that the other does not follow fails here.
func TestLintMatchesCIsLintPowershellJob(t *testing.T) {
	job := ciJob(t, "lint-powershell")
	// A parameter value must end where the job's does: "-Severity Error" is a
	// prefix of "-Severity Error,Warning" and must not pass for it.
	for _, want := range []string{
		"-Settings " + SettingsFile,
		"-Severity " + Severity,
		"-Include " + strings.Join(globs(), ", "),
		"'^" + PinKey + "='",
		"Import-Module PSScriptAnalyzer -RequiredVersion $env:" + PinKey,
	} {
		if !regexp.MustCompile(regexp.QuoteMeta(want) + `(\s|\)|$)`).MatchString(job) {
			t.Errorf("ci.yml's lint-powershell job no longer contains %q; change `dotf lint powershell` (package pwsh) with it", want)
		}
	}
}

func globs() []string {
	g := make([]string, len(Extensions))
	for i, e := range Extensions {
		g[i] = "*" + e
	}
	return g
}
