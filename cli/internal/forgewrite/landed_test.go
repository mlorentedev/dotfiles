package forgewrite

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// giteaPR renders one pull request as Gitea 1.25 lists it once its branch is
// deleted, which is what a squash merge leaves: head.ref is then the pull ref
// and only head.label names the branch (measured on gitea.kubelab.live,
// 2026-10-19).
func giteaPR(branch, sha string, merged bool, headRepo int) string {
	return fmt.Sprintf(`{"merged":%t,"head":{"label":%q,"ref":"refs/pull/9/head","sha":%q,"repo_id":%d},"base":{"ref":"main","repo_id":1}}`, merged, branch, sha, headRepo)
}

// giteaLookup answers the repository read and the by-head lookup: pulls maps a
// request path under the repository to its body, and any other path is a 404.
func giteaLookup(t *testing.T, pulls map[string]string) (*httptest.Server, *[]string) {
	t.Helper()
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.EscapedPath())
		if r.URL.Path == "/api/v1/repos/o/n" {
			_, _ = w.Write([]byte(`{"default_branch":"trunk"}`))
			return
		}
		body, ok := pulls[strings.TrimPrefix(r.URL.Path, "/api/v1/repos/o/n/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &asked
}

func TestGiteaLandedHeadsReadsThePullOfThisBranchAgainstTheDefaultBranch(t *testing.T) {
	for name, tc := range map[string]struct {
		pr   string
		want []string
	}{
		"merged here":                {giteaPR("fix/x", "aaa", true, 1), []string{"aaa"}},
		"closed without merging":     {giteaPR("fix/x", "bbb", false, 1), nil},
		"a fork's branch":            {giteaPR("fix/x", "ccc", true, 7), nil},
		"a branch that still exists": {`{"merged":true,"head":{"label":"fix/x","ref":"fix/x","sha":"eee","repo_id":1},"base":{"repo_id":1}}`, []string{"eee"}},
	} {
		t.Run(name, func(t *testing.T) {
			srv, asked := giteaLookup(t, map[string]string{"pulls/trunk/fix/x": tc.pr})
			g := Gitea{BaseURL: srv.URL, Token: tokenFn}
			heads, err := g.LandedHeads(context.Background(), Repo{Host: "gitea.example", Owner: "o", Name: "n"}, "fix/x")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(heads, tc.want) {
				t.Fatalf("heads = %v, want %v", heads, tc.want)
			}
			if len(*asked) != 2 {
				t.Fatalf("asked %v, want the repository and one lookup", *asked)
			}
		})
	}
}

// A 404 on the lookup is the answer "this branch has no pull request", and
// only there: an error on either read must not read as "never landed".
func TestGiteaLandedHeadsReadsA404AsNoPullRequest(t *testing.T) {
	srv, _ := giteaLookup(t, nil)
	heads, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err != nil || len(heads) != 0 {
		t.Fatalf("heads = %v, err %v", heads, err)
	}
}

func TestGiteaLandedHeadsRefusesAMissingRepository(t *testing.T) {
	srv, _ := giteaLookup(t, nil)
	_, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "gone"}, "fix/x")
	if err == nil {
		t.Fatal("a repository that is not there read as a branch with no pull request")
	}
}

func TestGiteaLandedHeadsRefusesARepositoryWithNoDefaultBranch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	_, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err == nil || !strings.Contains(err.Error(), "default branch") {
		t.Fatalf("err = %v", err)
	}
}

func TestGiteaLandedHeadsSaysWhenTheLookupFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/o/n" {
			_, _ = w.Write([]byte(`{"default_branch":"main"}`))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	_, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v", err)
	}
}

func TestGiteaLandedHeadsEscapesTheBranchButKeepsItsSlash(t *testing.T) {
	srv, asked := giteaLookup(t, nil)
	if _, err := (Gitea{BaseURL: srv.URL, Token: tokenFn}).LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/a#b"); err != nil {
		t.Fatal(err)
	}
	if got := (*asked)[1]; got != "/api/v1/repos/o/n/pulls/trunk/fix/a%23b" {
		t.Fatalf("asked %s", got)
	}
}

// #3089: a body cut off mid-read is that failure, not an unreadable answer.
func TestGiteaReadCutOffSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte(`{"default`))
	}))
	t.Cleanup(srv.Close)
	_, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err == nil || strings.Contains(err.Error(), "not readable") || !strings.Contains(err.Error(), "reading") {
		t.Fatalf("err = %v", err)
	}
}

func TestGiteaLandedHeadsNeverLeaksTheToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("bad token " + fakeToken))
	}))
	t.Cleanup(srv.Close)
	_, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err == nil || strings.Contains(err.Error(), fakeToken) {
		t.Fatalf("want a redacted error, got %v", err)
	}
}

func TestGitHubLandedHeadsAsksRESTForTheBranchAndKeepsMergedOnly(t *testing.T) {
	var got []string
	g := GitHub{Run: func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		got = args
		// two pages, as --paginate prints them
		return []byte(`[{"merged_at":"2026-10-18T00:00:00Z","head":{"sha":"aaa","repo":{"full_name":"O/n"}}},` +
			`{"merged_at":null,"head":{"sha":"bbb","repo":{"full_name":"o/n"}}}]` +
			`[{"merged_at":"2026-10-18T00:00:00Z","head":{"sha":"ccc","repo":{"full_name":"o/n-fork"}}},` +
			`{"merged_at":"2026-10-18T00:00:00Z","head":{"sha":"ddd","repo":null}},` +
			`{"merged_at":"2026-10-18T00:00:00Z","head":{"sha":"eee","repo":{"full_name":"o/n"}}}]`), nil
	}}
	heads, err := g.LandedHeads(context.Background(), Repo{Host: "github.com", Owner: "o", Name: "n"}, "fix/a&b#2")
	if err != nil {
		t.Fatal(err)
	}
	// bbb never merged, ccc is the owner's fork, ddd's repository is gone
	if !reflect.DeepEqual(heads, []string{"aaa", "eee"}) {
		t.Fatalf("heads = %v, want [aaa eee]", heads)
	}
	// REST, not `gh pr list` (GraphQL); the branch is escaped so & and # stay
	// part of it, and the slash stays as it is.
	want := []string{"api", "--paginate", "repos/o/n/pulls?state=closed&per_page=100&head=o:fix/a%26b%232"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("gh %v, want %v", got, want)
	}
}

func TestGitHubLandedHeadsSaysWhatFailed(t *testing.T) {
	g := GitHub{Run: func(context.Context, []byte, ...string) ([]byte, error) {
		return nil, errors.New("exit status 1: HTTP 404")
	}}
	_, err := g.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("want gh's reason in the error, got %v", err)
	}
}
