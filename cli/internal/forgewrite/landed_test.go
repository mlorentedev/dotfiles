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

// giteaPulls serves page N of the closed-PR listing from pages[N-1].
func giteaPulls(t *testing.T, pages ...[]string) (*httptest.Server, *[]string) {
	t.Helper()
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.RequestURI())
		var n int
		_, _ = fmt.Sscanf(r.URL.Query().Get("page"), "%d", &n)
		if n < 1 || n > len(pages) {
			_, _ = w.Write([]byte("[]"))
			return
		}
		_, _ = w.Write([]byte("[" + strings.Join(pages[n-1], ",") + "]"))
	}))
	t.Cleanup(srv.Close)
	return srv, &asked
}

func TestGiteaLandedHeadsKeepsOnlyMergedPRsOfThisBranchInThisRepo(t *testing.T) {
	srv, asked := giteaPulls(t, []string{
		giteaPR("fix/x", "aaa", true, 1),  // merged here: counts
		giteaPR("fix/x", "bbb", false, 1), // closed without merging: the case B could not tell apart
		giteaPR("fix/x", "ccc", true, 7),  // a fork's branch of the same name
		giteaPR("fix/y", "ddd", true, 1),  // another branch
		// a branch that still exists: head.ref names it too
		`{"merged":true,"head":{"label":"fix/x","ref":"fix/x","sha":"eee","repo_id":1},"base":{"repo_id":1}}`,
	})
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	heads, err := g.LandedHeads(context.Background(), Repo{Host: "gitea.example", Owner: "o", Name: "n"}, "fix/x")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(heads, []string{"aaa", "eee"}) {
		t.Fatalf("heads = %v, want [aaa eee]", heads)
	}
	if want := "/api/v1/repos/o/n/pulls?state=closed&sort=recentclose&limit=50&page=1"; (*asked)[0] != want {
		t.Fatalf("asked %s, want %s", (*asked)[0], want)
	}
}

func TestGiteaLandedHeadsReadsPastTheFirstPage(t *testing.T) {
	full := make([]string, giteaPullsPage)
	for i := range full {
		full[i] = giteaPR("other", "x", true, 1)
	}
	srv, asked := giteaPulls(t, full, []string{giteaPR("fix/x", "aaa", true, 1)})
	heads, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err != nil || !reflect.DeepEqual(heads, []string{"aaa"}) {
		t.Fatalf("heads = %v, err %v; want [aaa] from page 2", heads, err)
	}
	if len(*asked) != 3 {
		t.Fatalf("asked %d pages, want 3 (only the empty page 3 ends the listing)", len(*asked))
	}
}

// A server whose MAX_RESPONSE_ITEMS is below the asked limit answers every
// page short. Stopping at the first short page would read "never landed" for
// a branch whose pull request is on page 2.
func TestGiteaLandedHeadsReadsOnWhenTheServerClampsThePage(t *testing.T) {
	short := []string{giteaPR("other", "x", true, 1)}
	srv, _ := giteaPulls(t, short, []string{giteaPR("fix/x", "aaa", true, 1)})
	heads, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err != nil || !reflect.DeepEqual(heads, []string{"aaa"}) {
		t.Fatalf("heads = %v, err %v; want [aaa] from the second short page", heads, err)
	}
}

// Past the page cap the answer is partial, even with a match: an older pull
// request of the branch may hold the tip, and done would read "not landed".
func TestGiteaLandedHeadsRefusesACutOffListing(t *testing.T) {
	full := make([]string, giteaPullsPage)
	for i := range full {
		full[i] = giteaPR("other", "x", true, 1)
	}
	full[0] = giteaPR("fix/x", "aaa", true, 1)
	pages := make([][]string, maxPullPages)
	for i := range pages {
		pages[i] = full
	}
	srv, _ := giteaPulls(t, pages...)
	_, err := Gitea{BaseURL: srv.URL, Token: tokenFn}.LandedHeads(context.Background(), Repo{Owner: "o", Name: "n"}, "fix/x")
	if err == nil || !strings.Contains(err.Error(), "past 10 pages") {
		t.Fatalf("want an error naming the cut-off listing, got %v", err)
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
