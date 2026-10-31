package mem

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update", false, "rewrite the resume golden files")

// AC1: resume is a pure render of the thread file. The same input renders the
// same bytes, and both formats are pinned by a golden file.
func TestResumeIsAPureRenderOfTheThreadFile(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("testdata", "resume", "thread.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"context", "prompt"} {
		first, err := RenderResume(string(text), "memory/threads/feat-x.md", format)
		if err != nil {
			t.Fatal(err)
		}
		second, _ := RenderResume(string(text), "memory/threads/feat-x.md", format)
		if first != second {
			t.Errorf("%s: two renders differ", format)
		}
		golden := filepath.Join("testdata", "resume", format+".golden")
		if *updateGolden {
			if err := os.WriteFile(golden, []byte(first), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if first != string(want) {
			t.Errorf("%s render differs from %s:\n%s", format, golden, first)
		}
	}
	if _, err := RenderResume(string(text), "x", "yaml"); err == nil {
		t.Error("an unknown format rendered")
	}
}

// AC2: Verify and Awaiting Manu both render. Every Verify command renders as
// written under the note, and none is marked or presented as safe to run.
func TestResumeRendersVerifyAndAwaiting(t *testing.T) {
	text, _ := os.ReadFile(filepath.Join("testdata", "resume", "thread.md"))
	for _, format := range []string{"context", "prompt"} {
		out, err := RenderResume(string(text), "f", format)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{
			"`gh pr view 2852 --json state`, `go test ./internal/mem` and `rm -rf ~/x`",
			"A or B on #2841",
			"whether lane-status-ff may be deleted",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: lacks %q:\n%s", format, want, out)
			}
		}
		if n := strings.Count(out, verifyNote); n != 1 {
			t.Errorf("%s: the note appears %d times, want 1:\n%s", format, n, out)
		}
	}
}

// AC2: a thread that carries a field under its canonical label and again under
// an alias renders both, in both formats (MEMORY-008 review round 6).
func TestResumeRendersEveryFieldOfALabel(t *testing.T) {
	text := "### thread: x\n\n**Next action:** n.<br>\n" +
		"**Verify:** `go test ./a`<br>\n**Verify at start:** `gh pr view 1`<br>\n" +
		"**Awaiting Manu:** - one<br>\n**Judgment calls left open:** - two<br>\n"
	for _, format := range []string{"context", "prompt"} {
		out, err := RenderResume(text, "f", format)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"`go test ./a`", "`gh pr view 1`", "- one", "- two"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: lacks %q:\n%s", format, want, out)
			}
		}
		if n := strings.Count(out, verifyNote); n != 1 {
			t.Errorf("%s: the note appears %d times, want 1:\n%s", format, n, out)
		}
	}
}

// AC2: a working directory that names no single thread for this agent lists
// the candidates and picks none.
func TestResumeListsCandidatesWhenTheThreadIsAmbiguous(t *testing.T) {
	doc := "# M\n\n## Session Handoff\n\n" +
		"### thread: master@msi (writer: pi)\n\n**Next action:** pi's.\n\n" +
		"### thread: feat-a (writer: claude)\n\n**Next action:** a.\n\n" +
		"### thread: feat-b\n\n**Next action:** b.\n\n" +
		"### thread: feat-j\n\n**Next action:** pi's step.<br>\nJournal: sessions/2026-10-20-dotfiles-pi-feat-j.md\n"
	for _, tc := range []struct {
		key, agent, want string
	}{
		{"feat-a", "claude", "feat-a"},
		{"feat-b", "claude", "feat-b"}, // unstamped: nothing says it is another's
		{"feat-b", "", "feat-b"},
		// Owned by its journal, as threadOwner reads it: not claude's (F-B).
		{"feat-j", "claude", ""},
		{"feat-j", "pi", "feat-j"},
		// An owned thread is not handed to a caller that names no agent (F-C).
		{"feat-a", "", ""},
		{"master@msi", "pi", "master@msi"},
		{"master@msi", "claude", ""}, // pi's, and claude has no fork of it
		{"feat-none", "claude", ""},
	} {
		got, candidates := SelectThread(doc, tc.key, tc.agent)
		if got != tc.want {
			t.Errorf("%s as %q: chose %q, want %q", tc.key, tc.agent, got, tc.want)
		}
		if tc.want == "" && strings.Join(candidates, ",") != "master@msi,feat-a,feat-b,feat-j" {
			t.Errorf("%s as %q: candidates %v, want every thread in order", tc.key, tc.agent, candidates)
		}
	}
	withFork := doc + "\n### thread: master@msi+claude (writer: claude)\n\n**Next action:** mine.\n"
	if got, _ := SelectThread(withFork, "master@msi", "claude"); got != "master@msi+claude" {
		t.Errorf("claude's fork was not chosen: %q", got)
	}
}

