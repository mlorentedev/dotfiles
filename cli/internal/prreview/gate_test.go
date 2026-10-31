package prreview

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// The bats cases of tests/pr-agent-push-gate.bats, ported, plus the edge cases
// that pin what jq does. What the port does differently on purpose is in
// TestGate_Divergences.

const (
	guideFull        = "## PR Reviewer Guide \xf0\x9f\x94\x8d\n\n<!-- pr-agent:review:full -->\n\nfindings"
	guideIncremental = "## Incremental PR Reviewer Guide \xf0\x9f\x94\x8d\n\n<!-- pr-agent:review:incremental -->\n\nmore"
	gateBotLogin     = "github-actions[bot]"
)

func reviewWithSHA(sha string) string {
	return guideFull + "\n\n<!-- pr-agent-review-state:v1\n{\"last_run\":{\"head_sha\":\"" + sha + "\"},\"schema_version\":1}\n-->"
}

// gateFixture is a PR as the REST API returns it. A raw value replaces the list
// it stands for; "" leaves that list in charge, and missing* leaves no file.
type gateFixture struct {
	comments, commits               []map[string]any
	rawComments, rawCommits         string
	missingComments, missingCommits bool
}

func (f *gateFixture) comment(login, at, body string) {
	f.comments = append(f.comments, map[string]any{"user": map[string]any{"login": login},
		"created_at": at, "updated_at": at, "body": body})
}

// commit appends one commit with an author date and no committer, as the bats
// fixture does. parents defaults to 1.
func (f *gateFixture) commit(date string, parents ...int) {
	f.commitOf(map[string]any{"author": map[string]any{"date": date}, "message": "change"}, parents...)
}

// rebased appends one commit as a rebase leaves it.
func (f *gateFixture) rebased(author, committer string) {
	f.commitOf(map[string]any{"author": map[string]any{"date": author},
		"committer": map[string]any{"date": committer}, "message": "change"})
}

func (f *gateFixture) commitOf(commit map[string]any, parents ...int) {
	n := 1
	if len(parents) > 0 {
		n = parents[0]
	}
	list := make([]map[string]any, n)
	for i := range list {
		list[i] = map[string]any{"sha": "p"}
	}
	f.commits = append(f.commits, map[string]any{"sha": fmt.Sprintf("c%d", len(f.commits)),
		"parents": list, "commit": commit})
}

