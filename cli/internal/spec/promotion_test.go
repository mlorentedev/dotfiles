package spec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// promotionsWith is a verification.md whose three candidate lines carry the
// given answers, in the template's wording.
func promotionsWith(lesson, adr, pattern string) string {
	return "# Verification\n\n## Promotion candidates\n\n" +
		"- [ ] Lesson for the repo's `docs/lessons/`? " + lesson + "\n" +
		"- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? " + adr + "\n" +
		"- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. " + pattern + "\n" +
		"\n## Archive checklist\n\n- [ ] Folder moved\n"
}

func noVault() (string, error) { return "", errors.New("no knowledge vault found") }

func promotionSpec(t *testing.T, verification string) (root, dir string) {
	t.Helper()
	root = t.TempDir()
	dir = writeSpec(t, root, "AI-001-x", map[string]string{"verification.md": verification})
	return root, dir
}

func TestPromotionsAnsweredNoWithAReasonPass(t *testing.T) {
	root, dir := promotionSpec(t, promotionsWith("no: nothing new", "no: no decision", "no: one project only"))
	if problems := CheckPromotions(root, dir, noVault); len(problems) != 0 {
		t.Fatalf("want no problems, got %q", problems)
	}
}

func TestPromotionsRefuseTheTemplatePlaceholder(t *testing.T) {
	root, dir := promotionSpec(t, promotionsWith(
		"<yes / no - one line of what>", "<yes / no - one line of what>", "<yes / no - one line>"))
	problems := CheckPromotions(root, dir, noVault)
	if len(problems) != 3 {
		t.Fatalf("want 3 problems, got %q", problems)
	}
	for _, p := range problems {
		if !strings.Contains(p, "unanswered") {
			t.Errorf("problem should say unanswered: %q", p)
		}
	}
	if !strings.Contains(problems[0], "Lesson") {
		t.Errorf("a problem should name its line: %q", problems[0])
	}
}

func TestPromotionsRefuseANoWithoutAReason(t *testing.T) {
	for _, answer := range []string{"no", "no:", "No.", "no: <reason>"} {
		root, dir := promotionSpec(t, promotionsWith(answer, "no: x", "no: y"))
		problems := CheckPromotions(root, dir, noVault)
		if len(problems) != 1 || !strings.Contains(problems[0], "reason") {
			t.Errorf("answer %q: want one reason problem, got %q", answer, problems)
		}
	}
}

func TestPromotionsRefuseAYesThatNamesNoFile(t *testing.T) {
	root, dir := promotionSpec(t, promotionsWith("yes", "no: x", "no: y"))
	problems := CheckPromotions(root, dir, noVault)
	if len(problems) != 1 || !strings.Contains(problems[0], "names no file") {
		t.Fatalf("want one names-no-file problem, got %q", problems)
	}
}

func TestPromotionsRefuseAYesWhoseFileIsMissing(t *testing.T) {
	root, dir := promotionSpec(t, promotionsWith("yes: docs/lessons/lesson-999-ghost.md", "no: x", "no: y"))
	problems := CheckPromotions(root, dir, noVault)
	if len(problems) != 1 || !strings.Contains(problems[0], "docs/lessons/lesson-999-ghost.md") {
		t.Fatalf("want one problem naming the missing file, got %q", problems)
	}
}

