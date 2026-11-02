package prreview

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

const testRegistry = `{"escape":{"label":"no-review","section":"## Why unreviewed"}}`

// fakeForge is a PR on a forge that keeps state, so a test reads what is on the
// PR afterwards and not which calls were made.
type fakeForge struct {
	registry    string // "" answers 404
	failReg     bool
	body        *string
	failBody    bool
	labels      []string
	failLabels  bool
	dropLabel   bool // the POST succeeds and the label does not land
	deleteError string
	writes      []string
	calls       []string
}

func (f *fakeForge) run(_ context.Context, args ...string) ([]byte, []byte, error) {
	joined := strings.Join(args, " ")
	f.calls = append(f.calls, joined)
	failed := func(reason string) ([]byte, []byte, error) {
		return []byte(`{"message":"failed"}`), []byte("gh: " + reason + "\n"), errors.New("exit status 1")
	}
	switch {
	case strings.Contains(joined, "/contents/"):
		return f.answerRegistry(failed)
	case strings.Contains(joined, "-X PATCH"):
		return f.patchBody(args)
	case strings.Contains(joined, "-X POST"):
		f.writes = append(f.writes, "POST")
		if !f.dropLabel {
			f.labels = append(f.labels, strings.TrimPrefix(args[len(args)-1], "labels[]="))
		}
		return nil, nil, nil
	case strings.Contains(joined, "-X DELETE"):
		f.writes = append(f.writes, args[3])
		if f.deleteError != "" {
			return failed(f.deleteError)
		}
		f.labels = nil
		return nil, nil, nil
	case strings.HasSuffix(args[1], "/pulls/7"):
		if f.failBody {
			return failed("Bad Gateway (HTTP 502)")
		}
		out, _ := json.Marshal(map[string]*string{"body": f.body})
		return out, nil, nil
	case strings.HasSuffix(args[1], "/labels"):
		return f.answerLabels(args, failed)
	}
	return failed("unexpected call: " + joined)
}

func (f *fakeForge) answerRegistry(failed func(string) ([]byte, []byte, error)) ([]byte, []byte, error) {
	if f.failReg {
		return failed("Gateway Timeout (HTTP 504)")
	}
	if f.registry == "" {
		return failed("Not Found (HTTP 404)")
	}
	return []byte(base64.StdEncoding.EncodeToString([]byte(f.registry)) + "\n"), nil, nil
}

func (f *fakeForge) patchBody(args []string) ([]byte, []byte, error) {
	raw, err := os.ReadFile(args[len(args)-1])
	if err != nil {
		return nil, []byte(err.Error()), err
	}
	var in struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, []byte(err.Error()), err
	}
	f.body = &in.Body
	f.writes = append(f.writes, "PATCH")
	return nil, nil, nil
}

// answerLabels splits the labels over two pages, as a PR past 30 labels does:
// without --paginate only the first page answers.
func (f *fakeForge) answerLabels(args []string, failed func(string) ([]byte, []byte, error)) ([]byte, []byte, error) {
	if f.failLabels {
		return failed("Bad Gateway (HTTP 502)")
	}
	page := func(names []string) []byte {
		items := make([]map[string]string, 0, len(names))
		for _, n := range names {
			items = append(items, map[string]string{"name": n})
		}
		out, _ := json.Marshal(items)
		return out
	}
	if len(f.labels) < 2 {
		return page(f.labels), nil, nil
	}
	if !slices.Contains(args, "--paginate") {
		return page(f.labels[:1]), nil, nil
	}
	return append(page(f.labels[:1]), page(f.labels[1:])...), nil, nil
}

func strptr(s string) *string { return &s }

func declare(t *testing.T, f *fakeForge, diff string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := DeclareUnreviewed(context.Background(),
		DeclareInputs{Repo: "o/r", PR: "7", DefaultBranch: "trunk", Diff: diff},
		DeclareEnv{Out: &out, GH: f.run})
	return out.String(), err
}

func TestDeclare_WritesTheRegistrysSectionAndLabel(t *testing.T) {
	f := &fakeForge{registry: testRegistry, body: strptr("Intro."), labels: []string{"area/ci"}}
	if out, err := declare(t, f, DiffEmpty); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.HasPrefix(*f.body, "Intro.\n\n## Why unreviewed\n\nPR-Agent") {
		t.Errorf("body: %q", *f.body)
	}
	if !slices.Equal(f.labels, []string{"area/ci", "no-review"}) {
		t.Errorf("labels: %q; an add must keep the others", f.labels)
	}
}

