package spec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// IssueState is what the forge says about one tracking issue.
type IssueState int

const (
	// StateUnknown: the forge could not answer. Never read as clean.
	StateUnknown IssueState = iota
	StateOpen
	StateClosed
	// StateNotFound: the forge answered, and there is no such issue.
	StateNotFound
	// StateNotIssue: the number is a pull request, which tracks nothing.
	StateNotIssue
)

// IssueStateLookup answers the state of one issue. A non-nil error means the
// question could not be answered — distinct from StateNotFound, which is an
// answer.
type IssueStateLookup func(IssueRef) (IssueState, error)

// GHIssueStateLookup builds a lookup over `gh api`, on the REST issues
// endpoint: it has its own rate budget, separate from the GraphQL one that
// board automation exhausts, and its 404 cleanly means "no such issue".
// run executes gh with args and returns stdout, stderr and the exit error.
//
// One attempt by default. Attempts(n) retries a failure GitHub did not answer
// (a network or TLS timeout, a 5xx), never an HTTP 4xx, which is its answer,
// nor a run that returns context.DeadlineExceeded: that is the caller's own
// bound, and retrying it would multiply the wait the caller chose.
// The retry is the caller's policy because only the caller knows its bound: a
// CI audit that fails its leg on one blip wants it (#2450); the doctor, which
// reports an unanswerable spec as a WARN and kills gh at its own deadline,
// would only triple its wait on a hung call.
func GHIssueStateLookup(run func(args ...string) (stdout, stderr string, err error), opts ...LookupOption) IssueStateLookup {
	cfg := lookupConfig{attempts: 1, wait: linearBackoff}
	for _, o := range opts {
		o(&cfg)
	}
	return func(ref IssueRef) (IssueState, error) {
		var out, errOut string
		var err error
		for attempt := 1; attempt <= cfg.attempts; attempt++ {
			out, errOut, err = run("api", fmt.Sprintf("repos/%s/issues/%d", ref.Repo, ref.Number),
				"--jq", `[.state, (.pull_request != null)] | @tsv`)
			if err == nil || httpClientError.MatchString(errOut) || errors.Is(err, context.DeadlineExceeded) || attempt == cfg.attempts {
				break
			}
			cfg.wait(attempt)
		}
		if err != nil {
			if strings.Contains(errOut, "HTTP 404") {
				return StateNotFound, nil
			}
			return StateUnknown, fmt.Errorf("gh api %s: %s", ref, firstLine(errOut, err))
		}
		switch strings.TrimSpace(out) {
		case "open\tfalse":
			return StateOpen, nil
		case "closed\tfalse":
			return StateClosed, nil
		case "open\ttrue", "closed\ttrue":
			return StateNotIssue, nil
		}
		return StateUnknown, fmt.Errorf("gh api %s: unexpected output %q", ref, strings.TrimSpace(out))
	}
}

// LookupOption tunes GHIssueStateLookup.
type LookupOption func(*lookupConfig)

type lookupConfig struct {
	attempts int
	wait     func(attempt int)
}

// Attempts makes a lookup try up to n times (n < 1 counts as 1), backing off
// one second more before each retry.
func Attempts(n int) LookupOption {
	return func(c *lookupConfig) { c.attempts = max(n, 1) }
}

// httpClientError is gh's report of a 4xx, with or without parentheses
// ("Not Found (HTTP 404)", "HTTP 403"): an answer from GitHub, not a failure to
// reach it.
var httpClientError = regexp.MustCompile(`\bHTTP 4[0-9]{2}\b`)

// RetryWait replaces the back-off before retry number attempt; nil keeps the
// default. Tests pass a no-op: a retrying test without one sleeps for real.
func RetryWait(wait func(attempt int)) LookupOption {
	return func(c *lookupConfig) {
		if wait != nil {
			c.wait = wait
		}
	}
}

// linearBackoff waits one second more before each retry.
func linearBackoff(attempt int) { time.Sleep(time.Duration(attempt) * time.Second) }

