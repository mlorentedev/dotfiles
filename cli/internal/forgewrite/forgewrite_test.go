package forgewrite

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fakeToken = "tok-3f9a-SECRET-do-not-print"

func TestParseOrigin(t *testing.T) {
	cases := map[string]Repo{
		"https://github.com/mlorentedev/dotfiles.git":            {"github.com", "mlorentedev", "dotfiles"},
		"https://github.com/mlorentedev/dotfiles":                {"github.com", "mlorentedev", "dotfiles"},
		"git@github.com:mlorentedev/dotfiles.git":                {"github.com", "mlorentedev", "dotfiles"},
		"ssh://git@gitea.kubelab.live:2222/personal/imagesensor": {"gitea.kubelab.live", "personal", "imagesensor"},
		"https://user:pw@Gitea.Kubelab.Live/personal/x.git/":     {"gitea.kubelab.live", "personal", "x"},
	}
	for in, want := range cases {
		got, err := ParseOrigin(in)
		if err != nil || got != want {
			t.Errorf("ParseOrigin(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
}

func TestDetectForgeFromOrigin(t *testing.T) {
	hosts := Hosts{"gitea.kubelab.live": {Kind: KindGitea, Secret: "GITEA_KUBELAB_TOKEN"}}
	for host, want := range map[string]Kind{"github.com": KindGitHub, "gitea.kubelab.live": KindGitea} {
		k, _, err := Select(Repo{Host: host, Owner: "o", Name: "n"}, hosts)
		if err != nil || k != want {
			t.Errorf("Select(%s) = %q, %v; want %q", host, k, err, want)
		}
	}
}

func TestUndetectableForgeIsAnError(t *testing.T) {
	_, _, err := Select(Repo{Host: "git.example.org", Owner: "o", Name: "n"}, Hosts{})
	if err == nil {
		t.Fatal("an undeclared host must not be guessed")
	}
	for _, part := range []string{"ERROR:", "WHY:", "FIX:", "forge/hosts.json"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("error lacks %q: %v", part, err)
		}
	}
	if _, err := ParseOrigin("not a url"); err == nil || !strings.Contains(err.Error(), "FIX:") {
		t.Errorf("an unreadable origin must say how to fix it, got %v", err)
	}
}

func TestLoadHosts(t *testing.T) {
	dir := t.TempDir()
	if h, err := LoadHosts(filepath.Join(dir, "absent.json")); err != nil || len(h) != 0 {
		t.Errorf("a missing file is an empty declaration, got %v, %v", h, err)
	}
	good := filepath.Join(dir, "hosts.json")
	if err := os.WriteFile(good, []byte(`{"hosts":{"g.example":{"kind":"gitea","secret":"S"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if h, err := LoadHosts(good); err != nil || h["g.example"].Secret != "S" {
		t.Errorf("LoadHosts = %v, %v", h, err)
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"hosts":{"g.example":{"kind":"gitea"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadHosts(bad); err == nil {
		t.Error("a host without a secret must be refused")
	}
}

// The contract: no merge, no close. Merging is `dotf pr land` behind the grant.
func TestNoMergeOrCloseVerb(t *testing.T) {
	bt := reflect.TypeOf((*Backend)(nil)).Elem()
	for i := 0; i < bt.NumMethod(); i++ {
		name := strings.ToLower(bt.Method(i).Name)
		for _, banned := range []string{"merge", "close", "delete", "reopen"} {
			if strings.Contains(name, banned) {
				t.Errorf("Backend has a %q verb: %s", banned, bt.Method(i).Name)
			}
		}
	}
}

func TestGiteaCommentPostsBodyAndToken(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"html_url":"https://gitea.example/o/n/issues/7#issuecomment-1"}`)
	}))
	defer srv.Close()
	g := Gitea{BaseURL: srv.URL, Token: func() (string, error) { return fakeToken, nil }}
	url, err := g.CommentIssue(context.Background(), Repo{"h", "o", "n"}, 7, "line one\nline two")
	if err != nil || !strings.HasSuffix(url, "#issuecomment-1") {
		t.Fatalf("CommentIssue = %q, %v", url, err)
	}
	if gotPath != "/api/v1/repos/o/n/issues/7/comments" || gotAuth != "token "+fakeToken || gotBody["body"] != "line one\nline two" {
		t.Errorf("request was path=%q auth=%q body=%v", gotPath, gotAuth, gotBody)
	}
}

// A happy path that leaks nothing proves little: every failure shape must keep
// the token out of the error as well.
func TestGiteaTokenNeverLeaks(t *testing.T) {
	echo := func(status int, body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			// A hostile or sloppy server that echoes the credential back.
			_, _ = io.WriteString(w, body+r.Header.Get("Authorization"))
		}))
	}
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()

	cases := map[string]Gitea{
		"401":       {BaseURL: echo(401, "bad credentials ").URL},
		"500":       {BaseURL: echo(500, "boom ").URL},
		"malformed": {BaseURL: echo(200, "not json ").URL},
		"transport": {BaseURL: closedURL},
	}
	for name, g := range cases {
		g.Token = func() (string, error) { return fakeToken, nil }
		_, err := g.CommentIssue(context.Background(), Repo{"h", "o", "n"}, 1, "b")
		if err == nil {
			t.Errorf("%s: want an error", name)
			continue
		}
		if strings.Contains(err.Error(), fakeToken) {
			t.Errorf("%s: the error carries the token: %v", name, err)
		}
	}

	// The resolver's own failure may quote what the store said; it must not
	// be echoed past the package either.
	g := Gitea{BaseURL: "http://127.0.0.1:0", Token: func() (string, error) { return "", errors.New("bw: locked") }}
	if _, err := g.CommentIssue(context.Background(), Repo{"h", "o", "n"}, 1, "b"); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Errorf("a resolver failure should surface, got %v", err)
	}
	empty := Gitea{BaseURL: "http://127.0.0.1:0", Token: func() (string, error) { return "", nil }}
	if _, err := empty.CommentIssue(context.Background(), Repo{"h", "o", "n"}, 1, "b"); err == nil {
		t.Error("an empty token must be refused before any request")
	}
}

func TestGitHubCommentSendsBodyOnStdinNotArgs(t *testing.T) {
	var gotArgs []string
	var gotStdin []byte
	g := GitHub{Run: func(_ context.Context, stdin []byte, args ...string) ([]byte, error) {
		gotArgs, gotStdin = args, stdin
		return []byte(`{"html_url":"https://github.com/o/n/issues/3#issuecomment-9"}`), nil
	}}
	secretish := "a body mentioning dotf secrets show and a token"
	url, err := g.CommentIssue(context.Background(), Repo{"github.com", "o", "n"}, 3, secretish)
	if err != nil || !strings.HasSuffix(url, "#issuecomment-9") {
		t.Fatalf("CommentIssue = %q, %v", url, err)
	}
	if strings.Contains(strings.Join(gotArgs, " "), "secrets show") {
		t.Errorf("the body reached the command line: %v", gotArgs)
	}
	if !strings.Contains(string(gotStdin), "secrets show") {
		t.Errorf("the body did not travel on stdin: %s", gotStdin)
	}
}
