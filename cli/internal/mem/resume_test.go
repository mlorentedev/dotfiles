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

// AC2: Verify and Awaiting Manu both render, and a Verify command outside the
// read-only allowlist is marked, never presented as safe to run.
func TestResumeRendersVerifyAndAwaiting(t *testing.T) {
	text, _ := os.ReadFile(filepath.Join("testdata", "resume", "thread.md"))
	for _, format := range []string{"context", "prompt"} {
		out, err := RenderResume(string(text), "f", format)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{
			"`gh pr view 2852 --json state`",
			"`go test ./internal/mem`",
			"`rm -rf ~/x` (unverified",
			"A or B on #2841",
			"whether lane-status-ff may be deleted",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: lacks %q:\n%s", format, want, out)
			}
		}
		if strings.Contains(out, "`go test ./internal/mem` (unverified") {
			t.Errorf("%s: a read-only command was marked unverified", format)
		}
	}
}

func TestReadOnlyCommand(t *testing.T) {
	for cmd, want := range map[string]bool{
		"git status":                    true,
		"gh pr view 2852 --json state":  false,
		"dotf spec audit":               true,
		"go test ./internal/mem":        true,
		"rm -rf ~/x":                    false,
		"git status; rm -rf ~":          false,
		"gh pr view 1 && gh pr merge 1": false,
		"cat x | sh":                    false,
		"ls $(rm x)":                    false,
		"git statusx":                   false,
		"gh pr merge 1":                 false,
		"cat secrets > out":             false,
		"git status -s\nrm -rf ~":       false,
		"git status \r\nrm -rf ~":       false,
		"git branch -D main":            false,
		"git fetch":                     false,
		"git diff --output=file":        false,
		"git diff --output file":        false,
		"go test -o bin ./x":            false,
		"go build ./...":                false,
		"git diff main -- x.go":         false,
		"cat ~/.config/secrets.env":     false,
		"grep -r token ~":               false,
		// The code runners take only flags that neither name a program nor an
		// output, and paths inside the checkout (MEMORY-008 review, F-A).
		"go test -exec=/tmp/evil-shim ./...":             false,
		"go test -exec /tmp/shim ./...":                  false,
		"go test --outputdir=/tmp/x ./...":               false,
		"go test -cpuprofile cpu.out ./x":                false,
		"go test -toolexec=x ./...":                      false,
		"go vet -vettool=/tmp/x ./...":                   false,
		"bats /tmp/evil.bats":                            false,
		"bats ../outside.bats":                           false,
		"bats --formatter /tmp/f tests/x.bats":           false,
		"shellcheck ~/x.sh":                              false,
		"go test ./... -run '^TestX$' -count=1 -v":       false,
		"go test ./internal/mem ./internal/cmd -count=1": false,
		"go vet ./...":                                   true,
		"bats tests/x.bats --filter 'under 8000'":        false,
		"shellcheck scripts/a.sh":                        true,
		// Round 3: only flagless plain commands are read-only. A flag, a quote,
		// a variable or a colon makes it unverified, however harmless the one
		// in hand (the -count, --json, --filter and -- rows above flipped).
		"go test ${IFS}-exec=/tmp/evil ./...": false,
		"go test '-exec=/tmp/x' ./...":        false,
		"dotf doctor --fix":                   false,
		"gh pr view 1 --web":                  false,
		"git show HEAD:secrets/x":             false,
		"git status\tx":                       false,
		"gh issue view 1689":                  true,
		"gh pr view 2852":                     true,
		"go test ./...":                       true,
		"bats tests/x.bats":                   true,
		"git log origin/main":                 true,
	} {
		if got := readOnlyCommand(cmd); got != want {
			t.Errorf("readOnlyCommand(%q) = %v, want %v", cmd, got, want)
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

// A Verify field's prose is part of the record: a caveat written beside the
// commands survives the render in both formats (MEMORY-008 review round 2).
func TestResumeKeepsVerifyProseBesideTheCommands(t *testing.T) {
	for name, tc := range map[string]struct {
		field string
		want  []string
	}{
		"one line": {
			"do NOT run this on mba. `go test ./...` is fine, but `rm -rf /tmp/x` only after review.<br>",
			[]string{"do NOT run this on mba. `go test ./...` is fine, but `rm -rf /tmp/x` (unverified) only after review."},
		},
		// Round 3: every line of the field, not its first.
		"two lines": {
			"`git status`<br>\nnot on mba, and see #1234 for the caveat<br>",
			[]string{"`git status`\nnot on mba, and see #1234 for the caveat"},
		},
		"bullets": {
			"\n- `gh issue view 1689`\n- the queue is answered by hand\n- `dotf doctor --fix`",
			[]string{"- `gh issue view 1689`\n- the queue is answered by hand\n- `dotf doctor --fix` (unverified)"},
		},
	} {
		text := "### thread: feat-a\n\n**Next action:** x.<br>\n**Verify:** " + tc.field + "\n"
		for _, format := range []string{"context", "prompt"} {
			out, err := RenderResume(text, "src", format)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("%s, %s: lacks %q:\n%s", name, format, want, out)
				}
			}
			if strings.Contains(out, "(unverified)") != strings.Contains(out, unverifiedNote) {
				t.Errorf("%s, %s: the note and the markers disagree:\n%s", name, format, out)
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
