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

// tagComment captures the tag a trailing comment names for a pinned SHA.
var tagComment = regexp.MustCompile(`^\s*#\s*(v\d\S*)`)

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
			tag := tagComment.FindStringSubmatch(m[2])
			if tag == nil {
				t.Errorf("%s:%d: %s has no trailing tag comment (# vN)", p, i+1, m[1])
			}
			// An action the repository does not run has no pin to agree with, and
			// a SHA checked only for shape can be one nobody published.
			want, ok := repoPins[pin[1]]
			switch {
			case !ok:
				t.Errorf("%s:%d: %s is not used by this repository's workflows, so nothing checks its SHA", p, i+1, pin[1])
			case want.sha != pin[2]:
				t.Errorf("%s:%d: %s pins %s, this repository's workflows pin %s", p, i+1, pin[1], pin[2], want.sha)
			case tag != nil && want.tag != tag[1]:
				t.Errorf("%s:%d: %s is commented %s, this repository's workflows comment the same SHA %s", p, i+1, pin[1], tag[1], want.tag)
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

// repoPin is one action's pinned commit and the tag its comment names.
type repoPin struct{ sha, tag string }

// repositoryPins maps each action this repository pins to its commit SHA and tag, read
// from the workflows and composite actions it runs. Two different SHAs for one
// action fail here: the template would have no single commit to agree with.
func repositoryPins(t *testing.T, repo string) map[string]repoPin {
	t.Helper()
	pins := map[string]repoPin{}
	for _, glob := range []string{
		filepath.Join(repo, ".github", "workflows", "*.yml"),
		filepath.Join(repo, ".github", "workflows", "*.yaml"),
		filepath.Join(repo, ".github", "actions", "*", "action.yml"),
		filepath.Join(repo, ".github", "actions", "*", "action.yaml"),
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
				var tag string
				if c := tagComment.FindStringSubmatch(m[2]); c != nil {
					tag = c[1]
				}
				if prev, ok := pins[pin[1]]; ok && prev.sha != pin[2] {
					t.Fatalf("%s: %s is pinned to both %s and %s across this repository's workflows", f, pin[1], prev.sha, pin[2])
				} else if ok && prev.tag != tag {
					t.Fatalf("%s: %s@%s is commented both %s and %s across this repository's workflows", f, pin[1], pin[2], prev.tag, tag)
				}
				pins[pin[1]] = repoPin{pin[2], tag}
			}
		}
	}
	if len(pins) == 0 {
		t.Fatal("no pinned action found in this repository's workflows: the path is wrong")
	}
	return pins
}
