package forgewrite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// prServer stands in for Gitea's pull request endpoints: GET answers the pull
// request as it stands, and a write answers with draft set by whether the
// title it was given starts with a prefix the server recognises.
func prServer(t *testing.T, title string, prefixes ...string) (*httptest.Server, *[]seen) {
	t.Helper()
	var log []seen
	draftOf := func(title string) bool {
		for _, p := range prefixes {
			if strings.HasPrefix(strings.ToUpper(title), strings.ToUpper(p)) {
				return true
			}
		}
		return false
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		log = append(log, seen{r.Method, r.URL.RequestURI(), body})
		if r.Method != http.MethodGet {
			title, _ = body["title"].(string)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"number":7,"title":%q,"draft":%t,"html_url":"https://gitea.example/o/n/pulls/7"}`, title, draftOf(title))
	}))
	t.Cleanup(srv.Close)
	return srv, &log
}

func TestGiteaCreatePRDraftSendsAPrefixedTitle(t *testing.T) {
	srv, log := prServer(t, "", "WIP:", "[WIP]")
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	url, err := g.CreatePR(context.Background(), Repo{"h", "o", "n"}, NewPR{Title: "feat: x", Body: "b", Head: "feat/x", Base: "main", Draft: true})
	if err != nil || !strings.HasSuffix(url, "/pulls/7") {
		t.Fatalf("CreatePR = %q, %v", url, err)
	}
	if p := (*log)[0]; p.method != http.MethodPost || p.path != "/api/v1/repos/o/n/pulls" || p.body["title"] != "WIP: feat: x" {
		t.Errorf("request was %+v", p)
	}
}

// An instance configured with other prefixes opens the pull request ready.
// It exists by then, so the error names it rather than reading as "nothing
// happened".
func TestGiteaCreatePRDraftFailsWhenTheInstanceIgnoresThePrefix(t *testing.T) {
	srv, _ := prServer(t, "", "DRAFT:")
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	_, err := g.CreatePR(context.Background(), Repo{"h", "o", "n"}, NewPR{Title: "t", Head: "h", Base: "b", Draft: true})
	if err == nil || !strings.Contains(err.Error(), "https://gitea.example/o/n/pulls/7") || !strings.Contains(err.Error(), "FIX:") {
		t.Fatalf("want ERROR/WHY/FIX naming the pull request, got %v", err)
	}
}

func TestGiteaCreatePRWithoutDraftKeepsTheTitle(t *testing.T) {
	srv, log := prServer(t, "", "WIP:")
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	if _, err := g.CreatePR(context.Background(), Repo{"h", "o", "n"}, NewPR{Title: "t", Head: "h", Base: "b"}); err != nil {
		t.Fatal(err)
	}
	if got := (*log)[0].body["title"]; got != "t" {
		t.Errorf("title sent = %v", got)
	}
}

func TestGiteaReadyPRStripsThePrefix(t *testing.T) {
	for _, title := range []string{"WIP: feat: x", "wip:feat: x", "[WIP]  feat: x", "[wip] feat: x"} {
		srv, log := prServer(t, title, "WIP:", "[WIP]")
		g := Gitea{BaseURL: srv.URL, Token: tokenFn}
		url, err := g.ReadyPR(context.Background(), Repo{"h", "o", "n"}, 7)
		if err != nil || !strings.HasSuffix(url, "/pulls/7") {
			t.Fatalf("%q: ReadyPR = %q, %v", title, url, err)
		}
		last := (*log)[len(*log)-1]
		if last.method != http.MethodPatch || last.path != "/api/v1/repos/o/n/pulls/7" || last.body["title"] != "feat: x" {
			t.Errorf("%q: write was %+v", title, last)
		}
	}
}

func TestGiteaReadyPRFailsWhenTheTitleStillReadsAsDraft(t *testing.T) {
	// The instance treats "DRAFT:" as a prefix too; dotf only knows the defaults.
	srv, _ := prServer(t, "DRAFT: WIP: t", "WIP:", "DRAFT:")
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	if _, err := g.ReadyPR(context.Background(), Repo{"h", "o", "n"}, 7); err == nil || !strings.Contains(err.Error(), "FIX:") {
		t.Fatalf("want ERROR/WHY/FIX, got %v", err)
	}
}

func TestGiteaReadyPRWritesNothingForAReadyPullRequest(t *testing.T) {
	srv, log := prServer(t, "feat: x", "WIP:")
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	url, err := g.ReadyPR(context.Background(), Repo{"h", "o", "n"}, 7)
	if err != nil || !strings.HasSuffix(url, "/pulls/7") {
		t.Fatalf("ReadyPR = %q, %v", url, err)
	}
	for _, s := range *log {
		if s.method != http.MethodGet {
			t.Errorf("a ready pull request was written: %+v", s)
		}
	}
}

// ghFake answers `gh api repos/.../pulls/N` with the pull request and records
// every call, the GraphQL one included.
func ghFake(draft bool, calls *[][]string) GHRunner {
	return ghFakeAnswering(draft, `{"data":{"markPullRequestReadyForReview":{"pullRequest":{"isDraft":false}}}}`, calls)
}

func ghFakeAnswering(draft bool, mutation string, calls *[][]string) GHRunner {
	return func(_ context.Context, _ []byte, a ...string) ([]byte, error) {
		*calls = append(*calls, a)
		switch {
		case len(a) >= 2 && a[1] == "repos/o/n/pulls/9":
			return []byte(fmt.Sprintf(`{"node_id":"PR_kw9","draft":%t,"html_url":"https://github.com/o/n/pull/9"}`, draft)), nil
		case len(a) >= 2 && a[1] == "graphql":
			return []byte(mutation), nil
		}
		return nil, errors.New("unexpected gh call: " + strings.Join(a, " "))
	}
}

func TestGitHubReadyPRSendsTheMutationWithTheNodeID(t *testing.T) {
	var calls [][]string
	g := GitHub{Run: ghFake(true, &calls)}
	url, err := g.ReadyPR(context.Background(), Repo{"github.com", "o", "n"}, 9)
	if err != nil || url != "https://github.com/o/n/pull/9" {
		t.Fatalf("ReadyPR = %q, %v", url, err)
	}
	if len(calls) != 2 {
		t.Fatalf("gh calls = %v", calls)
	}
	m := strings.Join(calls[1], " ")
	if !strings.HasPrefix(m, "api graphql ") || !strings.Contains(m, "markPullRequestReadyForReview") || !strings.Contains(m, "id=PR_kw9") {
		t.Errorf("mutation call = %s", m)
	}
}

func TestGitHubReadyPRSkipsTheMutationForAReadyPullRequest(t *testing.T) {
	var calls [][]string
	g := GitHub{Run: ghFake(false, &calls)}
	if _, err := g.ReadyPR(context.Background(), Repo{"github.com", "o", "n"}, 9); err != nil {
		t.Fatal(err)
	}
	for _, c := range calls {
		if len(c) > 1 && c[1] == "graphql" {
			t.Errorf("a ready pull request got the mutation: %v", calls)
		}
	}
}

// Verified, never assumed: an answer that does not say isDraft: false fails.
func TestGitHubReadyPRFailsWhenTheAnswerIsNotReady(t *testing.T) {
	for _, answer := range []string{
		`{"data":{"markPullRequestReadyForReview":{"pullRequest":{"isDraft":true}}}}`,
		`{"data":null}`,
		`not json`,
	} {
		var calls [][]string
		g := GitHub{Run: ghFakeAnswering(true, answer, &calls)}
		if _, err := g.ReadyPR(context.Background(), Repo{"github.com", "o", "n"}, 9); err == nil || !strings.Contains(err.Error(), "FIX:") {
			t.Errorf("answer %s: want ERROR/WHY/FIX, got %v", answer, err)
		}
	}
}
