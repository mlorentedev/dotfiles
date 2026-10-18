// Package prland merges a pull request only when CI, review triage and
// freshness all hold on one head commit (CLI-098, #2034). It is the mechanical
// half of the owner's merge rule; the judgement on reviewer output stays with
// the pr-review-triage skill, whose record this package only reads.
package prland

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Facts is what Decide needs, read from GitHub for one head commit.
type Facts struct {
	State      string // OPEN | CLOSED | MERGED
	IsDraft    bool
	HeadSHA    string
	HeadRef    string
	BaseRef    string
	MergeState string // CLEAN | BEHIND | DIRTY | BLOCKED | UNSTABLE | ...
	Checks     []Check
	Untriaged  bool // the triage queue lists this PR
	// GrantDeclared is whether the base branch carries GrantFile, the
	// repository's opt-in to the standing merge grant.
	GrantDeclared bool
	// Excluded is the grant's `excluded:` list, read from the base branch
	// with the grant itself.
	Excluded []string
	// Files is every path the PR changes, a renamed file's old path
	// included; FilesComplete is false when GitHub listed fewer files than
	// the PR changes, so a path may be missing.
	Files         []string
	FilesComplete bool
	// changedFiles is GitHub's count of the files the PR changes, the
	// figure Files is checked against.
	changedFiles int
	// UnknownFor is how long --wait waited on an UNKNOWN merge state with
	// every check green, so a refusal can say that waiting was tried.
	UnknownFor time.Duration
}

// Check is one status check and gh's bucket for it: pass, fail, pending,
// skipping or cancel.
type Check struct {
	Name   string `json:"name"`
	Bucket string `json:"bucket"`
}

// releasePrefix marks release-please branches. Cutting a release is the
// owner's decision, so a match only ever forfeits the merge, never grants one.
const releasePrefix = "release-please--"

// GrantFile opts a repository in to the standing merge grant: an agent may
// merge a PR it opened once CI, conflicts and triage hold (pattern-git-workflow
// §9, #2178). It is read from the base branch, so a PR cannot add it to grant
// itself the merge.
const GrantFile = ".github/merge-grant.yml"

// The exclusion kinds that take an argument. A branch prefix only ever
// forfeits the merge: anyone who can push a branch can choose its name, so a
// name never grants anything (pattern-git-workflow §9). A path excludes every
// PR that changes it, whatever its branch is called, so it also catches a
// promotion opened on a branch nobody named for it.
const (
	branchExclusion = "branch:"
	pathExclusion   = "path:" // an exact path, or a directory when it ends in "/"
)

// Decide returns every reason the PR cannot land; none means it can.
func Decide(f Facts) []string {
	var r []string
	if f.State != "OPEN" {
		r = append(r, "PR is not open ("+f.State+")")
	}
	if f.IsDraft {
		r = append(r, "PR is a draft")
	}
	if strings.HasPrefix(f.HeadRef, releasePrefix) {
		r = append(r, "release-please PRs are merged by the owner")
	}
	r = append(r, excludedReasons(f, f.Excluded)...)
	if !f.FilesComplete {
		r = append(r, "the PR's file list could not be read whole, so it may change "+GrantFile)
	} else if slices.Contains(f.Files, GrantFile) {
		r = append(r, "the PR changes "+GrantFile+": a PR that changes the grant is never merged under it")
	}
	if len(f.Checks) == 0 {
		r = append(r, "no checks reported")
	}
	for _, c := range f.Checks {
		if c.Bucket != "pass" && c.Bucket != "skipping" {
			r = append(r, c.Name+": "+c.Bucket)
		}
	}
	switch {
	case f.MergeState == "UNKNOWN" && f.UnknownFor > 0:
		r = append(r, fmt.Sprintf("merge state is UNKNOWN after waiting %s for GitHub to compute it; a new head (a re-created merge of the base) makes it compute again", f.UnknownFor))
	case f.MergeState != "CLEAN":
		r = append(r, "merge state is "+f.MergeState)
	}
	if f.Untriaged {
		r = append(r, "reviewer output awaits triage")
	}
	if !f.GrantDeclared {
		r = append(r, "the repository has not opted in to the standing merge grant ("+GrantFile+" on "+f.BaseRef+"): the owner merges this PR")
	}
	return r
}

