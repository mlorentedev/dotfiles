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
		if strings.Contains(out, "`gh pr view 2852 --json state` (unverified") {
			t.Errorf("%s: a read-only command was marked unverified", format)
		}
	}
}

func TestReadOnlyCommand(t *testing.T) {
	for cmd, want := range map[string]bool{
		"git status":                    true,
		"gh pr view 2852 --json state":  true,
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
		"cat ~/.config/secrets.env":     false,
		"grep -r token ~":               false,
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
		"### thread: feat-b\n\n**Next action:** b.\n"
	for _, tc := range []struct {
		key, agent, want string
	}{
		{"feat-a", "claude", "feat-a"},
		{"feat-b", "claude", "feat-b"}, // unstamped: nothing says it is another's
		{"feat-b", "", "feat-b"},
		{"master@msi", "pi", "master@msi"},
		{"master@msi", "claude", ""}, // pi's, and claude has no fork of it
		{"feat-none", "claude", ""},
	} {
		got, candidates := SelectThread(doc, tc.key, tc.agent)
		if got != tc.want {
			t.Errorf("%s as %q: chose %q, want %q", tc.key, tc.agent, got, tc.want)
		}
		if tc.want == "" && strings.Join(candidates, ",") != "master@msi,feat-a,feat-b" {
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
	text, source := ThreadText(doc, "feat-a", dir)
	if !strings.Contains(text, "only in the file") || source != filepath.Join(dir, "threads", "feat-a.md") {
		t.Errorf("feat-a read %q from %q", text, source)
	}
	text, source = ThreadText(doc, "feat-b", dir)
	if !strings.Contains(text, "**Next action:** b.") || source != filepath.Join(dir, "MEMORY.md") {
		t.Errorf("feat-b read %q from %q", text, source)
	}
}
