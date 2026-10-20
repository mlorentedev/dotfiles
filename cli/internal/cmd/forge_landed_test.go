package cmd

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/forgewrite"
)

func repoWithOrigin(t *testing.T, origin string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", origin}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// #2759: worktree done asks the forge its origin names, Gitea included.
func TestForgeLandedHeadsAsksTheOriginsForge(t *testing.T) {
	rec := &commentRecorder{landed: []string{"aaa"}}
	var built forgewrite.Repo
	old := forgeWriteBackend
	forgeWriteBackend = func(r forgewrite.Repo) (forgewrite.Backend, error) { built = r; return rec, nil }
	t.Cleanup(func() { forgeWriteBackend = old })

	heads, err := forgeLandedHeads(repoWithOrigin(t, "git@gitea.example.test:manu/kubelab.git"), "fix/x")
	if err != nil {
		t.Fatal(err)
	}
	want := forgewrite.Repo{Host: "gitea.example.test", Owner: "manu", Name: "kubelab"}
	if built != want || rec.repo != want || rec.branch != "fix/x" || !reflect.DeepEqual(heads, []string{"aaa"}) {
		t.Fatalf("built %v, asked %v about %q, got %v", built, rec.repo, rec.branch, heads)
	}
}

// An origin no backend serves is an answer nobody gave: the error says why,
// and done keeps the branch with it.
func TestForgeLandedHeadsNamesWhyNoForgeCanBeAsked(t *testing.T) {
	for origin, want := range map[string]string{
		"file:///srv/git/kubelab.git":              "cannot read a repository from origin",
		"https://gitea.undeclared.test/manu/x.git": "no forge backend for gitea.undeclared.test/manu/x",
	} {
		t.Run(origin, func(t *testing.T) {
			old := forgeWriteBackend
			t.Cleanup(func() { forgeWriteBackend = old })
			forgeWriteBackend = func(r forgewrite.Repo) (forgewrite.Backend, error) {
				_, _, err := forgewrite.Select(r, forgewrite.Hosts{})
				return nil, err
			}
			_, err := forgeLandedHeads(repoWithOrigin(t, origin), "fix/x")
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want an error containing %q, got %v", want, err)
			}
		})
	}
}