// excludedReasons applies the grant's exclusions to the head branch. An
// exclusion of a kind this package does not know refuses the merge rather
// than being skipped: a grant that says more than its actor reads would be a
// promise nothing keeps.
func excludedReasons(f Facts, excluded []string) []string {
	var r []string
	for _, e := range excluded {
		switch {
		case e == "release-please":
			// Decide checks releasePrefix whatever the grant says.
		case strings.HasPrefix(e, branchExclusion) && len(e) > len(branchExclusion):
			if p := strings.TrimPrefix(e, branchExclusion); strings.HasPrefix(f.HeadRef, p) {
				r = append(r, "the grant excludes branches starting "+p+": the owner merges this PR")
			}
		case strings.HasPrefix(e, pathExclusion) && len(e) > len(pathExclusion):
			p := strings.TrimPrefix(e, pathExclusion)
			if i := slices.IndexFunc(f.Files, func(file string) bool {
				return file == p || strings.HasSuffix(p, "/") && strings.HasPrefix(file, p)
			}); i >= 0 {
				r = append(r, "the grant excludes PRs that change "+p+" ("+f.Files[i]+"): the owner merges this PR")
			}
		default:
			r = append(r, fmt.Sprintf("the grant declares an exclusion this version does not understand (%q): the owner merges this PR", e))
		}
	}
	return r
}

// Runner runs gh with args and returns its stdout.
type Runner func(ctx context.Context, args ...string) ([]byte, error)

// Options wires Land to GitHub.
type Options struct {
	Run       Runner
	Untriaged func(ctx context.Context) ([]int, error) // PR numbers the triage queue lists
	Repo      string                                   // owner/name; "" is the current repository
	Wait      bool                                     // wait for the checks before reading them
	// Sleep pauses between --wait rounds, so checks a push has not registered
	// yet get time to appear; nil means time.Sleep.
	Sleep func(time.Duration)
	// UpdateBranch merges the base into a PR whose only failing condition is
	// BEHIND, waits for the new CI, and decides again on the new head.
	UpdateBranch bool
	// UnknownWait bounds how long --wait waits for GitHub to compute the merge
	// state of a PR whose checks are all green; 0 means DefaultUnknownWait.
	UnknownWait time.Duration
}

// Result is what Land did.
type Result struct {
	Number     int // the PR this is the result for
	Merged     bool
	Updated    bool // the base was merged into the branch first
	HeadSHA    string
	Reasons    []string // why it did not merge
	Retargeted []int    // dependents pointed at the base before the merge
	Warnings   []string // what went wrong after the merge landed
}

// Land merges the PR when Decide finds no reason against it and the head did
// not move while the facts were read. Before the merge deletes the head branch,
// every PR based on it is pointed at this PR's base, because GitHub closes a
// PR whose base branch is deleted. An error means a fact could not be read; a
// refusal is a Result with reasons.
func Land(ctx context.Context, o Options, number int) (Result, error) {
	gh := o.gh(ctx)
	n := strconv.Itoa(number)
	f, err := settle(ctx, gh, o, number, o.Wait)
	if err != nil {
		return Result{Number: number}, err
	}
	g, err := readGrant(ctx, o, f.BaseRef)
	if err != nil {
		return Result{Number: number}, err
	}
	decide := func(f Facts) []string {
		f.GrantDeclared, f.Excluded = g.declared, g.excluded
		return Decide(f)
	}
	res := Result{Number: number, HeadSHA: f.HeadSHA, Reasons: decide(f)}
	// Again while the base moves under the new head's CI, which a busy queue
	// does: one update left the PR behind the next merge (#2041).
	updates := 0
	for ; o.UpdateBranch && onlyBehind(res.Reasons) && updates < maxUpdates; updates++ {
		// A merge of the base, not a rebase: one merge commit, which the
		// reviewer's push gate does not count, and the squash flattens anyway.
		if _, err := gh("pr", "update-branch", n); err != nil {
			return res, fmt.Errorf("gh pr update-branch: %w", err)
		}
		res.Updated = true
		if f, err = settle(ctx, gh, o, number, true); err != nil {
			return res, err
		}
		res.HeadSHA, res.Reasons = f.HeadSHA, decide(f)
	}
	// A bare "BEHIND" after three updates reads as if none was tried. Say the
	// base outran CI, which is the race #2083 describes, not a stale branch.
	if updates > 0 && onlyBehind(res.Reasons) {
		res.Reasons = []string{fmt.Sprintf("merge state is BEHIND after merging the base in %d times: the base moved again during each CI run (#2083)", updates)}
	}
	if len(res.Reasons) > 0 {
		return res, nil
	}
	return merge(gh, n, f, res)
}

