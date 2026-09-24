package spec

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// W1.4 precondition (#1625, SDD-042 AC4). The bypass flags used to leave no
// durable trace — Archive simply skipped the check — so a sweep that bans them
// could only be audited by trusting commit messages. A bypass now needs a
// stated reason, and the archived proposal.md records what was overridden.

func bypassRecords(t *testing.T, archived string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(archived, "proposal.md"))
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "review_bypass:") {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestArchiveBypassRecordedRequiresReason(t *testing.T) {
	for _, opts := range []ArchiveOptions{{ForceWithoutReview: true}, {ForceWithDrafts: true}, {ForceWithoutReview: true, BypassReason: "   "}} {
		root := t.TempDir()
		writeSpec(t, root, "AI-001-x", map[string]string{"proposal.md": "---\nstatus: implementing\n---\n"})
		_, err := Archive(root, "AI-001-x", opts)
		if err == nil || !strings.Contains(err.Error(), "--reason") {
			t.Errorf("%+v: a bypass without a reason must be refused, naming --reason: %v", opts, err)
		}
		if _, statErr := os.Stat(filepath.Join(root, "specs", "AI-001-x")); statErr != nil {
			t.Errorf("%+v: a refused archive must not move the spec", opts)
		}
	}
}

func TestArchiveBypassRecordedWhatTheGateWouldHaveRefused(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{"proposal.md": "---\nid: \"AI-001-x\"\nstatus: implementing\n---\n# body\n"})
	target, err := Archive(root, "AI-001-x", ArchiveOptions{
		ForceWithoutReview: true, BypassReason: `retro spec for #906, "shipped" in 4f1c2d0`, Date: "2026-09-23",
	})
	if err != nil {
		t.Fatal(err)
	}
	recs := bypassRecords(t, target)
	if len(recs) != 1 {
		t.Fatalf("want exactly one review_bypass: line, got %v", recs)
	}
	f := frontmatterFields(readProposal(t, target))
	got := f["review_bypass"]
	for _, want := range []string{"force-without-review", "overrode: no review.md", `retro spec for #906, "shipped" in 4f1c2d0`, "2026-09-23"} {
		if !strings.Contains(got, want) {
			t.Errorf("review_bypass should record %q, got %q", want, got)
		}
	}
	if f["status"] != "archived" {
		t.Errorf("the status rewrite must survive the record: %v", f)
	}
}

// A bypass that overrode nothing still leaves its record — "the flag was not
// needed" is itself worth knowing, and the record must never claim a refusal
// that did not happen.
func TestArchiveBypassRecordedWhenNothingWasOverridden(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: implementing\n---\n",
		ReviewFile:    "---\nspec: \"AI-001-x\"\nverdict: \"PASS\"\nreviewed_sha: \"abc\"\n---\n",
	})
	target, err := Archive(root, "AI-001-x", ArchiveOptions{
		ForceWithoutReview: true, BypassReason: "belt and braces", Date: "2026-09-23",
		Staleness: fakeStaleness{known: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := frontmatterFields(readProposal(t, target))["review_bypass"]; !strings.Contains(got, "overrode: nothing") {
		t.Errorf("a bypass the gate did not need must say so, got %q", got)
	}
}

func TestArchiveBypassRecordedBothFlags(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: implementing\n---\n<!-- [AGENT-DRAFT] undecided -->\n",
	})
	target, err := Archive(root, "AI-001-x", ArchiveOptions{
		ForceWithDrafts: true, ForceWithoutReview: true, BypassReason: "abandoned draft", Date: "2026-09-23",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := frontmatterFields(readProposal(t, target))["review_bypass"]
	for _, want := range []string{"force-with-drafts", "force-without-review", "1 unresolved draft tag", "no review.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("review_bypass should record %q, got %q", want, got)
		}
	}
}

func TestArchiveBypassRecordedRefusesWithoutAProposal(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{"tasks.md": "- [ ] x\n"})
	_, err := Archive(root, "AI-001-x", ArchiveOptions{ForceWithoutReview: true, BypassReason: "r"})
	if err == nil || !strings.Contains(err.Error(), "proposal.md") {
		t.Fatalf("a bypass with nowhere to record it must be refused: %v", err)
	}
}

func TestArchiveBypassRecordedInAProposalWithoutFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{"proposal.md": "# a pre-template proposal\n"})
	target, err := Archive(root, "AI-001-x", ArchiveOptions{ForceWithoutReview: true, BypassReason: "legacy", Date: "2026-09-23"})
	if err != nil {
		t.Fatal(err)
	}
	if got := frontmatterFields(readProposal(t, target))["review_bypass"]; !strings.Contains(got, "legacy") {
		t.Fatalf("the record must be readable as frontmatter even when the proposal had none, got %q", got)
	}
}

func readProposal(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "proposal.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// SDD-042 AC5 ([QW] R-2): a refusal names the recovery path, never the bypass.
// During the W1.4 sweep the bypass flags are banned, and a refusal that
// advertises one invites exactly the move the sweep forbids. The flags stay
// documented in `dotf spec archive --help`, which also says they are recorded.
//
// Checked at the source, so every refusal path is covered — including ones no
// fixture below happens to reach. The single allowed site is the message that
// names the flags because the operator just typed them.
func TestArchiveRefusalsNameNoBypassFlag(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	checked := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name == "checkBypassRequest" {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				checked++
				if strings.Contains(lit.Value, "--force-with") {
					t.Errorf("%s: %s names a bypass flag in a message: %s", fset.Position(lit.Pos()), fn.Name.Name, lit.Value)
				}
				return true
			})
		}
	}
	if checked == 0 {
		t.Fatal("no string literal was inspected — the source walk is broken")
	}

	// And by behaviour, on the refusals the sweep will actually meet.
	cases := map[string]map[string]string{
		"draft tags":         {"proposal.md": "<!-- [AGENT-DRAFT] x -->\n"},
		"no review":          {"proposal.md": "---\nstatus: verifying\n---\n"},
		"FAIL verdict":       {"proposal.md": "x\n", ReviewFile: "---\nspec: \"AI-001-x\"\nverdict: \"FAIL\"\nreviewed_sha: \"abc\"\n---\n"},
		"reasonless waiver":  {"proposal.md": "---\nreview: waived\n---\n"},
		"wrong spec in file": {"proposal.md": "x\n", ReviewFile: "---\nspec: \"B-002-y\"\nverdict: \"PASS\"\nreviewed_sha: \"abc\"\n---\n"},
	}
	for name, files := range cases {
		root := t.TempDir()
		writeSpec(t, root, "AI-001-x", files)
		_, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{known: true}})
		if err == nil {
			t.Errorf("%s: expected a refusal", name)
			continue
		}
		if strings.Contains(err.Error(), "--force") {
			t.Errorf("%s: the refusal advertises a bypass flag:\n%s", name, err)
		}
	}
}
