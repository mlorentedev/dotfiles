package initrepo

import (
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// usesRef matches a YAML `uses:` key and captures its reference and what
// follows it on the line. Comments and scalar values that merely contain
// "uses:" do not match, the same scope scripts/check-actions-pinned.sh judges.
var usesRef = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s+["']?([^"'\s#]+)["']?(.*)$`)

// pinnedRef is a full commit SHA reference; its tag rides in a trailing comment.
var pinnedRef = regexp.MustCompile(`^(.+)@([0-9a-f]{40})$`)

// The workflows a repository scaffolded by `dotf init` receives are not
// exempt from the rule this repository applies to its own: every third-party
// action is a full commit SHA with its tag as a trailing comment (#3094). A tag
// can be moved, and these workflows run in repositories this one never sees.
// Where this repository already pins the same action, the template must pin the
// same commit, so one bump moves both.
func TestWorkflowTemplatesPinActionsToTheRepositorysSHAs(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	repoPins := repositoryPins(t, repo)

	checked := 0
	err := fs.WalkDir(templatesFS, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || (path.Ext(p) != ".yml" && path.Ext(p) != ".yaml") {
			return err
		}
		raw, err := templatesFS.ReadFile(p)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(raw), "\n") {
			m := usesRef.FindStringSubmatch(line)
			if m == nil || strings.HasPrefix(m[1], "./") || strings.HasPrefix(m[1], "docker://") {
				continue
			}
			checked++
			pin := pinnedRef.FindStringSubmatch(m[1])
			if pin == nil {
				t.Errorf("%s:%d: %s is not pinned to a commit SHA; pin it with the tag as a trailing comment", p, i+1, m[1])
				continue
			}
			if !regexp.MustCompile(`^\s*# v\d`).MatchString(m[2]) {
				t.Errorf("%s:%d: %s has no trailing tag comment (# vN)", p, i+1, m[1])
			}
			if want, ok := repoPins[pin[1]]; ok && want != pin[2] {
				t.Errorf("%s:%d: %s pins %s, this repository's workflows pin %s", p, i+1, pin[1], pin[2], want)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no uses: reference found in any workflow template: the walk is looking in the wrong place")
	}
}

// repositoryPins maps each action this repository pins to its commit SHA, read
// from the workflows and composite actions it runs. Two different SHAs for one
// action fail here: the template would have no single commit to agree with.
func repositoryPins(t *testing.T, repo string) map[string]string {
	t.Helper()
	pins := map[string]string{}
	for _, glob := range []string{
		filepath.Join(repo, ".github", "workflows", "*.yml"),
		filepath.Join(repo, ".github", "actions", "*", "action.yml"),
	} {
		files, err := filepath.Glob(glob)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			for _, line := range strings.Split(readFile(t, f), "\n") {
				m := usesRef.FindStringSubmatch(line)
				if m == nil {
					continue
				}
				pin := pinnedRef.FindStringSubmatch(m[1])
				if pin == nil {
					continue
				}
				if prev, ok := pins[pin[1]]; ok && prev != pin[2] {
					t.Fatalf("%s: %s is pinned to both %s and %s across this repository's workflows", f, pin[1], prev, pin[2])
				}
				pins[pin[1]] = pin[2]
			}
		}
	}
	if len(pins) == 0 {
		t.Fatal("no pinned action found in this repository's workflows: the path is wrong")
	}
	return pins
}