func firstLine(s string, fallback error) string {
	if line, _, _ := strings.Cut(strings.TrimSpace(s), "\n"); line != "" {
		return line
	}
	return fallback.Error()
}

// AuditSeverity grades one active spec.
type AuditSeverity string

const (
	SeverityOK           AuditSeverity = "ok"
	SeverityWarn         AuditSeverity = "warn"
	SeverityFail         AuditSeverity = "fail"
	SeverityUnanswerable AuditSeverity = "unanswerable"
)

// AuditFinding is the verdict on one active spec's tracking issue.
type AuditFinding struct {
	SpecID   string
	Ref      IssueRef // zero when the spec is unlinked or its link is malformed
	Source   LinkSource
	Severity AuditSeverity
	Detail   string
	zombie   bool // its issue is CLOSED: the one verdict a stale checkout can fake
}

// AuditNeedsAttention reports whether an audit is anything other than clean.
// An unanswerable lookup counts: a question nobody could answer must not be
// mistaken for one answered "all clear" — the `dotf pr triage-queue` contract.
func AuditNeedsAttention(findings []AuditFinding) bool {
	for _, f := range findings {
		if f.Severity == SeverityFail || f.Severity == SeverityUnanswerable {
			return true
		}
	}
	return false
}

// auditWorkers bounds concurrent forge lookups: enough that ~40 specs cost
// about one round-trip of wall time, few enough not to look like a burst.
const auditWorkers = 8

// AuditIssueState grades every active spec (specs/*/, never specs/archive/)
// by the state of the issue it tracks. It exists because archive-on-merge
// only sees issues closed by a PR's closing keyword; an issue closed any
// other way leaves its spec active forever (#1087), and only asking the forge
// can find it. The error return is for the filesystem only — a forge that
// cannot answer yields SeverityUnanswerable findings, never an error.
func AuditIssueState(repoRoot, homeRepo string, lookup IssueStateLookup) ([]AuditFinding, error) {
	findings, err := resolveActiveSpecs(repoRoot, homeRepo)
	if err != nil {
		return nil, err
	}
	states := lookupAll(findings, lookup)
	for i := range findings {
		if findings[i].Severity == "" {
			grade(&findings[i], states[findings[i].Ref], homeRepo)
		}
	}
	return findings, nil
}

// resolveActiveSpecs reads each active spec's link. Findings whose verdict is
// already known without the forge (unlinked, malformed) get their severity
// here; the rest are left blank for grade.
func resolveActiveSpecs(repoRoot, homeRepo string) ([]AuditFinding, error) {
	dir := filepath.Join(repoRoot, "specs")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var findings []AuditFinding
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "proposal.md"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		f := AuditFinding{SpecID: e.Name()}
		f.Ref, f.Source, err = ResolveIssueLink(string(data), homeRepo)
		switch {
		case err != nil:
			f.Ref, f.Severity, f.Detail = IssueRef{}, SeverityFail, "unresolvable link: "+err.Error()
		case f.Source == LinkNone:
			f.Severity, f.Detail = SeverityWarn, "unlinked: proposal.md names no tracking issue"
		}
		findings = append(findings, f)
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].SpecID < findings[j].SpecID })
	return findings, nil
}

type lookupResult struct {
	state IssueState
	err   error
}

// lookupAll asks the forge once per distinct issue, with bounded concurrency.
func lookupAll(findings []AuditFinding, lookup IssueStateLookup) map[IssueRef]lookupResult {
	refs := map[IssueRef]bool{}
	for _, f := range findings {
		if f.Severity == "" {
			refs[f.Ref] = true
		}
	}
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		sem     = make(chan struct{}, auditWorkers)
		results = make(map[IssueRef]lookupResult, len(refs))
	)
	for ref := range refs {
		wg.Add(1)
		sem <- struct{}{}
		go func(ref IssueRef) {
			defer wg.Done()
			defer func() { <-sem }()
			s, err := lookup(ref)
			mu.Lock()
			results[ref] = lookupResult{s, err}
			mu.Unlock()
		}(ref)
	}
	wg.Wait()
	return results
}

