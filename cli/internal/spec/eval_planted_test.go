package spec

import (
	"path/filepath"
	"strings"
	"testing"
)

// A planted case has no commit and no spec: its prompt is the whole task, and
// it goes to the candidate as written, plus where to put the answer and the
// time budget. The answer lands in the same place a replay's does, so one
// grading path reads both.
func TestPlantedLaunchSendsThePromptAsWritten(t *testing.T) {
	tree := t.TempDir()
	prompt := "# Review this bash script\n\n((count++))\n\n## How to answer\n\nverdict: and a findings table.\n"
	p, err := PlanPlantedLaunch(PlantedLaunch{
		Candidate: evalCandidate, Prompt: prompt, Tree: tree,
		OutDir: filepath.Join(t.TempDir(), "run", "p1-1"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.Prompt, prompt) {
		t.Errorf("the prompt was not sent as written:\n%s", p.Prompt)
	}
	if want := filepath.Join(tree, EvalAnswerDir, EvalSlot.Review); p.Answer != want || !strings.Contains(p.Prompt, want) {
		t.Errorf("the answer goes to %s and the prompt must say so: %s\n%s", want, p.Answer, p.Prompt)
	}
	if !strings.Contains(p.Prompt, "Time budget:") || p.Reviewer != "eval:nan/glm5.3-flash" || len(p.Argv) == 0 {
		t.Errorf("plan = %+v", p)
	}
	for _, banned := range []string{"reviewer-pool", "specs/", "dotf spec archive", "reviewed_sha"} {
		if strings.Contains(p.Prompt[len(prompt):], banned) {
			t.Errorf("a planted prompt names %q, which belongs to a spec review:\n%s", banned, p.Prompt)
		}
	}
}

func TestPlantedLaunchRefusesWhatItCannotKeepApart(t *testing.T) {
	tree := t.TempDir()
	ok := PlantedLaunch{Candidate: evalCandidate, Prompt: "review this", Tree: tree, OutDir: filepath.Join(t.TempDir(), "out")}
	for _, tc := range []struct {
		name string
		mut  func(*PlantedLaunch)
		want string
	}{
		{"an empty prompt", func(l *PlantedLaunch) { l.Prompt = " \n" }, "prompt"},
		{"an eval-prefixed candidate", func(l *PlantedLaunch) { l.Candidate.ID = "eval:x" }, "eval:"},
		{"an out dir inside the tree", func(l *PlantedLaunch) { l.OutDir = filepath.Join(tree, "out") }, "inside"},
		{"a relative tree", func(l *PlantedLaunch) { l.Tree = "rel" }, "absolute"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := ok
			tc.mut(&l)
			if _, err := PlanPlantedLaunch(l); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want a refusal naming %q, got %v", tc.want, err)
			}
		})
	}
}
