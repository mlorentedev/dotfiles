package spec

import "testing"

const homeRepo = "mlorentedev/dotfiles"

func frontmatter(issueLine string) string {
	return "---\nid: \"X-001-demo\"\ntype: spec\n" + issueLine + "\ntags: [spec]\n---\n\n# X-001-demo\n"
}

func TestIssueStateResolveFrontmatter(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		want    IssueRef
		wantErr bool
	}{
		{"full slug, scaffolded with a trailing comment", `issue: "mlorentedev/dotfiles#1087"   # repo#NNN — GitHub issue`, IssueRef{"mlorentedev/dotfiles", 1087}, false},
		{"full slug, cross-repo", `issue: "mlorentedev/knowledge#90"`, IssueRef{"mlorentedev/knowledge", 90}, false},
		{"name-only shorthand takes the home owner", `issue: "dotfiles#1486"   # repo#NNN`, IssueRef{"mlorentedev/dotfiles", 1486}, false},
		{"bare number is the home repo", `issue: "#42"`, IssueRef{"mlorentedev/dotfiles", 42}, false},
		{"unquoted value", `issue: mlorentedev/hive#267`, IssueRef{"mlorentedev/hive", 267}, false},
		{"issue URL", `issue: "https://github.com/mlorentedev/hive/issues/380"`, IssueRef{"mlorentedev/hive", 380}, false},
		{"placeholder is malformed, not absent", `issue: "TBD"`, IssueRef{}, true},
		{"two refs is malformed", `issue: "#1 #2"`, IssueRef{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, src, err := ResolveIssueLink(frontmatter(tc.line), homeRepo)
			if src != LinkFrontmatter {
				t.Fatalf("source = %v, want LinkFrontmatter", src)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want an error for %q, got %v", tc.line, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIssueStateResolveEmptyFrontmatterIsUnlinked(t *testing.T) {
	for _, line := range []string{`issue: ""`, `issue: ""   # repo#NNN — GitHub issue`, `status: draft`} {
		got, src, err := ResolveIssueLink(frontmatter(line), homeRepo)
		if err != nil || src != LinkNone || got != (IssueRef{}) {
			t.Errorf("%q: got (%v, %v, %v), want an unlinked spec", line, got, src, err)
		}
	}
}

// Every positive shape below was measured in a real proposal.md on 2026-09-23.
// The spec folder it came from is named so a future reader can re-check it.
func TestIssueStateResolveProse(t *testing.T) {
	cases := []struct {
		name string
		home string
		line string
		want IssueRef
	}{
		{"GOV-004: labelled list item, markdown link", homeRepo,
			"- GH issue: [#673](https://github.com/mlorentedev/dotfiles/issues/673) (parts b+c in #728)",
			IssueRef{"mlorentedev/dotfiles", 673}},
		{"hive FEAT-015: quoted and bold label", "mlorentedev/hive",
			"> **Issue:** [mlorentedev/hive#380](https://github.com/mlorentedev/hive/issues/380)",
			IssueRef{"mlorentedev/hive", 380}},
		{"AI-022: another repo of the same owner", homeRepo,
			"- hive issue: [mlorentedev/hive#176](https://github.com/mlorentedev/hive/issues/176)",
			IssueRef{"mlorentedev/hive", 176}},
		{"IDEAS-007: bare GH label with an autolink", homeRepo,
			"- GH: <https://github.com/mlorentedev/dotfiles/issues/103>.",
			IssueRef{"mlorentedev/dotfiles", 103}},
		{"SDD-007: code-spanned bare number", homeRepo,
			"- Issue: GitHub `#100` (Antigravity CLI Circular Symlink Recursion) — closed by AC4.",
			IssueRef{"mlorentedev/dotfiles", 100}},
		{"hive HIVE-115: plural label takes the first", "mlorentedev/hive",
			"- Issues: [#110](https://github.com/mlorentedev/hive/issues/110), [#111](https://github.com/mlorentedev/hive/issues/111)",
			IssueRef{"mlorentedev/hive", 110}},
		{"hive HIVE-97: snake_case label", "mlorentedev/hive",
			"  - github_issue: https://github.com/mlorentedev/hive/issues/97",
			IssueRef{"mlorentedev/hive", 97}},
		{"hive HIVE-119: GitHub issue label", "mlorentedev/hive",
			"- GitHub issue: [#151](https://github.com/mlorentedev/hive/issues/151)",
			IssueRef{"mlorentedev/hive", 151}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := frontmatter(`status: draft`) + "\n## References\n\n" + tc.line + "\n"
			got, src, err := ResolveIssueLink(doc, tc.home)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if src != LinkProse {
				t.Fatalf("source = %v, want LinkProse", src)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// Each of these names an issue that is NOT the spec's tracking issue. Reading
// any of them as one would audit the wrong issue, or a project we do not own.
func TestIssueStateResolveProseRejectsNonTrackingRefs(t *testing.T) {
	cases := []struct{ name, line string }{
		{"BUG-004: upstream project", "- Upstream issue: [`anthropics/claude-code#59870`](https://github.com/anthropics/claude-code/issues/59870)"},
		{"hive HIVE-104: related upstream", "- Related upstream issue: modelcontextprotocol/python-sdk#2610 (target for removal)"},
		{"AI-020: sister issue", "- Sister sunset issue: `agy` (Antigravity) was auto-installed via PR #121 last session"},
		{"WORKMODE-001: unlabelled ref in running text", "- **Concrete regression (GH #197):** `kubelab` is a personal project"},
		{"prose sentence", "This was first reported in #455 and fixed in #460."},
		{"another owner under a tracking label", "- GH issue: [other/repo#12](https://github.com/other/repo/issues/12)"},
		// Same owner and a one-word qualifier, so only the non-tracking word
		// keeps it from being read as this spec's tracker.
		{"same-owner related issue", "- Related issue: [#455](https://github.com/mlorentedev/dotfiles/issues/455)"},
		{"same-owner upstream issue", "- Upstream issue: mlorentedev/hive#12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := frontmatter(`status: draft`) + "\n" + tc.line + "\n"
			got, src, err := ResolveIssueLink(doc, homeRepo)
			if err != nil || src != LinkNone || got != (IssueRef{}) {
				t.Fatalf("got (%v, %v, %v), want no link", got, src, err)
			}
		})
	}
}

func TestIssueStateFrontmatterWinsOverProse(t *testing.T) {
	doc := frontmatter(`issue: "mlorentedev/dotfiles#1087"`) + "\n- GH issue: [#673](https://github.com/mlorentedev/dotfiles/issues/673)\n"
	got, src, err := ResolveIssueLink(doc, homeRepo)
	if err != nil || src != LinkFrontmatter || got != (IssueRef{"mlorentedev/dotfiles", 1087}) {
		t.Fatalf("got (%v, %v, %v), want the frontmatter link", got, src, err)
	}
}