// grade turns one lookup into a verdict. A 404 is proof of absence only in the
// repository the token is scoped to: GitHub answers 404 for a missing issue and
// for a repository the token cannot see alike, so elsewhere it is "not verified"
// (#1909), the verdict an unauthenticated run already gets.
func grade(f *AuditFinding, r lookupResult, homeRepo string) {
	switch {
	case r.err != nil:
		f.Severity, f.Detail = SeverityUnanswerable, fmt.Sprintf("%s could not be checked: %v", f.Ref, r.err)
	case r.state == StateClosed:
		f.Severity, f.Detail = SeverityFail, fmt.Sprintf("zombie: %s is CLOSED but the spec is still active", f.Ref)
		f.zombie = true
	case r.state == StateNotFound && !strings.EqualFold(f.Ref.Repo, homeRepo):
		f.Severity, f.Detail = SeverityUnanswerable, fmt.Sprintf("%s could not be verified: GitHub answered 404, which is also what a repository this token cannot see answers", f.Ref)
	case r.state == StateNotFound:
		f.Severity, f.Detail = SeverityFail, fmt.Sprintf("unresolvable link: %s does not exist", f.Ref)
	case r.state == StateNotIssue:
		f.Severity, f.Detail = SeverityFail, fmt.Sprintf("unresolvable link: %s is a pull request, not an issue", f.Ref)
	case r.state != StateOpen:
		f.Severity, f.Detail = SeverityUnanswerable, fmt.Sprintf("%s returned no usable state", f.Ref)
	case f.Source == LinkProse:
		f.Severity, f.Detail = SeverityWarn, fmt.Sprintf("%s is open, but linked only in prose — move it to the `issue:` frontmatter field", f.Ref)
	default:
		f.Severity, f.Detail = SeverityOK, fmt.Sprintf("%s is open", f.Ref)
	}
}

// ArchiveLookup answers whether the forge's default branch already holds
// specs/archive/<specID>. A non-nil error means it could not be asked.
type ArchiveLookup func(specID string) (bool, error)

// archiveRoutes are where `spec archive` moves a spec: the plain route, and
// the one `--abandoned` takes.
var archiveRoutes = []string{"specs/archive/", "specs/archive/_abandoned/"}

// GHArchiveLookup builds an ArchiveLookup over the REST contents endpoint of
// homeRepo. It names no ref, so the forge answers for its default branch.
func GHArchiveLookup(run func(args ...string) (stdout, stderr string, err error), homeRepo string) ArchiveLookup {
	return func(specID string) (bool, error) {
		for _, route := range archiveRoutes {
			_, errOut, err := run("api", fmt.Sprintf("repos/%s/contents/%s%s", homeRepo, route, specID), "--jq", ".[0].name")
			switch {
			case err == nil:
				return true, nil
			case !strings.Contains(errOut, "HTTP 404"):
				return false, fmt.Errorf("gh api %s%s: %s", route, specID, firstLine(errOut, err))
			}
		}
		return false, nil
	}
}

// ExcuseArchivedZombies re-grades a zombie as a WARN when the default branch
// has already archived its spec (#2918). The audit reads the specs from a
// checkout cut when the run started and the issue state live, so an archive
// that merges in between leaves an active spec beside a closed issue. A base
// that cannot be asked keeps the FAIL: the zombie question was answered.
func ExcuseArchivedZombies(findings []AuditFinding, archived ArchiveLookup) {
	for i := range findings {
		f := &findings[i]
		if !f.zombie {
			continue
		}
		ok, err := archived(f.SpecID)
		switch {
		case err != nil:
			f.Detail += fmt.Sprintf("; whether the default branch archived it could not be checked: %v", err)
		case ok:
			f.Severity, f.Detail = SeverityWarn, fmt.Sprintf("%s is CLOSED and the default branch already archived this spec, after this checkout was cut: merge the base", f.Ref)
		}
	}
}
