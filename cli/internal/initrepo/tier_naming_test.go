package initrepo

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ADR-060's "lanes" are the persistent named worktrees (feat-lane-primary and
// the others). AGENTS.md used the same word for the three implementation tiers
// (Inline Direct, Fast-Track Development, Full SDD), so an agent told to "start
// in a lane" could read either. HARNESS-195 AC8 renames the tiers; this keeps
// the word from coming back, in the repository's own AGENTS.md and in the
// template that seeds every other repository's.

var (
	laneWord = regexp.MustCompile(`(?i)\blanes?\b`)
	// Any one of these names a tier, so a paragraph that carries one is about
	// tiers. They are the tiers' own names, not the word under test.
	tierName = regexp.MustCompile(`Inline Direct|Fast-Track Development|Full SDD`)
)

// tierParagraphs returns the blank-line separated paragraphs of doc that name
// a tier.
func tierParagraphs(doc string) []string {
	var out []string
	for _, p := range regexp.MustCompile(`\n\s*\n`).Split(doc, -1) {
		if tierName.MatchString(p) {
			out = append(out, p)
		}
	}
	return out
}

func tierDocs(t *testing.T) map[string]string {
	t.Helper()
	agents, err := os.ReadFile(filepath.Join("..", "..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := ReadTemplate("agents-spec-section.md")
	if err != nil {
		t.Fatal(err)
	}
	return map[string]string{"AGENTS.md": string(agents), "agents-spec-section.md": string(tmpl)}
}

func TestNoTierIsCalledALane(t *testing.T) {
	for name, doc := range tierDocs(t) {
		paragraphs := tierParagraphs(doc)
		if len(paragraphs) == 0 {
			t.Errorf("%s: no paragraph names a tier, so this test no longer reads the tiers", name)
		}
		for _, p := range paragraphs {
			if m := laneWord.FindString(p); m != "" {
				t.Errorf("%s calls a tier a %q; lanes are ADR-060's persistent worktrees, the tiers are tiers:\n%s", name, m, p)
			}
		}
	}
}

// The doctrine names the way in: in a repository that runs lanes, work starts
// with one of two commands.
func TestDoctrineNamesTheLaneEntrances(t *testing.T) {
	for name, doc := range tierDocs(t) {
		for _, entrance := range []string{"dotf lane switch", "dotf lane next"} {
			if !strings.Contains(doc, entrance) {
				t.Errorf("%s does not name the lane entrance `%s`", name, entrance)
			}
		}
	}
}
