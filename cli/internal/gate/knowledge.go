package gate

import (
	"fmt"
	"regexp"
	"strings"
)

// Every pull request names where its knowledge went (HARNESS-160): a
// "## Knowledge" section whose Lesson, ADR and Runbook lines each name a path
// the pull request adds, changes or renames under the kind's directory, or say
// "none: <reason>". A port of scripts/check-knowledge-gate.sh; its bats suite
// runs both and requires the same verdict and output.

// KnowledgeKinds are the lines the section must answer, in order, with the
// directory each kind's paths sit under.
var KnowledgeKinds = []struct{ Kind, Dir string }{
	{"Lesson", "docs/lessons/"},
	{"ADR", "docs/adr/"},
	{"Runbook", "docs/runbooks/"},
}

// KnowledgeResult is the gate's verdict. Skipped names the dependency bot
// whose pull request was not judged; Summary is one line per answered kind.
type KnowledgeResult struct {
	Skipped  string
	Problems []string
	Summary  []string
}

// KnowledgeSection returns the lines under "## Knowledge" in a body already
// stripped of markup, up to the next heading of level 1 or 2, and whether the
// section exists. The open fence's line, when there is one, explains a
// missing section.
func KnowledgeSection(text string) ([]string, bool) {
	var lines []string
	on, found := false, false
	for _, l := range strings.Split(text, "\n") {
		switch {
		case knowledgeHeadRe.MatchString(strings.ToLower(l)):
			on, found = true, true
		case on && headingRe.MatchString(l):
			return lines, found
		case on:
			lines = append(lines, l)
		}
	}
	return lines, found
}

var (
	knowledgeHeadRe = regexp.MustCompile(`^##[[:space:]]+knowledge[[:space:]]*$`)
	headingRe       = regexp.MustCompile(`^##?[[:space:]]`)
	noneRe          = regexp.MustCompile(`(?i)^none[[:space:]]*(:[[:space:]]*(.*))?$`)
	placeholderRe   = regexp.MustCompile(`^<[^>]*>$`)
	// Leftmost-longest, as the shell gate's grep -oE reads tokens.
	mdTokenRe = regexp.MustCompilePOSIX(`[A-Za-z0-9._/-]+\.md`)
)

// lineRe matches a kind's line, a list bullet optional, in any letter case.
func lineRe(kind string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)^[[:space:]]*([-*+][[:space:]]+)?` + regexp.QuoteMeta(kind) + `[[:space:]]*:[[:space:]]*(.*)$`)
}

// JudgeKnowledge judges a section's lines against the paths the pull request
// adds, modifies or renames.
func JudgeKnowledge(section []string, changed []string) KnowledgeResult {
	var r KnowledgeResult
	isChanged := map[string]bool{}
	for _, c := range changed {
		isChanged[c] = true
	}
	for _, k := range KnowledgeKinds {
		values := kindValues(section, k.Kind)
		switch len(values) {
		case 0:
			r.Problems = append(r.Problems, k.Kind+": the line is missing")
		case 1:
			r.Problems = append(r.Problems, judgeValue(k.Kind, k.Dir, values[0], isChanged)...)
			r.Summary = append(r.Summary, fmt.Sprintf("  %-8s %s", k.Kind, values[0]))
		default:
			r.Problems = append(r.Problems, fmt.Sprintf("%s: the line appears %d times; give one answer", k.Kind, len(values)))
		}
	}
	return r
}

// kindValues is every answer the section gives for a kind, bold markers
// removed and trailing blanks trimmed.
func kindValues(section []string, kind string) []string {
	re := lineRe(kind)
	var values []string
	for _, l := range section {
		if m := re.FindStringSubmatch(strings.ReplaceAll(l, "**", "")); m != nil {
			values = append(values, strings.TrimRight(m[2], " \t\n\v\f\r"))
		}
	}
	return values
}

// judgeValue judges one line's answer for a kind.
func judgeValue(kind, dir, value string, isChanged map[string]bool) []string {
	value = strings.ReplaceAll(value, "`", "")
	if m := noneRe.FindStringSubmatch(value); m != nil {
		reason := m[2]
		switch {
		case strings.ReplaceAll(reason, " ", "") == "":
			return []string{kind + `: "none" needs a reason, as "none: <reason>"`}
		case placeholderRe.MatchString(reason):
			return []string{fmt.Sprintf(`%s: "%s" is the template's placeholder; write the reason`, kind, reason)}
		}
		return nil
	}
	tokens := mdTokenRe.FindAllString(value, -1)
	if len(tokens) == 0 {
		return []string{fmt.Sprintf(`%s: "%s" is neither a path nor "none: <reason>"`, kind, value)}
	}
	var problems []string
	for _, t := range tokens {
		target := strings.TrimPrefix(t, "./")
		switch {
		case !strings.HasPrefix(target, dir):
			problems = append(problems, fmt.Sprintf("%s: %s is not under %s", kind, target, dir))
		case target[strings.LastIndex(target, "/")+1:] == "_index.md":
			problems = append(problems, fmt.Sprintf("%s: %s is an index, not a %s", kind, target, kind))
		case !isChanged[target]:
			problems = append(problems, fmt.Sprintf("%s: %s is not changed by this PR (added, modified or renamed)", kind, target))
		}
	}
	return problems
}

// KnowledgeExplain is the expected shape, printed on failure with --explain.
const KnowledgeExplain = `
The PR body needs this section, one line per kind. Each line names the files
this PR adds or changes for that kind, or says why there are none:

  ## Knowledge

  - Lesson: docs/lessons/lesson-NNN-<slug>.md
  - ADR: none: <reason>
  - Runbook: none: <reason>

Edit the PR body; the gate re-runs on the edit.`
