package vault

// backlog.go — the Go port of scripts/check-backlog-integrity.sh (SDD-012) and
// scripts/check-backlog-merged.sh (SDD-012b), the two checks `vault health`
// runs in its Backlog Integrity section (#492). Section 7 used to exec them
// through bash, so `dotf vault health` needed bash and a dotfiles checkout.
//
// Each function returns what its script prints on stdout, byte for byte, and
// the verdict its exit status carried. The scripts stay beside this port until
// vault-health.sh retires: it still execs them, and the golden corpus is
// replayed against both. backlog_test.go runs the two implementations on the
// same fixtures and compares them while the shell exists.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// backlogEntryRe is the scripts' entry line: `- [<status>] **<full id>**`. The
// full id is the bold token, so BUG-020-pwsh-profile and BUG-020-splitpath are
// two tickets, and WIN-002 and WIN-002a are too.
var backlogEntryRe = regexp.MustCompile(`^- \[([ xX~-])\] \*\*([^*]+)\*\*`)

// ticketNumberRe is the number prefix of a full id, which only drives the
// advisory NOTE about two tickets sharing a number.
var ticketNumberRe = regexp.MustCompile(`^[A-Z]+-[0-9]+`)

// backlogLines splits like awk's record reader: a final line without a newline
// is still a line, and a trailing newline does not add an empty one.
func backlogLines(b []byte) []string {
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// BacklogIntegrity checks one tasks file for structural drift: the same full id
// on two or more entry lines (DUPLICATE), or marked both open and done
// (CONTRADICTION). A number reused by different ids is an advisory NOTE. The
// report is check-backlog-integrity.sh's stdout for that file ("" when it has
// nothing to say), and drift is its exit 1. NOTE lines alone are not drift.
func BacklogIntegrity(path string) (report string, drift bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}

	count := map[string]int{}
	done := map[string]bool{}
	open := map[string]bool{}
	var order []string
	numIDs := map[string][]string{}
	var numOrder []string

	for _, line := range backlogLines(b) {
		m := backlogEntryRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		status, id := m[1], m[2]
		if count[id] == 0 {
			order = append(order, id)
		}
		count[id]++
		if status == "x" || status == "X" {
			done[id] = true
		} else {
			open[id] = true
		}
		num := ticketNumberRe.FindString(id)
		if num == "" {
			continue
		}
		if _, seen := numIDs[num]; !seen {
			numOrder = append(numOrder, num)
		}
		if !slices.Contains(numIDs[num], id) {
			numIDs[num] = append(numIDs[num], id)
		}
	}

	var out []string
	for _, id := range order {
		if count[id] < 2 {
			continue
		}
		if done[id] && open[id] {
			out = append(out, fmt.Sprintf("  CONTRADICTION: %s — %d entries, marked BOTH open and done", id, count[id]))
		} else {
			out = append(out, fmt.Sprintf("  DUPLICATE: %s — %d entries", id, count[id]))
		}
	}
	for _, num := range numOrder {
		if ids := numIDs[num]; len(ids) > 1 {
			out = append(out, fmt.Sprintf("  NOTE: number %s reused by %d different tickets (%s) — advisory, not drift",
				num, len(ids), strings.Join(ids, ", ")))
		}
	}
	if len(out) == 0 {
		return "", false, nil
	}
	report = path + ":\n" + strings.Join(out, "\n") + "\n"
	// The script's own test is `grep -qE 'DUPLICATE|CONTRADICTION'` over its
	// output, so an id spelling either word in a NOTE line counts too.
	drift = strings.Contains(report, "DUPLICATE") || strings.Contains(report, "CONTRADICTION")
	return report, drift, nil
}

// BacklogMerged flags open entries (`[ ]`, `[~]`, `[-]`) whose full id has an
// archived spec, `<repo>/specs/archive/<id>/`: the work shipped and the tick is
// stale. It is advisory. The report is check-backlog-merged.sh's stdout and
// stale is its exit 1. An empty repo is inferred the way the script does,
// <vault>/10_projects/<proj>/11-tasks.md -> $HOME/Projects/<proj>, and a repo
// with no specs/archive has nothing to cross-reference.
func BacklogMerged(tasks, repo string) (report string, stale bool, err error) {
	b, err := os.ReadFile(tasks)
	if err != nil {
		return "", false, err
	}
	if repo == "" {
		repo = filepath.Join(backlogHome(), "Projects", filepath.Base(filepath.Dir(tasks)))
	}
	if !isDir(filepath.Join(repo, "specs", "archive")) {
		return "", false, nil
	}

	var sb strings.Builder
	for _, line := range backlogLines(b) {
		m := backlogEntryRe.FindStringSubmatch(line)
		if m == nil || m[1] == "x" || m[1] == "X" {
			continue
		}
		id := m[2]
		if isDir(filepath.Join(repo, "specs", "archive", id)) {
			fmt.Fprintf(&sb, "  STALE-OPEN: %s — archived spec exists (specs/archive/%s/); verify + tick [x]\n", id, id)
			stale = true
		}
	}
	return sb.String(), stale, nil
}

// backlogHome is the script's $HOME: the variable when set, else the OS home
// directory (USERPROFILE on Windows, where HOME is usually unset).
func backlogHome() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}
