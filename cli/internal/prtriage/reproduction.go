package prtriage

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Unreproduced is a REAL finding whose disposition lacks the reproduction it
// needs (TOOL-017 AC4, AC6).
type Unreproduced struct {
	Finding Comment
	Problem string
}

// The rows `dotf pr reproduce` prints. A disposition is checked against
// these exact strings, never against a paraphrase of them.
const (
	reproducedRow = "Reproduction: reproduced"
	refutedRow    = "Reproduction: did-not-reproduce"
)

// dispositions maps the words a triage row opens its disposition cell with
// to the disposition they name.
var dispositions = map[string]string{
	"apply": "apply", "applied": "apply",
	"defer": "defer", "deferred": "defer", "ticketed": "defer",
	"skip": "skip", "skipped": "skip", "declined": "skip",
	"refuted": "refuted",
}

// UnreproducedFindings checks the newest triage record with standing against
// the REAL findings it covers, under the owner's ruling on #1062 (option A,
// and the gate refuses). A REAL finding is applied only with a row showing
// it was reproduced, and refuted only with a row showing it was not; it is
// never skipped, because a failed reproduction is not a skip (#1059). It may
// be deferred. Other classes, and findings newer than the record, are not
// judged here: the second already make the PR pending.
//
// A finding's row is the first table row naming its path and line, the base
// name standing for the path. The disposition is the first cell, after the
// one naming the finding, whose first word is a disposition.
func UnreproducedFindings(pr PR, inline []Comment, reg Registry) []Unreproduced {
	record, ok := triageRecord(pr, reg)
	if !ok {
		return nil
	}
	rows := tableRows(record.Body)
	var out []Unreproduced
	for _, f := range inline {
		if f.Spoken().After(record.CreatedAt) || Classify(f.Body, reg) != "REAL" {
			continue
		}
		if problem := judge(f, rows); problem != "" {
			out = append(out, Unreproduced{Finding: f, Problem: problem})
		}
	}
	return out
}

func judge(f Comment, rows [][]string) string {
	cells, at := rowFor(f, rows)
	if cells == nil {
		return "the triage record has no row for it"
	}
	row := strings.Join(cells, " | ")
	switch dispositionOf(cells[at+1:]) {
	case "apply":
		if !strings.Contains(row, reproducedRow) {
			return fmt.Sprintf("applied without a `%s` row from `dotf pr reproduce`", reproducedRow)
		}
	case "refuted":
		if !strings.Contains(row, refutedRow) {
			return fmt.Sprintf("refuted without a `%s` row from `dotf pr reproduce`", refutedRow)
		}
	case "skip":
		return "a REAL finding is refuted with its evidence or deferred, never skipped"
	case "defer":
	default:
		return "its row names no disposition (apply, defer, refuted)"
	}
	return ""
}

// tableRows splits every markdown table row of body into trimmed cells.
func tableRows(body string) [][]string {
	var rows [][]string
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		var cells []string
		for c := range strings.SplitSeq(strings.Trim(line, "|"), "|") {
			cells = append(cells, strings.TrimSpace(c))
		}
		rows = append(rows, cells)
	}
	return rows
}

// rowFor returns the first row naming f, and the index of the cell that
// does. A name is bounded on both sides, so a.go:4 is not ba.go:4 or a.go:40.
func rowFor(f Comment, rows [][]string) ([]string, int) {
	if f.Path == "" || f.Line == 0 {
		return nil, 0
	}
	name := regexp.MustCompile(`(^|[^\w.\-])` + regexp.QuoteMeta(path.Base(f.Path)) + fmt.Sprintf(`:%d(\D|$)`, f.Line))
	for _, cells := range rows {
		for i, c := range cells {
			if name.MatchString(c) {
				return cells, i
			}
		}
	}
	return nil, 0
}

func dispositionOf(cells []string) string {
	for _, c := range cells {
		word, _, _ := strings.Cut(strings.TrimLeft(c, "*_` "), " ")
		word = strings.ToLower(strings.TrimRight(word, "*_`:.,"))
		if d, ok := dispositions[word]; ok {
			return d
		}
	}
	return ""
}