// files is the two documents, nil where the fixture leaves no file: nil is what
// the command passes for a file it could not read.
func (f *gateFixture) files(t *testing.T) (comments, commits []byte) {
	t.Helper()
	encode := func(raw string, missing bool, list []map[string]any) []byte {
		switch {
		case missing:
			return nil
		case raw != "":
			return []byte(raw)
		case list == nil:
			return []byte("[]")
		}
		out, err := json.Marshal(list)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	return encode(f.rawComments, f.missingComments, f.comments), encode(f.rawCommits, f.missingCommits, f.commits)
}

type gateCase struct {
	name      string
	build     func(*gateFixture)
	threshold int // 0 means the default, 3, unless zero is set
	zero      bool
	sender    string
	want      []string
}

func (c gateCase) limit() int {
	switch {
	case c.zero:
		return 0
	case c.threshold != 0:
		return c.threshold
	}
	return 3
}

func threeCommits(f *gateFixture) {
	f.commit("2026-09-25T11:00:00Z")
	f.commit("2026-09-25T12:00:00Z")
	f.commit("2026-09-25T13:00:00Z")
}

func quotedBlock(body string) func(*gateFixture) {
	return func(f *gateFixture) {
		f.commit("2026-09-25T08:00:00Z")
		f.comment(gateBotLogin, "2026-09-25T10:00:00Z", body)
		threeCommits(f)
	}
}

func portedGateCases() []gateCase {
	return []gateCase{
		{name: "below the threshold", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.commit("2026-09-25T11:00:00Z")
			f.commit("2026-09-25T12:00:00Z")
		}, want: []string{"run=false", "2 new commit"}},
		{name: "at the threshold", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			threeCommits(f)
		}, want: []string{"run=true"}},
		{name: "commits older than the review do not count", build: func(f *gateFixture) {
			f.commit("2026-09-25T08:00:00Z")
			f.commit("2026-09-25T09:00:00Z")
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.commit("2026-09-25T11:00:00Z")
		}, want: []string{"run=false", "1 new commit"}},
		{name: "merge commits do not count", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.commit("2026-09-25T11:00:00Z")
			f.commit("2026-09-25T12:00:00Z", 2)
			f.commit("2026-09-25T13:00:00Z", 2)
			f.commit("2026-09-25T14:00:00Z", 2)
		}, want: []string{"run=false"}},
		{name: "an incremental review resets the count", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			threeCommits(f)
			f.comment(gateBotLogin, "2026-09-25T14:00:00Z", guideIncremental)
			f.commit("2026-09-25T15:00:00Z")
		}, want: []string{"run=false", "1 new commit"}},
		{name: "a review is recognised by its heading alone", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", "## PR Reviewer Guide\n\nno identity line")
			f.commit("2026-09-25T11:00:00Z")
		}, want: []string{"run=false"}},
		{name: "prose is not a review, nor an identity past line 5", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			threeCommits(f)
			f.comment("mlorentedev", "2026-09-25T14:00:00Z", "The ## PR Reviewer Guide above is triaged.")
			f.comment("mlorentedev", "2026-09-25T14:30:00Z", "a\nb\nc\nd\ne\n<!-- pr-agent:review:full -->")
		}, want: []string{"run=true", "mode=incremental"}},
		{name: "an identity line with CRLF endings", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", "## Something else\r\n\r\n<!-- pr-agent:review:full -->\r\n")
			f.commit("2026-09-25T11:00:00Z")
		}, want: []string{"run=false"}},
		{name: "no previous review", build: func(f *gateFixture) {
			f.comment("mlorentedev", "2026-09-25T10:00:00Z", "a note")
			f.commit("2026-09-25T11:00:00Z")
		}, want: []string{"run=true", "no previous review", "mode=full"}},
		{name: "a non-bot marker is never the baseline (CWE-345)", build: func(f *gateFixture) {
			f.comment("randomuser", "2026-09-25T10:00:00Z", guideFull)
			f.commit("2026-09-25T09:00:00Z")
		}, want: []string{"run=true", "no previous review", "mode=full"}},
		{name: "a forged marker newer than the review forces a full review", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.commit("2026-09-25T11:00:00Z")
			f.comment("randomuser", "2026-09-25T12:00:00Z", guideFull)
		}, want: []string{"run=true", "mode=full"}},
		{name: "a head_sha anchors the count by position", build: func(f *gateFixture) {
			f.commit("2026-09-25T08:00:00Z")
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", reviewWithSHA("c0"))
			f.commit("2026-09-25T09:00:00Z")
			f.commit("2026-09-25T09:15:00Z")
			f.commit("2026-09-25T09:30:00Z")
		}, want: []string{"run=true", "mode=incremental"}},
		{name: "a head_sha respects the threshold", build: func(f *gateFixture) {
			f.commit("2026-09-25T08:00:00Z")
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", reviewWithSHA("c0"))
			f.commit("2026-09-25T09:00:00Z")
			f.commit("2026-09-25T09:15:00Z")
		}, want: []string{"run=false", "2 new commit"}},
		{name: "a head_sha absent from the PR", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", reviewWithSHA("deadbeef"))
			f.commit("2026-09-25T11:00:00Z")
		}, want: []string{"run=true", "mode=full", "rebase"}},
		{name: "a state block in an incremental review is quoted text",
			build: quotedBlock(guideIncremental + "\n\n<!-- pr-agent-review-state:v1\n{\"last_run\":{\"head_sha\":\"c3\"}}\n-->"),
			want:  []string{"run=true", "mode=incremental", "since the review of"}},
		{name: "a state block followed by more text is quoted text",
			build: quotedBlock(guideFull + "\n\n```html\n<!-- pr-agent-review-state:v1\n{\"last_run\":{\"head_sha\":\"c3\"}}\n-->\n```\nmore findings"),
			want:  []string{"run=true", "since the review of"}},
		{name: "a state block without its closing marker is quoted text",
			build: quotedBlock(guideFull + "\n\nquoted: pr-agent-review-state:v1\n{\"last_run\":{\"head_sha\":\"c3\"}}"),
			want:  []string{"run=true", "since the review of"}},
		{name: "a state block without a head_sha counts by date", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull+"\n\n<!-- pr-agent-review-state:v1\n{\"schema_version\":1}\n-->")
			threeCommits(f)
		}, want: []string{"run=true", "mode=incremental", "since the review of 2026-09-25T10:00:00Z"}},
		{name: "a head_sha review with unreadable commits", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", reviewWithSHA("c0"))
			f.rawCommits = "not json"
		}, want: []string{"run=true", "mode=full"}},
		{name: "a push by a bot is skipped", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			threeCommits(f)
		}, sender: "Bot", want: []string{"run=false", "bot"}},
		{name: "absent comments review", build: func(f *gateFixture) { f.missingComments = true },
			want: []string{"run=true", "mode=full", "comments cannot be read"}},
		{name: "unreadable commits review", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.rawCommits = "not json"
		}, want: []string{"run=true", "mode=full", "commits cannot be read"}},
		{name: "--threshold changes the bound", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.commit("2026-09-25T11:00:00Z")
		}, threshold: 1, want: []string{"run=true"}},
		{name: "a rebase with no state block counts committer dates", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.rebased("2026-09-25T08:00:00Z", "2026-09-25T11:00:00Z")
			f.rebased("2026-09-25T08:30:00Z", "2026-09-25T11:00:00Z")
			f.rebased("2026-09-25T09:00:00Z", "2026-09-25T11:00:00Z")
		}, want: []string{"run=true", "3 new commit"}},
		{name: "a forged marker in the link-reference form", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.comment("someone", "2026-09-25T11:00:00Z", "[pr-agent:review:full]: https://github.com/The-PR-Agent/pr-agent\n\nlooks fine")
			f.commit("2026-09-25T12:00:00Z")
		}, want: []string{"run=true", "mode=full"}},
		{name: "the bot's review in the link-reference form", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", "[pr-agent:review:incremental]: https://github.com/The-PR-Agent/pr-agent\n\nmore")
			f.commit("2026-09-25T11:00:00Z")
		}, want: []string{"run=false", "1 new commit"}},
	}
}

