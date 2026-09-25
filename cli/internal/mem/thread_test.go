package mem

import (
	"strings"
	"testing"
)

// A thread as sessions write it today: blockquote date, bold labels with the
// markdown hard break, a plain Journal line.
const canonicalThread = "### thread: master@msi\n\n" +
	"> Updated: 2026-09-22  \n" +
	"**Last task:** Researched prior-art.  \n" +
	"**Decisions:** (1) Separate 30_career.  \n" +
	"**Open threads:** (1) web: BRAND-002.  \n" +
	"**Next action:** Branch RES-065 PR 2.  \n" +
	"**Verify:** `gh pr view 1705 --json state`  \n" +
	"**Awaiting Manu:** the merge order.  \n" +
	"Journal: sessions/2026-09-22-knowledge-antigravity-master@msi.md\n"

// AC2: every canonical label is read into its field, the hard break trimmed.
func TestParseThreadReadsEveryCanonicalLabel(t *testing.T) {
	th := ParseThread(canonicalThread)
	if th.Key != "master@msi" {
		t.Errorf("key = %q, want master@msi", th.Key)
	}
	want := map[string]string{
		"Updated":       "2026-09-22",
		"Last task":     "Researched prior-art.",
		"Decisions":     "(1) Separate 30_career.",
		"Open threads":  "(1) web: BRAND-002.",
		"Next action":   "Branch RES-065 PR 2.",
		"Verify":        "`gh pr view 1705 --json state`",
		"Awaiting Manu": "the merge order.",
		"Journal":       "sessions/2026-09-22-knowledge-antigravity-master@msi.md",
	}
	if len(want) != len(ThreadLabels) {
		t.Fatalf("the test covers %d labels, the canonical set has %d", len(want), len(ThreadLabels))
	}
	for _, label := range ThreadLabels {
		got, ok := th.Get(label)
		if !ok || got != want[label] {
			t.Errorf("%s = %q (present %v), want %q", label, got, ok, want[label])
		}
	}
	if u := th.Unknown(); len(u) != 0 {
		t.Errorf("canonical labels read as unknown: %+v", u)
	}
}

// AC2: the labels sessions improvised before the set existed are read as the
// fields they meant, so no existing block needs migrating by hand.
func TestParseThreadReadsTheImprovisedLabelsAsTheirFields(t *testing.T) {
	th := ParseThread("**Verify at start:** `git log -1`\n**Judgment calls left open:** keep the alias or drop it.\n")
	if v, _ := th.Get("Verify"); v != "`git log -1`" {
		t.Errorf("Verify = %q", v)
	}
	if v, _ := th.Get("Awaiting Manu"); v != "keep the alias or drop it." {
		t.Errorf("Awaiting Manu = %q", v)
	}
}

// AC2: a label outside the set is kept as written, in order, never dropped;
// a qualified canonical label is one of them, because the qualifier is content.
func TestParseThreadKeepsAnUnknownLabel(t *testing.T) {
	th := ParseThread("**Last task:** x.\n**Trap:** the vault auto-commits.\n**Decisions (Manu, 2026-09-24):** y.\n**Next action:** z.\n")
	u := th.Unknown()
	if len(u) != 2 || u[0] != (Field{"Trap", "the vault auto-commits."}) || u[1] != (Field{"Decisions (Manu, 2026-09-24)", "y."}) {
		t.Errorf("unknown fields = %+v", u)
	}
	if _, ok := th.Get("Decisions"); ok {
		t.Error("a qualified label was read as the canonical one, losing its qualifier")
	}
}

// No line of a body is lost: text before the first label, and lines after a
// label up to the next one, stay with the field they follow.
func TestParseThreadDropsNoLine(t *testing.T) {
	block := "### thread: wt-cli-023 (feat/cli-050-crystallize-cutover)\n\n" +
		"Written before the labels settled.\n\n" +
		"**Open threads:**\n- (1) #1705 release\n- (2) #1709 digest\n\n" +
		"### Next Actions\n\n" +
		"**Next action:** merge #1705.\n"
	th := ParseThread(block)
	if th.Key != "wt-cli-023" {
		t.Errorf("key = %q, want wt-cli-023", th.Key)
	}
	if len(th.Fields) == 0 || th.Fields[0] != (Field{"", "Written before the labels settled."}) {
		t.Errorf("the text before the first label was not kept: %+v", th.Fields)
	}
	open, _ := th.Get("Open threads")
	if open != "- (1) #1705 release\n- (2) #1709 digest\n\n### Next Actions" {
		t.Errorf("Open threads = %q", open)
	}
	if next, _ := th.Get("Next action"); next != "merge #1705." {
		t.Errorf("Next action = %q", next)
	}
}

// A bold Updated and a blockquoted Journal are the same fields.
func TestParseThreadReadsEitherFormOfTheDateAndJournal(t *testing.T) {
	th := ParseThread("**Updated:** 2026-09-05\n> Journal: sessions/x.md\n")
	if v, _ := th.Get("Updated"); v != "2026-09-05" {
		t.Errorf("Updated = %q", v)
	}
	if v, _ := th.Get("Journal"); v != "sessions/x.md" {
		t.Errorf("Journal = %q", v)
	}
	if strings.Contains(ParseThread("Note: prose, not a field.\n").Fields[0].Label, "Note") {
		t.Error("a plain word and a colon read as a label; only Updated and Journal are plain")
	}
}

// A body in the canonical shape draws no warning; an empty Next action is as
// good as a missing one, because the next session has no first step either way.
func TestThreadWarningsNameOnlyWhatIsMissingOrUnknown(t *testing.T) {
	if w := ThreadWarnings(canonicalThread); len(w) != 0 {
		t.Errorf("a canonical thread drew warnings: %q", w)
	}
	w := ThreadWarnings("**Last task:** x.\n**Next action:**\n**Trap:** y.\n")
	if len(w) != 2 || !strings.Contains(w[0], "Next action") || !strings.Contains(w[1], `"Trap"`) {
		t.Errorf("warnings = %q, want the empty Next action and the unknown Trap", w)
	}
}
