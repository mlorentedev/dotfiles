package lessons

import (
	"os"
	"path/filepath"
	"runtime"
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
			l, out, err := Normalize("lesson-215-x.md", []byte(raw))
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
			_, again, err := Normalize("lesson-215-x.md", out)
			if err != nil || string(again) != s {
				t.Errorf("second normalisation changed the file:\n%s\n---\n%s", s, again)
			}
		})
	}
}

func TestNormalize_KeepsTheOtherFrontmatterKeysAndTheBody(t *testing.T) {
	_, out, err := Normalize("lesson-026-x.md", []byte(shapes["vault"]))
	if err != nil {
		t.Fatal(err)
	}
	for _, must := range []string{"owner: manu\n", "tags: [a, b]\n", "## Context\n"} {
		if !strings.Contains(string(out), must) {
			t.Errorf("lost %q:\n%s", must, out)
		}
	}
}

func TestNormalize_RefusesWhatItCannotRead(t *testing.T) {
	cases := map[string]struct{ name, raw string }{
		"not a lesson file": {"notes.md", "# x\n"},
		"no date":           {"lesson-001-x.md", "# x\n\nbody\n"},
		"no title":          {"lesson-001-x.md", "---\ncreated: \"2026-01-01\"\n---\n\nbody\n"},
	}
	for name, tc := range cases {
		if _, _, err := Normalize(tc.name, []byte(tc.raw)); err == nil {
			t.Errorf("%s: want an error", name)
		}
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
		"lesson-001-a.md":          shapes["inline"],
		"lesson-002-b.md":          shapes["recent"],
		"category/lesson-003-c.md": shapes["vault"],
		"category/not-a-lesson.md": "ignored\n",
	})
	changes, err := Plan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 3 { // the three lessons; there is no index to write
		t.Fatalf("want 3 changes, got %d", len(changes))
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
		"lesson-001-a.md":          shapes["recent"],
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
		"lesson-001-a.md":          linking,
		"lesson-002-b.md":          shapes["recent"],
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
		l, _, err := Normalize("lesson-001-x.md", []byte(h1+"\n**Date:** 2026-01-01\n"))
		if err != nil {
			t.Fatalf("%q: %v", h1, err)
		}
		if l.Title != want {
			t.Errorf("%q: title %q, want %q", h1, l.Title, want)
		}
	}
}

// CLI-131: fmt needs no index, and never creates one.
func TestPlan_NeedsNoIndexAndWritesNone(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"lesson-001-a.md":          shapes["recent"],
		"category/lesson-002-b.md": shapes["inline"],
	})
	changes, err := Plan(dir)
	if err != nil {
		t.Fatalf("a tree with lessons and no _index.md was refused: %v", err)
	}
	for _, c := range changes {
		if filepath.Base(c.Path) == "_index.md" {
			t.Errorf("Plan wants to write %s", c.Path)
		}
	}
	if err := Apply(changes); err != nil {
		t.Fatal(err)
	}
	if got, _ := StrayIndexes(dir); len(got) != 0 {
		t.Errorf("fmt created an index: %v", got)
	}
}

func TestIndex_OneTablePerDirectorySortedByNumber(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"lesson-002-b.md":          shapes["inline"],
		"lesson-001-a.md":          shapes["recent"],
		"category/lesson-003-c.md": shapes["vault"],
		"category/not-a-lesson.md": "ignored\n",
		"empty/readme.md":          "no lessons here\n",
	})
	out, err := Index(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	i1, i2, i3 := strings.Index(got, "| [001 - "), strings.Index(got, "| [002 - "), strings.Index(got, "| [003 - ")
	if i1 < 0 || i2 < i1 || i3 < i2 {
		t.Errorf("rows missing or out of order:\n%s", got)
	}
	for _, must := range []string{"# Lessons\n", "](lesson-001-a.md)", "\n## category\n", "](category/lesson-003-c.md)"} {
		if !strings.Contains(got, must) {
			t.Errorf("index lacks %q:\n%s", must, got)
		}
	}
	if strings.Contains(got, "## empty") || strings.Contains(got, "not-a-lesson") {
		t.Errorf("index lists what is not a lesson:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "_index.md")); err == nil {
		t.Error("Index wrote a file")
	}
}

func TestIndex_EscapesAPipeInATitleAndRefusesAnUnreadableLesson(t *testing.T) {
	piped := strings.Replace(shapes["recent"], "# A checksum proves the bytes", "# A | B", 1)
	piped = strings.Replace(piped, `title: "A checksum proves the bytes"`, `title: "A | B"`, 1)
	dir := writeTree(t, map[string]string{"lesson-001-a.md": piped})
	out, err := Index(dir)
	if err != nil || !strings.Contains(string(out), `A \| B`) {
		t.Fatalf("pipe not escaped (err %v):\n%s", err, out)
	}
	bad := writeTree(t, map[string]string{"lesson-001-a.md": "# no date\n"})
	if _, err := Index(bad); err == nil {
		t.Error("a lesson that cannot be normalised was skipped, not reported")
	}
	dup := writeTree(t, map[string]string{"lesson-001-a.md": shapes["recent"], "category/lesson-001-b.md": shapes["recent"]})
	if _, err := Index(dup); err == nil || !strings.Contains(err.Error(), "used twice") {
		t.Errorf("a number used twice across directories was rendered: %v", err)
	}
}

func TestIndex_ReportsAnUnreadableCategoryDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a directory cannot be made unreadable here")
	}
	dir := writeTree(t, map[string]string{"lesson-001-a.md": shapes["recent"], "category/lesson-002-b.md": shapes["inline"]})
	locked := filepath.Join(dir, "category")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	if _, err := Index(dir); err == nil {
		t.Error("an unreadable category directory dropped its lessons from the index without an error")
	}
}

func TestStrayIndexes_NamesACommittedIndexAtAnyLevel(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"_index.md":                "# Index\n",
		"lesson-001-a.md":          shapes["recent"],
		"category/_index.md":       "# Category\n",
		"category/lesson-002-b.md": shapes["inline"],
	})
	got, err := StrayIndexes(dir)
	if err != nil || len(got) != 2 {
		t.Fatalf("want both indexes named, got %v (err %v)", got, err)
	}
}
