package evals

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// PreflightDeps are the checks that look outside the repository: whether a
// runner is installed, and whether a candidate's secret resolves. A secret is
// checked by consequence only; its value is never read here.
type PreflightDeps struct {
	LookPath    func(bin string) (string, error)
	CheckSecret func(id string) error
}

// Preflight checks everything a run would otherwise discover one paid attempt
// at a time: each candidate's runner and secret, each replay's commit, and
// that no replay tree holds the answer it is graded against. It names every
// problem at once.
func Preflight(repo string, p Plan, cases []Case, candidates map[string]spec.ReviewerEntry, d PreflightDeps) error {
	var problems []string
	seen := map[string]bool{}
	byID := map[string]Case{}
	for _, c := range cases {
		byID[c.ID] = c
	}
	for _, cell := range p.Cells {
		if !seen["candidate "+cell.Candidate] {
			seen["candidate "+cell.Candidate] = true
			problems = append(problems, candidateProblems(cell.Candidate, candidates, d)...)
		}
		if !seen["case "+cell.Case] {
			seen["case "+cell.Case] = true
			c, ok := byID[cell.Case]
			if !ok {
				problems = append(problems, fmt.Sprintf("%s: the case is not loaded", cell.Case))
				continue
			}
			problems = append(problems, caseProblems(repo, c)...)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return errors.New("eval run preflight:\n  " + strings.Join(problems, "\n  "))
}

func candidateProblems(id string, candidates map[string]spec.ReviewerEntry, d PreflightDeps) []string {
	e, ok := candidates[id]
	if !ok {
		return []string{fmt.Sprintf("%s: not in candidates.json", id)}
	}
	var out []string
	if _, err := d.LookPath(e.Runner); err != nil {
		out = append(out, fmt.Sprintf("%s: runner %s is not on PATH", id, e.Runner))
	}
	if strings.TrimSpace(e.Auth) != "login" && strings.TrimSpace(e.SecretID) != "" {
		// The runner is launched as `dotf secrets run --only <id> --`, the only
		// way a credential reaches it (ADR-028), so dotf has to be found too.
		if _, err := d.LookPath("dotf"); err != nil {
			return append(out, fmt.Sprintf("%s: dotf is not on PATH, and it injects secret %s", id, e.SecretID))
		}
		if err := d.CheckSecret(e.SecretID); err != nil {
			out = append(out, fmt.Sprintf("%s: secret %s does not resolve (%v)", id, e.SecretID, err))
		}
	}
	return out
}

func caseProblems(repo string, c Case) []string {
	if c.Kind == "planted" {
		full, err := insideRoot(repo, c.Prompt)
		if err == nil {
			_, err = os.Stat(full)
		}
		if err != nil {
			return []string{fmt.Sprintf("%s: prompt: %v", c.ID, err)}
		}
		return nil
	}
	if err := resolveCommit(repo, c); err != nil {
		return []string{fmt.Sprintf("%s: %v", c.ID, err)}
	}
	var out []string
	for _, p := range sourceReviewPaths(c.SourceReview) {
		if hasPath(repo, c.ReviewedSHA, p) {
			out = append(out, fmt.Sprintf("%s: the reviewed commit holds the source review at %s, so the candidate could read the answer", c.ID, p))
		}
	}
	if key := ReplayCaseDir + "/" + c.ID + ".json"; hasPath(repo, c.ReviewedSHA, key) {
		out = append(out, fmt.Sprintf("%s: the reviewed commit holds the case's own key %s", c.ID, key))
	}
	return out
}

// sourceReviewPaths is where the source review can sit at its reviewed commit:
// the archive path the case records, and the live spec folder it was written
// to before the spec was archived.
func sourceReviewPaths(archived string) []string {
	paths := []string{archived}
	if live := strings.Replace(archived, "specs/archive/", "specs/", 1); live != archived {
		paths = append(paths, live)
	}
	return paths
}

// hasPath reports whether path exists in the tree of commit sha.
func hasPath(repo, sha, path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := gitIn(repo, "cat-file", "-e", sha+":"+path)
	return err == nil
}