// gh runs gh for the repository the options name.
func (o Options) gh(ctx context.Context) ghFunc {
	return func(args ...string) ([]byte, error) {
		if o.Repo != "" {
			args = append(args, "-R", o.Repo)
		}
		return o.Run(ctx, args...)
	}
}

// Blockers is every reason the PR cannot land right now, read the way Land
// reads them and from the same Decide, and none when it can. It reads once and
// waits for nothing, and it changes nothing: a PR still running its checks is
// a reason, not a pause. An error means a fact could not be read, which is not
// the same as "nothing blocks it".
func Blockers(ctx context.Context, o Options, number int) ([]string, error) {
	f, err := settle(ctx, o.gh(ctx), o, number, false)
	if err != nil {
		return nil, err
	}
	g, err := readGrant(ctx, o, f.BaseRef)
	if err != nil {
		return nil, err
	}
	f.GrantDeclared, f.Excluded = g.declared, g.excluded
	return Decide(f), nil
}

// apiRepo names the repository in a `gh api` path, which takes no -R flag.
func apiRepo(o Options) string {
	if o.Repo == "" {
		return "{owner}/{repo}" // gh api fills these from the current repository
	}
	return o.Repo
}

// baseGrant is what the base branch declares in GrantFile.
type baseGrant struct {
	declared bool
	excluded []string
}

