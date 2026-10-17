package cmd

import (
	"bytes"
	"context"
	"errors"
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
	newPR  forgewrite.NewPR
	edit   forgewrite.IssueEdit
	calls  []string
}

func (r *commentRecorder) CreateIssue(_ context.Context, repo forgewrite.Repo, in forgewrite.NewIssue) (string, error) {
	r.repo, r.newIss = repo, in
	r.calls = append(r.calls, "create")
	return "https://forge.example/i/9", nil
}

func (r *commentRecorder) CreatePR(_ context.Context, repo forgewrite.Repo, in forgewrite.NewPR) (string, error) {
	r.repo, r.newPR = repo, in
	r.calls = append(r.calls, "createpr")
	return "https://forge.example/p/3", nil
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

func TestForgePRCreateSendsHeadBaseAndBody(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	old := forgeWriteBranch
	forgeWriteBranch = func() (string, error) { return "feat/x", nil }
	t.Cleanup(func() { forgeWriteBranch = old })
	out, err := runForge(t, "pr", "create", "--title", "feat: x", "--base", "main", "--draft", "--body-file", writeBody(t, "why\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := forgewrite.NewPR{Title: "feat: x", Body: "why\n", Head: "feat/x", Base: "main", Draft: true}
	if rec.newPR != want {
		t.Errorf("backend got %+v, want %+v", rec.newPR, want)
	}
	if !strings.Contains(out, "https://forge.example/p/3") {
		t.Errorf("the pull request URL was not printed: %q", out)
	}
	if _, err := runForge(t, "pr", "create", "--title", "t", "--base", "main", "--head", "other", "--body-file", writeBody(t, "b")); err != nil || rec.newPR.Head != "other" {
		t.Errorf("an explicit --head must win: err=%v head=%q", err, rec.newPR.Head)
	}
}

func TestForgePRCreateNeverGuessesTheBase(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	for name, args := range map[string][]string{
		"no base":  {"--title", "t", "--body-file", writeBody(t, "b")},
		"no title": {"--base", "main", "--body-file", writeBody(t, "b")},
		"no body":  {"--title", "t", "--base", "main"},
		"inline":   {"--title", "t", "--base", "main", "--body", "prose"},
	} {
		if _, err := runForge(t, append([]string{"pr", "create"}, args...)...); err == nil {
			t.Errorf("%s: want a refusal", name)
		}
	}
	if len(rec.calls) != 0 {
		t.Errorf("a refused create reached the backend: %v", rec.calls)
	}
}

// Cross-fork heads are out of scope, so the scope line is a guard, not a hope.
func TestForgePRCreateRefusesACrossForkHead(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	_, err := runForge(t, "pr", "create", "--title", "t", "--base", "main", "--head", "fork:branch", "--body-file", writeBody(t, "b"))
	if err == nil || !strings.Contains(err.Error(), "FIX:") || len(rec.calls) != 0 {
		t.Errorf("err=%v calls=%v", err, rec.calls)
	}
}

func TestForgePREditIsTheIssueEdit(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	if _, err := runForge(t, "pr", "edit", "6", "--title", "retitled"); err != nil {
		t.Fatal(err)
	}
	if rec.number != 6 || rec.edit.Title == nil || *rec.edit.Title != "retitled" || strings.Join(rec.calls, ",") != "edit" {
		t.Errorf("backend got number=%d edit=%+v calls=%v", rec.number, rec.edit, rec.calls)
	}
}

// boardRecorder is a backend that can also place a card on a board.
type boardRecorder struct {
	*commentRecorder
	resolved  forgewrite.BoardFields
	placedURL string
	placeErr  error
}

func (b *boardRecorder) ResolveBoard(_ context.Context, board forgewrite.Board, f forgewrite.BoardFields) (forgewrite.BoardPlan, error) {
	b.resolved = f
	b.calls = append(b.calls, "resolve")
	return forgewrite.BoardPlan{Board: board, ProjectID: "P"}, nil
}

func (b *boardRecorder) PlaceOnBoard(_ context.Context, _ forgewrite.BoardPlan, url string) error {
	b.placedURL = url
	b.calls = append(b.calls, "place")
	return b.placeErr
}

func stubBoards(t *testing.T, boards map[string]forgewrite.Board) {
	t.Helper()
	old := forgeWriteBoards
	forgeWriteBoards = func() (map[string]forgewrite.Board, error) { return boards, nil }
	t.Cleanup(func() { forgeWriteBoards = old })
}

func useBackend(t *testing.T, b forgewrite.Backend) {
	t.Helper()
	old := forgeWriteBackend
	forgeWriteBackend = func(forgewrite.Repo) (forgewrite.Backend, error) { return b, nil }
	t.Cleanup(func() { forgeWriteBackend = old })
}

func createArgs(t *testing.T, extra ...string) []string {
	return append([]string{"create", "--title", "TOOL-1: x", "--body-file", writeBody(t, "b")}, extra...)
}

func TestForgeIssueCreateBoardResolvesThenCreatesThenPlaces(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/mlorentedev/dotfiles.git")
	br := &boardRecorder{commentRecorder: rec}
	useBackend(t, br)
	stubBoards(t, map[string]forgewrite.Board{"github.com": {Owner: "mlorentedev", Number: 1}})
	if _, err := runForgeIssue(t, createArgs(t, "--board", "--type", "chore", "--id", "TOOL-1-x")...); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(rec.calls, ","); got != "resolve,create,place" {
		t.Errorf("order = %s, want resolve before create, place after", got)
	}
	want := forgewrite.BoardFields{Status: "Backlog", Priority: "P2", Type: "chore", ID: "TOOL-1-x"}
	if br.resolved != want || br.placedURL != "https://forge.example/i/9" {
		t.Errorf("resolved=%+v placed=%q", br.resolved, br.placedURL)
	}
}

// A refused --board writes nothing, not even the issue.
func TestForgeIssueCreateBoardRefusalsWriteNothing(t *testing.T) {
	stubBoards(t, map[string]forgewrite.Board{"github.com": {Owner: "o", Number: 1}})
	cases := map[string]struct {
		origin string
		args   []string
		want   string
	}{
		"gitea has no projects": {"git@gitea.kubelab.live:p/r.git", []string{"--board", "--type", "bug"}, "not supported"},
		"board not declared":    {"https://ghe.example/o/n.git", []string{"--board", "--type", "bug"}, "no board is declared"},
		"type is a decision":    {"https://github.com/o/n.git", []string{"--board"}, "--type"},
	}
	for name, c := range cases {
		rec := stubForgeWrite(t, c.origin)
		if !strings.Contains(c.origin, "gitea") {
			useBackend(t, &boardRecorder{commentRecorder: rec})
		}
		_, err := runForgeIssue(t, createArgs(t, c.args...)...)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to mention %q", name, err, c.want)
		}
		for _, call := range rec.calls {
			if call == "create" || call == "place" {
				t.Errorf("%s: a refused --board still wrote: %v", name, rec.calls)
			}
		}
	}
}

// If the card cannot be placed the issue already exists; the error must say so
// and tell the caller not to create it again.
func TestForgeIssueCreateBoardFailureNamesTheIssue(t *testing.T) {
	rec := stubForgeWrite(t, "https://github.com/o/n.git")
	useBackend(t, &boardRecorder{commentRecorder: rec, placeErr: errors.New("boom")})
	stubBoards(t, map[string]forgewrite.Board{"github.com": {Owner: "o", Number: 1}})
	_, err := runForgeIssue(t, createArgs(t, "--board", "--type", "bug")...)
	if err == nil || !strings.Contains(err.Error(), "https://forge.example/i/9") || !strings.Contains(err.Error(), "do not run create again") {
		t.Errorf("got %v", err)
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
