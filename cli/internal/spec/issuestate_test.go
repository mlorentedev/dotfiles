package spec

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestIssueStateGHLookup(t *testing.T) {
	cases := []struct {
		name           string
		stdout, stderr string
		err            error
		want           IssueState
		wantErr        bool
	}{
		{"open issue", "open\tfalse\n", "", nil, StateOpen, false},
		{"closed issue", "closed\tfalse\n", "", nil, StateClosed, false},
		{"number is a pull request", "closed\ttrue\n", "", nil, StateNotIssue, false},
		{"no such issue", "", "gh: Not Found (HTTP 404)\n", errors.New("exit status 1"), StateNotFound, false},
		{"auth failure is unanswerable", "", "gh: Bad credentials (HTTP 401)\n", errors.New("exit status 1"), StateUnknown, true},
		{"rate limit is unanswerable", "", "gh: API rate limit exceeded (HTTP 403)\n", errors.New("exit status 1"), StateUnknown, true},
		{"unparseable output is unanswerable", "<html>", "", nil, StateUnknown, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotArgs []string
			lookup := GHIssueStateLookup(func(args ...string) (string, string, error) {
				gotArgs = args
				return tc.stdout, tc.stderr, tc.err
			})
			got, err := lookup(IssueRef{"mlorentedev/hive", 267})
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("got (%v, %v), want (%v, err=%v)", got, err, tc.want, tc.wantErr)
			}
			if len(gotArgs) < 2 || gotArgs[0] != "api" || gotArgs[1] != "repos/mlorentedev/hive/issues/267" {
				t.Fatalf("gh called with %v, want the REST issues endpoint", gotArgs)
			}
		})
	}
}

// writeAuditSpec creates specs/<id>/proposal.md under root.
func writeAuditSpec(t *testing.T, root, dir, proposal string) {
	t.Helper()
	p := filepath.Join(root, "specs", dir)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "proposal.md"), []byte(proposal), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeLookup answers from a table keyed by IssueRef.String(); a missing key is
// an unanswerable lookup. It records every call so dedupe can be asserted.
type fakeLookup struct {
	mu     sync.Mutex
	states map[string]IssueState
	calls  []string
}

func (f *fakeLookup) lookup(ref IssueRef) (IssueState, error) {
	f.mu.Lock()
	f.calls = append(f.calls, ref.String())
	f.mu.Unlock()
	if s, ok := f.states[ref.String()]; ok {
		return s, nil
	}
	return StateUnknown, errors.New("gh: connection refused")
}

