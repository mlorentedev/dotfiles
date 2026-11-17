package spec

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// OwnCommit is one commit of a spec's own work inside its review range.
type OwnCommit struct {
	SHA     string
	Subject string
}

// OwnCommitSet is the spec's own commits in a review range, parents before
// children (topological, so commits sharing a timestamp keep one order), and
// how many non-merge commits the whole range holds.
type OwnCommitSet struct {
	Commits   []OwnCommit
	RangeSize int
}

// OwnCommits derives which commits in base..head are the spec's own
// (HARNESS-158, #1727): those that touch the spec folder under any of the
// names in folders, and, when issue is non-zero, those whose message
// references #issue. Nothing is recorded at merge time, so a hand-merged slice
// and an already-archived spec answer the same way.
//
// A spec whose slices merge to main one at a time is otherwise reviewed over
// every commit other work merged in between: HARNESS-142's closing review ran
// over 14 commits of which 5 were its own.
//
// Merge commits are left out: what a merge brings in is judged by its own
// commits, and a merge message naming the issue says nothing about its content.
// Default history simplification is kept on purpose: a slice on a side branch
// merged --no-ff is listed, and one whose change the merge discarded is not,
// since it is not part of what the diff shows.
//
// Any git failure is an error, never an empty set: a broken range must not
// read as a spec with no commits of its own.
func OwnCommits(repoRoot, base, head string, folders []string, issue int) (OwnCommitSet, error) {
	return OwnCommitsWidened(repoRoot, base, "", head, folders, issue)
}

// OwnCommitsWidened is OwnCommits for a base widened to the fork point
// (BUG-166): every commit in base..widenedFrom is the branch's work before it
// committed the spec folder, and is the spec's own whatever its message says.
// The commit the widening exists to include is a fix that names no issue.
func OwnCommitsWidened(repoRoot, base, widenedFrom, head string, folders []string, issue int) (OwnCommitSet, error) {
	rng := base + ".." + head
	all, err := gitLines(repoRoot, "log", "--no-merges", "--topo-order", "--reverse", "--format=%H%x00%s", rng)
	if err != nil {
		return OwnCommitSet{}, err
	}
	own := map[string]bool{}
	byPath, err := gitLines(repoRoot, append([]string{"log", "--no-merges", "--format=%H", rng, "--"}, folders...)...)
	if err != nil {
		return OwnCommitSet{}, err
	}
	for _, sha := range byPath {
		own[sha] = true
	}
	if widenedFrom != "" {
		before, err := gitLines(repoRoot, "log", "--no-merges", "--format=%H", base+".."+widenedFrom)
		if err != nil {
			return OwnCommitSet{}, err
		}
		for _, sha := range before {
			own[sha] = true
		}
	}
	if issue > 0 {
		// "#1727" not followed by a digit, so #17270 does not match.
		byIssue, err := gitLines(repoRoot, "log", "--no-merges", "-E",
			fmt.Sprintf("--grep=#%d([^0-9]|$)", issue), "--format=%H", rng)
		if err != nil {
			return OwnCommitSet{}, err
		}
		for _, sha := range byIssue {
			own[sha] = true
		}
	}
	set := OwnCommitSet{RangeSize: len(all)}
	for _, line := range all {
		sha, subject, _ := strings.Cut(line, "\x00")
		if own[sha] {
			set.Commits = append(set.Commits, OwnCommit{SHA: sha, Subject: subject})
		}
	}
	return set, nil
}

// gitLines runs git in repoRoot and returns its non-empty output lines. A
// failure carries git's stderr, so the caller can say why.
func gitLines(repoRoot string, args ...string) ([]string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines, nil
}
