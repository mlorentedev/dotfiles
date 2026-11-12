package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gate"
	"io"
	"os"
)

// The gate's verdicts, in the order scripts/check-spec-gate.sh reaches them
// and in its words, so a caller switching from the script reads the same
// lines on the same streams.

// specIssues is every spec under specs/<root> at ref whose proposal names an
// issue, in tree order.
func specIssues(ref, root string) ([]gate.SpecIssue, error) {
	out, err := exec.Command("git", "ls-tree", "-r", "--name-only", ref, "--", "specs/"+root).Output()
	if err != nil {
		return nil, err
	}
	var specs []gate.SpecIssue
	for _, path := range strings.Split(string(out), "\n") {
		id := gate.SpecProposalID(path, root)
		if id == "" {
			continue
		}
		content, err := exec.Command("git", "show", ref+":"+path).Output()
		if err != nil {
			continue
		}
		if n := gate.FrontmatterIssue(string(content)); n != "" {
			specs = append(specs, gate.SpecIssue{Issue: n, ID: id})
		}
	}
	return specs, nil
}

// activeSpecs is the active specs at base and at head: base sees the spec a
// pull request archives, head the one it creates and closes at once.
func (g *specGate) activeSpecs() ([]gate.SpecIssue, error) {
	base, err := specIssues(g.o.base, "")
	if err != nil {
		return nil, g.fail(2, fmt.Sprintf("[ERROR] Could not read specs/ at base ref \"%s\".\n"+
			"        The gate fails closed: it will not pass a PR whose spec state cannot be read.\n", g.o.base))
	}
	head, err := specIssues(g.o.head, "")
	if err != nil {
		return nil, g.fail(2, fmt.Sprintf("[ERROR] Could not read specs/ at head ref \"%s\".\n", g.o.head))
	}
	var lines []string
	for _, s := range append(base, head...) {
		lines = append(lines, s.Issue+"\t"+s.ID)
	}
	var specs []gate.SpecIssue
	for _, l := range gate.SortedUnique(lines) {
		issue, id, _ := strings.Cut(l, "\t")
		specs = append(specs, gate.SpecIssue{Issue: issue, ID: id})
	}
	return specs, nil
}

// archiveOnMerge fails a pull request that closes an issue whose spec is
// still active at head (SDD-038).
func (g *specGate) archiveOnMerge() error {
	issues, err := g.closedIssues()
	if err != nil {
		return g.fail(2, fmt.Sprintf("[ERROR] git log \"%s\" failed: the commit messages cannot be read.\n"+
			"        The gate fails closed: a keyword in an unread commit still closes the issue.\n", g.o.base+".."+g.o.head))
	}
	if len(issues) == 0 {
		return nil
	}
	specs, err := g.activeSpecs()
	if err != nil {
		return err
	}
	var violations []string
	for _, n := range issues {
		for _, s := range specs {
			if s.Issue == n && !g.archivedAtHead(s.ID) {
				violations = append(violations, fmt.Sprintf("%s (#%s)", s.ID, n))
			}
		}
	}
	if len(violations) == 0 {
		return nil
	}
	if g.pr.HasLabel("skip-archive") {
		if gate.SkipRationale(g.pr.Body, "Archive skip rationale") {
			_, _ = fmt.Fprintln(g.out, `[OK] archive-on-merge skipped: "skip-archive" label + non-empty "## Archive skip rationale" in PR body`)
			return nil
		}
		return g.fail(1, "[FAIL] \"skip-archive\" label present but the \"## Archive skip rationale\" section is empty or missing in the PR body.\n")
	}
	// A draft is still under review, and the push may carry the fix a review
	// round needs CI to prove (#3364). The invariant is about the merge, so
	// at push time it is a warning. Only the pre-push adapter sets the
	// variable; the event's own draft field is deliberately not read, so CI
	// keeps the refusal until the spec is archived in the PR.
	if os.Getenv("SDD_PR_DRAFT") == "true" {
		_, _ = io.WriteString(g.err, archiveDraftWarning(violations))
		return nil
	}
	return g.fail(1, archiveViolation(violations))
}

func archiveDraftWarning(violations []string) string {
	var b strings.Builder
	b.WriteString("[WARN] SDD archive-on-merge: this draft PR closes an issue whose spec is still active:\n")
	for _, v := range violations {
		fmt.Fprintf(&b, "         %s\n", v)
	}
	b.WriteString("       The push goes through; spec-gate in CI refuses the merge until the spec is\n       archived in this PR. Before marking it ready, run:\n")
	for _, v := range violations {
		fmt.Fprintf(&b, "         dotf spec archive %s\n", strings.SplitN(v, " ", 2)[0])
	}
	return b.String()
}

// archivedAtHead asks the head tree, not the diff: a move can surface as a
// rename or as a delete and an add, and presence reads the same either way.
func (g *specGate) archivedAtHead(id string) bool {
	out, err := exec.Command("git", "ls-tree", "-r", "--name-only", g.o.head, "--",
		"specs/archive/"+id+"/", "specs/archive/_abandoned/"+id+"/").Output()
	return err == nil && len(out) > 0
}

func archiveViolation(violations []string) string {
	var b strings.Builder
	b.WriteString("[FAIL] SDD archive-on-merge violation:\n       This PR closes an issue whose spec is still active:\n")
	for _, v := range violations {
		fmt.Fprintf(&b, "         %s\n", v)
	}
	b.WriteString("\n       The SDD lifecycle ends by archiving the spec in the SAME PR\n       (Discipline Gate step 7). Run:\n\n")
	for _, v := range violations {
		fmt.Fprintf(&b, "         dotf spec archive %s\n", strings.SplitN(v, " ", 2)[0])
	}
	b.WriteString(`
       If the work genuinely continues elsewhere, reference the issue
       without a closing keyword (e.g. "Refs #N") in the PR body AND in
       every commit message (they reach the squash commit), or add the
       "skip-archive" label AND a non-empty "## Archive skip rationale"
       section to the PR body.
`)
	return b.String()
}

// escapes is the dependency-bot and skip-sdd exits; done means the verdict
// is given.
func (g *specGate) escapes() (bool, error) {
	if g.pr.HasLabel("dependencies") {
		if g.pr.IsDependencyBotPR() {
			_, _ = fmt.Fprintf(g.out, "[OK] spec-gate skipped: \"dependencies\" label on a bot-authored PR (%s)\n", g.pr.Author)
			return true, nil
		}
		author := g.pr.Author
		if author == "" {
			author = "<unset>"
		}
		_, _ = fmt.Fprintf(g.err, "[INFO] \"dependencies\" label present but author \"%s\" is not a recognised dependency bot — not honouring the skip; evaluating the gate normally.\n", author)
	}
	if !g.pr.HasLabel("skip-sdd") {
		return false, nil
	}
	if gate.SkipRationale(g.pr.Body, "SDD skip rationale") {
		_, _ = fmt.Fprintln(g.out, `[OK] spec-gate skipped: "skip-sdd" label + non-empty "## SDD skip rationale" in PR body`)
		return true, nil
	}
	return true, g.fail(1, `[FAIL] "skip-sdd" label present but the "## SDD skip rationale" section is empty or missing in the PR body.
       Add a "## SDD skip rationale" section to the PR body with a real reason
       (e.g. "mechanical rename, no logic change"), or open a spec folder instead.
`)
}