func TestIssueStateAuditClassifies(t *testing.T) {
	root := t.TempDir()
	writeAuditSpec(t, root, "A-001-open", frontmatter(`issue: "mlorentedev/dotfiles#1"`))
	writeAuditSpec(t, root, "A-002-zombie", frontmatter(`issue: "mlorentedev/dotfiles#2"`))
	writeAuditSpec(t, root, "A-003-gone", frontmatter(`issue: "mlorentedev/kubelab#611"`))
	writeAuditSpec(t, root, "A-004-pull", frontmatter(`issue: "#4"`))
	writeAuditSpec(t, root, "A-005-prose", frontmatter(`status: draft`)+"\n- GH issue: [#5](https://github.com/mlorentedev/dotfiles/issues/5)\n")
	writeAuditSpec(t, root, "A-006-unlinked", frontmatter(`status: draft`))
	writeAuditSpec(t, root, "A-007-malformed", frontmatter(`issue: "TBD"`))
	writeAuditSpec(t, root, "A-008-shares-a-zombie", frontmatter(`issue: "dotfiles#2"`))
	writeAuditSpec(t, root, "A-009-prose-zombie", frontmatter(`status: draft`)+"\n- GH issue: #2\n")
	// Never audited: the archive, a stray file, a folder with no proposal.
	writeAuditSpec(t, root, filepath.Join("archive", "Z-001-done"), frontmatter(`issue: "#999"`))
	if err := os.WriteFile(filepath.Join(root, "specs", "README.md"), []byte("# specs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "specs", "B-001-no-proposal"), 0o755); err != nil {
		t.Fatal(err)
	}

	f := &fakeLookup{states: map[string]IssueState{
		"mlorentedev/dotfiles#1":  StateOpen,
		"mlorentedev/dotfiles#2":  StateClosed,
		"mlorentedev/kubelab#611": StateNotFound,
		"mlorentedev/dotfiles#4":  StateNotIssue,
		"mlorentedev/dotfiles#5":  StateOpen,
	}}
	got, err := AuditIssueState(root, homeRepo, f.lookup)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]AuditSeverity{
		"A-001-open":            SeverityOK,
		"A-002-zombie":          SeverityFail,
		"A-003-gone":            SeverityFail,
		"A-004-pull":            SeverityFail,
		"A-005-prose":           SeverityWarn,
		"A-006-unlinked":        SeverityWarn,
		"A-007-malformed":       SeverityFail,
		"A-008-shares-a-zombie": SeverityFail,
		"A-009-prose-zombie":    SeverityFail,
	}
	var order []string
	gotSev := map[string]AuditSeverity{}
	for _, fd := range got {
		order = append(order, fd.SpecID)
		gotSev[fd.SpecID] = fd.Severity
		if fd.Detail == "" {
			t.Errorf("%s: empty Detail — every finding must say why", fd.SpecID)
		}
	}
	if !reflect.DeepEqual(gotSev, want) {
		t.Errorf("severities:\n got %v\nwant %v", gotSev, want)
	}
	for i := 1; i < len(order); i++ {
		if order[i-1] > order[i] {
			t.Errorf("findings not sorted by spec id: %v", order)
			break
		}
	}
	// One lookup per distinct issue: A-002, A-008 and A-009 share #2, and the
	// malformed and unlinked specs are never looked up.
	if len(f.calls) != 5 {
		t.Errorf("lookups = %v, want 5 distinct issues", f.calls)
	}
}

func TestIssueStateAuditUnanswerable(t *testing.T) {
	root := t.TempDir()
	writeAuditSpec(t, root, "A-001-open", frontmatter(`issue: "#1"`))
	writeAuditSpec(t, root, "A-002-offline", frontmatter(`issue: "#2"`))
	f := &fakeLookup{states: map[string]IssueState{"mlorentedev/dotfiles#1": StateOpen}}
	got, err := AuditIssueState(root, homeRepo, f.lookup)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Severity != SeverityUnanswerable || !strings.Contains(got[1].Detail, "connection refused") {
		t.Fatalf("want A-002 unanswerable naming the cause, got %+v", got)
	}
	if !AuditNeedsAttention(got) {
		t.Fatal("an unanswerable lookup must never read as a clear audit")
	}
	if AuditNeedsAttention(got[:1]) {
		t.Fatal("an all-open audit needs no attention")
	}
}

func TestIssueStateAuditNoSpecsDir(t *testing.T) {
	got, err := AuditIssueState(t.TempDir(), homeRepo, (&fakeLookup{}).lookup)
	if err != nil || len(got) != 0 {
		t.Fatalf("a repo without specs/ audits clean: got (%v, %v)", got, err)
	}
}

// TestIssueStateNoActiveSpecIsProseLinked holds this repository to the
// contract the audit only warns about: every active spec links its issue in
// frontmatter. A prose-only link resolves, but it is invisible to the
// archive-on-merge gate — which is how GOV-004 became a zombie (#1087).
func TestIssueStateNoActiveSpecIsProseLinked(t *testing.T) {
	// Absolute on purpose: RepoRoot(".") never climbs, because
	// filepath.Dir(".") is ".", and a skip here would prove nothing.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := RepoRoot(wd)
	if err != nil {
		t.Fatalf("this guard must run inside the checkout: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "specs"))
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "specs", e.Name(), "proposal.md"))
		if err != nil {
			continue
		}
		checked++
		if _, src, _ := ResolveIssueLink(string(data), homeRepo); src == LinkProse {
			t.Errorf("specs/%s links its issue only in prose — move it to the `issue:` frontmatter field", e.Name())
		}
	}
	if checked == 0 {
		t.Fatal("no active spec was checked — the walk found nothing, so this test proves nothing")
	}
}
