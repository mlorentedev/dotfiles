package prtriage

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/execerr"
)

// ghRunner runs a `gh` subcommand and returns its stdout.
//
// It is the seam CLI-071 introduced, and introducing it was the point of that
// change rather than a means to it. Before it, FetchWithRegistry called
// exec.CommandContext directly, so the fetch path could not be reached from a
// test — and the path that had actually been observed failing in production
// (#1454) was therefore the only one in this package with no coverage.
type ghRunner func(ctx context.Context, args ...string) ([]byte, error)

func execGH(ctx context.Context, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "gh", args...).Output()
	return out, execerr.WithStderr(err)
}

const (
	// prLimit and commentLimit are page sizes AND boundary alarms. REST
	// paginates where the old GraphQL query did not, so a full page means
	// "there may be more" and this package refuses rather than answering from
	// part of the data.
	prLimit      = 100
	commentLimit = 100
	filesLimit   = 100

	// commentFanout bounds the per-pull-request calls. The session-start probe
	// runs on every session start under a five-second budget (see the caller in
	// cmd/mem.go), and REST needs one call per pull request where GraphQL
	// needed none — so these must overlap. Unbounded would be its own failure
	// mode: a burst against a rate limit is what this change is escaping.
	commentFanout = 8
)

// restPR is one item of `GET /repos/{owner}/{repo}/pulls`.
//
// Note `html_url`: the GraphQL shape this replaced called the same field `url`,
// and a silent zero value there would have produced a queue whose entries point
// nowhere rather than an error.
type restPR struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"` // read by fetchPRWith; the list asks for open ones only
}

