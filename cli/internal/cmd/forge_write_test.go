package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/forgewrite"
)

type commentRecorder struct {
	repo   forgewrite.Repo
	number int
	body   string
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

func runForgeIssue(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newForgeCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"issue"}, args...))
	err := root.Execute()
	return out.String(), err
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

// The command tree has no merge or close verb.
func TestForgeHasNoMergeOrCloseCommand(t *testing.T) {
	var names []string
	for _, c := range newForgeCmd().Commands() {
		names = append(names, c.Name())
		for _, s := range c.Commands() {
			names = append(names, s.Name())
		}
	}
	for _, n := range names {
		for _, banned := range []string{"merge", "close", "delete", "reopen"} {
			if strings.Contains(n, banned) {
				t.Errorf("dotf forge has a %q command", n)
			}
		}
	}
}