// ThreadText resolves a key to its full text: the stored file when the entry
// points at one, the inline block otherwise.
func TestThreadTextForKey(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "threads"), 0o755); err != nil {
		t.Fatal(err)
	}
	full := "### thread: feat-a\n\n**Decisions:** only in the file.\n"
	if err := os.WriteFile(filepath.Join(dir, "threads", "feat-a.md"), []byte(full), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := "## Session Handoff\n\n### thread: feat-a\n\n**Next action:** a.<br>\n[full thread](threads/feat-a.md)\n\n### thread: feat-b\n\n**Next action:** b.\n"
	memory := filepath.Join(dir, "memory2.md") // not MEMORY.md: the source is the file read (F-D)
	text, source := ThreadText(doc, "feat-a", memory)
	if !strings.Contains(text, "only in the file") || source != filepath.Join(dir, "threads", "feat-a.md") {
		t.Errorf("feat-a read %q from %q", text, source)
	}
	text, source = ThreadText(doc, "feat-b", memory)
	if !strings.Contains(text, "**Next action:** b.") || source != memory {
		t.Errorf("feat-b read %q from %q", text, source)
	}
}

// A one-item list renders under its label like a longer one, not glued to the
// label line (MEMORY-008 review, F-G).
func TestResumeRendersAOneItemListAsABlock(t *testing.T) {
	text := "### thread: feat-a\n\n**Next action:** x.<br>\n**Awaiting Manu:**\n- A or B on #1\n"
	out, err := RenderResume(text, "src", "context")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "**Awaiting Manu:**\n- A or B on #1") {
		t.Errorf("a one-item list is not a block:\n%s", out)
	}
}

// A Verify field renders as written, every line and the prose beside the
// commands, under one note. A command written as prose gets the same note as a
// backticked one: nothing in the field reads as safe (MEMORY-008 review
// rounds 2 to 4).
func TestResumeKeepsVerifyProseBesideTheCommands(t *testing.T) {
	for name, tc := range map[string]struct {
		field, want string
	}{
		"one line": {
			"do NOT run this on mba. `go test ./...` is fine, but `rm -rf /tmp/x` only after review.<br>",
			"do NOT run this on mba. `go test ./...` is fine, but `rm -rf /tmp/x` only after review.",
		},
		"two lines": {
			"`git status`<br>\nnot on mba, and see #1234 for the caveat<br>",
			"`git status`\nnot on mba, and see #1234 for the caveat",
		},
		"bullets": {
			"\n- `gh issue view 1689`\n- the queue is answered by hand\n- `dotf doctor --fix`",
			"- `gh issue view 1689`\n- the queue is answered by hand\n- `dotf doctor --fix`",
		},
		// Round 4, F-1: a command with no backticks.
		"prose command": {
			"run go test ./... then cleanup with rm -rf /tmp/x<br>",
			"run go test ./... then cleanup with rm -rf /tmp/x",
		},
	} {
		text := "### thread: feat-a\n\n**Next action:** x.<br>\n**Verify:** " + tc.field + "\n"
		for _, format := range []string{"context", "prompt"} {
			out, err := RenderResume(text, "src", format)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, verifyNote+"\n"+tc.want+"\n") {
				t.Errorf("%s, %s: want the note, then %q:\n%s", name, format, tc.want, out)
			}
			if strings.Contains(out, "(unverified)") {
				t.Errorf("%s, %s: a command is marked:\n%s", name, format, out)
			}
		}
	}
}

// A source is matched as a harness may spell it (MEMORY-008 review round 2).
func TestSessionStartResumeNormalisesTheSource(t *testing.T) {
	if got, want := normaliseSource(" Compact\n"), "compact"; got != want {
		t.Errorf("normaliseSource = %q, want %q", got, want)
	}
}
