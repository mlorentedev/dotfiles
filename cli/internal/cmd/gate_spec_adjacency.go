package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gate"
)

// reportAdjacency prints the advisory adjacency report, plus a workflow
// annotation under Actions and a table in the --summary file. It runs before
// every verdict, so a pull request that skips SDD or fails archive-on-merge
// still shows its neighbours, and every failure is silence: an advisory check
// that can break the gate is worse than none.
func (g *specGate) reportAdjacency() {
	if g.o.adjacency == "" {
		return
	}
	feed, err := os.ReadFile(g.o.adjacency)
	if err != nil {
		return
	}
	// --no-renames lists a moved file under both names, so an issue that still
	// cites the old path surfaces (HARNESS-178).
	out, err := exec.Command("git", "diff", "--name-only", "--no-renames", g.o.base+"..."+g.o.head).Output()
	if err != nil {
		return
	}
	files := gate.AdjacencyCandidates(strings.Split(string(out), "\n"))
	if len(files) == 0 {
		return
	}
	closed := gate.SortedUnique(gate.ClosingIssueNumbers(g.slug, g.pr.Body))
	rows := gate.AdjacentIssues(string(feed), files, closed)
	if len(rows) == 0 {
		return
	}
	_, _ = fmt.Fprintln(g.out, "[INFO] Adjacent open issues (advisory — this does not gate the PR):")
	for _, r := range rows {
		_, _ = fmt.Fprintf(g.out, "         #%-6s names %s\n                 %s\n", r.Issue, r.Match, gate.Clip(gate.BreakCommands(r.Text), 88))
	}
	_, _ = fmt.Fprint(g.out, "       One of these may describe the same defect on an input shape this\n"+
		"       change does not cover. Fold it in, or record why it is separate,\n"+
		"       before merging.\n")
	if os.Getenv("GITHUB_ACTIONS") != "" {
		var b strings.Builder
		for _, r := range rows {
			fmt.Fprintf(&b, "#%s (%s) ", r.Issue, r.Match)
		}
		_, _ = fmt.Fprintf(g.out, "::warning title=Adjacent open issues::%s\n", strings.ReplaceAll(b.String(), "%", "%25"))
	}
	g.summarizeAdjacency(rows)
}

// titleEscaper backslash-escapes markdown's inline link and image syntax.
var titleEscaper = strings.NewReplacer("[", `\[`, "]", `\]`, "!", `\!`)

func (g *specGate) summarizeAdjacency(rows []gate.Adjacent) {
	if g.o.summary == "" {
		return
	}
	// The summary is rendered as markdown with inline HTML, so a title's angle
	// brackets would reach the page as tags (#3389). Clip before escaping: an
	// entity cut in half would show on the page.
	esc := strings.NewReplacer("|", `\|`, "<", "&lt;", ">", "&gt;").Replace
	// The title cell is markdown text, so link and image syntax would render
	// (#3414); the path cell is a code span and takes no backslashes.
	title := func(t string) string { return titleEscaper.Replace(esc(t)) }
	var b strings.Builder
	b.WriteString("\n### Adjacent open issues (advisory)\n\n| Issue | Names | Title |\n|---|---|---|\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| #%s | `%s` | %s |\n", r.Issue, esc(r.Match), title(gate.Clip(r.Text, 88)))
	}
	_ = appendText(g.o.summary, b.String()) // advisory: a summary it cannot write is silence
}
