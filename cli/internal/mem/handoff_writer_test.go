package mem

import (
	"strings"
	"testing"
)

// The block the live dry run of 2026-09-24 would have removed: antigravity's
// handoff on knowledge's master@msi, unstamped, attributed only by its Journal
// line (#1690).
const memoryWithAnAntigravityThread = `# knowledge — auto-memory

## Session Handoff

### thread: master@msi

> Updated: 2026-09-22
**Last task:** Researched prior-art and mapped it to five repositories.
**Next action:** Run a parallel session in ~/Projects/web.
Journal: sessions/2026-09-22-knowledge-antigravity-master@msi.md
`

const memoryStampedByPi = `# ts-bridge

## Session Handoff

### thread: master@msi (writer: pi)

**Last task:** pi's work.
**Next action:** pi's next step.
`

// AC4: without an agent nothing changes, so every caller that does not pass the
// flag keeps today's output exactly, legacy migration included.
func TestAWriteWithNoAgentIsUnchanged(t *testing.T) {
	legacy := "# M\n\n## Session Handoff\n> Updated: 2026-09-05  \n**Next action:** merge #412.\n\n### thread: master@msi\n\nlive\n"
	docs := map[string]string{
		"two threads":      memoryWithTwoThreads,
		"a journal line":   memoryWithAnAntigravityThread,
		"a stamped thread": memoryStampedByPi,
		"a legacy block":   legacy,
	}
	for name, doc := range docs {
		for _, key := range []string{"wt-pi-harness", "master@msi", "feat-x"} {
			want, wantChanged, wantErr := WriteThread(doc, key, "**Next action:** y.")
			got, err := WriteThreadAs(doc, key, "", "**Next action:** y.")
			if (err != nil) != (wantErr != nil) {
				t.Fatalf("%s, %s: error %v, WriteThread's %v", name, key, err, wantErr)
			}
			if got.Content != want || got.Changed != wantChanged {
				t.Errorf("%s, %s: without an agent the document differs from WriteThread's", name, key)
			}
			if got.Key != key || got.Kept != "" {
				t.Errorf("%s, %s: without an agent the write went to %q (kept %q)", name, key, got.Key, got.Kept)
			}
		}
	}
}

// AC3: the stamp names the writer, and the same writer replaces its own block
// rather than forking from itself.
func TestTheSameAgentRewritesItsOwnBlockInPlace(t *testing.T) {
	first, err := WriteThreadAs(memoryWithTwoThreads, "master@msi", "claude", "**Next action:** one.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.Content, "### thread: master@msi (writer: claude)\n") {
		t.Fatalf("the heading carries no writer stamp:\n%s", first.Content)
	}
	second, err := WriteThreadAs(first.Content, "master@msi", "claude", "**Next action:** two.")
	if err != nil {
		t.Fatal(err)
	}
	if second.Key != "master@msi" || second.Kept != "" {
		t.Errorf("claude's rewrite of its own block went to %q (kept %q)", second.Key, second.Kept)
	}
	if strings.Contains(second.Content, "one.") || !strings.Contains(second.Content, "two.") {
		t.Errorf("the block was not replaced in place:\n%s", second.Content)
	}
	if n := strings.Count(second.Content, "### thread: master@msi"); n != 1 {
		t.Errorf("%d threads start with master@msi, want 1", n)
	}
	again, err := WriteThreadAs(second.Content, "master@msi", "claude", "**Next action:** two.")
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || again.Content != second.Content {
		t.Error("an identical stamped rewrite changed the document")
	}

	// An unstamped block whose journal is the writer's own is the writer's too.
	own := strings.Replace(memoryWithAnAntigravityThread, "antigravity", "claude", 1)
	res, err := WriteThreadAs(own, "master@msi", "claude", "**Next action:** mine.")
	if err != nil {
		t.Fatal(err)
	}
	if res.Key != "master@msi" || strings.Contains(res.Content, "prior-art") {
		t.Errorf("a block whose journal names claude was not replaced by claude (went to %q)", res.Key)
	}
}

// AC1: the regression. pi's stamped block survives a write by claude under the
// same key, byte for byte, and claude's handoff lands in its fork.
func TestAnotherAgentsStampedBlockIsForkedNotReplaced(t *testing.T) {
	res, err := WriteThreadAs(memoryStampedByPi, "master@msi", "claude", "**Next action:** claude's step.")
	if err != nil {
		t.Fatal(err)
	}
	if res.Key != "master@msi+claude" || res.Kept != "pi" {
		t.Errorf("the write went to %q keeping %q, want master@msi+claude keeping pi", res.Key, res.Kept)
	}
	pi := "### thread: master@msi (writer: pi)\n\n**Last task:** pi's work.\n**Next action:** pi's next step.\n"
	if !strings.Contains(res.Content, pi) {
		t.Errorf("pi's block was not kept byte for byte:\n%s", res.Content)
	}
	if !strings.Contains(res.Content, "### thread: master@msi+claude (writer: claude)\n\n**Next action:** claude's step.\n") {
		t.Errorf("claude's handoff is not in its stamped fork:\n%s", res.Content)
	}

	// Writing again replaces the fork, not pi's block, and forks no further.
	again, err := WriteThreadAs(res.Content, "master@msi", "claude", "**Next action:** claude's second step.")
	if err != nil {
		t.Fatal(err)
	}
	if again.Key != "master@msi+claude" || !strings.Contains(again.Content, pi) {
		t.Errorf("the second write went to %q or lost pi's block", again.Key)
	}
	if n := strings.Count(again.Content, "### thread: master@msi+claude"); n != 1 || strings.Contains(again.Content, "claude's step.") {
		t.Errorf("the fork was not replaced in place (%d fork headings):\n%s", n, again.Content)
	}
}

