package mem

import (
	"strings"
	"testing"
)

// A section written before threads existed: un-threaded text, then a thread.
const memoryWithLegacyBlock = `# Project Memory

## Session Handoff
> Updated: 2026-09-05  
**Last task:** shipped #410.  
**Next action:** merge #412.

### thread: master@msi

**Last task:** live work.
**Next action:** keep going.

## Findings

Moved to a topic file.
`

// THE LEGACY BLOCK HAS NO OWNER, SO NO WRITE EVER REPLACED IT (#1651).
//
// Every real thread is left byte-identical, which is right for threads. But
// text written before threads existed is not a thread: nobody's handoff-write
// will ever replace it, and it sits first under the heading, so the next
// session read a stale Next action ("merge #412", merged weeks earlier) as the
// current handoff. The command is the only way the file may change, so the
// command moves that text into a thread named by its own Updated date, at the
// end of the section, where it reads as history rather than as the handoff.
func TestWriteThreadMovesTheUnthreadedBlockIntoALegacyThread(t *testing.T) {
	out, changed, err := WriteThread(memoryWithLegacyBlock, "feat-x", "**Next action:** new work.")
	if err != nil || !changed {
		t.Fatalf("WriteThread: changed=%v err=%v", changed, err)
	}
	lines := strings.Split(out, "\n")
	start, end := handoffSection(lines)
	for i := start + 1; i < end; i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if _, ok := threadHeadingKey(lines[i]); !ok {
			t.Fatalf("the section still opens with un-threaded text %q:\n%s", lines[i], out)
		}
		break
	}
	legacy := strings.Index(out, "### thread: legacy-2026-09-05\n")
	if legacy < 0 {
		t.Fatalf("no legacy thread named by the block's Updated date:\n%s", out)
	}
	if !strings.Contains(out[legacy:], "**Next action:** merge #412.") {
		t.Errorf("the legacy text was lost instead of moved:\n%s", out)
	}
	for _, live := range []string{"### thread: master@msi", "### thread: feat-x"} {
		if i := strings.Index(out, live); i < 0 || i > legacy {
			t.Errorf("%q must precede the legacy thread:\n%s", live, out)
		}
	}
	if !strings.Contains(out, "### thread: master@msi\n\n**Last task:** live work.\n**Next action:** keep going.\n") {
		t.Errorf("a real thread changed:\n%s", out)
	}
	if !strings.Contains(out, "## Findings\n\nMoved to a topic file.\n") {
		t.Errorf("content after the section changed:\n%s", out)
	}
}

func TestWriteThreadMigratesEvenWhenItsOwnThreadIsUnchanged(t *testing.T) {
	out, changed, err := WriteThread(memoryWithLegacyBlock, "master@msi", "**Last task:** live work.\n**Next action:** keep going.")
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !strings.Contains(out, "### thread: legacy-2026-09-05") {
		t.Fatalf("an unchanged thread must not skip the migration: changed=%v\n%s", changed, out)
	}
	again, changed, err := WriteThread(out, "master@msi", "**Last task:** live work.\n**Next action:** keep going.")
	if err != nil || changed || again != out {
		t.Errorf("a second write must find nothing left to migrate: changed=%v err=%v", changed, err)
	}
}

func TestWriteThreadNamesAnUndatedLegacyBlockAndNeverCollides(t *testing.T) {
	undated := strings.Replace(memoryWithLegacyBlock, "> Updated: 2026-09-05  \n", "", 1)
	if key, ok := LegacyThreadKey(undated); !ok || key != "legacy-undated" {
		t.Errorf("an undated block: got %q, %v; want legacy-undated", key, ok)
	}
	taken := strings.Replace(memoryWithLegacyBlock, "### thread: master@msi", "### thread: legacy-2026-09-05\n\nmigrated earlier\n\n### thread: master@msi", 1)
	if key, ok := LegacyThreadKey(taken); !ok || key != "legacy-2026-09-05-2" {
		t.Errorf("a taken name: got %q, %v; want legacy-2026-09-05-2", key, ok)
	}
	if _, ok := LegacyThreadKey(memoryWithTwoThreads); ok {
		t.Error("a section with no un-threaded text reported a legacy block")
	}
}
