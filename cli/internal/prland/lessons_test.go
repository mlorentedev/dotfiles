package prland

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// #2752: two PRs that each took the next free lesson number pass CI against
// the base they were checked on, and collide only once both have merged.
func TestDecide_RefusesALessonNumberTheBaseAlreadyHolds(t *testing.T) {
	cases := map[string]struct {
		files, freed []string
		base         map[int]string
		want         string // "" means no lesson reason
	}{
		"a new lesson on a taken number": {
			files: []string{"docs/lessons/lesson-497-b.md"},
			base:  map[int]string{497: "docs/lessons/lesson-497-a.md"},
			want:  "lesson 497 is already docs/lessons/lesson-497-a.md on main",
		},
		"a category directory shares the numbering": {
			files: []string{"docs/lessons/ci/lesson-12-x.md"},
			base:  map[int]string{12: "docs/lessons/lesson-12-y.md"},
			want:  "lesson 12 is already docs/lessons/lesson-12-y.md",
		},
		"a free number": {
			files: []string{"docs/lessons/lesson-498-b.md"},
			base:  map[int]string{497: "docs/lessons/lesson-497-a.md"},
		},
		"an edit of the lesson that holds it": {
			files: []string{"docs/lessons/lesson-497-a.md"},
			base:  map[int]string{497: "docs/lessons/lesson-497-a.md"},
		},
		"a slug rename frees the old name": {
			files: []string{"docs/lessons/lesson-497-b.md", "docs/lessons/lesson-497-a.md"},
			freed: []string{"docs/lessons/lesson-497-a.md"},
			base:  map[int]string{497: "docs/lessons/lesson-497-a.md"},
		},
		"a file deeper than a category is not a lesson": {
			files: []string{"docs/lessons/ci/old/lesson-497-b.md"},
			base:  map[int]string{497: "docs/lessons/lesson-497-a.md"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := readyFacts()
			f.Files, f.Freed, f.BaseLessons = tc.files, tc.freed, tc.base
			got := strings.Join(Decide(f), "; ")
			if tc.want == "" && got != "" || !strings.Contains(got, tc.want) {
				t.Errorf("reasons %q, want %q", got, tc.want)
			}
		})
	}
}

// lessonGH wraps fakeGH with the git trees endpoint, recording whether it was
// read and from which ref.
type lessonGH struct {
	fakeGH
	tree     string // the trees endpoint's --jq output
	treeErr  error
	treeRead string
}

func (g *lessonGH) run(ctx context.Context, args ...string) ([]byte, error) {
	if line := strings.Join(args, " "); strings.HasPrefix(line, "api ") && strings.Contains(line, "/git/trees/") {
		g.treeRead = line
		return []byte(g.tree), g.treeErr
	}
	return g.fakeGH.run(ctx, args...)
}

func TestLand_RefusesALessonWhoseNumberTheBaseTook(t *testing.T) {
	g := &lessonGH{
		fakeGH: fakeGH{view: readyView, checks: greenChecks, deps: `[]`,
			files: `{"filename":"docs/lessons/lesson-497-b.md","previous_filename":null,"status":"added"}`},
		tree: "lesson-001-x.md\nlesson-497-a.md\n_index.md\n",
	}
	spaced := *g
	spaced.tree = "lesson-497-a name with spaces.md\n"
	if res, err := Land(context.Background(), Options{Run: spaced.run, Untriaged: noneUntriaged}, 30); err != nil ||
		!strings.Contains(strings.Join(res.Reasons, ";"), "docs/lessons/lesson-497-a name with spaces.md") {
		t.Errorf("a base lesson whose name has spaces must hold its number whole, got %+v, %v", res, err)
	}
	res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Merged || len(g.calls) != 0 || !strings.Contains(strings.Join(res.Reasons, ";"), "lesson 497") {
		t.Errorf("want a refusal naming lesson 497 and no mutation, got %+v, calls %v", res, g.calls)
	}
	if !strings.Contains(g.treeRead, "/git/trees/main:docs/lessons") {
		t.Errorf("the lessons were not read from the base: %q", g.treeRead)
	}
}

// A slug rename lists the old name as previous_filename, and the old name is
// the one the base holds: the PR frees it, so the number is not taken.
func TestLand_ARenamedLessonKeepsItsOwnNumber(t *testing.T) {
	g := &lessonGH{
		fakeGH: fakeGH{view: readyView, checks: greenChecks, deps: `[]`,
			files: `{"filename":"docs/lessons/lesson-497-b.md","previous_filename":"docs/lessons/lesson-497-a.md","status":"renamed"}`},
		tree: "lesson-497-a.md\n",
	}
	res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Merged {
		t.Errorf("want the rename to land, got %+v", res)
	}
}

func TestLand_AnUnreadableLessonListIsAnErrorButAMissingOneIsEmpty(t *testing.T) {
	addsLesson := `{"filename":"docs/lessons/lesson-1-first.md","previous_filename":null,"status":"added"}`
	missing := &lessonGH{
		fakeGH:  fakeGH{view: readyView, checks: greenChecks, deps: `[]`, files: addsLesson},
		treeErr: errors.New("gh: Not Found (HTTP 404)"),
	}
	if res, err := Land(context.Background(), Options{Run: missing.run, Untriaged: noneUntriaged}, 30); err != nil || !res.Merged {
		t.Errorf("a base with no lessons holds no number: got %+v, %v", res, err)
	}
	for name, g := range map[string]*lessonGH{
		"a failed read": {fakeGH: fakeGH{view: readyView, checks: greenChecks, deps: `[]`, files: addsLesson},
			treeErr: errors.New("gh: Bad credentials (HTTP 401)")},
		"a truncated tree": {fakeGH: fakeGH{view: readyView, checks: greenChecks, deps: `[]`, files: addsLesson},
			tree: "truncated\n"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
			if err == nil || res.Merged || len(g.calls) != 0 {
				t.Errorf("want an error and no mutation, got %+v, %v, calls %v", res, err, g.calls)
			}
		})
	}
}

// Most PRs touch no lesson, and they pay no extra read.
func TestLand_APRWithoutALessonDoesNotReadTheBaseLessons(t *testing.T) {
	g := &lessonGH{fakeGH: fakeGH{view: readyView, checks: greenChecks, deps: `[]`}}
	if _, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30); err != nil {
		t.Fatal(err)
	}
	if g.treeRead != "" {
		t.Errorf("read the base lessons for a PR that touches none: %q", g.treeRead)
	}
}

// A stacked PR's base is a branch with a slash in its name. The trees route
// takes it as written (measured live on fix/pr-agent-image-digest), so the
// lessons are read from that branch, not reported missing.
func TestLand_ReadsTheLessonsOfASlashedBase(t *testing.T) {
	view := strings.Replace(readyView, `"baseRefName":"main"`, `"baseRefName":"feat/base"`, 1)
	if view == readyView {
		t.Fatal("readyView no longer names main as its base; update this test")
	}
	g := &lessonGH{
		fakeGH: fakeGH{view: view, checks: greenChecks, deps: `[]`,
			files: `{"filename":"docs/lessons/lesson-497-b.md","previous_filename":null,"status":"added"}`},
		tree: "lesson-497-a.md\n",
	}
	res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(g.treeRead, "/git/trees/feat/base:docs/lessons") || !strings.Contains(strings.Join(res.Reasons, ";"), "on feat/base") {
		t.Errorf("want the lessons read from feat/base and a refusal naming it, got read %q, %+v", g.treeRead, res)
	}
}