// edgeGateCases pin what the script's jq does where no bats case looks. Their
// expected output is edgeGateWants, in gate_edge_test.go.
func edgeGateCases() []gateCase {
	bot := func(body string, then ...func(*gateFixture)) func(*gateFixture) {
		return func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", body)
			for _, g := range then {
				g(f)
			}
		}
	}
	block := func(state string) string {
		return guideFull + "\n\n<!-- pr-agent-review-state:v1\n" + state + "\n-->"
	}
	return []gateCase{
		{name: "the baseline is the last review by position, not by date", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T14:00:00Z", guideFull)
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.commit("2026-09-25T11:00:00Z")
		}},
		{name: "a marker older than the latest review is spent", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.comment("eve", "2026-09-25T11:00:00Z", guideFull)
			f.comment(gateBotLogin, "2026-09-25T12:00:00Z", guideIncremental)
			f.commit("2026-09-25T13:00:00Z")
		}},
		{name: "a commit dated at the review is not new", build: bot(guideFull, threeCommits, func(f *gateFixture) {
			f.commit("2026-09-25T10:00:00Z")
		})},
		{name: "a commit dated at the review, three of them", build: bot(guideFull, func(f *gateFixture) {
			f.commit("2026-09-25T10:00:00Z")
			f.commit("2026-09-25T10:00:00Z")
			f.commit("2026-09-25T10:00:00Z")
		})},
		{name: "the committer date wins over a later author date", build: bot(guideFull, func(f *gateFixture) {
			f.rebased("2026-09-25T11:00:00Z", "2026-09-25T09:00:00Z")
		})},
		{name: "commits without parents count", build: bot(guideFull, func(f *gateFixture) {
			f.commits = append(f.commits, map[string]any{"sha": "x",
				"commit": map[string]any{"author": map[string]any{"date": "2026-09-25T11:00:00Z"}}})
		})},
		{name: "a commit with exactly two parents is a merge", build: bot(guideFull, func(f *gateFixture) {
			f.commit("2026-09-25T11:00:00Z", 2)
			f.commit("2026-09-25T11:00:00Z", 2)
			f.commit("2026-09-25T11:00:00Z", 2)
		})},
		{name: "a root commit counts", build: bot(guideFull, func(f *gateFixture) {
			f.commit("2026-09-25T11:00:00Z", 0)
		})},
		{name: "merges after the head_sha do not count", build: func(f *gateFixture) {
			f.commit("2026-09-25T08:00:00Z")
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", reviewWithSHA("c0"))
			threeCommits(f)
			f.commit("2026-09-25T14:00:00Z", 2)
		}},
		{name: "a head_sha at the tip has nothing new", build: func(f *gateFixture) {
			f.commit("2026-09-25T08:00:00Z")
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", reviewWithSHA("c0"))
		}},
		{name: "an identity on line 5 marks a review", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", "a\nb\nc\nd\n<!-- pr-agent:review:full -->")
			f.commit("2026-09-25T11:00:00Z")
		}},
		{name: "an identity on line 6 does not", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", "a\nb\nc\nd\ne\n<!-- pr-agent:review:full -->")
			f.commit("2026-09-25T11:00:00Z")
		}},
		{name: "an indented heading is not a review", build: bot(" ## PR Reviewer Guide")},
		{name: "an identity behind NBSP and NEL", build: bot("\u00a0<!-- pr-agent:review:full -->\u0085",
			func(f *gateFixture) { f.commit("2026-09-25T11:00:00Z") })},
		{name: "a state block under the link-reference identity", build: func(f *gateFixture) {
			f.commit("2026-09-25T08:00:00Z")
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", "[pr-agent:review:full]: https://github.com/The-PR-Agent/pr-agent\n\n"+
				"<!-- pr-agent-review-state:v1\n{\"last_run\":{\"head_sha\":\"c0\"}}\n-->")
			f.commit("2026-09-25T09:00:00Z")
		}},
		{name: "the last of two state blocks is the one read", build: func(f *gateFixture) {
			f.commit("2026-09-25T08:00:00Z")
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", block(`{"last_run":{"head_sha":"zz"}}`)+
				"\n\n<!-- pr-agent-review-state:v1 {\"last_run\":{\"head_sha\":\"c0\"}} -->")
		}},
		{name: "a block with trailing blank lines", build: func(f *gateFixture) {
			f.commit("2026-09-25T08:00:00Z")
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", reviewWithSHA("c0")+"\n\n")
		}},
		{name: "malformed state JSON counts by date", build: bot(block(`{"last_run":`), threeCommits)},
		{name: "an empty head_sha counts by date", build: bot(block(`{"last_run":{"head_sha":""}}`), threeCommits)},
		{name: "a null head_sha counts by date", build: bot(block(`{"last_run":{"head_sha":null}}`), threeCommits)},
		{name: "a last_run that is not an object counts by date", build: bot(block(`{"last_run":"s"}`), threeCommits)},
		{name: "keys are case-sensitive", build: bot(block(`{"last_run":{"HEAD_SHA":"deadbeef"}}`), threeCommits)},
		{name: "a comment by a deleted account", build: func(f *gateFixture) {
			f.rawComments = `[{"user":null,"created_at":"2026-09-25T09:00:00Z","body":"hi"}]`
		}},
		{name: "a comment with a null body", build: func(f *gateFixture) {
			f.rawComments = `[{"user":{"login":"github-actions[bot]"},"created_at":"2026-09-25T09:00:00Z","body":null}]`
		}},
		{name: "a comments listing of nulls", build: func(f *gateFixture) { f.rawComments = "[null]" }},
		{name: "comments that are null", build: func(f *gateFixture) { f.rawComments = "null" }},
		{name: "comments that are numbers", build: func(f *gateFixture) { f.rawComments = "[1]" }},
		{name: "commits that are numbers", build: bot(guideFull, func(f *gateFixture) { f.rawCommits = "[1]" })},
		{name: "commits that are null", build: bot(guideFull, func(f *gateFixture) { f.rawCommits = "null" })},
		{name: "absent commits with a review", build: bot(guideFull, func(f *gateFixture) { f.missingCommits = true })},
		{name: "absent commits with no review", build: func(f *gateFixture) { f.missingCommits = true }},
		{name: "a bot push never reads the input", build: func(f *gateFixture) {
			f.missingComments, f.missingCommits = true, true
		}, sender: "Bot"},
		{name: "the sender type is case-sensitive", build: bot(guideFull), sender: "bot"},
		{name: "a threshold of zero always runs", build: bot(guideFull), zero: true},
	}
}

