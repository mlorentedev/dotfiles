package initrepo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const twoEcosystems = `version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/cli"
    labels: ["dependencies"]
  - package-ecosystem: "github-actions"
    directory: "/"
    labels: ["dependencies", "ci"]
`

// stubLabelsGh installs a fake gh that serves dependabot (the contents of
// .github/dependabot.yml, or a 404 when empty), lists existing (one label per
// line), and fails `label create` for failName. It returns the calls log.
func stubLabelsGh(t *testing.T, dependabot, existing, failName string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("gh stub uses a POSIX shell script")
	}
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls.log")
	dep := filepath.Join(dir, "dependabot.yml")
	labels := filepath.Join(dir, "labels.txt")
	if dependabot != "" {
		if err := os.WriteFile(dep, []byte(dependabot), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(labels, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	failCase := ""
	if failName != "" {
		failCase = "  'label create '" + shq(failName) + "*) echo 'HTTP 403' >&2; exit 1 ;;\n"
	}
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + shq(calls) + "\n" +
		"case \"$*\" in\n" +
		"  *contents/.github/dependabot.yml*)\n" +
		"    [ -f " + shq(dep) + " ] || { echo 'gh: Not Found (HTTP 404)' >&2; exit 1; }\n" +
		"    base64 < " + shq(dep) + " ;;\n" +
		"  *contents/.github/dependabot.yaml*) echo 'gh: Not Found (HTTP 404)' >&2; exit 1 ;;\n" +
		"  'label list'*) cat " + shq(labels) + " ;;\n" +
		failCase +
		"  'label create'*) : ;;\n" +
		"  *) printf 'true' ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin:/bin")
	return calls
}

func creates(t *testing.T, calls string) []string {
	t.Helper()
	var out []string
	for _, l := range strings.Split(readFileOrEmpty(calls), "\n") {
		if strings.HasPrefix(l, "label create ") {
			out = append(out, strings.Fields(l)[2])
		}
	}
	return out
}

func readFileOrEmpty(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}

func TestApplyDependabotLabelsCreatesOnlyTheMissingOnes(t *testing.T) {
	calls := stubLabelsGh(t, twoEcosystems, "bug\nDependencies\n", "")
	res, err := ApplyDependabotLabels("owner/repo", false)
	if err != nil {
		t.Fatalf("ApplyDependabotLabels: %v", err)
	}
	// "Dependencies" already exists: GitHub label names compare case-insensitively.
	if got := creates(t, calls); len(got) != 1 || got[0] != "ci" {
		t.Errorf("created %v, want [ci]; calls:\n%s", got, readFileOrEmpty(calls))
	}
	if res.Action != "created" || strings.Join(res.Labels, ",") != "ci" {
		t.Errorf("result = %+v, want created [ci]", res)
	}
	if !strings.Contains(readFileOrEmpty(calls), "--color 0366d6") {
		t.Errorf("a created label should carry the default color; calls:\n%s", readFileOrEmpty(calls))
	}
}

func TestApplyDependabotLabelsSecondRunCreatesNothing(t *testing.T) {
	calls := stubLabelsGh(t, twoEcosystems, "dependencies\nci\n", "")
	res, err := ApplyDependabotLabels("owner/repo", false)
	if err != nil {
		t.Fatalf("ApplyDependabotLabels: %v", err)
	}
	if got := creates(t, calls); len(got) != 0 {
		t.Errorf("created %v on a converged repo", got)
	}
	if res.Action != "present" {
		t.Errorf("Action = %q, want present", res.Action)
	}
}

func TestApplyDependabotLabelsDryRunCreatesNone(t *testing.T) {
	calls := stubLabelsGh(t, twoEcosystems, "", "")
	res, err := ApplyDependabotLabels("owner/repo", true)
	if err != nil {
		t.Fatalf("ApplyDependabotLabels: %v", err)
	}
	if got := creates(t, calls); len(got) != 0 {
		t.Errorf("dry-run created %v", got)
	}
	if res.Action != "dry-run" || strings.Join(res.Labels, ",") != "ci,dependencies" {
		t.Errorf("result = %+v, want dry-run [ci dependencies]", res)
	}
}

func TestApplyDependabotLabelsWithoutDeclarationsDoesNothing(t *testing.T) {
	for name, dep := range map[string]string{
		"no dependabot.yml": "",
		"no labels":         "version: 2\nupdates:\n  - package-ecosystem: gomod\n    directory: /\n",
	} {
		t.Run(name, func(t *testing.T) {
			calls := stubLabelsGh(t, dep, "", "")
			res, err := ApplyDependabotLabels("owner/repo", false)
			if err != nil {
				t.Fatalf("ApplyDependabotLabels: %v", err)
			}
			if res.Action != "none" {
				t.Errorf("Action = %q, want none", res.Action)
			}
			if log := readFileOrEmpty(calls); strings.Contains(log, "label ") {
				t.Errorf("no declared label should mean no label call; calls:\n%s", log)
			}
		})
	}
}

func TestApplyDependabotLabelsFailsLoudlyWhenACreateFails(t *testing.T) {
	stubLabelsGh(t, twoEcosystems, "", "ci")
	_, err := ApplyDependabotLabels("owner/repo", false)
	if err == nil || !strings.Contains(err.Error(), `"ci"`) || !strings.Contains(err.Error(), "owner/repo") {
		t.Errorf("err = %v, want a failure naming the label and the repo", err)
	}
}

func TestApplyDependabotLabelsSkipsWithoutGh(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	res, err := ApplyDependabotLabels("owner/repo", false)
	if err != nil || res.Action != "skipped" {
		t.Errorf("got %+v, %v; want a graceful skip", res, err)
	}
}