func TestDeclare_ReadsTheRegistryAtTheDefaultBranch(t *testing.T) {
	f := &fakeForge{registry: testRegistry, body: strptr("")}
	if _, err := declare(t, f, DiffEmpty); err != nil {
		t.Fatal(err)
	}
	want := "repos/o/r/contents/harness/review-attestation.json?ref=trunk"
	if !strings.Contains(f.calls[0], want) {
		t.Errorf("first call %q; a PR must not choose its own escape", f.calls[0])
	}
}

func TestDeclare_NullBodyGetsOnlyTheSection(t *testing.T) {
	f := &fakeForge{registry: testRegistry}
	if _, err := declare(t, f, DiffEmpty); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(*f.body, "## Why unreviewed\n\n") {
		t.Errorf("body: %q", *f.body)
	}
}

func TestDeclare_SecondRunChangesNothing(t *testing.T) {
	f := &fakeForge{registry: testRegistry, body: strptr("Intro.")}
	if _, err := declare(t, f, DiffEmpty); err != nil {
		t.Fatal(err)
	}
	first := *f.body
	f.writes = nil
	if _, err := declare(t, f, DiffEmpty); err != nil {
		t.Fatal(err)
	}
	if *f.body != first || slices.Contains(f.writes, "PATCH") {
		t.Errorf("second run rewrote the body: writes %q, body %q", f.writes, *f.body)
	}
}

// The gate strips \r and trailing blanks before it matches the heading, because
// GitHub's web editor saves CRLF. An exact line compare here would append a
// second copy to a PR the gate already reads as declared.
func TestDeclare_ASectionTheGateReadsIsNotWrittenAgain(t *testing.T) {
	for name, body := range map[string]string{
		"lf":       "Intro.\n\n## Why unreviewed\n\nA reason.\n",
		"crlf":     "Intro.\r\n\r\n## Why unreviewed\r\n\r\nA reason.\r\n",
		"trailing": "Intro.\n\n## Why unreviewed  \t\n\nA reason.\n",
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeForge{registry: testRegistry, body: strptr(body)}
			if _, err := declare(t, f, DiffEmpty); err != nil {
				t.Fatal(err)
			}
			if slices.Contains(f.writes, "PATCH") || *f.body != body {
				t.Errorf("rewrote the body: %q", *f.body)
			}
			if !slices.Contains(f.labels, "no-review") {
				t.Errorf("labels: %q", f.labels)
			}
		})
	}
}

// The gate reads the FIRST heading and refuses an empty one, so a PR template
// that carries the heading would stay red if the text landed anywhere else.
func TestDeclare_FillsAnEmptyHeadingInPlace(t *testing.T) {
	for name, body := range map[string]string{
		"at the end":      "Intro.\n\n## Why unreviewed\n",
		"before the next": "## Why unreviewed\n\n\n## Evidence\n\nbats 11/11.",
		"crlf":            "## Why unreviewed\r\n\r\n## Evidence\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeForge{registry: testRegistry, body: strptr(body)}
			if _, err := declare(t, f, DiffEmpty); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.ReplaceAll(*f.body, "\r", ""), "## Why unreviewed\n\nPR-Agent") {
				t.Errorf("no blank line under the heading: %q", *f.body)
			}
			if n := strings.Count(*f.body, "## Why unreviewed"); n != 1 {
				t.Errorf("%d headings: %q", n, *f.body)
			}
			if !strings.Contains(gateSection(*f.body, "## Why unreviewed"), "PR-Agent") {
				t.Errorf("the gate reads an empty section: %q", *f.body)
			}
		})
	}
}

// gateSection reads the section as check-review-attestation.sh does.
func gateSection(body, section string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r", ""), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	start := slices.Index(lines, section)
	if start < 0 {
		return ""
	}
	rest := lines[start+1:]
	for i, l := range rest {
		if strings.HasPrefix(l, "## ") {
			rest = rest[:i]
			break
		}
	}
	return strings.TrimSpace(strings.Join(rest, "\n"))
}

