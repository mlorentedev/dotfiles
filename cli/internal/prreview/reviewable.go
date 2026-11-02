package prreview

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// A review is skipped only on a definite answer. Whatever this cannot read keeps
// the review, or narrows what a skip may rest on, and a skip never grants a
// review: review-attestation still reads the PR as unreviewed (#1618).
//
// Three questions, each with its incident. The run's event is a snapshot from
// when the run was queued, so the PR is read live: a merged PR held the
// one-at-a-time lane 18 minutes (#1923), and a stale draft flag killed a PR's
// only run (#2591). The branch is the second question, and the repository's
// choice. Only `pull_request` runs ask these two: a `/review` comment is a human
// asking on purpose. The last question, for every run, is whether PR-Agent would
// receive an empty diff, on which it publishes nothing, a failure the guard used
// to blame on NaN (#1618, #1417).

// ErrNoAuditedRef means the commit of PR-Agent to read its filters at is not
// known. It fails the step: guessing would read filters of another version.
var ErrNoAuditedRef = errors.New("pr reviewable: no audited PR-Agent commit")

const upstreamRepo = "The-PR-Agent/pr-agent"

var auditedRefPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// ReviewableInputs are the environment of the step this replaces: Repo
// (GITHUB_REPOSITORY), PR (PR_NUMBER), Event (GITHUB_EVENT_NAME) and
// DefaultBranch (DEFAULT_BRANCH), where PR-Agent reads its configuration.
// AuditedRef is the PR-Agent commit the pinned image was built from.
// SkipPrefixes lists branch prefixes that forfeit the review, separated by
// commas or whitespace.
type ReviewableInputs struct {
	Repo, PR, Event, DefaultBranch string
	AuditedRef, SkipPrefixes       string
}

// ReviewableEnv is what the decision reaches outside itself through.
type ReviewableEnv struct {
	Out io.Writer // workflow annotations
	GH  GHRunner  // nil: the gh on PATH
}

// AuditedRefFromContract reads audited_ref from the upstream contract at path. It
// answers "" when it cannot, which Reviewable refuses.
func AuditedRefFromContract(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var contract struct {
		AuditedRef string `json:"audited_ref"`
	}
	if json.Unmarshal(raw, &contract) != nil {
		return ""
	}
	return contract.AuditedRef
}

// Reviewable answers whether PR-Agent should run on this PR. The only error is
// ErrNoAuditedRef, once the PR's own state has not already answered.
func Reviewable(ctx context.Context, in ReviewableInputs, env ReviewableEnv) (bool, error) {
	if env.GH == nil {
		env.GH = execGH
	}
	r := reviewableRun{in: in, env: env}
	if in.Event == "pull_request" {
		if why := forfeit(r.readPR(ctx), in.PR, in.SkipPrefixes); why != "" {
			r.say("::notice::%s", why)
			return false, nil
		}
	}
	if !auditedRefPattern.MatchString(in.AuditedRef) {
		r.say("::error::cannot read the audited PR-Agent commit from the executing contract")
		return false, ErrNoAuditedRef
	}
	filter, err := r.loadFilter(ctx)
	if err != nil {
		annotate(env.Out, "cannot read PR-Agent's filters, so the review runs: %v", err)
		return true, nil
	}
	files, ok := r.readFiles(ctx)
	if !ok {
		annotate(env.Out, "cannot list this PR's files, so the review runs.")
		return true, nil
	}
	if filter.kept(files) > 0 {
		return true, nil
	}
	r.say("::notice::every file in this PR is in .pr_agent.toml's [ignore] list or of a type PR-Agent " +
		"never reads (#1417), so PR-Agent has nothing it may read; not running it. This is not a review: " +
		"review-attestation still needs CodeRabbit or a human (#1618).")
	return false, nil
}

// livePR is the PR as the API tells it now. The zero value, a failed read, is
// an open PR on no branch, which nothing skips.
type livePR struct {
	Merged bool   `json:"merged"`
	State  string `json:"state"`
	Draft  bool   `json:"draft"`
	Head   struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

// forfeit is why this PR is not reviewed, or "" when it may be. The prefixes only
// ever remove a review: the head ref is chosen by whoever pushed the branch, so a
// name can cost them the reviewer and can buy nothing. The branch itself is not
// repeated, for the same reason.
func forfeit(pr livePR, number, prefixes string) string {
	state := ""
	switch {
	case pr.Merged:
		state = "merged"
	case pr.State == "closed":
		state = "closed"
	case pr.Draft:
		state = "draft"
	}
	if state != "" {
		return fmt.Sprintf("PR #%s is %s now (read from the API, not the event payload); not reviewing it.", number, state)
	}
	for _, prefix := range splitPrefixes(prefixes) {
		if strings.HasPrefix(pr.Head.Ref, prefix) {
			return fmt.Sprintf("PR #%s's branch starts with %s, a prefix this repository does not review; "+
				"not reviewing it. This forfeits the reviewer and grants nothing: "+
				"review-attestation still needs CodeRabbit or a human.", number, prefix)
		}
	}
	return ""
}

// splitPrefixes never yields an empty prefix: every branch starts with "", so a
// trailing comma would skip every review in the repository.
func splitPrefixes(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
}

// reviewableRun is one decision, with what it reads from GitHub.
type reviewableRun struct {
	in  ReviewableInputs
	env ReviewableEnv
}

func (r reviewableRun) say(format string, a ...any) {
	_, _ = fmt.Fprintf(r.env.Out, format+"\n", a...)
}

func (r reviewableRun) readPR(ctx context.Context) livePR {
	var pr livePR
	out, _, err := r.env.GH(ctx, "api", "repos/"+r.in.Repo+"/pulls/"+r.in.PR)
	if err != nil || json.Unmarshal(out, &pr) != nil {
		return livePR{}
	}
	return pr
}

// content is a file through the contents API, empty when the read fails.
func (r reviewableRun) content(ctx context.Context, repo, path, ref string) []byte {
	out, _, err := r.env.GH(ctx, "api", "repos/"+repo+"/contents/"+path+"?ref="+ref, "--jq", ".content")
	if err != nil {
		return nil
	}
	// The API wraps its base64, and the decoder skips the newlines.
	raw, err := base64.StdEncoding.DecodeString(string(out))
	if err != nil {
		return nil
	}
	return raw
}

func (r reviewableRun) loadFilter(ctx context.Context) (*fileFilter, error) {
	upstream := func(path string) []byte { return r.content(ctx, upstreamRepo, path, r.in.AuditedRef) }
	cfg := r.content(ctx, r.in.Repo, ".pr_agent.toml", r.in.DefaultBranch)
	warn := func(format string, a ...any) { annotate(r.env.Out, format, a...) }
	return newFileFilter(cfg, upstream("pr_agent/settings/language_extensions.toml"),
		upstream("pr_agent/algo/language_handler.py"), warn)
}

// readFiles is the PR's changed files. A failed or empty listing is no answer:
// an API hiccup must not read as "nothing to review".
func (r reviewableRun) readFiles(ctx context.Context) ([]string, bool) {
	out, _, err := r.env.GH(ctx, "api", "repos/"+r.in.Repo+"/pulls/"+r.in.PR+"/files", "--paginate")
	if err != nil {
		return nil, false
	}
	var files []string
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for {
		var page []struct {
			Filename string `json:"filename"`
		}
		if err := dec.Decode(&page); err == io.EOF {
			return files, len(files) > 0
		} else if err != nil {
			return nil, false
		}
		for _, f := range page {
			if f.Filename != "" {
				files = append(files, f.Filename)
			}
		}
	}
}
