package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	evalReviewedSHA = "1111111111111111111111111111111111111111"
	evalBaseSHA     = "2222222222222222222222222222222222222222"
)

var evalCandidate = ReviewerEntry{ID: "nan/glm5.3-flash", Runner: "pi", Provider: "nan", Model: "glm5.3-flash", SecretID: "NAN_API_KEY"}

// evalTree is a replay tree whose reviewer pool cannot be parsed: a launch that
// read it would fail, so a launch that succeeds did not.
func evalTree(t *testing.T) string {
	t.Helper()
	tree := t.TempDir()
	for rel, body := range map[string]string{
		ReviewerPoolFile:       "{ this is not a pool",
		"specs/X/proposal.md":  "---\nstatus: implementing\n---\n",
		"specs/X/review.md":    "---\nverdict: PASS\n---\nthe previous round\n",
		"cli/internal/main.go": "package main\n",
	} {
		p := filepath.Join(tree, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return tree
}

// treeDigests maps every file under root to its content digest.
func treeDigests(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		out[p] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func evalLaunch(t *testing.T, tree string) EvalLaunch {
	t.Helper()
	return EvalLaunch{
		Candidate:   evalCandidate,
		SpecID:      "X",
		Tree:        tree,
		ReviewedSHA: evalReviewedSHA,
		BaseSHA:     evalBaseSHA,
		OutDir:      filepath.Join(t.TempDir(), "run", "harness-105-1"),
		SkillPath:   "/skills/adversarial-review/SKILL.md",
	}
}

func TestEvalSlotReadsNoPoolAndWritesNothingUnderSpecs(t *testing.T) {
	tree := evalTree(t)
	before := treeDigests(t, tree)
	l := evalLaunch(t, tree)

	plan, err := PlanEvalLaunch(l)
	if err != nil {
		t.Fatalf("an eval launch read the pool, or refused a valid launch: %v", err)
	}
	if err := WriteEvalRequest(plan, l); err != nil {
		t.Fatal(err)
	}
	if after := treeDigests(t, tree); len(after) != len(before) {
		t.Errorf("the launch changed the replay tree: %d files before, %d after", len(before), len(after))
	} else {
		for p, sum := range before {
			if after[p] != sum {
				t.Errorf("the launch changed %s", p)
			}
		}
	}

	if plan.Reviewer != "eval:nan/glm5.3-flash" {
		t.Errorf("reviewer = %q, want the candidate id under the eval prefix", plan.Reviewer)
	}
	if rel, err := filepath.Rel(tree, plan.Answer); err != nil || strings.HasPrefix(rel, "..") || strings.HasPrefix(filepath.ToSlash(rel), "specs/") {
		t.Errorf("answer %s: want a path inside the replay tree and outside specs/", plan.Answer)
	}
	for name, p := range map[string]string{"result": plan.Result, "request": plan.Request, "transcript": plan.Transcript} {
		if filepath.Dir(p) != l.OutDir {
			t.Errorf("%s %s: want it in the run directory %s", name, p, l.OutDir)
		}
	}

	var req ReviewRequest
	raw, err := os.ReadFile(plan.Request)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	if req.Reviewer != plan.Reviewer || req.ReviewedSHA != evalReviewedSHA || req.BaseSHA != evalBaseSHA {
		t.Errorf("request = %+v, want the eval reviewer and both explicit commits", req)
	}

	for _, want := range []string{plan.Answer, plan.Reviewer, evalBaseSHA, "harness/evals/candidates.json", "Time budget:"} {
		if !strings.Contains(plan.Prompt, want) {
			t.Errorf("the prompt does not name %q", want)
		}
	}
	for _, banned := range []string{"`specs/X/review.md`", "reviewer-pool.json", "reviewer pool", "dotf spec archive", "spec folder"} {
		if strings.Contains(plan.Prompt, banned) {
			t.Errorf("the prompt still says %q, which is the review slot's, not the eval slot's", banned)
		}
	}
	if !strings.Contains(strings.Join(plan.Argv, " "), "glm5.3-flash") {
		t.Errorf("argv does not pin the candidate's model: %q", plan.Argv)
	}
}

func TestEvalSlotRefusesALaunchItCannotKeepApart(t *testing.T) {
	tree := evalTree(t)
	link := filepath.Join(t.TempDir(), "tree-link")
	if err := os.Symlink(tree, link); err != nil {
		link = "" // no symlinks here; the case that needs one is skipped
	}
	for _, tc := range []struct {
		name string
		edit func(*EvalLaunch)
		want string
	}{
		{"no base", func(l *EvalLaunch) { l.BaseSHA = "" }, "base"},
		{"a short base", func(l *EvalLaunch) { l.BaseSHA = "2222222" }, "base"},
		{"no reviewed commit", func(l *EvalLaunch) { l.ReviewedSHA = "" }, "reviewed"},
		{"no candidate id", func(l *EvalLaunch) { l.Candidate.ID = "" }, "candidate"},
		{"a candidate already prefixed", func(l *EvalLaunch) { l.Candidate.ID = "eval:x" }, "candidate"},
		{"a relative run directory", func(l *EvalLaunch) { l.OutDir = "run/x" }, "absolute"},
		{"a run directory in the tree", func(l *EvalLaunch) { l.OutDir = filepath.Join(tree, "out") }, "replay tree"},
		{"a run directory in the tree through a symlink", func(l *EvalLaunch) { l.OutDir = filepath.Join(link, "out") }, "replay tree"},
		{"a run directory under specs/", func(l *EvalLaunch) { l.OutDir = filepath.Join(t.TempDir(), "specs", "X") }, "specs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if link == "" && strings.Contains(tc.name, "symlink") {
				t.Skip("no symlinks here")
			}
			l := evalLaunch(t, tree)
			tc.edit(&l)
			_, err := PlanEvalLaunch(l)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want a refusal naming %q, got %v", tc.want, err)
			}
		})
	}
}
