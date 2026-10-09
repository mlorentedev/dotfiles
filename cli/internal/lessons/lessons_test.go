package lessons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The three shapes the repository's lessons were written in, measured on
// 2026-10-06: no frontmatter with an inline date, the vault's created key, and
// the newest title/date frontmatter.
var shapes = map[string]string{
	"inline": "# Lesson 215 — A parser reads the other runner's review as empty\n\n**Date:** 2026-08-21\n**Context:** reviewing.\n\n## What happened\n",
	"vault":  "---\nid: lesson-026-x\ntype: lesson\nstatus: active\ncreated: \"2026-03-25\"\nowner: manu\ntags: [a, b]\n---\n\n# Lesson 026: Config guards vs tool guards\n\n## Context\n",
	"recent": "---\ntitle: \"A checksum proves the bytes\"\ndate: \"2026-10-05\"\n---\n\n# A checksum proves the bytes\n\n## Context\n",
	// Blank lines after the H1 and the removed date line must survive: an H1
	// pattern ending in \s* ate them, one per run, so fmt never converged.
	"blank": "# Lesson 215: A blank-line title\n\n**Date:** 2026-08-21\n\nbody\n",
	// A number prefix without the word Lesson, as lesson 213's H1 had it.
	"bare": "# 215 — A bare-number title\n\n**Date:** 2026-08-21\n",
}

func TestNormalize_EveryShapeReachesTheCanonicalForm(t *testing.T) {
	want := map[string]struct{ title, created string }{
		"inline": {"A parser reads the other runner's review as empty", "2026-08-21"},
		"vault":  {"Config guards vs tool guards", "2026-03-25"},
		"recent": {"A checksum proves the bytes", "2026-10-05"},
		"blank":  {"A blank-line title", "2026-08-21"},
		"bare":   {"A bare-number title", "2026-08-21"},
	}
	for shape, raw := range shapes {
		t.Run(shape, func(t *testing.T) {
			l, out, err := Normalize("lesson-215-x.md", []byte(raw), "")
			if err != nil {
				t.Fatal(err)
			}
			if l.Title != want[shape].title || l.Created != want[shape].created || l.Number != 215 {
				t.Errorf("parsed %+v", l)
			}
			s := string(out)
			for _, must := range []string{
				"---\nid: \"lesson-215-x\"\ntype: lesson\nstatus: active\ntitle: ",
				"created: \"" + want[shape].created + "\"\n",
				"\n# " + want[shape].title + "\n",
			} {
				if !strings.Contains(s, must) {
					t.Errorf("output lacks %q:\n%s", must, s)
				}
			}
			for _, mustNot := range []string{"**Date:**", "\ndate:", "# Lesson "} {
				if strings.Contains(s, mustNot) {
					t.Errorf("output keeps %q:\n%s", mustNot, s)
				}
			}
			// Idempotent: the canonical form is a fixed point.
			_, again, err := Normalize("lesson-215-x.md", out, "")
			if err != nil || string(again) != s {
				t.Errorf("second normalisation changed the file:\n%s\n---\n%s", s, again)
			}
		})
	}
}

