package evals

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// The run loop names the tree; a real checkout must land exactly there, be the
// reviewed commit while the runner works, and be gone afterwards.
func TestLaunchChecksOutAReplayWhereTheLoopExpectsIt(t *testing.T) {
	clone, onMain, _ := replayRemote(t)
	c := Case{ID: "c1", Kind: "replay", Spec: "AI-055", ReviewedSHA: onMain, BaseSHA: onMain}
	a := attemptFor(t.TempDir(), c, "agy/x", 1)
	var head string
	r := &fakeRunner{act: func(tree string, _ io.Writer) (spec.RunResult, error) {
		head = git(t, tree, "rev-parse", "HEAD")
		return spec.RunResult{}, nil
	}}
	l := testLauncher(t.TempDir(), r, DefaultLaunchDeps(clone).Checkout)
	if _, err := l.Launch(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if r.limits.Dir != a.Tree || head != onMain {
		t.Errorf("ran in %s at %s, want %s at %s", r.limits.Dir, head, a.Tree, onMain)
	}
	if wt := git(t, clone, "worktree", "list", "--porcelain"); strings.Count(wt, "worktree ") != 1 {
		t.Errorf("the replay worktree is still registered:\n%s", wt)
	}
}

// leakRepo is a repository whose only commit holds a review and a case key,
// the two files a replay must not be able to read.
func leakRepo(t *testing.T) (repo, sha string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo = t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	for _, f := range []string{"specs/archive/X/review.md", ReplayCaseDir + "/leaky.json", "code.go"} {
		p := filepath.Join(repo, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git(t, repo, "add", ".")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "c")
	return repo, git(t, repo, "rev-parse", "HEAD")
}

func preflightDeps(missing ...string) PreflightDeps {
	return PreflightDeps{
		LookPath: func(bin string) (string, error) {
			for _, m := range missing {
				if m == bin {
					return "", errors.New("not found")
				}
			}
			return "/bin/" + bin, nil
		},
		CheckSecret: func(id string) error {
			for _, m := range missing {
				if m == id {
					return errors.New("exit 1")
				}
			}
			return nil
		},
	}
}

var preflightCandidates = map[string]spec.ReviewerEntry{
	"agy/x":   {ID: "agy/x", Runner: "agy", Model: "x", Auth: "login"},
	"nan/glm": {ID: "nan/glm", Runner: "pi", Provider: "nan", Model: "glm", SecretID: "NAN_API_KEY"},
}

// Every check runs before the first launch, so a broken matrix costs nothing,
// and every problem is named at once.
func TestPreflightNamesEverythingThatWouldFailARun(t *testing.T) {
	repo, sha := leakRepo(t)
	cases := []Case{
		{ID: "clean", Kind: "replay", ReviewedSHA: sha, SourceReview: "specs/archive/X/review-round2.md"},
		{ID: "sees-review", Kind: "replay", ReviewedSHA: sha, SourceReview: "specs/archive/X/review.md"},
		{ID: "leaky", Kind: "replay", ReviewedSHA: sha, SourceReview: "specs/archive/X/none.md"},
		{ID: "gone", Kind: "replay", ReviewedSHA: strings.Repeat("d", 40), SourceReview: "x.md"},
	}
	p := Plan{}
	for _, c := range cases {
		for _, id := range []string{"agy/x", "nan/glm"} {
			p.Cells = append(p.Cells, Cell{Case: c.ID, Kind: c.Kind, Candidate: id, Epochs: 1})
		}
	}
	err := Preflight(repo, p, cases, preflightCandidates, preflightDeps("agy", "NAN_API_KEY"))
	if err == nil {
		t.Fatal("preflight passed a matrix that cannot run")
	}
	for _, want := range []string{
		"agy/x: runner agy is not on PATH",
		"nan/glm: secret NAN_API_KEY does not resolve",
		"sees-review: the reviewed commit holds the source review specs/archive/X/review.md",
		"leaky: the reviewed commit holds the case's own key " + ReplayCaseDir + "/leaky.json",
		"gone",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("preflight does not name %q:\n%v", want, err)
		}
	}
	if strings.Contains(err.Error(), "clean:") {
		t.Errorf("a clean case was refused:\n%v", err)
	}
	if strings.Count(err.Error(), "agy/x: runner") != 1 {
		t.Errorf("a candidate is checked once, not once per cell:\n%v", err)
	}
}

func TestPreflightPassesAMatrixThatCanRun(t *testing.T) {
	repo, sha := leakRepo(t)
	c := Case{ID: "clean", Kind: "replay", ReviewedSHA: sha, SourceReview: "specs/archive/X/review-round2.md"}
	p := Plan{Cells: []Cell{{Case: "clean", Kind: "replay", Candidate: "nan/glm", Epochs: 3}}}
	if err := Preflight(repo, p, []Case{c}, preflightCandidates, preflightDeps()); err != nil {
		t.Fatal(err)
	}
}
