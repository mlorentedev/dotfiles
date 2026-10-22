package lessons

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func lessonBody(title, extra string) string {
	return "---\ntitle: \"" + title + "\"\ndate: \"2026-10-05\"\n---\n\n# " + title + "\n\n" + extra
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The state `dotf pr land` leaves a branch in (#2752, #2837): main merged in,
// and the base's lesson took the number this branch's lesson uses.
func collided(t *testing.T) string {
	return writeTree(t, map[string]string{
		"lesson-497-mine.md":   lessonBody("Mine", "Self link [[lesson-497-mine]].\n"),
		"lesson-497-theirs.md": lessonBody("Theirs", ""),
		"lesson-498-cites.md":  lessonBody("Cites", "See [[lesson-497-mine|alias]], [[lesson-497-mine#Heading]] and [[lesson-497]].\n"),
		"cat/lesson-510-x.md":  lessonBody("Category", "Also [[lesson-497-mine]].\n"),
	})
}

func TestRenumber_NextTakesMaxPlusOneAcrossCategoriesAndRewritesFullStemLinks(t *testing.T) {
	dir := collided(t)
	got, err := Renumber(dir, filepath.Join(dir, "lesson-497-mine.md"), 0, os.Rename)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "lesson-511-mine.md")
	if got.To != want {
		t.Fatalf("moved to %s, want %s (max+1 over the root and its categories)", got.To, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "lesson-497-mine.md")); !os.IsNotExist(err) {
		t.Errorf("the old file is still there: %v", err)
	}
	moved := read(t, want)
	for _, must := range []string{"id: \"lesson-511-mine\"", "[[lesson-511-mine]]"} {
		if !strings.Contains(moved, must) {
			t.Errorf("moved file lacks %q:\n%s", must, moved)
		}
	}
	cites := read(t, filepath.Join(dir, "lesson-498-cites.md"))
	for _, must := range []string{"[[lesson-511-mine|alias]]", "[[lesson-511-mine#Heading]]"} {
		if !strings.Contains(cites, must) {
			t.Errorf("citing lesson lacks %q:\n%s", must, cites)
		}
	}
	if cat := read(t, filepath.Join(dir, "cat", "lesson-510-x.md")); !strings.Contains(cat, "[[lesson-511-mine]]") {
		t.Errorf("category lesson's link was not rewritten:\n%s", cat)
	}
	if read(t, filepath.Join(dir, "lesson-497-theirs.md")) != lessonBody("Theirs", "") {
		t.Error("the other lesson holding 497 was touched")
	}
}

// [[lesson-497]] names both lessons that hold 497. After the move it still
// resolves, to the other one, so fmt --check stays green and a wrong guess
// would be silent: it is left alone and reported.
func TestRenumber_LeavesAnAmbiguousNumberLinkAndReportsIt(t *testing.T) {
	dir := collided(t)
	got, err := Renumber(dir, filepath.Join(dir, "lesson-497-mine.md"), 0, os.Rename)
	if err != nil {
		t.Fatal(err)
	}
	if cites := read(t, filepath.Join(dir, "lesson-498-cites.md")); !strings.Contains(cites, "[[lesson-497]]") {
		t.Errorf("the ambiguous link was rewritten:\n%s", cites)
	}
	if len(got.Ambiguous) != 1 || got.Ambiguous[0] != "lesson-498-cites.md links [[lesson-497]]" {
		t.Errorf("ambiguous = %q", got.Ambiguous)
	}
}

// With the number held by this lesson alone, the number form names it and is
// rewritten too.
func TestRenumber_RewritesTheNumberFormWhenTheNumberIsUnique(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"lesson-007-a.md": lessonBody("A", ""),
		"lesson-008-b.md": lessonBody("B", "See [[lesson-007]] and [[lesson-007-a]].\n"),
	})
	if _, err := Renumber(dir, filepath.Join(dir, "lesson-007-a.md"), 20, os.Rename); err != nil {
		t.Fatal(err)
	}
	b := read(t, filepath.Join(dir, "lesson-008-b.md"))
	if !strings.Contains(b, "See [[lesson-020]] and [[lesson-020-a]].") {
		t.Errorf("links not rewritten in both forms:\n%s", b)
	}
}

func TestRenumber_LeavesTheTreeFormattedAndASecondRunChangesNothing(t *testing.T) {
	dir := collided(t)
	if _, err := Plan(dir); err == nil {
		t.Fatal("the collided fixture should fail fmt before the renumber")
	}
	src := filepath.Join(dir, "lesson-497-mine.md")
	if _, err := Renumber(dir, src, 0, os.Rename); err != nil {
		t.Fatal(err)
	}
	changes, err := Plan(dir)
	if err != nil {
		t.Fatalf("fmt fails after the renumber: %v", err)
	}
	if err := Apply(changes); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, dir)
	again, err := Renumber(dir, src, 0, os.Rename)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Already || again.To != filepath.Join(dir, "lesson-511-mine.md") {
		t.Errorf("second run = %+v, want already at lesson-511-mine.md", again)
	}
	if after := snapshot(t, dir); after != before {
		t.Errorf("second run changed the tree:\n%s\n---\n%s", before, after)
	}
}

func TestRenumber_RefusesATakenNumberAndWritesNothing(t *testing.T) {
	dir := collided(t)
	before := snapshot(t, dir)
	moves := 0
	move := func(a, b string) error { moves++; return os.Rename(a, b) }
	for _, to := range []int{498, 510, 497} {
		if _, err := Renumber(dir, filepath.Join(dir, "lesson-497-mine.md"), to, move); err == nil || !strings.Contains(err.Error(), "taken") {
			t.Errorf("--to %d: want a taken-number refusal, got %v", to, err)
		}
	}
	if moves != 0 || snapshot(t, dir) != before {
		t.Errorf("a refused renumber moved or wrote something (moves=%d)", moves)
	}
}

func TestRenumber_AFailedMoveWritesNoLink(t *testing.T) {
	dir := collided(t)
	before := snapshot(t, dir)
	_, err := Renumber(dir, filepath.Join(dir, "lesson-497-mine.md"), 0, func(string, string) error { return errors.New("boom") })
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want the move error, got %v", err)
	}
	if snapshot(t, dir) != before {
		t.Error("links were rewritten although the move failed")
	}
}

func TestRenumber_RefusesWhatIsNotALessonOfTheTree(t *testing.T) {
	dir := collided(t)
	other := writeTree(t, map[string]string{"lesson-001-z.md": lessonBody("Z", "")})
	for name, file := range map[string]string{
		"missing":      filepath.Join(dir, "lesson-600-nowhere.md"),
		"not a lesson": filepath.Join(dir, "README.md"),
		"outside":      filepath.Join(other, "lesson-001-z.md"),
	} {
		if _, err := Renumber(dir, file, 0, os.Rename); err == nil {
			t.Errorf("%s: renumbered %s", name, file)
		}
	}
}

// snapshot renders every file under dir, path and content, for before/after
// comparison.
func snapshot(t *testing.T, dir string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b.WriteString(p + "\n" + read(t, p) + "\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
