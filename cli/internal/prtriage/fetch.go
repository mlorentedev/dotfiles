package prtriage

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"
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
	return exec.CommandContext(ctx, "gh", args...).Output()
}

const (
	// prLimit and commentLimit are page sizes AND boundary alarms. REST
	// paginates where the old GraphQL query did not, so a full page means
	// "there may be more" and this package refuses rather than answering from
	// part of the data.
	prLimit      = 100
	commentLimit = 100

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
}

// restComment is one item of `GET /repos/{owner}/{repo}/issues/{n}/comments`.
// Pull request conversation comments are issue comments; this is the same set
// the previous `gh pr list --json comments` returned.
type restComment struct {
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
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
			prs[i] = PR{Number: w.Number, Title: w.Title, URL: w.HTMLURL, Comments: comments}
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

	return Queue(prs, reg), nil
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
		return nil, fmt.Errorf("pull request #%d returned %d comments (a full page); its triage verdict would be computed from partial data", number, commentLimit)
	}

	comments := make([]Comment, 0, len(wire))
	for _, w := range wire {
		comments = append(comments, Comment{
			Author: w.User.Login, Body: w.Body, CreatedAt: w.CreatedAt,
		})
	}
	return comments, nil
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
