package forge

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseOAuthScopes(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    []string
		present bool
	}{
		{name: "classic token", in: "HTTP/2.0 200 OK\nX-Oauth-Scopes: gist, read:org, repo, workflow\nX-Accepted-Oauth-Scopes: \n\n{}", want: []string{"gist", "read:org", "repo", "workflow"}, present: true},
		{name: "header case does not matter", in: "HTTP/2.0 200 OK\r\nx-oauth-scopes: repo\r\n\r\n", want: []string{"repo"}, present: true},
		{name: "a token with no scopes still reports the header", in: "HTTP/2.0 200 OK\nX-Oauth-Scopes: \n\n", want: nil, present: true},
		{name: "a token that does not report scopes", in: "HTTP/2.0 200 OK\nContent-Type: application/json\n\n{}", want: nil, present: false},
		{name: "the accepted-scopes header is not the token's", in: "HTTP/2.0 200 OK\nX-Accepted-Oauth-Scopes: repo\n\n", want: nil, present: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, present := ParseOAuthScopes(tc.in)
			if present != tc.present || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseOAuthScopes = %v, %v; want %v, %v", got, present, tc.want, tc.present)
			}
		})
	}
}

func TestMissingGHScopes(t *testing.T) {
	cases := []struct {
		name string
		have []string
		want []string
	}{
		{name: "everything granted", have: RequiredGHScopes, want: nil},
		{name: "a token without the project scope", have: []string{"gist", "read:org", "repo", "workflow"}, want: []string{"read:project"}},
		{name: "project implies read:project", have: []string{"gist", "read:org", "repo", "workflow", "project"}, want: nil},
		{name: "admin:org implies read:org", have: []string{"gist", "admin:org", "repo", "workflow", "read:project"}, want: nil},
		{name: "no scopes at all", have: nil, want: RequiredGHScopes},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MissingGHScopes(tc.have); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("MissingGHScopes(%v) = %v, want %v", tc.have, got, tc.want)
			}
		})
	}
}

func TestGHLoginCommands(t *testing.T) {
	if got, want := GHLoginCommand(), "gh auth login -s gist,read:org,read:project,repo,workflow"; got != want {
		t.Errorf("GHLoginCommand = %q, want %q", got, want)
	}
	if got, want := GHRefreshCommand([]string{"read:project"}), "gh auth refresh -h github.com -s read:project"; got != want {
		t.Errorf("GHRefreshCommand = %q, want %q", got, want)
	}
}

// The runbooks that walk a new machine through `gh auth login` quote the one
// list, so the login step cannot ask for fewer scopes than doctor checks (#2204).
func TestRunbooksRequestEveryDeclaredScope(t *testing.T) {
	for _, rb := range []string{"guide-new-machine.md", "guide-git-config.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "runbooks", rb))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), GHLoginCommand()) {
			t.Errorf("%s does not quote %q", rb, GHLoginCommand())
		}
	}
}