func runGateCase(t *testing.T, c gateCase) string {
	t.Helper()
	f := &gateFixture{}
	c.build(f)
	comments, commits := f.files(t)
	return Gate(GateInputs{Comments: comments, Commits: commits, Threshold: c.limit(), SenderType: c.sender}).Output()
}

func TestGate(t *testing.T) {
	for _, c := range portedGateCases() {
		t.Run(c.name, func(t *testing.T) {
			got := runGateCase(t, c)
			for _, want := range c.want {
				if !strings.Contains(got, want) {
					t.Errorf("output = %q, want it to contain %q", got, want)
				}
			}
		})
	}
}

// What the port decides differently from the script, on purpose. Each is a gap
// in the script's inputs: run=false, or a forged output line, that no data
// justified. The PR that added this lists them.
func TestGate_Divergences(t *testing.T) {
	for _, c := range []gateCase{
		// The script interpolates head_sha into the reason, so a newline in it
		// wrote a second line to $GITHUB_OUTPUT, and the last run= wins.
		{name: "a newline in head_sha cannot forge an output line", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull+
				"\n\n<!-- pr-agent-review-state:v1\n{\"last_run\":{\"head_sha\":\"x\\nrun=false\\r\\nmode=y\"}}\n-->")
			f.commit("2026-09-25T09:00:00Z")
		}, want: []string{"run=true\nreason=the reviewed commit (x run=false  mode=y) is no longer", "mode=full\n"}},
		// The script compares with >, so a marker stamped in the review's own
		// second was missed; PR-Agent picks by position and would take it.
		{name: "a forged marker in the review's own second", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.comment("eve", "2026-09-25T10:00:00Z", guideFull)
			f.commit("2026-09-25T11:00:00Z")
		}, want: []string{"run=true", "mode=full", "non-bot commenter"}},
		// A date the API did not send is not an old date.
		{name: "a review without created_at is unreadable", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "", guideFull)
			f.commit("2026-09-25T11:00:00Z")
		}, want: []string{"run=true", "mode=full", "comments cannot be read"}},
		{name: "a commit without a date is unreadable", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull)
			f.commitOf(map[string]any{"message": "change"})
		}, want: []string{"run=true", "mode=full", "commits cannot be read"}},
		// jq takes `{}` and an empty file as no comments; neither is a listing.
		{name: "comments that are not a list", build: func(f *gateFixture) { f.rawComments = "{}" },
			want: []string{"run=true", "mode=full", "comments cannot be read"}},
		// jq prints a number as text, and it then matches no commit.
		{name: "a head_sha that is not a string counts by date", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull+
				"\n\n<!-- pr-agent-review-state:v1\n{\"last_run\":{\"head_sha\":123}}\n-->")
			threeCommits(f)
		}, want: []string{"mode=incremental", "since the review of"}},
		// jq reads two documents and joins their answers; the block is one object.
		{name: "two documents in a state block count by date", build: func(f *gateFixture) {
			f.comment(gateBotLogin, "2026-09-25T10:00:00Z", guideFull+
				"\n\n<!-- pr-agent-review-state:v1\n{\"last_run\":{\"head_sha\":\"a\"}} {\"last_run\":{\"head_sha\":\"b\"}}\n-->")
			threeCommits(f)
		}, want: []string{"mode=incremental", "since the review of"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := runGateCase(t, c)
			for _, want := range c.want {
				if !strings.Contains(got, want) {
					t.Errorf("output = %q, want it to contain %q", got, want)
				}
			}
		})
	}
}

