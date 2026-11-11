package gate

import (
	"reflect"
	"testing"
)

// The cases are the shapes scripts/check-spec-gate.sh's comments name; the
// end-to-end verdicts are held by its bats suites once dotf gate spec runs
// through them.

func TestStripCode(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"prose stays", "Closes #1", "Closes #1"},
		{"inline span goes", "see `Closes #2` here", "see  here"},
		{"two spans", "`a` b `c`", " b "},
		{"empty span", "x``y", "xy"},
		{"unpaired backtick stays", "a ` b", "a ` b"},
		{"fenced block goes", "a\n```\nCloses #3\n```\nb", "a\nb"},
		{"tilde fence", "~~~\nCloses #3\n~~~", ""},
		{"indented up to three", "   ```\nx\n   ```\ny", "y"},
		{"a tab before the marker opens too", "\t```\nx\n```\ny", "y"},
		{"shorter marker does not close", "````\n```\nx\n````\ny", "y"},
		{"other character does not close", "```\n~~~\nx\n```\ny", "y"},
		// The spec gate closes on any long-enough marker, info string or not
		// (#2407 records the drift from the knowledge gate).
		{"marker with info string closes", "```\nx\n```js\ny", "y"},
		{"unclosed fence hides the rest", "a\n```\nCloses #4", "a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StripCode(c.in); got != c.want {
				t.Errorf("StripCode(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestClosingIssueNumbers(t *testing.T) {
	const slug = "me/repo"
	cases := []struct {
		name, slug, body string
		want             []string
	}{
		{"every verb form", slug, "close #1 closes #2 closed #3 fix #4 fixes #5 fixed #6 resolve #7 resolves #8 resolved #9",
			[]string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}},
		{"case and colon", slug, "CLOSES: #10", []string{"10"}},
		{"refs does not close", slug, "Refs #11, part of #12", nil},
		{"own qualified", slug, "Closes me/repo#13", []string{"13"}},
		{"other repo ignored", slug, "Closes other/repo#14", nil},
		{"qualified with no slug ignored", "", "Closes me/repo#15 and fixes #16", []string{"16"}},
		{"full URL folds", slug, "Fixes https://github.com/me/repo/issues/17", []string{"17"}},
		{"other repo URL ignored", slug, "Fixes https://github.com/x/y/issues/18", nil},
		{"inside code ignored", slug, "`Closes #19`\n```\nCloses #20\n```", nil},
		{"word boundary", slug, "disclosed #21 prefix", nil},
		{"no line crossing", slug, "Closes\n#22", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ClosingIssueNumbers(c.slug, c.body); !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestFrontmatterIssue(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"bare", "---\nissue: 42\n---\n", "42"},
		{"bare with comment", "---\nissue: 42 # tracked\n---\n", "42"},
		{"owner/repo quoted", "---\nissue: \"me/repo#43\" # x\n---\n", "43"},
		{"repo quoted", "---\nissue: \"repo#44\"\n---\n", "44"},
		{"no frontmatter", "issue: 45\n", ""},
		{"after frontmatter", "---\nid: x\n---\nissue: 46\n", ""},
		{"not a number", "---\nissue: tbd\n---\n", ""},
		{"absent", "---\nid: x\n---\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FrontmatterIssue(c.in); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestExcludedFromLOC(t *testing.T) {
	cases := map[string]bool{
		"docs/adr/adr-001.md":             true,
		"docs/runbooks/a/b.md":            true,
		"docs/diagram.svg":                false,
		"tests/x.bats":                    true,
		"specs/archive/X/proposal.md":     true,
		".agent/runs/r1/report.md":        true,
		".agent/runs/r1/deep/report.md":   false,
		".agent/unattended.yml":           false,
		".agent/runs/report.md":           false,
		"cli/internal/x/x_test.go":        true,
		"cli/go.sum":                      true,
		"web/package-lock.json":           true,
		"bun.lockb":                       true,
		"sub/.gitignore":                  true,
		"CHANGELOG.md":                    true,
		"fix.patch":                       true,
		"a.rej":                           true,
		"cli/internal/generated_names.go": false,
		"scripts/x.sh":                    false,
		"README.md":                       false,
	}
	for p, want := range cases {
		if got := ExcludedFromLOC(p); got != want {
			t.Errorf("ExcludedFromLOC(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestSpecPaths(t *testing.T) {
	active := map[string]bool{
		"specs/X/proposal.md":         true,
		"specs/X/deep/a.md":           true,
		"specs/README.md":             false,
		"specs/archive/X/proposal.md": false,
		"docs/specs/X/a.md":           false,
	}
	for p, want := range active {
		if got := IsActiveSpecPath(p); got != want {
			t.Errorf("IsActiveSpecPath(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestNormalizeRenamePath(t *testing.T) {
	cases := map[string]string{
		"specs/{ => archive}/X/proposal.md": "specs/archive/X/proposal.md",
		"specs/{X => archive/X}/tasks.md":   "specs/archive/X/tasks.md",
		"old/a.sh => new/b.sh":              "new/b.sh",
		"a/{b => }/c.sh":                    "a/c.sh",
		"plain/path.go":                     "plain/path.go",
	}
	for in, want := range cases {
		if got := NormalizeRenamePath(in); got != want {
			t.Errorf("NormalizeRenamePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseNumstat(t *testing.T) {
	out := "3\t1\tscripts/a.sh\n-\t-\tbin/blob\n2\t0\tspecs/{ => archive}/X/proposal.md\n\n"
	want := []FileStat{
		{Path: "scripts/a.sh", Added: 3, Removed: 1},
		{Path: "bin/blob"},
		{Path: "specs/archive/X/proposal.md", Added: 2},
	}
	got, err := ParseNumstat(out)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, %v; want %+v", got, err, want)
	}
	if _, err := ParseNumstat("x\t1\ta\n"); err == nil {
		t.Error("a non-numeric count must be an error, not a zero")
	}
}
