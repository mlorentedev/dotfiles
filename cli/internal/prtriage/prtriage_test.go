package prtriage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func at(min int) time.Time {
	return time.Date(2026, 8, 18, 1, min, 0, 0, time.UTC)
}

func reg() Registry {
	var r Registry
	r.Reviewers = []Reviewer{
		{Login: "coderabbitai", ReviewMarkers: nil},
		{Login: "github-actions", ReviewMarkers: []string{"## PR Reviewer Guide"}},
	}
	r.Triage.Marker = "## Review triage"
	r.Triage.Standing = DefaultStanding
	return r
}

// AC7/AC8: the queue's whole contract, as a table. Each row is a state the loop
// actually reaches, not a synthetic permutation.
func TestEvaluate(t *testing.T) {
	cases := []struct {
		name     string
		comments []Comment
		pending  bool
	}{
		{
			name:     "no comments at all",
			comments: nil,
			pending:  false,
		},
		{
			name: "a reviewer spoke and nobody triaged",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfindings", CreatedAt: at(10)},
			},
			pending: true,
		},
		{
			name: "triaged after the review",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfindings", CreatedAt: at(10)},
				{Author: "manu", AuthorAssociation: "OWNER", Body: "## Review triage\n| item | applied |", CreatedAt: at(20)},
			},
			pending: false,
		},
		{
			// The case that matters. Pushing a fix makes the reviewer re-review,
			// and the earlier disposition does not cover what it said this time.
			name: "the reviewer spoke again after the triage",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfirst", CreatedAt: at(10)},
				{Author: "manu", AuthorAssociation: "OWNER", Body: "## Review triage\ndone", CreatedAt: at(20)},
				{Author: "github-actions", Body: "## PR Reviewer Guide\nsecond", CreatedAt: at(30)},
			},
			pending: true,
		},
		{
			// TOOL-019 (#1422). The same event as the row above, in the shape both
			// reviewers actually produce it: ONE comment, edited in place. The body
			// grows new findings and createdAt does not move, so a queue reading
			// createdAt sees 10 < 20 and reports clear over unread findings.
			// Measured on #1543: CodeRabbit created 02:52:56, updated 03:28:29,
			// three real findings in the edit.
			name: "the reviewer edited its comment in place after the triage",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfirst, then more",
					CreatedAt: at(10), UpdatedAt: at(30)},
				{Author: "manu", AuthorAssociation: "OWNER", Body: "## Review triage\ndone", CreatedAt: at(20)},
			},
			pending: true,
		},
		{
			// The asymmetry, asserted rather than left to the comment. Editing a
			// TRIAGE must not clear a queue: if it counted, a typo fix on an old
			// disposition would silently dismiss findings the reviewer added since,
			// and anyone could empty the queue with one keystroke and no reading.
			name: "editing the triage does not dismiss a review that came after it",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfindings",
					CreatedAt: at(10), UpdatedAt: at(30)},
				{Author: "manu", AuthorAssociation: "OWNER", Body: "## Review triage\ndone (typo fixed later)",
					CreatedAt: at(20), UpdatedAt: at(40)},
			},
			pending: true,
		},
		{
			// Why Spoken() is max() and not UpdatedAt: an unedited comment carries a
			// zero UpdatedAt, and reading that as "spoken at the epoch" would answer
			// "nothing pending" for the whole queue -- the one direction this
			// package must never fail in.
			name: "a never-edited comment (zero UpdatedAt) still counts as spoken",
			comments: []Comment{
				{Author: "manu", AuthorAssociation: "OWNER", Body: "## Review triage\ndone", CreatedAt: at(20)},
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfindings", CreatedAt: at(30)},
			},
			pending: true,
		},
		{
			// GraphQL says `github-actions`, REST says `github-actions[bot]`.
			// Matching raw made the attestation gate depend on which API produced
			// the payload (#1033); the queue must not inherit that.
			name: "the REST spelling of the same reviewer",
			comments: []Comment{
				{Author: "github-actions[bot]", Body: "## PR Reviewer Guide\nx", CreatedAt: at(10)},
			},
			pending: true,
		},
		{
			// A reviewer with no markers publishes proper reviews; its ordinary
			// comments are not review output and must not enqueue anything.
			name: "a declared reviewer commenting without a marker",
			comments: []Comment{
				{Author: "coderabbitai", Body: "auto-summary, not a review", CreatedAt: at(10)},
			},
			pending: false,
		},
		{
			name: "an undeclared author using the marker verbatim",
			comments: []Comment{
				{Author: "somebody-else", Body: "## PR Reviewer Guide\nnot a declared reviewer", CreatedAt: at(10)},
			},
			pending: false,
		},
		{
			// The marker is matched at the start of a line, so quoting it in
			// prose — as this repo's own docs do — is not a disposition.
			name: "the triage marker quoted mid-sentence does not count",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nx", CreatedAt: at(10)},
				{Author: "manu", Body: "we should write a ## Review triage section soon", CreatedAt: at(20)},
			},
			pending: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(PR{Number: 1, Comments: tc.comments}, reg())
			if got.Pending != tc.pending {
				t.Fatalf("pending = %v, want %v (reason: %s)", got.Pending, tc.pending, got.Reason)
			}
			if got.Reason == "" {
				t.Fatal("Reason is empty: a verdict that cannot say why is the defect this repo keeps finding")
			}
		})
	}
}

