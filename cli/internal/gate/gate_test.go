package gate

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The judging cases live in tests/check-knowledge-gate.bats, which runs them
// through this package and the shell gate and requires the same output. These
// cover what that suite cannot reach: the event file itself.

func writeEvent(t *testing.T, body string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestReadEvent(t *testing.T) {
	cases := []struct {
		name, json string
		want       PR
		wantErr    error
	}{
		{"full", `{"pull_request":{"number":7,"body":"b","labels":[{"name":"x"},{"name":"dependencies"}],"user":{"login":"renovate[bot]"}}}`,
			PR{Number: 7, Body: "b", Labels: []string{"x", "dependencies"}, Author: "renovate[bot]"}, nil},
		// GitHub sends null for a pull request opened without a description.
		{"null body is empty", `{"pull_request":{"number":1,"body":null,"user":{"login":"u"}}}`,
			PR{Number: 1, Author: "u"}, nil},
		{"push event has no PR", `{"ref":"refs/heads/main"}`, PR{}, ErrNoPR},
		{"explicit null PR", `{"pull_request":null}`, PR{}, ErrNoPR},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ReadEvent(writeEvent(t, c.json))
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestReadEventRefusesUnreadable(t *testing.T) {
	if _, err := ReadEvent(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("a missing event file must be an error")
	}
	if _, err := ReadEvent(writeEvent(t, "{")); err == nil || errors.Is(err, ErrNoPR) {
		t.Errorf("malformed JSON must be a parse error, got %v", err)
	}
}

func TestIsDependencyBotPR(t *testing.T) {
	cases := []struct {
		pr   PR
		want bool
	}{
		{PR{Author: "dependabot[bot]", Labels: []string{"dependencies"}}, true},
		{PR{Author: "dependabot[bot]"}, false},                                    // no label: judged
		{PR{Author: "manu", Labels: []string{"dependencies"}}, false},             // a human: judged
		{PR{Author: "dependabot[bot]x", Labels: []string{"dependencies"}}, false}, // exact match only
	}
	for _, c := range cases {
		if got := c.pr.IsDependencyBotPR(); got != c.want {
			t.Errorf("%+v: got %v, want %v", c.pr, got, c.want)
		}
	}
}

func TestStripMarkupReportsTheOpenFence(t *testing.T) {
	text, open := StripMarkup("a\n```\nb\n## Knowledge")
	if text != "a" || open != 2 {
		t.Errorf("got %q open=%d, want %q open=2", text, open, "a")
	}
	if _, open := StripMarkup("```\nx\n```\n"); open != 0 {
		t.Errorf("a closed fence reported open at %d", open)
	}
}