// readGrant reads GrantFile on base. Only a 404 means it is not declared; any
// other failure, a file that does not decode included, is an error, because
// an unanswered question must not read as either answer.
func readGrant(ctx context.Context, o Options, base string) (baseGrant, error) {
	out, err := o.Run(ctx, "api", "repos/"+apiRepo(o)+"/contents/"+GrantFile+"?ref="+base)
	switch {
	case err == nil:
	case strings.Contains(err.Error(), "HTTP 404"):
		return baseGrant{}, nil
	default:
		return baseGrant{}, fmt.Errorf("read %s on %s: %w", GrantFile, base, err)
	}
	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(out, &file); err != nil {
		return baseGrant{}, fmt.Errorf("read %s on %s: %w", GrantFile, base, err)
	}
	if file.Encoding != "base64" {
		return baseGrant{}, fmt.Errorf("read %s on %s: content encoding %q, want base64", GrantFile, base, file.Encoding)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil {
		return baseGrant{}, fmt.Errorf("decode %s on %s: %w", GrantFile, base, err)
	}
	var doc struct {
		Excluded []string `yaml:"excluded"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return baseGrant{}, fmt.Errorf("parse %s on %s: %w", GrantFile, base, err)
	}
	return baseGrant{declared: true, excluded: doc.Excluded}, nil
}

// settle reads the facts and, when wait is set, keeps pausing and re-reading
// while more waiting can still change them.
func settle(ctx context.Context, gh ghFunc, o Options, number int, wait bool) (Facts, error) {
	sleep := o.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	budget := o.UnknownWait
	if budget <= 0 {
		budget = DefaultUnknownWait
	}
	f, err := readFacts(ctx, gh, o, number)
	rounds, uncomputedFor := 0, time.Duration(0)
	for wait && err == nil && unsettled(f) {
		// Two budgets, because the two waits fail differently. Checks finish
		// in minutes or fail; a merge state GitHub has not computed was
		// measured null for 27 minutes with every check green (#2118), and
		// six rounds gave up long before it was computed.
		pause := waitPause
		if uncomputed(f) {
			if uncomputedFor >= budget {
				break
			}
			// The last pause is cut to what is left, so --unknown-wait 45s
			// waits 45s, not two whole pauses.
			pause = min(pause, budget-uncomputedFor)
			uncomputedFor += pause
		} else {
			if rounds >= maxWaitRounds {
				break
			}
			rounds++
		}
		sleep(pause)
		_, _ = gh("pr", "checks", strconv.Itoa(number), "--watch", "--interval", "30") // the facts below decide
		f, err = readFacts(ctx, gh, o, number)
	}
	if f.MergeState == "UNKNOWN" {
		f.UnknownFor = uncomputedFor
	}
	return f, err
}

// merge re-reads the head, refuses if it moved, retargets the dependents and
// squash-merges the head it checked.
func merge(gh ghFunc, n string, f Facts, res Result) (Result, error) {
	again, err := readView(gh, n)
	if err != nil {
		return res, err
	}
	if again.HeadSHA != f.HeadSHA {
		res.Reasons = []string{fmt.Sprintf("head moved from %s to %s while the facts were read", short(f.HeadSHA), short(again.HeadSHA))}
		return res, nil
	}
	moved, err := retargetDependents(gh, f.HeadRef, f.BaseRef)
	if err != nil {
		return res, errors.Join(err, restoreDependents(gh, moved, f.HeadRef))
	}
	// The retarget and the merge are one step: a merge that fails (the head
	// moved, the base moved, a permission) puts the dependents back on the
	// head branch, so a refused landing leaves the stack as it found it.
	if _, err := gh("pr", "merge", n, "--squash", "--delete-branch", "--match-head-commit", f.HeadSHA); err != nil {
		// gh also fails after the merge landed: the head branch deletion is
		// refused, or the client times out. A PR that reads MERGED landed, and
		// its dependents belong on the base, so it is a warning, not a restore.
		after, verr := readView(gh, n)
		if verr != nil || after.State != "MERGED" {
			return res, errors.Join(fmt.Errorf("gh pr merge: %w", err), restoreDependents(gh, moved, f.HeadRef))
		}
		res.Warnings = append(res.Warnings, fmt.Sprintf("merged, but gh pr merge reported: %v", err))
	}
	res.Retargeted, res.Merged = moved, true
	return res, nil
}

// restoreDependents points each PR back at the head branch it was stacked on.
// A PR it cannot restore is named in the error, never left silently.
func restoreDependents(gh ghFunc, moved []int, head string) error {
	var errs []error
	for _, d := range moved {
		if _, err := gh("pr", "edit", strconv.Itoa(d), "--base", head); err != nil {
			errs = append(errs, fmt.Errorf("#%d is left based on the target branch, not %s: %w", d, head, err))
		}
	}
	return errors.Join(errs...)
}

// maxUpdates bounds --update-branch: a base that keeps moving is a refusal
// that names BEHIND, not an endless chase.
const maxUpdates = 3

func onlyBehind(reasons []string) bool {
	return len(reasons) == 1 && reasons[0] == "merge state is BEHIND"
}

// maxWaitRounds bounds --wait while checks settle: each round is a pause and
// one `gh pr checks --watch`. DefaultUnknownWait bounds the separate wait for a
// merge state GitHub has not computed; it is 20 minutes because the longest
// measured on 2026-10-07 was 27, and past that a new head is the remedy, not
// more waiting.
const (
	maxWaitRounds      = 6
	waitPause          = 30 * time.Second
	DefaultUnknownWait = 20 * time.Minute
)

// unsettled reports facts that more waiting can still change: a check still
// pending, or a merge state GitHub has not finished computing. Right after a
// push, `gh pr checks --watch` can return before every check is registered,
// leaving the state BLOCKED with every reported check green.
func unsettled(f Facts) bool {
	if len(f.Checks) == 0 {
		return true // CI has not registered its checks yet
	}
	for _, c := range f.Checks {
		if c.Bucket == "pending" {
			return true
		}
	}
	switch f.MergeState {
	case "UNKNOWN", "BLOCKED", "UNSTABLE":
		return len(f.Checks) == 0 || allGreen(f.Checks)
	}
	return false
}

// uncomputed reports the wait only GitHub can end: the merge state is still
// UNKNOWN and no check is pending or failing. No checks at all counts too: on
// #2105 no pull_request workflow ran on the head until GitHub computed it, so
// waiting for checks could never end that wait. BLOCKED is not this case,
// since a PR can be blocked for good with every check green.
func uncomputed(f Facts) bool {
	return f.MergeState == "UNKNOWN" && allGreen(f.Checks)
}

func allGreen(cs []Check) bool {
	for _, c := range cs {
		if c.Bucket != "pass" && c.Bucket != "skipping" {
			return false
		}
	}
	return true
}

type ghFunc func(args ...string) ([]byte, error)

// ghNoChecks is the error gh returns when a head has no checks yet
// (pkg/cmd/pr/checks/checks.go in cli/cli, as of gh 2.102).
const ghNoChecks = "no checks reported on the"

// ghRemedy names the fix for a gh too old for `pr checks --json`.
func ghRemedy(err error) string {
	if strings.Contains(err.Error(), "unknown flag: --json") {
		return " (this gh predates `gh pr checks --json`; install a newer gh)"
	}
	return ""
}

func readFacts(ctx context.Context, gh ghFunc, o Options, number int) (Facts, error) {
	n := strconv.Itoa(number)
	f, err := readView(gh, n)
	if err != nil {
		return f, err
	}
	changed, err := readChangedFiles(ctx, o, number)
	if err != nil {
		return f, err
	}
	f.Files = changed.paths
	f.FilesComplete = changed.listed >= f.changedFiles
	// gh exits non-zero while a check fails or is pending, and still prints
	// the JSON; only an output that does not parse is an error.
	out, runErr := gh("pr", "checks", n, "--json", "name,bucket")
	// Before CI has registered any check, gh exits non-zero, prints nothing
	// and says why on stderr: that is "no checks yet" (Decide refuses it,
	// --wait waits for it), not an unreadable answer. Any other failure with
	// nothing on stdout is an error. A gh that predates `pr checks --json`
	// fails the same way with "unknown flag", and reading that as "no checks"
	// refused a PR with twenty green checks for the wrong reason (#2056).
	if len(bytes.TrimSpace(out)) == 0 {
		if runErr != nil && !strings.Contains(runErr.Error(), ghNoChecks) {
			return f, fmt.Errorf("gh pr checks: %w%s", runErr, ghRemedy(runErr))
		}
		f.Checks = nil
	} else if err := json.Unmarshal(out, &f.Checks); err != nil {
		return f, fmt.Errorf("gh pr checks: %v (%w)", runErr, err)
	}
	pending, err := o.Untriaged(ctx)
	if err != nil {
		return f, fmt.Errorf("triage queue: %w", err)
	}
	f.Untriaged = slices.Contains(pending, number)
	return f, nil
}

func readView(gh ghFunc, n string) (Facts, error) {
	out, err := gh("pr", "view", n, "--json", "state,isDraft,headRefOid,headRefName,baseRefName,mergeStateStatus,changedFiles")
	if err != nil {
		return Facts{}, fmt.Errorf("gh pr view: %w", err)
	}
	var v struct {
		State        string `json:"state"`
		IsDraft      bool   `json:"isDraft"`
		HeadSHA      string `json:"headRefOid"`
		HeadRef      string `json:"headRefName"`
		BaseRef      string `json:"baseRefName"`
		MergeState   string `json:"mergeStateStatus"`
		ChangedFiles int    `json:"changedFiles"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return Facts{}, fmt.Errorf("gh pr view: %w", err)
	}
	return Facts{State: v.State, IsDraft: v.IsDraft, HeadSHA: v.HeadSHA, HeadRef: v.HeadRef, BaseRef: v.BaseRef, MergeState: v.MergeState, changedFiles: v.ChangedFiles}, nil
}

// changedFiles is what the pull request files endpoint lists.
type changedFiles struct {
	paths  []string // every filename, and every renamed file's old name
	listed int      // files listed, which counts a rename once
}

// readChangedFiles lists the PR's files through the REST endpoint, not `gh pr
// view --json files`: GraphQL names only a renamed file's new path, so a PR
// that renamed the grant away would read as leaving it alone. REST names the
// old path too (previous_filename) and pages up to 3000 files.
func readChangedFiles(ctx context.Context, o Options, number int) (changedFiles, error) {
	out, err := o.Run(ctx, "api", "repos/"+apiRepo(o)+"/pulls/"+strconv.Itoa(number)+"/files",
		"--paginate", "--jq", ".[] | {filename, previous_filename}")
	if err != nil {
		return changedFiles{}, fmt.Errorf("list the files of #%d: %w", number, err)
	}
	var c changedFiles
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var file struct {
			Filename string `json:"filename"`
			Previous string `json:"previous_filename"`
		}
		if err := dec.Decode(&file); errors.Is(err, io.EOF) {
			return c, nil
		} else if err != nil {
			return changedFiles{}, fmt.Errorf("list the files of #%d: %w", number, err)
		}
		c.listed++
		c.paths = append(c.paths, file.Filename)
		if file.Previous != "" {
			c.paths = append(c.paths, file.Previous)
		}
	}
}

func retargetDependents(gh ghFunc, head, base string) ([]int, error) {
	out, err := gh("pr", "list", "--base", head, "--state", "open", "--json", "number")
	if err != nil {
		return nil, fmt.Errorf("gh pr list: %w", err)
	}
	var deps []struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(out, &deps); err != nil {
		return nil, fmt.Errorf("gh pr list: %w", err)
	}
	var moved []int
	for _, d := range deps {
		if _, err := gh("pr", "edit", strconv.Itoa(d.Number), "--base", base); err != nil {
			return moved, fmt.Errorf("retarget #%d to %s: %w", d.Number, base, err)
		}
		moved = append(moved, d.Number)
	}
	return moved, nil
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