func TestNormalize_KeepsTheOtherFrontmatterKeysAndTheBody(t *testing.T) {
	_, out, err := Normalize("lesson-026-x.md", []byte(shapes["vault"]), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, must := range []string{"owner: manu\n", "tags: [a, b]\n", "## Context\n"} {
		if !strings.Contains(string(out), must) {
			t.Errorf("lost %q:\n%s", must, out)
		}
	}
}

func TestNormalize_TheSeedTitleWinsOnlyWhileTheFileDeclaresNone(t *testing.T) {
	l, _, _ := Normalize("lesson-026-x.md", []byte(shapes["vault"]), "Config Deployment Guards, curated")
	if l.Title != "Config Deployment Guards, curated" {
		t.Errorf("seed ignored on a file with no title: %q", l.Title)
	}
	l, _, _ = Normalize("lesson-337-x.md", []byte(shapes["recent"]), "a stale index title")
	if l.Title != "A checksum proves the bytes" {
		t.Errorf("seed overrode the file's own title: %q", l.Title)
	}
}

func TestNormalize_RefusesWhatItCannotRead(t *testing.T) {
	cases := map[string]struct{ name, raw string }{
		"not a lesson file": {"notes.md", "# x\n"},
		"no date":           {"lesson-001-x.md", "# x\n\nbody\n"},
		"no title":          {"lesson-001-x.md", "---\ncreated: \"2026-01-01\"\n---\n\nbody\n"},
	}
	for name, tc := range cases {
		if _, _, err := Normalize(tc.name, []byte(tc.raw), ""); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestRenderIndex_ReplacesTheHandKeptTableOnceThenItsMarkers(t *testing.T) {
	hand := "# Index\n\n> prose\n\n| Lesson | Date | Scope |\n|---|---|---|\n| [001 - Old](lesson-001-a.md) | 2026-01-01 |  |\n\ntrailing prose\n"
	ls := []Lesson{{File: "lesson-002-b.md", Number: 2, Title: "B | pipe", Created: "2026-01-02"}, {File: "lesson-001-a.md", Number: 1, Title: "A", Created: "2026-01-01"}}

	first := string(RenderIndex([]byte(hand), ls))
	for _, must := range []string{"> prose\n", IndexBegin, "| [001 - A](lesson-001-a.md) | 2026-01-01 |\n| [002 - B \\| pipe](lesson-002-b.md) | 2026-01-02 |\n", IndexEnd, "trailing prose\n"} {
		if !strings.Contains(first, must) {
			t.Errorf("first render lacks %q:\n%s", must, first)
		}
	}
	if strings.Contains(first, "Scope") || strings.Contains(first, "Old") {
		t.Errorf("the hand-kept table survived:\n%s", first)
	}
	if second := string(RenderIndex([]byte(first), ls)); second != first {
		t.Errorf("rendering a generated index changed it:\n%s\n---\n%s", first, second)
	}
}

func TestIndexTitles_ReadsTheHandKeptRows(t *testing.T) {
	got := IndexTitles([]byte("| [302 - `producer \\| grep -q` fails](lesson-302-x.md) | 2026-09-25 |  |\n"))
	if got["lesson-302-x.md"] != "`producer | grep -q` fails" {
		t.Errorf("got %q", got)
	}
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestPlan_ConvergesAndThenHasNothingToDo(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"_index.md":                "# Index\n\n| Lesson | Date | Scope |\n|---|---|---|\n",
		"lesson-001-a.md":          shapes["inline"],
		"lesson-002-b.md":          shapes["recent"],
		"category/_index.md":       "# Category\n",
		"category/lesson-003-c.md": shapes["vault"],
		"category/not-a-lesson.md": "ignored\n",
	})
	changes, err := Plan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 5 { // three lessons and two indexes
		t.Fatalf("want 5 changes, got %d", len(changes))
	}
	if err := Apply(changes); err != nil {
		t.Fatal(err)
	}
	again, err := Plan(dir)
	if err != nil || len(again) != 0 {
		t.Errorf("a converged tree still plans %d change(s), err %v", len(again), err)
	}
}

func TestPlan_RefusesANumberUsedTwice(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"_index.md":                "# Index\n",
		"lesson-001-a.md":          shapes["recent"],
		"category/_index.md":       "# Category\n",
		"category/lesson-001-b.md": shapes["recent"],
	})
	if _, err := Plan(dir); err == nil || !strings.Contains(err.Error(), "used twice") {
		t.Fatalf("want a duplicate-number error, got %v", err)
	}
}

// A renumbering moves a file and leaves its inbound wikilinks behind (#1514:
// renumbering lesson-256 to 271 left lesson-264's link pointing at nothing).
// Both forms in use resolve: a number names a lesson, a full stem names a file,
// across categories.
func TestPlan_RefusesAWikilinkThatNamesNoLesson(t *testing.T) {
	linking := "---\ntitle: \"Links\"\ndate: \"2026-10-05\"\n---\n\n# Links\n\n" +
		"See [[lesson-002]], [[lesson-003-c]] and [[lesson-009-gone]].\n"
	dir := writeTree(t, map[string]string{
		"_index.md":                "# Index\n",
		"lesson-001-a.md":          linking,
		"lesson-002-b.md":          shapes["recent"],
		"category/_index.md":       "# Category\n",
		"category/lesson-003-c.md": shapes["recent"],
	})
	_, err := Plan(dir)
	if err == nil || !strings.Contains(err.Error(), "lesson-001-a.md links [[lesson-009-gone]]") {
		t.Fatalf("want the dangling wikilink named with its file, got %v", err)
	}
	for _, ok := range []string{"[[lesson-002]]", "[[lesson-003-c]]"} {
		if strings.Contains(err.Error(), ok) {
			t.Errorf("%s resolves and was reported: %v", ok, err)
		}
	}
}