// A registry with no triage marker must fail rather than silently match every
// comment, which would empty the queue and look like "nothing pending".
func TestLoadRegistryRefusesAnEmptyMarker(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "reg.json")
	if err := os.WriteFile(p, []byte(`{"reviewers":[],"triage":{"marker":"  "}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRegistry(p); err == nil {
		t.Fatal("want an error for an empty triage.marker, got nil")
	}
}

// AC9: the queue and the gate read the SAME file. This asserts the real one
// parses here and carries what both consumers need — a marker, and at least one
// reviewer whose output is comment-shaped. Without it the two could drift into
// separate truths, which is the failure class the registry exists to prevent.
func TestTheRealRegistryServesBothConsumers(t *testing.T) {
	r, err := LoadRegistry(filepath.Join("..", "..", "..", "harness", "review-attestation.json"))
	if err != nil {
		t.Fatalf("the shipped registry does not load: %v", err)
	}
	if len(r.Reviewers) == 0 {
		t.Fatal("the shipped registry declares no reviewers")
	}
	markers := 0
	for _, rv := range r.Reviewers {
		markers += len(rv.ReviewMarkers)
	}
	if markers == 0 {
		t.Fatal("no reviewer declares review_markers: the queue can never see comment-shaped output")
	}
}

// The boundary-truncation guard used to be tested here against `parseWire`, the
// helper that unpacked `gh pr list --json`'s nested GraphQL shape. CLI-071
// removed both that transport and that helper, so the test moved to
// `TestFetchRefusesTruncationOnBothAxes` in fetch_test.go — which is strictly
// stronger: it exercises the guard through the real fetch path rather than a
// helper, and it covers the SECOND axis REST introduced. Comments now paginate
// too, and a truncated comment list yields a wrong verdict rather than a
// visibly missing pull request.

// A release PR restates commits each reviewed when it merged, and the registry
// declares its diff signature exempt; the attestation gate already honours it.
// The queue did not, so every regeneration of the release branch made
// CodeRabbit re-edit its "Reviews paused" notice and put the PR back in the
// queue, once per merge to main (#1196). The match is the gate's: set
// equality, both directions, and an unread file list is never exempt.
func TestEvaluate_AnExemptDiffSignatureIsNotPending(t *testing.T) {
	r := reg()
	r.Exempt.Signatures = []Signature{{Name: "release-please", Files: []string{"CHANGELOG.md", "versions.conf", ".release-please-manifest.json"}}}
	notice := Comment{Author: "github-actions[bot]", Body: "## PR Reviewer Guide", CreatedAt: at(0), UpdatedAt: at(30)}
	for name, tc := range map[string]struct {
		files   []string
		pending bool
	}{
		"exactly the signature":   {[]string{".release-please-manifest.json", "CHANGELOG.md", "versions.conf"}, false},
		"one file more":           {[]string{".release-please-manifest.json", "CHANGELOG.md", "versions.conf", "cli/main.go"}, true},
		"one file fewer":          {[]string{"CHANGELOG.md", "versions.conf"}, true},
		"files never read":        {nil, true},
		"a duplicate is the same": {[]string{"CHANGELOG.md", "CHANGELOG.md", "versions.conf", ".release-please-manifest.json"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			st := Evaluate(PR{Number: 1, Files: tc.files, Comments: []Comment{notice}}, r)
			if st.Pending != tc.pending {
				t.Fatalf("pending = %v, want %v (%s)", st.Pending, tc.pending, st.Reason)
			}
			if !tc.pending && !strings.Contains(st.Reason, `exempt: the "release-please" diff signature`) {
				t.Errorf("an exemption must say which signature it matched: %q", st.Reason)
			}
		})
	}
}

func TestTheRealRegistryDeclaresTheReleaseSignature(t *testing.T) {
	r, err := LoadRegistry(filepath.Join("..", "..", "..", "harness", "review-attestation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Exempt.Signatures) == 0 || len(r.Exempt.Signatures[0].Files) == 0 {
		t.Fatalf("the queue reads exempt.signatures from the registry, and found none: %+v", r.Exempt)
	}
}

// TOOL-017 AC3: a finding's class comes from a declared marker line, and
// anything short of exactly one agreeing line is unclassified. The prose rows
// are the shapes measured on #2832 and #2833: each states a class a reader can
// see, and each must still come back unclassified, because prose drifts with
// the model and the prompt while a declared line is a contract.
func TestClassify(t *testing.T) {
	r := reg()
	r.Classification.Prefix = "Class: "
	r.Classification.Classes = []string{"REAL", "THEORETICAL", "SPECULATIVE"}
	cases := []struct {
		name string
		body string
		want string
	}{
		{"declared line", "**Race Window**\n\nClass: REAL\nTwo lands race.", "REAL"},
		{"declared line, indented", "**Header**\n  Class: SPECULATIVE  \nbody", "SPECULATIVE"},
		{"the same class twice", "Class: THEORETICAL\nbody\nClass: THEORETICAL", "THEORETICAL"},
		{"two classes disagree", "Class: REAL\nbody\nClass: SPECULATIVE", Unclassified},
		{"header suffix only", "**Race Window (REAL)**\n\nTwo lands race.", Unclassified},
		{"first word of the body only", "**Vacuous Test**\n\nREAL. The new test passes either way.", Unclassified},
		{"marker inside prose", "This is not `Class: REAL` but a quote.", Unclassified},
		{"an undeclared class", "Class: CRITICAL\nbody", Unclassified},
		{"lower case", "class: real\nbody", Unclassified},
		{"no marker at all", "**Check cannot fail**\n\nThe table is shared.", Unclassified},
		{"empty body", "", Unclassified},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.body, r); got != tc.want {
				t.Errorf("Classify(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

// A registry that declares no classification classifies nothing. An empty
// prefix would otherwise make a bare "REAL" line a marker, which is prose.
func TestClassify_WithoutADeclarationEverythingIsUnclassified(t *testing.T) {
	for _, body := range []string{"Class: REAL", "REAL", ""} {
		if got := Classify(body, reg()); got != Unclassified {
			t.Errorf("Classify(%q) with no declaration = %q, want %q", body, got, Unclassified)
		}
	}
}

func TestTheRealRegistryDeclaresTheClassificationMarker(t *testing.T) {
	r, err := LoadRegistry(filepath.Join("..", "..", "..", "harness", "review-attestation.json"))
	if err != nil {
		t.Fatalf("the shipped registry does not load: %v", err)
	}
	if got := Classify("Class: REAL", r); got != "REAL" {
		t.Fatalf("the shipped registry does not classify its own REAL line: got %q", got)
	}
}

// TOOL-017 AC3, the consumer: a finding the registry's classifying reviewers
// left without a declared class is reported, so a human sees it. Only findings
// newer than the last triage count, because older ones were already disposed.
func TestUnclassifiedFindings(t *testing.T) {
	r := reg()
	r.Classification.Prefix = "Class: "
	r.Classification.Classes = []string{"REAL", "THEORETICAL", "SPECULATIVE"}
	r.Classification.Reviewers = []string{"github-actions"}
	triage := Comment{Author: "mlorentedev", AuthorAssociation: "OWNER", Body: "## Review triage\n| ... |", CreatedAt: at(10)}
	inline := []Comment{
		{Author: "github-actions[bot]", Body: "**A**\n\nClass: REAL\nx", Path: "a.go", Line: 1, CreatedAt: at(20)},
		{Author: "github-actions[bot]", Body: "**B**\n\nREAL: prose only", Path: "b.go", Line: 2, CreatedAt: at(20)},
		{Author: "github-actions[bot]", Body: "**C**\n\nno class", Path: "c.go", Line: 3, CreatedAt: at(5)},
		{Author: "coderabbitai[bot]", Body: "🟠 Major", Path: "d.go", Line: 4, CreatedAt: at(20)},
	}

	got := UnclassifiedFindings(PR{Comments: []Comment{triage}}, inline, r)
	if len(got) != 1 || got[0].Path != "b.go" {
		t.Fatalf("got %+v, want only b.go: a.go is classified, c.go predates the triage, "+
			"and coderabbitai is not a reviewer the registry asks for a class", got)
	}

	if got := UnclassifiedFindings(PR{}, inline, r); len(got) != 2 {
		t.Errorf("with no triage recorded, every unclassified finding counts: got %d, want 2 (b.go, c.go)", len(got))
	}

	r.Classification.Reviewers = nil
	if got := UnclassifiedFindings(PR{}, inline, r); got != nil {
		t.Errorf("a registry that asks no reviewer for a class reports nothing: got %+v", got)
	}
}

// #2870, GUARD-003 review F1: the triage record is the one input that takes a
// PR out of the queue, so who wrote it is part of what it claims. A marker from
// an account without standing, or one whose association could not be read, is
// a comment like any other, and the reviewer's output stays awaiting a
// disposition.
func TestEvaluate_AnUndeclaredAuthorDoesNotClearTheQueue(t *testing.T) {
	review := Comment{Author: "github-actions", Body: "## PR Reviewer Guide\nfindings", CreatedAt: at(10)}
	for _, assoc := range []string{"NONE", "CONTRIBUTOR", "FIRST_TIMER", ""} {
		t.Run("association "+assoc, func(t *testing.T) {
			pr := PR{Number: 1, Comments: []Comment{review,
				{Author: "drive-by", AuthorAssociation: assoc, Body: "## Review triage\nall fine", CreatedAt: at(20)},
			}}
			if st := Evaluate(pr, reg()); !st.Pending {
				t.Fatalf("a %q triage record cleared the queue: %s", assoc, st.Reason)
			}
		})
	}
	pr := PR{Number: 1, Comments: []Comment{review,
		{Author: "maintainer", AuthorAssociation: "member", Body: "## Review triage\napplied", CreatedAt: at(20)},
	}}
	if st := Evaluate(pr, reg()); st.Pending {
		t.Fatalf("a MEMBER's triage record (any case) must clear the queue: %s", st.Reason)
	}
}

// The registry is hand-ported to every repository, so a copy without
// triage.standing must still answer, with the default; an explicit empty list
// is refused like an empty marker.
func TestParseRegistryStanding(t *testing.T) {
	r, err := ParseRegistry([]byte(`{"triage":{"marker":"## Review triage"}}`), "absent")
	if err != nil {
		t.Fatalf("a registry without triage.standing must parse: %v", err)
	}
	if strings.Join(r.Triage.Standing, ",") != strings.Join(DefaultStanding, ",") {
		t.Fatalf("absent standing = %v, want the default %v", r.Triage.Standing, DefaultStanding)
	}
	if _, err := ParseRegistry([]byte(`{"triage":{"marker":"## Review triage","standing":[]}}`), "empty"); err == nil {
		t.Fatal("want an error for an empty triage.standing, got nil")
	}
}

// dotfiles declares its standing explicitly, and the code default exists for
// copies that do not. Two spellings of one set drift unless something holds
// them together.
func TestTheRealRegistryDeclaresTheDefaultStanding(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "harness", "review-attestation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"standing"`) {
		t.Fatal("harness/review-attestation.json does not declare triage.standing")
	}
	r, err := ParseRegistry(raw, "real")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.Triage.Standing, ",") != strings.Join(DefaultStanding, ",") {
		t.Fatalf("registry standing %v differs from DefaultStanding %v", r.Triage.Standing, DefaultStanding)
	}
}
