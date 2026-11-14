package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EvalReviewerPrefix marks a reviewer id recorded by an evaluation run. No pool
// id carries it, and the archive gate refuses any signature that does.
const EvalReviewerPrefix = "eval:"

// IsEvalReviewer reports whether a recorded reviewer is an evaluation run's.
func IsEvalReviewer(reviewer string) bool {
	return strings.HasPrefix(strings.TrimSpace(reviewer), EvalReviewerPrefix)
}

// refuseEvalRecord keeps an evaluation answer from signing a spec. It reads
// both the answer's own claim and the launcher's request, because either one
// copied into a spec folder is enough to show where the record came from. It
// runs before the pool check, which has no opinion in a repository without a
// pool.
func refuseEvalRecord(specDir string, slot ReviewSlot, review Review) error {
	recorded := ""
	if IsEvalReviewer(review.Reviewer) {
		recorded = review.Reviewer
	} else if req, found, err := ReadReviewRequestIn(specDir, slot); err == nil && found && IsEvalReviewer(req.Reviewer) {
		recorded = req.Reviewer
	}
	if recorded == "" {
		return nil
	}
	return fmt.Errorf("%s records reviewer %q: it is an evaluation answer, graded against a key, not a review of this spec\n"+
		"an evaluation run never signs a spec; launch `dotf spec review %s%s` for a review that does",
		slot.Review, strings.TrimSpace(recorded), filepath.Base(specDir), slot.Flag())
}

// EvalAnswerDir is the folder, inside a replay tree, that an evaluation run
// writes its answer to. It sits outside specs/, so the answer never lands
// where the archive gate reads a signature, and inside the tree, so a runner
// whose writes are scoped to the tree can reach it.
const EvalAnswerDir = ".dotf-eval"

// EvalSlot names the files of an evaluation run, all kept in the run directory
// the caller supplies.
var EvalSlot = ReviewSlot{Name: "eval", Review: ReviewFile, Request: "eval-request.json", Transcript: "transcript.jsonl"}

// EvalLaunch is one evaluation run of a review: a candidate, given directly
// rather than drawn from the reviewer pool, reviewing a replay tree from an
// explicit base. Nothing here names a pool or a way to resolve a base, so the
// slot cannot consult either.
type EvalLaunch struct {
	Candidate   ReviewerEntry // from harness/evals/candidates.json
	SpecID      string
	Tree        string // the replay worktree, checked out at ReviewedSHA
	ReviewedSHA string
	BaseSHA     string
	OutDir      string // the run directory, outside Tree and outside any specs/
	SkillPath   string
	Timeout     time.Duration
}

// EvalPlan is what an evaluation launch runs and where it leaves its files.
type EvalPlan struct {
	Reviewer   string // the candidate id under EvalReviewerPrefix
	Prompt     string
	Argv       []string // the runner's command, not yet under a deadline
	Answer     string   // where the reviewer writes, inside the tree
	Result     string   // where the answer is kept, in the run directory
	Request    string
	Transcript string
}

// PlanEvalLaunch builds an evaluation run's prompt and command. It reads and
// writes nothing: WriteEvalRequest records the launch, in the run directory.
func PlanEvalLaunch(l EvalLaunch) (EvalPlan, error) {
	id := strings.TrimSpace(l.Candidate.ID)
	if id == "" || IsEvalReviewer(id) {
		return EvalPlan{}, fmt.Errorf("eval launch: candidate id %q: want a candidates.json id, which the launcher prefixes with %q itself", l.Candidate.ID, EvalReviewerPrefix)
	}
	if !isFullSHA(l.ReviewedSHA) {
		return EvalPlan{}, fmt.Errorf("eval launch for %s: reviewed commit %q is not a full sha", id, l.ReviewedSHA)
	}
	if !isFullSHA(l.BaseSHA) {
		return EvalPlan{}, fmt.Errorf("eval launch for %s: base %q is not a full sha; an eval run takes its base from the case, never resolves one", id, l.BaseSHA)
	}
	if err := checkEvalOutDir(l.Tree, l.OutDir); err != nil {
		return EvalPlan{}, err
	}
	timeout := l.Timeout
	if timeout <= 0 {
		timeout = DefaultReviewerTimeout
	}
	plan := EvalPlan{
		Reviewer:   EvalReviewerPrefix + id,
		Answer:     filepath.Join(l.Tree, EvalAnswerDir, EvalSlot.Review),
		Result:     filepath.Join(l.OutDir, EvalSlot.Review),
		Request:    filepath.Join(l.OutDir, EvalSlot.Request),
		Transcript: filepath.Join(l.OutDir, EvalSlot.Transcript),
	}
	plan.Prompt = reviewPrompt(EvalSlot, l.SpecID, l.Tree, plan.Reviewer, l.Candidate.Runner, l.SkillPath,
		ReviewScope{BaseSHA: l.BaseSHA}, promptTarget{verdict: plan.Answer, eval: true}) +
		"\n" + FreshWorktreeNote(l.ReviewedSHA) +
		"\n" + TimeBudget(time.Now(), timeout)
	argv, err := ReviewerCommand(l.Candidate, plan.Prompt, timeout, l.Tree)
	if err != nil {
		return EvalPlan{}, err
	}
	plan.Argv = argv
	return plan, nil
}

// WriteEvalRequest records the launch in the run directory, in the review
// request's shape so a result carries the same provenance a review does.
func WriteEvalRequest(p EvalPlan, l EvalLaunch) error {
	if err := os.MkdirAll(filepath.Dir(p.Request), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(ReviewRequest{
		ReviewedSHA: l.ReviewedSHA,
		Reviewer:    p.Reviewer,
		RequestedAt: time.Now().UTC().Format(time.RFC3339),
		BaseSHA:     l.BaseSHA,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.Request, append(raw, '\n'), 0o600)
}

// checkEvalOutDir refuses a run directory that is relative, inside the replay
// tree, or under a specs/ folder. Both sides are compared after resolving
// symlinks, since a temporary directory is often reached through one.
func checkEvalOutDir(tree, out string) error {
	if !filepath.IsAbs(out) || !filepath.IsAbs(tree) {
		return fmt.Errorf("eval launch: the run directory %q and the replay tree %q must both be absolute", out, tree)
	}
	realTree, realOut := ResolveExisting(tree), ResolveExisting(out)
	if rel, err := filepath.Rel(realTree, realOut); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("eval launch: the run directory %s is inside the replay tree %s, which is removed after the run", out, tree)
	}
	for _, part := range strings.Split(filepath.ToSlash(realOut), "/") {
		if part == "specs" {
			return fmt.Errorf("eval launch: the run directory %s is under a specs/ folder, where an answer would read as a review", out)
		}
	}
	return nil
}

// ResolveExisting resolves symlinks in the longest existing prefix of p and
// keeps the rest as given, so a path not created yet still compares by where
// it will be.
func ResolveExisting(p string) string {
	p = filepath.Clean(p)
	var rest []string
	for {
		if real, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(append([]string{real}, rest...)...)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return filepath.Join(append([]string{p}, rest...)...)
		}
		rest = append([]string{filepath.Base(p)}, rest...)
		p = parent
	}
}
