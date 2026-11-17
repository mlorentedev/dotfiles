package spec

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// PlantedLaunch is one evaluation run of a planted case: a candidate given a
// self-contained prompt in an empty tree. A planted case has no commit, no
// spec and no base, so nothing here names one.
type PlantedLaunch struct {
	Candidate ReviewerEntry // from harness/evals/candidates.json
	Prompt    string        // the case's prompt.md, sent as written
	Tree      string        // an empty directory the runner works in
	OutDir    string        // the run directory, outside Tree and outside any specs/
	Timeout   time.Duration
}

// PlanPlantedLaunch builds a planted run's prompt and command. The answer goes
// where a replay's does, so one grading path reads both. It reads and writes
// nothing.
func PlanPlantedLaunch(l PlantedLaunch) (EvalPlan, error) {
	id := strings.TrimSpace(l.Candidate.ID)
	if id == "" || IsEvalReviewer(id) {
		return EvalPlan{}, fmt.Errorf("planted launch: candidate id %q: want a candidates.json id, which the launcher prefixes with %q itself", l.Candidate.ID, EvalReviewerPrefix)
	}
	if strings.TrimSpace(l.Prompt) == "" {
		return EvalPlan{}, fmt.Errorf("planted launch for %s: the case prompt is empty", id)
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
	plan.Prompt = strings.TrimRight(l.Prompt, "\n") + "\n\n" +
		"Write your answer, in the format above, to " + plan.Answer + ". Create the folder if it does not exist.\n" +
		"\n" + TimeBudget(time.Now(), timeout)
	argv, err := ReviewerCommand(l.Candidate, plan.Prompt, timeout, l.Tree)
	if err != nil {
		return EvalPlan{}, err
	}
	plan.Argv = argv
	return plan, nil
}
