package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PromotionHeading is the verification.md section whose candidate lines the
// archive checks (HARNESS-160, ADR-039).
const PromotionHeading = "## Promotion candidates"

// VaultPrefix marks a promoted path that lives in the knowledge vault, not in
// the repository: the template's pattern line points at 00_meta/patterns/.
const VaultPrefix = "00_meta/"

var (
	// promotionAnswer reads the text after a candidate's question mark: yes or
	// no as a whole word, then whatever separates it from the rest. The rest is
	// the reason, or the promoted paths.
	promotionAnswer = regexp.MustCompile(`(?i)^(yes|no)\b[\s.:—–-]*(.*)$`)
	promotedPath    = regexp.MustCompile(`[A-Za-z0-9._/-]+\.md`)
	placeholder     = regexp.MustCompile(`^<[^>]*>$`)
	candidateLine   = regexp.MustCompile(`^\s*[-*]\s+\[[ xX]\]\s+(.*)$`)
)

// CheckPromotions reads the promotion candidates in specDir's verification.md
// and returns one problem per line that is not answered "yes: <path>", with
// every named path existing, or "no: <reason>". An empty result means every
// line is answered.
//
// Promotion used to be left to the operator: the archive printed that it "must
// be done separately" and accepted a "no", or no answer, without looking, and
// "yes" answers fell from about 30% to 5% in two months (vault note
// 2026-09-25-knowledge-capture-routing). A path under 00_meta/ resolves
// against vaultRoot; every other path against repoRoot. A nil or failing
// vaultRoot refuses a vault path rather than passing it unchecked.
func CheckPromotions(repoRoot, specDir string, vaultRoot func() (string, error)) []string {
	data, err := os.ReadFile(filepath.Join(specDir, "verification.md"))
	if err != nil {
		return []string{fmt.Sprintf("verification.md cannot be read (%v); it carries the %q the archive checks", err, PromotionHeading)}
	}
	lines, found := promotionSection(string(data))
	if !found {
		return []string{fmt.Sprintf("verification.md has no %q section", PromotionHeading)}
	}
	var problems []string
	candidates := 0
	for _, line := range lines {
		m := candidateLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		candidates++
		question, answer, _ := strings.Cut(m[1], "?")
		label := strings.TrimSpace(question) + "?"
		if problem := judgePromotion(repoRoot, vaultRoot, answer); problem != "" {
			problems = append(problems, label+" "+problem)
		}
	}
	if candidates == 0 {
		problems = append(problems, fmt.Sprintf("the %q section has no candidate lines", PromotionHeading))
	}
	return problems
}

// promotionSection returns the lines under PromotionHeading, up to the next
// level-1 or level-2 heading, and whether the heading was found.
func promotionSection(content string) ([]string, bool) {
	var out []string
	in := false
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r", ""), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, PromotionHeading) {
			in = true
			continue
		}
		if in && (strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ")) {
			break
		}
		if in {
			out = append(out, line)
		}
	}
	return out, in
}

// judgePromotion returns what is wrong with one answer, or "" when it holds.
func judgePromotion(repoRoot string, vaultRoot func() (string, error), answer string) string {
	answer = strings.TrimSpace(strings.NewReplacer("*", "", "`", "").Replace(answer))
	// The pattern line's question carries an extra sentence before the answer.
	answer = strings.TrimSpace(strings.TrimPrefix(answer, "Only if this recurs in >1 project."))
	m := promotionAnswer.FindStringSubmatch(answer)
	if m == nil {
		return `unanswered: write "yes: <path>" or "no: <reason>"`
	}
	rest := strings.TrimSpace(m[2])
	if strings.EqualFold(m[1], "no") {
		if rest == "" || placeholder.MatchString(rest) {
			return `"no" needs a reason: "no: <reason>"`
		}
		return ""
	}
	paths := promotedPath.FindAllString(rest, -1)
	if len(paths) == 0 {
		return `"yes" names no file: "yes: <path of the promoted file>"`
	}
	var missing []string
	for _, p := range paths {
		if problem := promotedFileProblem(repoRoot, vaultRoot, strings.TrimPrefix(p, "./")); problem != "" {
			missing = append(missing, problem)
		}
	}
	return strings.Join(missing, "; ")
}

// promotedFileProblem checks that one promoted path exists where it belongs.
func promotedFileProblem(repoRoot string, vaultRoot func() (string, error), p string) string {
	root := repoRoot
	where := "the repository"
	if strings.HasPrefix(p, VaultPrefix) {
		if vaultRoot == nil {
			return fmt.Sprintf("%s is a vault path, and no vault resolver was given", p)
		}
		v, err := vaultRoot()
		if err != nil || v == "" {
			return fmt.Sprintf("%s is a vault path, and the vault cannot be resolved (%v)", p, err)
		}
		root, where = v, "the vault"
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
		return fmt.Sprintf("%s does not exist in %s", p, where)
	}
	return ""
}
