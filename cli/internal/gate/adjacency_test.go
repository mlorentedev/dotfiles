package gate

import (
	"reflect"
	"testing"
)

func TestAdjacencyCandidates(t *testing.T) {
	got := AdjacencyCandidates([]string{"", "scripts/a.sh", "docs/lessons.md", "specs/X-1/proposal.md", "cli/x.go"})
	want := []string{"scripts/a.sh", "cli/x.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestContainsWord(t *testing.T) {
	cases := []struct {
		text, word string
		want       bool
	}{
		{"the config file", "config", true},
		{"the configuration", "config", false},
		{"config", "config", true},
		{"aXb.sh fails", "a.b.sh", false},
		{"a.b.sh fails", "a.b.sh", true},
		{"my_config", "config", false},
	}
	for _, c := range cases {
		if got := ContainsWord(c.text, c.word); got != c.want {
			t.Errorf("ContainsWord(%q, %q) = %v, want %v", c.text, c.word, got, c.want)
		}
	}
}

func TestAdjacentIssues(t *testing.T) {
	feed := "849\tfix `scripts/k.sh` now\n" +
		"850\tscripts/k.sh closed here\n" +
		"444\tthe Makefile is slow\n" +
		"556\tthe config file\n" +
		"\n" +
		"557\t\n" +
		"558\tlast line without newline names config"
	got := AdjacentIssues(feed, []string{"scripts/k.sh", "build/Makefile", "scripts/config"}, []string{"850"})
	want := []Adjacent{
		{"849", "scripts/k.sh", "fix `scripts/k.sh` now"},
		{"556", "config", "the config file"},
		{"558", "config", "last line without newline names config"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestClip(t *testing.T) {
	if got := Clip("abcdef", 3); got != "abc" {
		t.Errorf("got %q", got)
	}
	if got := Clip("ab", 3); got != "ab" {
		t.Errorf("got %q", got)
	}
	if got := Clip("aé", 2); got != "a" {
		t.Errorf("mid-rune cut: got %q", got)
	}
}

func TestStripControl(t *testing.T) {
	cases := map[string]string{
		"plain title":         "plain title",
		"a\r::error::b":       "a ::error::b",
		"tab\there":           "tab here",
		"esc\x1b[31m del\x7f": "esc [31m del ",
		"ñandú <b>":           "ñandú <b>",
	}
	for in, want := range cases {
		if got := StripControl(in); got != want {
			t.Errorf("StripControl(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAdjacentIssuesStripsControlCharacters(t *testing.T) {
	got := AdjacentIssues("7\tscripts/x.sh breaks\r::error::y\n", []string{"scripts/x.sh"}, nil)
	if len(got) != 1 || got[0].Text != "scripts/x.sh breaks ::error::y" {
		t.Fatalf("AdjacentIssues = %+v", got)
	}
}
