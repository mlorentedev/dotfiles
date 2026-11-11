package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gate"
)

// specDiff is the production diff, counted the way the Discipline Gate does.
type specDiff struct {
	total, specLOC     int
	touched            bool
	included, excluded []string
	mandated           []string
}

// mandatedArchives is the specs this pull request archived because it closes
// their issues (#800): such a move is the spec touch outright, since an
// archive barely registers as lines. An unreadable log credits none.
func (g *specGate) mandatedArchives() []string {
	issues, err := g.closedIssues()
	if err != nil || len(issues) == 0 {
		return nil
	}
	archived, err := specIssues(g.o.head, "archive/")
	if err != nil {
		return nil
	}
	var ids []string
	for _, n := range issues {
		for _, s := range archived {
			if s.Issue == n {
				ids = append(ids, s.ID)
			}
		}
	}
	return ids
}

func (g *specGate) countDiff() (specDiff, error) {
	d := specDiff{mandated: g.mandatedArchives()}
	out, err := exec.Command("git", "diff", "--numstat", g.o.base+"..."+g.o.head).Output()
	var stats []gate.FileStat
	if err == nil {
		stats, err = gate.ParseNumstat(string(out))
	}
	if err != nil {
		return d, g.fail(2, fmt.Sprintf("[ERROR] git diff \"%s...%s\" failed — base/head ref could not be resolved.\n"+
			"        The Discipline Gate fails closed (exit 2): it will not pass a PR whose diff cannot be computed.\n"+
			"        In CI, ensure the base ref is fetched (fetch-depth: 0 or an explicit git fetch origin <base>).\n", g.o.base, g.o.head))
	}
	for _, f := range stats {
		loc := f.Added + f.Removed
		if gate.IsActiveSpecPath(f.Path) {
			d.specLOC += loc
		} else if isMandatedArchive(f.Path, d.mandated) {
			d.touched = true
		}
		entry := fmt.Sprintf("%s:%d", f.Path, loc)
		if gate.ExcludedFromLOC(f.Path) {
			d.excluded = append(d.excluded, entry)
			continue
		}
		d.total += loc
		d.included = append(d.included, entry)
	}
	if d.specLOC >= gate.SpecFloor {
		d.touched = true
	}
	return d, nil
}

func isMandatedArchive(path string, ids []string) bool {
	rest, ok := strings.CutPrefix(path, "specs/archive/")
	if !ok || !strings.Contains(rest, "/") {
		return false
	}
	id, _, _ := strings.Cut(rest, "/")
	for _, m := range ids {
		if m == id {
			return true
		}
	}
	return false
}

func (g *specGate) judgeDiff() error {
	d, err := g.countDiff()
	if err != nil {
		return err
	}
	if g.o.explain {
		g.explainDiff(d)
	}
	if d.total < g.o.threshold {
		_, _ = fmt.Fprintf(g.out, "[OK] Production diff %d LOC < threshold %d (below threshold; spec not required)\n", d.total, g.o.threshold)
		return nil
	}
	if d.touched {
		_, _ = fmt.Fprintf(g.out, "[OK] Production diff %d LOC >= threshold %d AND spec folder touched in diff\n", d.total, g.o.threshold)
		return nil
	}
	return g.fail(1, fmt.Sprintf(`[FAIL] SDD Discipline Gate violation:
       Production diff: %d LOC (>= threshold %d)
       No specs/<feature-id>/ folder touched in this PR.

       Options:
         (a) Create a spec folder: dotf spec init <feature-id>
         (b) Add the "skip-sdd" label to the PR AND a non-empty
             "## SDD skip rationale" section in the PR body.

       If this already archived a spec to satisfy archive-on-merge but still
       failed here: the PR body or a commit message must close the spec's
       issue ("Closes #N"), and the spec's proposal.md must name that issue.

       Reference: AGENTS.md "Discipline Gate (NON-NEGOTIABLE)" section.
`, d.total, g.o.threshold))
}

func (g *specGate) explainDiff(d specDiff) {
	w := g.out
	_, _ = fmt.Fprintf(w, "[INFO] Spec-gate breakdown (%s...%s)\n", g.o.base, g.o.head)
	_, _ = fmt.Fprintf(w, "  Threshold: %d LOC\n", g.o.threshold)
	_, _ = fmt.Fprintf(w, "  Production LOC (added+removed): %d\n", d.total)
	_, _ = fmt.Fprintf(w, "  Active-spec LOC (added+removed): %d (floor %d to count as a spec touch)\n", d.specLOC, gate.SpecFloor)
	listLines(w, "  Specs archived for archive-on-merge (count as a spec touch):", d.mandated)
	touched := "no"
	if d.touched {
		touched = "yes"
	}
	_, _ = fmt.Fprintf(w, "  Spec folder touched: %s\n", touched)
	listLines(w, "  Files counted:", d.included)
	listLines(w, "  Files excluded:", d.excluded)
}

func listLines(w io.Writer, title string, items []string) {
	if len(items) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, title)
	for _, it := range items {
		_, _ = fmt.Fprintf(w, "    %s\n", it)
	}
}