func TestGate_OutputIsKeyValueLines(t *testing.T) {
	for name, tc := range map[string]struct {
		d    GateDecision
		want string
	}{
		"a skip has two lines": {GateDecision{Reason: "r"}, "run=false\nreason=r\n"},
		"a review has a mode":  {GateDecision{Run: true, Reason: "r", Mode: "full"}, "run=true\nreason=r\nmode=full\n"},
		"a reason is one line": {GateDecision{Reason: "a\nb\rc"}, "run=false\nreason=a b c\n"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := tc.d.Output(); got != tc.want {
				t.Errorf("Output() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The gate's rules are audited against upstream source identities, never
// against another hand-kept copy of the action's version (#1893, #2010). The
// script reads no value from the contract at run time and neither does the
// port: these two tests are that case of the bats suite, and the contract's
// promise that the sources the rules mirror are pinned.
func TestGate_SourceCitesTheContractAndNoVersion(t *testing.T) {
	raw, err := os.ReadFile("gate.go")
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`v0\.[0-9]+\.[0-9]+`).Match(raw) {
		t.Error("gate.go cites a PR-Agent version; cite the audited contract instead")
	}
	if !strings.Contains(string(raw), "pr-agent-upstream-contract.json") {
		t.Error("gate.go does not name harness/pr-agent-upstream-contract.json")
	}
}

func TestGate_ContractPinsTheSourcesItsRulesMirror(t *testing.T) {
	raw, err := os.ReadFile(repoFile("harness", "pr-agent-upstream-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"pr_agent/algo/comment_identity.py", "pr_agent/git_providers/github_provider.py",
		"pr_agent/algo/review_finding_state.py", "pr_agent/tools/pr_reviewer.py"} {
		if contract.Files[f] == "" {
			t.Errorf("the contract pins no identity for %s, a source the gate mirrors", f)
		}
	}
}