// A failed read must never become an empty body: the PATCH would wipe the PR's.
func TestDeclare_AFailedBodyReadWritesNothing(t *testing.T) {
	f := &fakeForge{registry: testRegistry, body: strptr("Intro."), failBody: true}
	out, err := declare(t, f, DiffEmpty)
	if !errors.Is(err, ErrNotDeclared) {
		t.Fatalf("err = %v", err)
	}
	if len(f.writes) != 0 || !strings.Contains(out, "HTTP 502") {
		t.Errorf("writes %q, out %q", f.writes, out)
	}
}

func TestDeclare_ReadsTheLabelBack(t *testing.T) {
	f := &fakeForge{registry: testRegistry, body: strptr(""), dropLabel: true, labels: []string{"area/ci"}}
	out, err := declare(t, f, DiffEmpty)
	if !errors.Is(err, ErrNotDeclared) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out, "no-review") || !strings.Contains(out, "area/ci") {
		t.Errorf("out: %q", out)
	}
}

func TestDeclare_AFailedReadBackIsAFailure(t *testing.T) {
	f := &fakeForge{registry: testRegistry, body: strptr(""), failLabels: true}
	if _, err := declare(t, f, DiffEmpty); !errors.Is(err, ErrNotDeclared) {
		t.Fatalf("err = %v", err)
	}
}

func TestDeclare_FindsTheLabelOnALaterPage(t *testing.T) {
	f := &fakeForge{registry: testRegistry, body: strptr(""), labels: []string{"a", "b"}}
	if out, err := declare(t, f, DiffEmpty); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// Hive's registry has no escape. Both directions refuse, so turning the input on
// in such a repository fails on its first PR, whichever way that PR goes.
func TestDeclare_ARegistryWithNoEscapeNamesTheFile(t *testing.T) {
	cases := map[string]*fakeForge{
		"absent":        {registry: `{"reviewers":[]}`},
		"null":          {registry: `{"escape":null}`},
		"no label":      {registry: `{"escape":{"section":"## s"}}`},
		"no section":    {registry: `{"escape":{"label":"l"}}`},
		"empty label":   {registry: `{"escape":{"label":"","section":"## s"}}`},
		"blank section": {registry: `{"escape":{"label":"l","section":"  "}}`},
		"not a string":  {registry: `{"escape":{"label":5,"section":"## s"}}`},
		"not json":      {registry: `{`},
		"no file":       {},
		"outage":        {failReg: true},
	}
	for name, f := range cases {
		for _, diff := range []string{DiffEmpty, DiffReviewable} {
			t.Run(name+"/"+diff, func(t *testing.T) {
				f.writes, f.body = nil, strptr("Intro.")
				f.labels = []string{"no-review"}
				out, err := declare(t, f, diff)
				if !errors.Is(err, ErrNotDeclared) {
					t.Fatalf("err = %v", err)
				}
				if !strings.Contains(out, "harness/review-attestation.json") || !strings.Contains(out, "trunk") {
					t.Errorf("out does not name the file and ref: %q", out)
				}
				if len(f.writes) != 0 {
					t.Errorf("writes: %q", f.writes)
				}
			})
		}
	}
}

func TestDeclare_AnOutageIsNotReportedAsAMissingEscape(t *testing.T) {
	out, _ := declare(t, &fakeForge{failReg: true}, DiffEmpty)
	if !strings.Contains(out, "HTTP 504") || strings.Contains(out, "declares no") {
		t.Errorf("out: %q", out)
	}
}

func TestClear_RemovesOnlyTheLabel(t *testing.T) {
	body := "## Why unreviewed\n\nA reason.\n"
	f := &fakeForge{registry: testRegistry, body: strptr(body), labels: []string{"no-review"}}
	out, err := declare(t, f, DiffReviewable)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(f.labels) != 0 {
		t.Errorf("labels: %q", f.labels)
	}
	if *f.body != body || slices.Contains(f.writes, "PATCH") {
		t.Errorf("the body is left for a human: %q, writes %q", *f.body, f.writes)
	}
	if !strings.Contains(out, "::warning::") || !strings.Contains(out, "body") {
		t.Errorf("out: %q", out)
	}
}

func TestClear_NoLabelNoWrite(t *testing.T) {
	f := &fakeForge{registry: testRegistry, labels: []string{"area/ci"}}
	out, err := declare(t, f, DiffReviewable)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(f.writes) != 0 || strings.Contains(out, "::warning::") {
		t.Errorf("writes %q, out %q", f.writes, out)
	}
}

func TestClear_FindsTheLabelOnALaterPage(t *testing.T) {
	f := &fakeForge{registry: testRegistry, labels: []string{"area/ci", "no-review"}}
	if out, err := declare(t, f, DiffReviewable); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(f.writes) != 1 {
		t.Errorf("writes: %q", f.writes)
	}
}

func TestClear_EscapesTheLabelInThePath(t *testing.T) {
	f := &fakeForge{registry: `{"escape":{"label":"no review/ok","section":"## s"}}`, labels: []string{"no review/ok"}}
	if out, err := declare(t, f, DiffReviewable); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if want := "repos/o/r/issues/7/labels/no%20review%2Fok"; f.writes[0] != want {
		t.Errorf("DELETE %q, want %q", f.writes[0], want)
	}
}

func TestClear_ALabelAlreadyGoneIsDone(t *testing.T) {
	f := &fakeForge{registry: testRegistry, labels: []string{"no-review"}, deleteError: "Not Found (HTTP 404)"}
	if out, err := declare(t, f, DiffReviewable); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestClear_AFailedWriteOrReadIsAFailure(t *testing.T) {
	for name, f := range map[string]*fakeForge{
		"delete": {registry: testRegistry, labels: []string{"no-review"}, deleteError: "Bad Gateway (HTTP 502)"},
		"read":   {registry: testRegistry, labels: []string{"no-review"}, failLabels: true},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := declare(t, f, DiffReviewable)
			if !errors.Is(err, ErrNotDeclared) || !strings.Contains(out, "HTTP 502") {
				t.Errorf("err %v, out %q", err, out)
			}
		})
	}
}

