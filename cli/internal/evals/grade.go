package evals

import (
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// Finding is one entry of a case's key: an expected, optional, decoy or noise
// pattern, matched against a findings-table row by its anchors.
type Finding struct {
	ID            string   `json:"id"`
	SeverityFloor string   `json:"severity_floor,omitempty"`
	AnchorsAll    []string `json:"anchors_all"`
	AnchorsAny    []string `json:"anchors_any,omitempty"`
	ConfirmedBy   string   `json:"confirmed_by,omitempty"`
	Note          string   `json:"note,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	Sample        string   `json:"sample,omitempty"`
}

// Case is one evaluation case (harness/evals/case.schema.json).
type Case struct {
	ID              string    `json:"id"`
	Repo            string    `json:"repo"`
	Kind            string    `json:"kind"`
	Role            string    `json:"role"`
	Spec            string    `json:"spec,omitempty"`
	SourceReview    string    `json:"source_review,omitempty"`
	Prompt          string    `json:"prompt,omitempty"`
	ReferenceAnswer string    `json:"reference_answer,omitempty"`
	KeyAuthor       string    `json:"key_author"`
	ReviewedSHA     string    `json:"reviewed_sha,omitempty"`
	BaseSHA         string    `json:"base_sha,omitempty"`
	BaseDerivation  string    `json:"base_derivation,omitempty"`
	FetchRef        string    `json:"fetch_ref,omitempty"`
	ExpectedVerdict string    `json:"expected_verdict"`
	Expected        []Finding `json:"expected"`
	Optional        []Finding `json:"optional"`
	Decoys          []Finding `json:"decoys"`
	Noise           []Finding `json:"noise"`
}

// Format is the state of an answer's findings table.
type Format string

const (
	FormatOK         Format = "ok"
	FormatNoTable    Format = "no-table"
	FormatEmptyTable Format = "empty-table"
)

// Row is one findings-table row.
type Row struct {
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Text     string `json:"text"`
}

// Match pairs a key entry with the row that satisfied it.
type Match struct {
	ID  string `json:"id"`
	Row Row    `json:"row"`
}

// Grade is the deterministic grading of one answer against one case.
type Grade struct {
	Format          Format   `json:"format"`
	Verdict         string   `json:"verdict"`
	VerdictMatch    bool     `json:"verdict_match"`
	Matched         []Match  `json:"matched"`
	Missed          []string `json:"missed"`
	OptionalMatched []Match  `json:"optional_matched"`
	DecoyHits       []Match  `json:"decoy_hits"`
	Noise           []Match  `json:"noise"`
	Unadjudicated   []Row    `json:"unadjudicated"`
	Recall          float64  `json:"recall"`
	WeightedRecall  float64  `json:"weighted_recall"`
}

// ParseFindings extracts the findings table from a review's markdown: the first
// table whose header's first cell is "Severity". Every review the launcher has
// archived uses that header; the evaluator rubric that follows it does not.
//
// A row's text is its raw line, never its cells: a pipe inside a code span
// splits a cell where the author meant none, and anchors match the whole row.
// Only the severity is read from a cell, the first one, which never holds code.
func ParseFindings(markdown string) (Format, []Row) {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if !isTableLine(line) || !strings.EqualFold(firstCell(line), "Severity") {
			continue
		}
		rows := []Row{}
		for j := i + 1; j < len(lines); j++ {
			l := strings.TrimRight(lines[j], "\r")
			if !isTableLine(l) {
				break
			}
			if isSeparator(l) {
				continue
			}
			rows = append(rows, Row{Line: j + 1, Severity: firstCell(l), Text: strings.TrimSpace(l)})
		}
		if len(rows) == 0 {
			return FormatEmptyTable, rows
		}
		return FormatOK, rows
	}
	return FormatNoTable, nil
}

func isTableLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "|")
}

func isSeparator(line string) bool {
	return strings.Trim(strings.TrimSpace(line), "|-: ") == ""
}

func firstCell(line string) string {
	cells := strings.SplitN(strings.TrimSpace(line), "|", 3)
	if len(cells) < 2 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(cells[1]), "*_` ")
}

// severityRank orders the labels reviews use. A label outside the set ranks
// lowest, so it never satisfies a floor.
var severityRank = map[string]int{"blocker": 3, "major": 2, "minor": 1}

// severityWeight weights recall: a missed Blocker costs more than a missed Minor.
func severityWeight(floor string) float64 {
	if w := severityRank[strings.ToLower(floor)]; w > 0 {
		return float64(w)
	}
	return 1
}

// matches reports whether a row satisfies a key entry: every anchors_all token
// and, when anchors_any is set, at least one of its tokens, case-insensitively,
// at or above the entry's severity floor.
func matches(f Finding, r Row) bool {
	if f.SeverityFloor != "" && severityRank[strings.ToLower(r.Severity)] < severityRank[strings.ToLower(f.SeverityFloor)] {
		return false
	}
	text := strings.ToLower(r.Text)
	for _, a := range f.AnchorsAll {
		if !strings.Contains(text, strings.ToLower(a)) {
			return false
		}
	}
	if len(f.AnchorsAny) == 0 {
		return true
	}
	for _, a := range f.AnchorsAny {
		if strings.Contains(text, strings.ToLower(a)) {
			return true
		}
	}
	return false
}

// assign finds a maximum one-to-one assignment of rows to key entries (Kuhn's
// augmenting paths). A greedy first fit can give a shared row to the wrong entry
// and strand another that only that row could satisfy. Rows already taken are
// skipped. It returns, per entry, the index of its row or -1.
func assign(entries []Finding, rows []Row, taken []bool) []int {
	rowOwner := make([]int, len(rows))
	for i := range rowOwner {
		rowOwner[i] = -1
	}
	var try func(e int, seen []bool) bool
	try = func(e int, seen []bool) bool {
		for r := range rows {
			if taken[r] || seen[r] || !matches(entries[e], rows[r]) {
				continue
			}
			seen[r] = true
			if rowOwner[r] == -1 || try(rowOwner[r], seen) {
				rowOwner[r] = e
				return true
			}
		}
		return false
	}
	for e := range entries {
		try(e, make([]bool, len(rows)))
	}
	byEntry := make([]int, len(entries))
	for i := range byEntry {
		byEntry[i] = -1
	}
	for r, e := range rowOwner {
		if e >= 0 {
			byEntry[e] = r
			taken[r] = true
		}
	}
	return byEntry
}

// GradeAnswer grades a reviewer's answer against the case's key.
//
// Each row lands in exactly one class, in this order:
//  1. noise: a trace of the replay environment, discarded;
//  2. decoy: a claim known to be false, a false positive even when the row
//     also names a real symbol (a row that asserts the false claim is wrong);
//  3. expected, then optional: assigned one to one;
//  4. unadjudicated: neither a miss nor a false positive (ADR-063 §3).
func GradeAnswer(c Case, answer string) Grade {
	g := Grade{
		Matched: []Match{}, Missed: []string{}, OptionalMatched: []Match{},
		DecoyHits: []Match{}, Noise: []Match{}, Unadjudicated: []Row{},
	}
	if r, err := spec.ParseReview(answer); err == nil {
		g.Verdict = string(r.Verdict)
	}
	g.VerdictMatch = g.Verdict != "" && g.Verdict == c.ExpectedVerdict

	var rows []Row
	g.Format, rows = ParseFindings(answer)
	taken := make([]bool, len(rows))

	g.Noise = claimEach(c.Noise, rows, taken)
	g.DecoyHits = claimEach(c.Decoys, rows, taken)

	var total, weighted, totalWeight float64
	for i, r := range assign(c.Expected, rows, taken) {
		w := severityWeight(c.Expected[i].SeverityFloor)
		totalWeight += w
		if r < 0 {
			g.Missed = append(g.Missed, c.Expected[i].ID)
			continue
		}
		total++
		weighted += w
		g.Matched = append(g.Matched, Match{ID: c.Expected[i].ID, Row: rows[r]})
	}
	for i, r := range assign(c.Optional, rows, taken) {
		if r >= 0 {
			g.OptionalMatched = append(g.OptionalMatched, Match{ID: c.Optional[i].ID, Row: rows[r]})
		}
	}
	for i, r := range rows {
		if !taken[i] {
			g.Unadjudicated = append(g.Unadjudicated, r)
		}
	}
	if n := len(c.Expected); n > 0 {
		g.Recall = total / float64(n)
		g.WeightedRecall = weighted / totalWeight
	}
	return g
}

// claimEach takes every untaken row that matches any entry, recording the first
// entry it matched. Noise and decoys are not one to one: one false claim
// repeated in two rows is two false positives.
func claimEach(entries []Finding, rows []Row, taken []bool) []Match {
	out := []Match{}
	for r := range rows {
		if taken[r] {
			continue
		}
		for _, f := range entries {
			if matches(f, rows[r]) {
				taken[r] = true
				out = append(out, Match{ID: f.ID, Row: rows[r]})
				break
			}
		}
	}
	return out
}