func TestPromotionsAcceptAYesWhoseFileExists(t *testing.T) {
	root, dir := promotionSpec(t, promotionsWith(
		"**Yes:** `docs/lessons/lesson-301-x.md`, extended from lesson 290", "no: x", "no: y"))
	lesson := filepath.Join(root, "docs", "lessons", "lesson-301-x.md")
	if err := os.MkdirAll(filepath.Dir(lesson), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lesson, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if problems := CheckPromotions(root, dir, noVault); len(problems) != 0 {
		t.Fatalf("want no problems, got %q", problems)
	}
}

func TestPromotionsResolveAPatternAgainstTheVault(t *testing.T) {
	root, dir := promotionSpec(t, promotionsWith("no: x", "no: y", "yes: 00_meta/patterns/pattern-x.md"))
	vaultRoot := t.TempDir()
	pattern := filepath.Join(vaultRoot, "00_meta", "patterns", "pattern-x.md")
	if err := os.MkdirAll(filepath.Dir(pattern), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pattern, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	vault := func() (string, error) { return vaultRoot, nil }
	if problems := CheckPromotions(root, dir, vault); len(problems) != 0 {
		t.Fatalf("want no problems, got %q", problems)
	}

	// The same path under the repo, and not in the vault, does not count.
	inRepo := filepath.Join(root, "00_meta", "patterns", "pattern-x.md")
	if err := os.MkdirAll(filepath.Dir(inRepo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inRepo, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty := func() (string, error) { return t.TempDir(), nil }
	if problems := CheckPromotions(root, dir, empty); len(problems) != 1 {
		t.Fatalf("a pattern must resolve against the vault, not the repo: got %q", problems)
	}
}

func TestPromotionsRefuseAPatternWhenTheVaultIsUnresolved(t *testing.T) {
	root, dir := promotionSpec(t, promotionsWith("no: x", "no: y", "yes: 00_meta/patterns/pattern-x.md"))
	problems := CheckPromotions(root, dir, noVault)
	if len(problems) != 1 || !strings.Contains(problems[0], "vault") {
		t.Fatalf("want one vault problem, got %q", problems)
	}
	if problems := CheckPromotions(root, dir, nil); len(problems) != 1 {
		t.Fatalf("a nil resolver must refuse, not panic or pass: got %q", problems)
	}
}

func TestPromotionsAcceptTheArchivedEmphasisForm(t *testing.T) {
	// SKILL-001's archived answer: bold, a full stop, then the reason.
	root, dir := promotionSpec(t, promotionsWith(
		"**No.** The candidate is already recorded where it acts.", "no: x", "no: y"))
	if problems := CheckPromotions(root, dir, noVault); len(problems) != 0 {
		t.Fatalf("want no problems, got %q", problems)
	}
}

func TestPromotionsRefuseAMissingFileSectionOrLines(t *testing.T) {
	root := t.TempDir()
	dir := writeSpec(t, root, "AI-001-x", map[string]string{"proposal.md": "x\n"})
	if err := os.Remove(filepath.Join(dir, "verification.md")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if problems := CheckPromotions(root, dir, noVault); len(problems) != 1 {
		t.Errorf("no verification.md: want one problem, got %q", problems)
	}

	for name, content := range map[string]string{
		"no section":    "# Verification\n\nall done\n",
		"empty section": "## Promotion candidates\n\nnothing here\n\n## Archive checklist\n",
	} {
		_, d := promotionSpec(t, content)
		if problems := CheckPromotions(root, d, noVault); len(problems) != 1 {
			t.Errorf("%s: want one problem, got %q", name, problems)
		}
	}
}

func TestArchiveRefusesUnansweredPromotionsAndMovesNothing(t *testing.T) {
	root := t.TempDir()
	dir := writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md":     "---\nstatus: implementing\n---\n# AI-001-x\n",
		"review.md":       passingReview("AI-001-x"),
		"verification.md": promotionsWith("<yes / no - one line of what>", "no: x", "no: y"),
	})
	_, err := Archive(root, "AI-001-x", ArchiveOptions{ForceWithoutReview: true, BypassReason: "fixture"})
	if err == nil {
		t.Fatal("want a refusal for an unanswered promotion line")
	}
	if !strings.Contains(err.Error(), "Lesson") || !strings.Contains(err.Error(), "no: <reason>") {
		t.Errorf("the refusal should name the line and the answer grammar: %v", err)
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Errorf("a refused archive must leave the spec in place: %v", statErr)
	}
}

// A spec fresh from the template has answered nothing, and the template states
// the grammar the check reads, so the two cannot drift apart unnoticed.
func TestPromotionsRefuseAFreshTemplateAndItStatesTheGrammar(t *testing.T) {
	tmpl, err := os.ReadFile(filepath.Join("templates", "verification.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{PromotionHeading, "yes: <path>", "no: <reason>", "dotf spec archive"} {
		if !strings.Contains(string(tmpl), want) {
			t.Errorf("the verification.md template should state %q", want)
		}
	}
	root, dir := promotionSpec(t, string(tmpl))
	problems := CheckPromotions(root, dir, noVault)
	if len(problems) != 3 {
		t.Fatalf("a fresh template has three unanswered lines, got %q", problems)
	}
	for _, p := range problems {
		if !strings.Contains(p, "unanswered") {
			t.Errorf("want unanswered, got %q", p)
		}
	}
}