// restComment is one item of `GET /repos/{owner}/{repo}/issues/{n}/comments`.
// Pull request conversation comments are issue comments; this is the same set
// the previous `gh pr list --json comments` returned.
type restReview struct {
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	State       string    `json:"state"`
	Body        string    `json:"body"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type restComment struct {
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	AuthorAssociation string    `json:"author_association"`
	Body              string    `json:"body"`
	CreatedAt         time.Time `json:"created_at"`

	// The field TOOL-019 (#1422) needed and could not reach. That issue
	// concluded the fix was "not reachable through the current data source",
	// which was true of `gh pr view --json comments` -- it exposes only
	// `createdAt` and a timestamp-less `includesCreatedEdit` boolean. CLI-071
	// then moved this package to REST for unrelated reasons (#1454), and REST
	// has always carried `updated_at`. The blocker was dissolved by a change
	// that was not trying to; nobody revisited the ticket.
	//
	// Neither of the options that issue proposed is needed: no GraphQL, which
	// is the surface that hit a rate limit on 2026-08-31, and no digest in the
	// triage comment, whose format the reviewer registry declares.
	UpdatedAt time.Time `json:"updated_at"`
}

// Fetch asks GitHub for the open pull requests and returns the ones whose
// reviewer output is newer than their last recorded triage.
//
// It lives here rather than in the command because there are two callers and
// only one question. `dotf pr triage-queue` is the one an agent runs on purpose;
// the session-start brief is the one that runs whether or not anyone remembered
// — and a second copy of the gh call would be a second place for the domain
// conversion to drift.
//
// An empty repo means "the current repository", matching gh's own default. The
// context bounds the calls: this runs on the session-start path, where an
// unreachable API must cost a bounded wait and a visible message rather than a
// hung shell.
//
// Every failure is returned, never softened into an empty queue. That rule is
// the entire point of this package: a queue that could not be computed must not
// read as a queue with nothing in it.
func Fetch(ctx context.Context, repo, registryPath string) ([]Status, error) {
	reg, err := LoadRegistry(registryPath)
	if err != nil {
		return nil, err
	}
	return FetchWithRegistry(ctx, repo, reg)
}

// FetchRegistry reads repo's reviewer registry from its default branch; an
// empty repo is the current one. A repository without the file is an error,
// never an empty registry: the queue it would compute could not be trusted.
func FetchRegistry(ctx context.Context, repo string) (Registry, error) {
	return fetchRegistry(ctx, execGH, repo)
}

func fetchRegistry(ctx context.Context, run ghRunner, repo string) (Registry, error) {
	name := repo
	if name == "" {
		name = "the current repository"
	}
	raw, err := run(ctx, "api", "-H", "Accept: application/vnd.github.raw+json", repoBase(repo)+"/contents/"+RegistryPath)
	switch {
	case err == nil:
		return ParseRegistry(raw, name+":"+RegistryPath)
	case strings.Contains(err.Error(), "HTTP 404"):
		// GitHub answers 404 for a repository the token cannot read as well as
		// for a missing file, so the message names both.
		return Registry{}, fmt.Errorf("%s has no %s on its default branch, or the token cannot read it (HTTP 404)", name, RegistryPath)
	}
	return Registry{}, fmt.Errorf("read %s from %s: %w", RegistryPath, name, err)
}

// FetchWithRegistry asks GitHub for open PRs using a pre-loaded Registry,
// avoiding redundant disk reads when the caller already has the registry.
func FetchWithRegistry(ctx context.Context, repo string, reg Registry) ([]Status, error) {
	return fetchWith(ctx, execGH, repo, reg)
}

// fetchWith is FetchWithRegistry with the GitHub access injected.
//
// It speaks REST and only REST. The previous implementation used
// `gh pr list --json …`, a GraphQL query that returned pull requests with their
// comments nested in one response; REST has no such nesting, so the join lives
// here as a bounded fan-out. That cost buys independence from the GraphQL
// bucket, which refused every list and write for a stretch on 2026-09-02 while
// REST served the same data throughout.
func fetchWith(ctx context.Context, run ghRunner, repo string, reg Registry) ([]Status, error) {
	base := repoBase(repo)

	out, err := run(ctx, "api", fmt.Sprintf("%s/pulls?state=open&per_page=%d", base, prLimit))
	if err != nil {
		return nil, fmt.Errorf("gh api pulls: %w", err)
	}
	var wire []restPR
	if err := json.Unmarshal(out, &wire); err != nil {
		return nil, fmt.Errorf("parse open pull requests: %w", err)
	}
	if len(wire) >= prLimit {
		return nil, fmt.Errorf("GitHub returned %d open pull requests (a full page); the queue may be truncated — page through with `gh api %s/pulls --paginate`", prLimit, base)
	}

	return pendingOf(ctx, run, base, wire, reg)
}

// FetchPR is FetchWithRegistry scoped to one pull request: the answer is that
// PR's verdict alone, so another session's untriaged PR cannot hold up
// `triage-queue --pr N && pr land N` (#2549). The open-PR list is never read.
//
// The repo-wide refusals hold here too: a PR that cannot be read, or whose
// comments cannot be, is an error. So is a PR that is not open, which the
// repo-wide queue never sees: an empty answer for a merged PR would read as
// "clear to land" for a PR that can no longer land.
func FetchPR(ctx context.Context, repo string, reg Registry, number int) ([]Status, error) {
	return fetchPRWith(ctx, execGH, repo, reg, number)
}

func fetchPRWith(ctx context.Context, run ghRunner, repo string, reg Registry, number int) ([]Status, error) {
	base := repoBase(repo)
	out, err := run(ctx, "api", fmt.Sprintf("%s/pulls/%d", base, number))
	if err != nil {
		return nil, fmt.Errorf("gh api pull request #%d: %w", number, err)
	}
	var wire restPR
	if err := json.Unmarshal(out, &wire); err != nil {
		return nil, fmt.Errorf("parse pull request #%d: %w", number, err)
	}
	if wire.Number != number {
		return nil, fmt.Errorf("asked for pull request #%d, GitHub answered with #%d", number, wire.Number)
	}
	if wire.State != "open" {
		return nil, fmt.Errorf("pull request #%d is %s, not open; its queue cannot be clear to land", number, wire.State)
	}
	return pendingOf(ctx, run, base, []restPR{wire}, reg)
}

// pendingOf joins each pull request to its conversation and keeps the ones the
// registry reports pending, less any whose diff is an exempt signature. It is
// the one verdict path both the repo-wide and the scoped queue take.
func pendingOf(ctx context.Context, run ghRunner, base string, wire []restPR, reg Registry) ([]Status, error) {
	prs, err := withComments(ctx, run, base, wire)
	if err != nil {
		return nil, err
	}
	pending := Queue(prs, reg)
	if len(reg.Exempt.Signatures) > 0 {
		if pending, err = withoutExempt(ctx, run, base, pending, reg); err != nil {
			return nil, err
		}
	}
	return withFindings(ctx, run, base, pending, reg), nil
}

// withFindings attaches each pending PR's unclassified inline findings
// (TOOL-017 AC3). Like the file list, they are read only for a PR the queue
// reports, and only when the registry asks some reviewer for a class.
//
// A read failure marks the PR FindingsUnread and keeps it listed: the findings
// are a report on top of a verdict already computed, so losing them must make
// the answer say "unknown", never drop the PR or the rest of the queue.
func withFindings(ctx context.Context, run ghRunner, base string, pending []Status, reg Registry) []Status {
	if len(reg.Classification.Reviewers) == 0 {
		return pending
	}
	for i, st := range pending {
		inline, err := fetchInline(ctx, run, base, st.PR.Number)
		if err != nil {
			pending[i].FindingsUnread = true
			continue
		}
		pending[i].Unclassified = UnclassifiedFindings(st.PR, inline, reg)
	}
	return pending
}

// withoutExempt drops a pending PR whose diff is an exempt signature. The file
// list is read only for a PR the queue would report, which is rarely more than
// a few, so the session-start path pays no call for a quiet PR.
//
// A file list that cannot be read keeps its PR listed and costs nothing else.
// The queue is already computed at this point; an exemption may only make it
// quieter, so failing to establish one leaves it as loud as it was, and never
// turns into an error that would discard every other entry (the session-start
// probe runs under a deadline, and a slow call here must not empty its answer).
func withoutExempt(ctx context.Context, run ghRunner, base string, pending []Status, reg Registry) ([]Status, error) {
	kept := pending[:0]
	for _, st := range pending {
		files, err := fetchFiles(ctx, run, base, st.PR.Number)
		if err != nil {
			kept = append(kept, st)
			continue
		}
		pr := st.PR
		pr.Files = files
		if ev := Evaluate(pr, reg); ev.Pending {
			kept = append(kept, st)
		}
	}
	return kept, nil
}

// fetchFiles reads one pull request's changed paths. A full page cannot be
// compared with a signature, so it returns nil, which is never exempt: the
// PR stays in the queue rather than leaving it on partial data.
func fetchFiles(ctx context.Context, run ghRunner, base string, number int) ([]string, error) {
	out, err := run(ctx, "api", fmt.Sprintf("%s/pulls/%d/files?per_page=%d", base, number, filesLimit))
	if err != nil {
		return nil, fmt.Errorf("gh api files for #%d: %w", number, err)
	}
	var wire []struct {
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(out, &wire); err != nil {
		return nil, fmt.Errorf("parse files for #%d: %w", number, err)
	}
	if len(wire) >= filesLimit {
		return nil, nil
	}
	files := make([]string, 0, len(wire))
	for _, w := range wire {
		files = append(files, w.Filename)
	}
	return files, nil
}

// withComments resolves every pull request's conversation, overlapping the calls
// because REST needs one per pull request where the old GraphQL query needed
// none, and the session-start probe that calls this is bounded at five seconds.
//
// The fan-out is capped rather than unbounded: a burst against a rate limit is
// the condition this package is escaping, so reproducing it here would be
// self-defeating.
func withComments(ctx context.Context, run ghRunner, base string, wire []restPR) ([]PR, error) {
	prs := make([]PR, len(wire))
	errs := make([]error, len(wire))
	sem := make(chan struct{}, commentFanout)
	var wg sync.WaitGroup

	for i, w := range wire {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errs[i] = ctx.Err()
				return
			}
			comments, err := fetchComments(ctx, run, base, w.Number)
			if err != nil {
				errs[i] = err
				return
			}
			reviews, err := fetchReviews(ctx, run, base, w.Number)
			if err != nil {
				errs[i] = err
				return
			}
			prs[i] = PR{Number: w.Number, Title: w.Title, URL: w.HTMLURL, Comments: comments, Reviews: reviews}
		}()
	}
	wg.Wait()

	// One unreadable pull request poisons the whole answer. A PR whose comments
	// could not be read is not a PR with no comments, and reporting the rest as
	// a complete queue is exactly the "cannot compute reads as nothing pending"
	// failure this package refuses.
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return prs, nil
}

// fetchComments reads one pull request's conversation.
func fetchComments(ctx context.Context, run ghRunner, base string, number int) ([]Comment, error) {
	out, err := run(ctx, "api", fmt.Sprintf("%s/issues/%d/comments?per_page=%d", base, number, commentLimit))
	if err != nil {
		return nil, fmt.Errorf("gh api comments for #%d: %w", number, err)
	}
	var wire []restComment
	if err := json.Unmarshal(out, &wire); err != nil {
		return nil, fmt.Errorf("parse comments for #%d: %w", number, err)
	}
	// The dangerous axis. A truncated pull request list makes a PR go missing,
	// which is visible; truncated comments make the verdict WRONG — an
	// untriaged PR can read as triaged, or a re-review can vanish behind an
	// older disposition.
	if len(wire) >= commentLimit {
		return nil, fmt.Errorf("pull request #%d returned %d comments (a full page); its triage verdict would be computed from partial data — read them with `gh api %s/issues/%d/comments --paginate`", number, commentLimit, base, number)
	}

	comments := make([]Comment, 0, len(wire))
	for _, w := range wire {
		comments = append(comments, Comment{
			Author: w.User.Login, AuthorAssociation: w.AuthorAssociation,
			Body: w.Body, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
		})
	}
	return comments, nil
}

// fetchReviews reads one pull request's submitted reviews. Issue comments alone
// miss a reviewer that files its findings as a review with inline comments and
// no marker: CodeRabbit does, and #2581 merged over three of them (#1686).
// Inline comments need no read of their own, since each belongs to a review.
func fetchReviews(ctx context.Context, run ghRunner, base string, number int) ([]Comment, error) {
	out, err := run(ctx, "api", fmt.Sprintf("%s/pulls/%d/reviews?per_page=%d", base, number, commentLimit))
	if err != nil {
		return nil, fmt.Errorf("gh api reviews for #%d: %w", number, err)
	}
	var wire []restReview
	if err := json.Unmarshal(out, &wire); err != nil {
		return nil, fmt.Errorf("parse reviews for #%d: %w", number, err)
	}
	if len(wire) >= commentLimit {
		return nil, fmt.Errorf("pull request #%d returned %d reviews (a full page); its triage verdict would be computed from partial data — read them with `gh api %s/pulls/%d/reviews --paginate`", number, commentLimit, base, number)
	}
	reviews := make([]Comment, 0, len(wire))
	for _, w := range wire {
		// A dismissed review was withdrawn and a pending one never submitted.
		if w.State == "DISMISSED" || w.State == "PENDING" || w.SubmittedAt.IsZero() {
			continue
		}
		reviews = append(reviews, Comment{Author: w.User.Login, Body: w.Body, CreatedAt: w.SubmittedAt})
	}
	return reviews, nil
}

// restInline is one item of `GET /repos/{owner}/{repo}/pulls/{n}/comments`.
// Line is null on a comment whose line left the diff; it reads as 0.
type restInline struct {
	restComment
	Path        string `json:"path"`
	Line        int    `json:"line"`
	InReplyToID int64  `json:"in_reply_to_id"`
}

// fetchInline reads one pull request's inline review comments, the findings a
// reviewer anchors to a file and line. A reply is conversation about a finding,
// not a finding, so it is skipped. A full page is an error: a finding past it
// would go unreported.
func fetchInline(ctx context.Context, run ghRunner, base string, number int) ([]Comment, error) {
	out, err := run(ctx, "api", fmt.Sprintf("%s/pulls/%d/comments?per_page=%d", base, number, commentLimit))
	if err != nil {
		return nil, fmt.Errorf("gh api inline comments for #%d: %w", number, err)
	}
	var wire []restInline
	if err := json.Unmarshal(out, &wire); err != nil {
		return nil, fmt.Errorf("parse inline comments for #%d: %w", number, err)
	}
	if len(wire) >= commentLimit {
		return nil, fmt.Errorf("pull request #%d returned %d inline comments (a full page)", number, commentLimit)
	}
	inline := make([]Comment, 0, len(wire))
	for _, w := range wire {
		if w.InReplyToID != 0 {
			continue
		}
		inline = append(inline, Comment{Author: w.User.Login, Body: w.Body,
			CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt, Path: w.Path, Line: w.Line})
	}
	return inline, nil
}

// repoBase renders the REST path prefix. An empty repo yields gh's own
// `{owner}/{repo}` placeholders, which gh resolves from the current repository —
// preserving the default the session-start probe relies on, since it passes ""
// and runs wherever the shell happens to be.
func repoBase(repo string) string {
	if repo == "" {
		return "repos/{owner}/{repo}"
	}
	return "repos/" + repo
}

// Marker returns the triage heading a registry declares, for a caller that needs
// to name it in a message. It re-reads the registry rather than caching: this is
// invoked once per session, and a stale marker in a long-lived process would be
// the same two-file disagreement the registry exists to prevent.
func Marker(registryPath string) (string, error) {
	reg, err := LoadRegistry(registryPath)
	if err != nil {
		return "", err
	}
	return reg.Triage.Marker, nil
}
