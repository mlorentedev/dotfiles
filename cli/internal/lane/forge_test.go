package lane

import (
	"errors"
	"strings"
	"testing"
)

func pullsRunner(t *testing.T, body string, err error, got *[]string) Runner {
	t.Helper()
	return func(name string, args ...string) (string, error) {
		*got = append(*got, name+" "+strings.Join(args, " "))
		return body, err
	}
}

// The forge is asked over REST, one request for a branch, never GraphQL: a
// fleet of worktrees would drain that quota (#2550).
func TestREST_AsksOnePullsRequestPerBranch(t *testing.T) {
	var calls []string
	f := REST{Run: pullsRunner(t, `[]`, nil, &calls)}
	if _, err := f.PRFor("fix/a-b"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || !strings.HasPrefix(calls[0], "gh api ") || !strings.Contains(calls[0], "repos/{owner}/{repo}/pulls") {
		t.Fatalf("calls = %q, want one gh api repos/{owner}/{repo}/pulls", calls)
	}
	if strings.Contains(calls[0], "graphql") || strings.Contains(calls[0], "gh pr ") {
		t.Errorf("call %q reaches GraphQL", calls[0])
	}
	if !strings.Contains(calls[0], "head={owner}:fix/a-b") || !strings.Contains(calls[0], "state=all") || !strings.Contains(calls[0], "sort=created&direction=desc") {
		t.Errorf("call %q must filter on owner:branch, state=all and sort newest first explicitly", calls[0])
	}
}

func TestREST_NewestPullRequestDecidesTheState(t *testing.T) {
	cases := []struct {
		name string
		body string
		want PRInfo
	}{
		{"no pull request", `[]`, PRInfo{}},
		{"open", `[{"number":9,"state":"open","merged_at":null}]`, PRInfo{Number: 9, State: PROpen}},
		{"merged", `[{"number":8,"state":"closed","merged_at":"2026-10-15T10:00:00Z"}]`, PRInfo{Number: 8, State: PRMerged}},
		{"closed unmerged", `[{"number":7,"state":"closed","merged_at":null}]`, PRInfo{Number: 7, State: PRClosed}},
		{"newest first: an open retry after a closed attempt", `[{"number":12,"state":"open","merged_at":null},{"number":7,"state":"closed","merged_at":null}]`, PRInfo{Number: 12, State: PROpen}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var calls []string
			got, err := REST{Run: pullsRunner(t, c.body, nil, &calls)}.PRFor("fix/x")
			if err != nil || got != c.want {
				t.Errorf("PRFor = %+v, %v; want %+v", got, err, c.want)
			}
		})
	}
}

func TestREST_AFailedRequestIsAnErrorNotNoPR(t *testing.T) {
	var calls []string
	if _, err := (REST{Run: pullsRunner(t, "", errors.New("HTTP 403"), &calls)}).PRFor("fix/x"); err == nil {
		t.Error("a forge failure read as \"no pull request\"")
	}
	if _, err := (REST{Run: pullsRunner(t, `not json`, nil, &calls)}).PRFor("fix/x"); err == nil {
		t.Error("an unparseable answer read as \"no pull request\"")
	}
}
