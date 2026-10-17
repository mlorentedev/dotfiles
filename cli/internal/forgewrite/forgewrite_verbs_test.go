package forgewrite

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type seen struct {
	method, path string
	body         map[string]any
}

// giteaServer answers the label list with two labels and any write with a URL,
// and records every request.
func giteaServer(t *testing.T, writeStatus int, writeBody string) (*httptest.Server, *[]seen) {
	t.Helper()
	var log []seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		log = append(log, seen{r.Method, r.URL.RequestURI(), body})
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `[{"id":11,"name":"bug"},{"id":12,"name":"ci"}]`)
			return
		}
		w.WriteHeader(writeStatus)
		_, _ = io.WriteString(w, writeBody)
	}))
	t.Cleanup(srv.Close)
	return srv, &log
}

func tokenFn() (string, error) { return fakeToken, nil }

func TestGiteaCreateIssueMapsLabelNamesToIDs(t *testing.T) {
	srv, log := giteaServer(t, 201, `{"html_url":"https://gitea.example/o/n/issues/4"}`)
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	url, err := g.CreateIssue(context.Background(), Repo{"h", "o", "n"}, NewIssue{Title: "BUG-1: x", Body: "details", Labels: []string{"ci", "bug"}})
	if err != nil || url != "https://gitea.example/o/n/issues/4" {
		t.Fatalf("CreateIssue = %q, %v", url, err)
	}
	post := (*log)[len(*log)-1]
	if post.method != http.MethodPost || post.path != "/api/v1/repos/o/n/issues" {
		t.Errorf("write was %s %s", post.method, post.path)
	}
	ids, _ := post.body["labels"].([]any)
	if len(ids) != 2 || ids[0] != float64(12) || ids[1] != float64(11) || post.body["title"] != "BUG-1: x" || post.body["body"] != "details" {
		t.Errorf("payload = %v", post.body)
	}
}

func TestGiteaCreateIssueRefusesAnUnknownLabel(t *testing.T) {
	srv, log := giteaServer(t, 201, `{}`)
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	_, err := g.CreateIssue(context.Background(), Repo{"h", "o", "n"}, NewIssue{Title: "t", Body: "b", Labels: []string{"nope"}})
	if err == nil || !strings.Contains(err.Error(), "FIX:") || !strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("an unknown label must fail with ERROR/WHY/FIX, got %v", err)
	}
	for _, s := range *log {
		if s.method != http.MethodGet {
			t.Errorf("the issue was written despite the unknown label: %s %s", s.method, s.path)
		}
	}
}

func TestGiteaCreateWithoutLabelsDoesNotListThem(t *testing.T) {
	srv, log := giteaServer(t, 201, `{"html_url":"u"}`)
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	if _, err := g.CreateIssue(context.Background(), Repo{"h", "o", "n"}, NewIssue{Title: "t", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if len(*log) != 1 {
		t.Errorf("requests = %d, want only the write", len(*log))
	}
}

func TestGiteaEditSendsOnlyWhatIsNamed(t *testing.T) {
	srv, log := giteaServer(t, 201, `{"html_url":"https://gitea.example/o/n/issues/5"}`)
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	title := "new"
	if _, err := g.EditIssue(context.Background(), Repo{"h", "o", "n"}, 5, IssueEdit{Title: &title}); err != nil {
		t.Fatal(err)
	}
	p := (*log)[0]
	if p.method != http.MethodPatch || p.path != "/api/v1/repos/o/n/issues/5" {
		t.Errorf("write was %s %s", p.method, p.path)
	}
	if _, hasBody := p.body["body"]; hasBody || p.body["title"] != "new" {
		t.Errorf("an edit of the title must not carry a body: %v", p.body)
	}
}

// A 2xx whose body cannot be read still means the write happened.
func TestGiteaVerbsTreatAnUnreadable2xxAsDone(t *testing.T) {
	srv, _ := giteaServer(t, 200, `{"html_url": trunc`)
	g := Gitea{BaseURL: srv.URL, Token: tokenFn}
	title := "t"
	if url, err := g.CreateIssue(context.Background(), Repo{"h", "o", "n"}, NewIssue{Title: "t", Body: "b"}); err != nil || url != "" {
		t.Errorf("create = %q, %v", url, err)
	}
	if url, err := g.EditIssue(context.Background(), Repo{"h", "o", "n"}, 1, IssueEdit{Title: &title}); err != nil || url != "" {
		t.Errorf("edit = %q, %v", url, err)
	}
}

func TestGitHubCreateAndEditUseTheIssuesEndpoint(t *testing.T) {
	var calls [][]string
	var stdins []string
	g := GitHub{Run: func(_ context.Context, stdin []byte, args ...string) ([]byte, error) {
		calls = append(calls, args)
		stdins = append(stdins, string(stdin))
		return []byte(`{"html_url":"https://github.com/o/n/issues/7"}`), nil
	}}
	secretish := "mentions dotf secrets show"
	if url, err := g.CreateIssue(context.Background(), Repo{"github.com", "o", "n"}, NewIssue{Title: "t", Body: secretish, Labels: []string{"bug"}}); err != nil || !strings.HasSuffix(url, "/7") {
		t.Fatalf("create = %q, %v", url, err)
	}
	body := "new body"
	if _, err := g.EditIssue(context.Background(), Repo{"github.com", "o", "n"}, 7, IssueEdit{Body: &body}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(calls[0], " "); got != "api -X POST repos/o/n/issues --input -" {
		t.Errorf("create ran gh %s", got)
	}
	if got := strings.Join(calls[1], " "); got != "api -X PATCH repos/o/n/issues/7 --input -" {
		t.Errorf("edit ran gh %s", got)
	}
	if !strings.Contains(stdins[0], `"labels":["bug"]`) || !strings.Contains(stdins[0], "secrets show") {
		t.Errorf("create payload = %s", stdins[0])
	}
	if strings.Contains(stdins[1], "title") {
		t.Errorf("a body-only edit carried a title: %s", stdins[1])
	}
	for _, c := range calls {
		if strings.Contains(strings.Join(c, " "), "secrets show") {
			t.Errorf("the body reached the command line: %v", c)
		}
	}
}
