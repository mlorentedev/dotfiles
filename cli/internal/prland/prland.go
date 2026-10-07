// Package prland merges a pull request only when CI, review triage and
// freshness all hold on one head commit (CLI-098, #2034). It is the mechanical
// half of the owner's merge rule; the judgement on reviewer output stays with
// the pr-review-triage skill, whose record this package only reads.
package prland

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
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
	if len(f.Checks) == 0 {
		r = append(r, "no checks reported")
	}
	for _, c := range f.Checks {
		if c.Bucket != "pass" && c.Bucket != "skipping" {
			r = append(r, c.Name+": "+c.Bucket)
		}
	}
	if f.MergeState != "CLEAN" {
		r = append(r, "merge state is "+f.MergeState)
	}
	if f.Untriaged {
		r = append(r, "reviewer output awaits triage")
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
}

// Result is what Land did.
type Result struct {
	Number     int // the PR this is the result for
	Merged     bool
	Updated    bool // the base was merged into the branch first
	HeadSHA    string
	Reasons    []string // why it did not merge
	Retargeted []int    // dependents pointed at the base before the merge
}

// Land merges the PR when Decide finds no reason against it and the head did
// not move while the facts were read. Before the merge deletes the head branch,
// every PR based on it is pointed at this PR's base, because GitHub closes a
// PR whose base branch is deleted. An error means a fact could not be read; a
// refusal is a Result with reasons.
func Land(ctx context.Context, o Options, number int) (Result, error) {
	gh := func(args ...string) ([]byte, error) {
		if o.Repo != "" {
			args = append(args, "-R", o.Repo)
		}
		return o.Run(ctx, args...)
	}
	n := strconv.Itoa(number)
	f, err := settle(ctx, gh, o, number, o.Wait)
	if err != nil {
		return Result{Number: number}, err
	}
	res := Result{Number: number, HeadSHA: f.HeadSHA, Reasons: Decide(f)}
	// Again while the base moves under the new head's CI, which a busy queue
	// does: one update left the PR behind the next merge (#2041).
	for i := 0; o.UpdateBranch && onlyBehind(res.Reasons) && i < maxUpdates; i++ {
		// A merge of the base, not a rebase: one merge commit, which the
		// reviewer's push gate does not count, and the squash flattens anyway.
		if _, err := gh("pr", "update-branch", n); err != nil {
			return res, fmt.Errorf("gh pr update-branch: %w", err)
		}
		res.Updated = true
		if f, err = settle(ctx, gh, o, number, true); err != nil {
			return res, err
		}
		res.HeadSHA, res.Reasons = f.HeadSHA, Decide(f)
	}
	if len(res.Reasons) > 0 {
		return res, nil
	}
	return merge(gh, n, f, res)
}

// settle reads the facts and, when wait is set, keeps pausing and re-reading
// while more waiting can still change them.
func settle(ctx context.Context, gh ghFunc, o Options, number int, wait bool) (Facts, error) {
	sleep := o.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	f, err := readFacts(ctx, gh, o.Untriaged, number)
	for round := 0; wait && err == nil && unsettled(f) && round < maxWaitRounds; round++ {
		sleep(waitPause)
		_, _ = gh("pr", "checks", strconv.Itoa(number), "--watch", "--interval", "30") // the facts below decide
		f, err = readFacts(ctx, gh, o.Untriaged, number)
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
		return res, errors.Join(fmt.Errorf("gh pr merge: %w", err), restoreDependents(gh, moved, f.HeadRef))
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

// maxWaitRounds bounds --wait: each round is a pause and one
// `gh pr checks --watch`.
const (
	maxWaitRounds = 6
	waitPause     = 30 * time.Second
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

func allGreen(cs []Check) bool {
	for _, c := range cs {
		if c.Bucket != "pass" && c.Bucket != "skipping" {
			return false
		}
	}
	return true
}

type ghFunc func(args ...string) ([]byte, error)

func readFacts(ctx context.Context, gh ghFunc, untriaged func(context.Context) ([]int, error), number int) (Facts, error) {
	n := strconv.Itoa(number)
	f, err := readView(gh, n)
	if err != nil {
		return f, err
	}
	// gh exits non-zero while a check fails or is pending, and still prints
	// the JSON; only an output that does not parse is an error.
	out, runErr := gh("pr", "checks", n, "--json", "name,bucket")
	// Before CI has registered any check, gh exits non-zero and prints
	// nothing: that is "no checks yet" (Decide refuses it, --wait waits for
	// it), not an unreadable answer.
	if len(bytes.TrimSpace(out)) == 0 {
		f.Checks = nil
	} else if err := json.Unmarshal(out, &f.Checks); err != nil {
		return f, fmt.Errorf("gh pr checks: %v (%w)", runErr, err)
	}
	pending, err := untriaged(ctx)
	if err != nil {
		return f, fmt.Errorf("triage queue: %w", err)
	}
	f.Untriaged = slices.Contains(pending, number)
	return f, nil
}

func readView(gh ghFunc, n string) (Facts, error) {
	out, err := gh("pr", "view", n, "--json", "state,isDraft,headRefOid,headRefName,baseRefName,mergeStateStatus")
	if err != nil {
		return Facts{}, fmt.Errorf("gh pr view: %w", err)
	}
	var v struct {
		State      string `json:"state"`
		IsDraft    bool   `json:"isDraft"`
		HeadSHA    string `json:"headRefOid"`
		HeadRef    string `json:"headRefName"`
		BaseRef    string `json:"baseRefName"`
		MergeState string `json:"mergeStateStatus"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return Facts{}, fmt.Errorf("gh pr view: %w", err)
	}
	return Facts{State: v.State, IsDraft: v.IsDraft, HeadSHA: v.HeadSHA, HeadRef: v.HeadRef, BaseRef: v.BaseRef, MergeState: v.MergeState}, nil
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