// The index's prose is hand-written above the generated table, so a link there
// dangles after a renumbering exactly as one in a lesson does. The deleted
// shell guard scanned every file under the tree, indexes included.
func TestPlan_RefusesAWikilinkInAnIndexThatNamesNoLesson(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"_index.md":                "# Index\n\nStart at [[lesson-001-a]].\n",
		"lesson-001-a.md":          shapes["recent"],
		"category/_index.md":       "# Category\n\nSee [[lesson-008-moved]].\n",
		"category/lesson-003-c.md": shapes["recent"],
	})
	_, err := Plan(dir)
	if err == nil || !strings.Contains(err.Error(), filepath.Join("category", "_index.md")+" links [[lesson-008-moved]]") {
		t.Fatalf("want the dangling wikilink named with its index, got %v", err)
	}
	if strings.Contains(err.Error(), "[[lesson-001-a]]") {
		t.Errorf("[[lesson-001-a]] resolves and was reported: %v", err)
	}
}

func TestPlan_RefusesLessonsWithoutAnIndex(t *testing.T) {
	dir := writeTree(t, map[string]string{"lesson-001-a.md": shapes["recent"]})
	if _, err := Plan(dir); err == nil {
		t.Fatal("lessons without an _index.md were accepted")
	}
}

func TestPlan_RequiresTheRootIndexWhenOnlyACategoryHasLessons(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"category/_index.md":       "# Category\n",
		"category/lesson-001-a.md": shapes["recent"],
	})
	if _, err := Plan(dir); err == nil || !strings.Contains(err.Error(), "root _index.md") {
		t.Fatalf("want a missing-root-index error, got %v", err)
	}
}

// Only the numbering forms measured in the repository are a prefix: "Lesson
// NNN" with a separator, or a bare NNN followed by a spaced dash. A title that
// merely starts with digits keeps them.
func TestNormalize_StripsOnlyTheMeasuredNumberingPrefixes(t *testing.T) {
	cases := map[string]string{
		"# Lesson 026: Config guards\n": "Config guards",
		"# Lesson 215 — A parser\n":     "A parser",
		"# Lesson 7 - Hyphen form\n":    "Hyphen form",
		"# 213 — A reviewer\n":          "A reviewer",
		"# 213 – En dash\n":             "En dash",
		"# 3-2-1 backup rule\n":         "3-2-1 backup rule",
		"# 12-factor apps\n":            "12-factor apps",
		"# 2026 roadmap\n":              "2026 roadmap",
		"# 2 shells, one script\n":      "2 shells, one script",
	}
	for h1, want := range cases {
		l, _, err := Normalize("lesson-001-x.md", []byte(h1+"\n**Date:** 2026-01-01\n"), "")
		if err != nil {
			t.Fatalf("%q: %v", h1, err)
		}
		if l.Title != want {
			t.Errorf("%q: title %q, want %q", h1, l.Title, want)
		}
	}
}

// A directory whose lessons all moved into categories still has its generated
// table regenerated, so stale rows cannot outlive the files they listed.
func TestPlan_RegeneratesAnIndexWhoseDirectoryHasNoLessonsLeft(t *testing.T) {
	stale := "# Index\n\n" + IndexBegin + "\n| Lesson | Date |\n|---|---|\n| [001 - A](lesson-001-a.md) | 2026-01-01 |\n" + IndexEnd + "\n"
	dir := writeTree(t, map[string]string{
		"_index.md":                stale,
		"category/_index.md":       "# Category\n",
		"category/lesson-001-a.md": shapes["recent"],
	})
	changes, err := Plan(dir)
	if err != nil {
		t.Fatal(err)
	}
	var root []byte
	for _, c := range changes {
		if c.Path == filepath.Join(dir, "_index.md") {
			root = c.Want
		}
	}
	if root == nil || strings.Contains(string(root), "lesson-001-a.md") {
		t.Fatalf("the stale root table was not emptied:\n%s", root)
	}
	if err := Apply(changes); err != nil {
		t.Fatal(err)
	}
	if again, err := Plan(dir); err != nil || len(again) != 0 {
		t.Errorf("a converged tree still plans %d change(s), err %v", len(again), err)
	}
}
