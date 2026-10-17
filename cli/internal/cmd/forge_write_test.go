package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/forgewrite"
)

type commentRecorder struct {
	repo   forgewrite.Repo
	number int
	body   string
	newIss forgewrite.NewIssue
	edit   forgewrite.IssueEdit
	calls  []string
}

func (r *commentRecorder) CreateIssue(_ context.Context, repo forgewrite.Repo, in forgewrite.NewIssue) (string, error) {
	r.repo, r.newIss = repo, in
	r.calls = append(r.calls, "create")
	return "https://forge.example/i/9", nil
}

func (r *commentRecorder) EditIssue(_ context.Context, repo forgewrite.Repo, n int, in forgewrite.IssueEdit) (string, error) {
	r.repo, r.number, r.edit = repo, n, in
	r.calls = append(r.calls, "edit")
	return "", nil
}

func (r *commentRecorder) CommentIssue(_ context.Context, repo forgewrite.Repo, n int, body string) (string, error) {
	r.repo, r.number, r.body = repo, n, body
	return "https://forge.example/c/1", nil
}

func stubForgeWrite(t *testing.T, origin string) *commentRecorder {
	t.Helper()
	rec := &commentRecorder{}
	oldO, oldB := forgeWriteOrigin, forgeWriteBackend
	forgeWriteOrigin = func() (string, error) { return origin, nil }
	forgeWriteBackend = func(forgewrite.Repo) (forgewrite.Backend, error) { return rec, nil }
	t.Cleanup(func() { forgeWriteOrigin, forgeWriteBackend = oldO, oldB })
	return rec
}

func runForge(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newForgeCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func runForgeIssue(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runForge(t, append([]string{"issue"}, args...)...)
}

func writeBody(t *testing.T, text string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "b.md")
	if err := os.WriteFile(f, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestForgeIssueCommentReadsBodyFromFile(t *testing.T) {
	rec := stubForgeWrite(t, "git@gitea.kubelab.live:personal/imagesensortool.git")
	f := filepath.Join(t.TempDir(), "b.md")
	if err := os.WriteFile(f, []byte("measurement comment\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runForgeIssue(t, "comment", "12", "--body-file", f)
	if err != nil {
		t.Fatal(err)
	}
	if rec.number != 12 || rec.repo.Owner != "personal" || rec.repo.Name != "imagesensortool" || rec.body != "measurement comment\n" {
		t.Errorf("backend got %+v", rec)
	}
	if !strings.Contains(out, "https://forge.example/c/1") {
		t.Errorf("the comment URL was not printed: %q", out)
	}
}

func TestForgeIssueCommentReadsStdin(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	old := forgeWriteStdin
	forgeWriteStdin = strings.NewReader("from stdin")
	t.Cleanup(func() { forgeWriteStdin = old })
	if _, err := runForgeIssue(t, "comment", "3", "--body-file", "-"); err != nil || rec.body != "from stdin" {
		t.Errorf("err=%v body=%q", err, rec.body)
	}
}

// The body is never an argument: there is no flag that takes the text.
func TestBodyIsNeverAnArgument(t *testing.T) {
	stubForgeWrite(t, "https://github.com/o/n.git")
	if _, err := runForgeIssue(t, "comment", "3", "--body", "inline prose"); err == nil {
		t.Error("an inline --body must not exist")
	}
	_, err := runForgeIssue(t, "comment", "3")
	if err == nil || !strings.Contains(err.Error(), "FIX:") {
		t.Errorf("a missing body must say how to fix it, got %v", err)
	}
	empty := filepath.Join(t.TempDir(), "e.md")
	if err := os.WriteFile(empty, []byte("  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runForgeIssue(t, "comment", "3", "--body-file", empty); err == nil {
		t.Error("an empty body must be refused")
	}
}

func TestForgeIssueCommentRefusesABadNumber(t *testing.T) {
	stubForgeWrite(t, "https://github.com/o/n.git")
	for _, n := range []string{"zero", "0", "-4"} {
		if _, err := runForgeIssue(t, "comment", n, "--body-file", "x"); err == nil {
			t.Errorf("%q should be refused", n)
		}
	}
}

func TestForgeIssueCreateSendsTitleBodyAndLabels(t *testing.T) {
	rec := stubForgeWrite(t, "git@gitea.kubelab.live:personal/imagesensortool.git")
	out, err := runForgeIssue(t, "create", "--title", "BUG-1: it breaks", "--body-file", writeBody(t, "details\n"), "--label", "bug", "--label", "ci")
	if err != nil {
		t.Fatal(err)
	}
	want := forgewrite.NewIssue{Title: "BUG-1: it breaks", Body: "details\n", Labels: []string{"bug", "ci"}}
	if rec.newIss.Title != want.Title || rec.newIss.Body != want.Body || strings.Join(rec.newIss.Labels, ",") != "bug,ci" {
		t.Errorf("backend got %+v", rec.newIss)
	}
	if !strings.Contains(out, "https://forge.example/i/9") {
		t.Errorf("the issue URL was not printed: %q", out)
	}
}

func TestForgeIssueCreateNeedsATitleAndABody(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	if _, err := runForgeIssue(t, "create", "--body-file", writeBody(t, "x")); err == nil {
		t.Error("a missing title must be refused")
	}
	if _, err := runForgeIssue(t, "create", "--title", "t"); err == nil {
		t.Error("a missing body must be refused")
	}
	if _, err := runForgeIssue(t, "create", "--title", "t", "--body", "inline"); err == nil {
		t.Error("an inline --body must not exist")
	}
	if len(rec.calls) != 0 {
		t.Errorf("a refused create reached the backend: %v", rec.calls)
	}
}

func TestForgeIssueEditChangesOnlyWhatIsNamed(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	out, err := runForgeIssue(t, "edit", "5", "--title", "new title")
	if err != nil {
		t.Fatal(err)
	}
	if rec.number != 5 || rec.edit.Title == nil || *rec.edit.Title != "new title" || rec.edit.Body != nil {
		t.Errorf("backend got %+v", rec.edit)
	}
	if !strings.Contains(out, "no URL") {
		t.Errorf("an empty URL must be stated, got %q", out)
	}
	if _, err := runForgeIssue(t, "edit", "5", "--body-file", writeBody(t, "b")); err != nil || rec.edit.Body == nil || *rec.edit.Body != "b" || rec.edit.Title != nil {
		t.Errorf("body-only edit: err=%v edit=%+v", err, rec.edit)
	}
}

func TestForgeIssueEditNeedsSomethingToChange(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	if _, err := runForgeIssue(t, "edit", "5"); err == nil {
		t.Error("an edit with no field must be refused")
	}
	if _, err := runForgeIssue(t, "edit", "5", "--title", " "); err == nil {
		t.Error("an empty title must be refused")
	}
	if len(rec.calls) != 0 {
		t.Errorf("a refused edit reached the backend: %v", rec.calls)
	}
}

func TestForgePRCommentUsesTheCommentEndpoint(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	if _, err := runForge(t, "pr", "comment", "8", "--body-file", writeBody(t, "triage")); err != nil {
		t.Fatal(err)
	}
	if rec.number != 8 || rec.body != "triage" {
		t.Errorf("backend got %+v", rec)
	}
}

// The command tree has no merge or close verb, at any depth.
func TestForgeHasNoMergeOrCloseCommand(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, banned := range []string{"merge", "close", "delete", "reopen"} {
			if strings.Contains(c.Name(), banned) {
				t.Errorf("dotf forge has a %q command", c.CommandPath())
			}
		}
		for _, s := range c.Commands() {
			walk(s)
		}
	}
	walk(newForgeCmd())
}