// AC1: blocks written before the stamp existed carry their writer in the journal
// they point at, <date>-<project>-<agent>[-<thread>].md.
func TestAnUnstampedBlockIsAttributedByItsJournalLine(t *testing.T) {
	res, err := WriteThreadAs(memoryWithAnAntigravityThread, "master@msi", "claude", "**Next action:** claude's step.")
	if err != nil {
		t.Fatal(err)
	}
	if res.Key != "master@msi+claude" || res.Kept != "antigravity" {
		t.Errorf("the write went to %q keeping %q, want master@msi+claude keeping antigravity", res.Key, res.Kept)
	}
	if !strings.Contains(res.Content, "Researched prior-art") || !strings.Contains(res.Content, "### thread: master@msi\n") {
		t.Errorf("antigravity's block was not kept:\n%s", res.Content)
	}

	// The journal may sit in another project's tree and carry a thread of its own.
	for journal, writer := range map[string]string{
		"Journal: 10_projects/dotfiles/sessions/2026-09-24-dotfiles-pi-fix-harness-remediation.md": "pi",
		"Journal: sessions/2026-09-24-fae-onboarding-webapp-copilot-main@egw-len029.md":            "copilot",
		"Journal: [record](sessions/2026-08-14-knowledge-agy.md)":                                  "agy",
	} {
		doc := strings.Replace(memoryWithAnAntigravityThread,
			"Journal: sessions/2026-09-22-knowledge-antigravity-master@msi.md", journal, 1)
		res, err := WriteThreadAs(doc, "master@msi", "claude", "x")
		if err != nil {
			t.Fatal(err)
		}
		if res.Kept != writer {
			t.Errorf("%q: attributed to %q, want %q", journal, res.Kept, writer)
		}
	}
}

// AC3: with no stamp and no journal naming an agent, nothing says the block is
// someone else's, so it is replaced as it always was, and stamped.
func TestABlockWithNoKnownWriterIsReplacedAndStamped(t *testing.T) {
	for name, doc := range map[string]string{
		"no journal line": strings.Replace(memoryWithAnAntigravityThread,
			"Journal: sessions/2026-09-22-knowledge-antigravity-master@msi.md\n", "", 1),
		"a journal naming no agent": strings.Replace(memoryWithAnAntigravityThread,
			"knowledge-antigravity-master@msi", "knowledge-master@msi", 1),
	} {
		res, err := WriteThreadAs(doc, "master@msi", "claude", "**Next action:** claude's step.")
		if err != nil {
			t.Fatal(err)
		}
		if res.Key != "master@msi" || res.Kept != "" {
			t.Errorf("%s: the write went to %q keeping %q", name, res.Key, res.Kept)
		}
		if strings.Contains(res.Content, "prior-art") || !strings.Contains(res.Content, "### thread: master@msi (writer: claude)\n") {
			t.Errorf("%s: the block was not replaced and stamped:\n%s", name, res.Content)
		}
	}
}

// A writer stamp is parsed back out of the heading, so it has to be one word.
func TestWriteThreadAsRejectsAWriterTheStampCannotCarry(t *testing.T) {
	for _, agent := range []string{"two words", "claude)", "Claude", "pi\n"} {
		if _, err := WriteThreadAs(memoryStampedByPi, "master@msi", agent, "x"); err == nil {
			t.Errorf("agent %q was accepted", agent)
		}
	}
}

// The legacy block still moves when the writer is named (#1651).
func TestAnAgentWriteStillMovesTheLegacyBlock(t *testing.T) {
	doc := "# M\n\n## Session Handoff\n> Updated: 2026-09-05  \n**Next action:** merge #412.\n"
	res, err := WriteThreadAs(doc, "feat-x", "claude", "**Next action:** new work.")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"### thread: feat-x (writer: claude)", "### thread: legacy-2026-09-05", "merge #412."} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("missing %q:\n%s", want, res.Content)
		}
	}
}

// The two moves of a write can meet: the un-threaded block is migrated, and the
// key holds another agent's block, so the write forks. Both have to happen,
// with the other agent's block untouched (MEMORY-009 closing review).
func TestAnAgentWriteMigratesTheLegacyBlockAndForksFromAnotherAgent(t *testing.T) {
	doc := "# M\n\n## Session Handoff\n> Updated: 2026-09-05  \n**Next action:** merge #412.\n\n" +
		"### thread: master@msi (writer: pi)\n\n**Next action:** pi's step.\n"
	res, err := WriteThreadAs(doc, "master@msi", "claude", "**Next action:** claude's step.")
	if err != nil {
		t.Fatal(err)
	}
	if res.Key != "master@msi+claude" || res.Kept != "pi" {
		t.Errorf("the write went to %q keeping %q, want master@msi+claude keeping pi", res.Key, res.Kept)
	}
	for _, want := range []string{
		"### thread: master@msi (writer: pi)\n\n**Next action:** pi's step.\n",
		"### thread: master@msi+claude (writer: claude)",
		"### thread: legacy-2026-09-05",
		"merge #412.",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("missing %q:\n%s", want, res.Content)
		}
	}
}