// Unread covers a draft, a forfeited branch and a failed read. None of them
// answers the question, so nothing is declared and nothing is cleared, and the
// forge is not touched: a label on a PR the read failed for may be right.
func TestDeclare_UnreadTouchesNothing(t *testing.T) {
	f := &fakeForge{registry: testRegistry, labels: []string{"no-review"}}
	if _, err := declare(t, f, DiffUnread); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 0 {
		t.Errorf("calls: %q", f.calls)
	}
}

// kubelab's dead-end test. A review asked for by comment reaches the same filter
// and the same empty diff, so the rationale must not send anyone there.
func TestRationale_PointsAtNoDeadEnd(t *testing.T) {
	f := &fakeForge{registry: testRegistry, body: strptr("")}
	if _, err := declare(t, f, DiffEmpty); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(*f.body, "/review") {
		t.Errorf("rationale names a command that cannot produce a review:\n%s", *f.body)
	}
	for _, want := range []string{"`trunk`", "`.pr_agent.toml`", "same empty diff", "decrypt", "key names"} {
		if !strings.Contains(*f.body, want) {
			t.Errorf("rationale lacks %q:\n%s", want, *f.body)
		}
	}
	// Another repository's issue number would link to the wrong thing there, and
	// a `## ` line would end the section the gate reads.
	if strings.Contains(strings.TrimPrefix(*f.body, "## Why unreviewed"), "#") {
		t.Errorf("rationale carries a reference or a heading:\n%s", *f.body)
	}
}

// kubelab's registry test: a literal here would be a second source of truth that
// could drift from the file the gate reads, with neither side erroring.
func TestDeclare_NeverHoldsTheEscapeAsALiteral(t *testing.T) {
	raw, err := os.ReadFile("../../../harness/review-attestation.json")
	if err != nil {
		t.Fatal(err)
	}
	var reg struct {
		Escape struct{ Label, Section string } `json:"escape"`
	}
	if err := json.Unmarshal(raw, &reg); err != nil {
		t.Fatal(err)
	}
	if reg.Escape.Label == "" || reg.Escape.Section == "" {
		t.Fatalf("the registry declares no escape: %+v", reg.Escape)
	}
	for _, file := range []string{"declare.go", "../cmd/pr_declare_unreviewed.go"} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, literal := range []string{reg.Escape.Label, reg.Escape.Section} {
			if bytes.Contains(src, []byte(literal)) {
				t.Errorf("%s holds %q; the registry is the source", file, literal)
			}
		}
	}
}
