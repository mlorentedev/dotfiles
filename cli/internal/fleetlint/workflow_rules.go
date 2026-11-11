package fleetlint

import (
	"errors"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	RuleCancelInProgress = "fleet/cancel-in-progress"
	RuleOSOnPullRequest  = "fleet/os-on-pull-request"
	RuleSkipCache        = "fleet/skip-cache"
	RuleBranchFilter     = "fleet/branch-filter"
)

// ErrNoDefaultBranch is a branch filter met with no default branch to check
// it against: the rule cannot answer, and an unanswered rule is not a pass.
var ErrNoDefaultBranch = errors.New("a workflow filters on branches and the default branch is unknown")

var matrixRef = regexp.MustCompile(`^\$\{\{\s*matrix\.([A-Za-z0-9_-]+)\s*\}\}$`)

// DefaultBranch is the branch origin/HEAD names in dir's checkout, or "".
func DefaultBranch(dir string) string {
	out, err := exec.Command("git", "-C", dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/")
}

// triggersPR reports whether on: names pull_request or pull_request_target,
// in any of its three shapes: a scalar, a sequence or a mapping.
func triggersPR(on *yaml.Node) bool {
	if on == nil {
		return false
	}
	isPR := func(s string) bool { return s == "pull_request" || s == "pull_request_target" }
	switch on.Kind {
	case yaml.ScalarNode:
		return isPR(on.Value)
	case yaml.SequenceNode:
		for _, n := range on.Content {
			if isPR(n.Value) {
				return true
			}
		}
	case yaml.MappingNode:
		for i := 0; i < len(on.Content); i += 2 {
			if isPR(on.Content[i].Value) {
				return true
			}
		}
	}
	return false
}

// rerunsOnPush reports whether a push to a pull request starts the workflow
// again: a pull-request trigger with no types: (the default set holds
// synchronize) or whose types: name synchronize. Only then can a run go stale.
func rerunsOnPush(on *yaml.Node) bool {
	if on == nil || on.Kind != yaml.MappingNode {
		return triggersPR(on)
	}
	for _, event := range []string{"pull_request", "pull_request_target"} {
		k, v := mapEntry(on, event)
		if k == nil {
			continue
		}
		types := scalars(mapValue(v, "types"))
		if types == nil || slices.Contains(types, "synchronize") {
			return true
		}
	}
	return false
}

// cancelFinding: a pull-request workflow cancels the run a new push makes
// stale. Only the workflow-level concurrency key is read; an expression
// counts as set.
func cancelFinding(path string, root, onKey *yaml.Node) []Finding {
	cKey, c := mapEntry(root, "concurrency")
	if cancel := mapValue(c, "cancel-in-progress"); cancel != nil && cancel.Value != "false" {
		return nil
	}
	line := onKey.Line
	if cKey != nil {
		line = cKey.Line
	}
	return []Finding{{Rule: RuleCancelInProgress, Path: path, Line: line,
		Message: "a pull-request workflow sets no concurrency with cancel-in-progress"}}
}

// osFinding: on a pull request, a Windows or macOS job runs only under an
// if:, so change detection decides. A matrix computed by an expression is
// conditional by construction; a literal one is read, include: entries too.
func osFinding(path string, key, job *yaml.Node) []Finding {
	if mapValue(job, "if") != nil {
		return nil
	}
	ro := mapValue(job, "runs-on")
	if ro != nil && ro.Kind == yaml.MappingNode {
		ro = mapValue(ro, "labels")
	}
	matrix := mapValue(mapValue(job, "strategy"), "matrix")
	for _, label := range scalars(ro) {
		values := []string{label}
		if m := matrixRef.FindStringSubmatch(label); m != nil {
			values = scalars(mapValue(matrix, m[1]))
			for _, inc := range seq(mapValue(matrix, "include")) {
				values = append(values, scalars(mapValue(inc, m[1]))...)
			}
		}
		for _, v := range values {
			if l := strings.ToLower(v); strings.Contains(l, "windows") || strings.Contains(l, "macos") {
				return []Finding{{Rule: RuleOSOnPullRequest, Path: path, Line: key.Line,
					Message: "job " + key.Value + " runs " + v + " on every pull request; gate it with if:"}}
			}
		}
	}
	return nil
}

// cacheFinding: a setup action keeps its cache and a linter does not skip
// its own. Only an explicit switch-off is read, not a cache left unset.
func cacheFinding(path string, uses, step *yaml.Node) []Finding {
	with := mapValue(step, "with")
	var off *yaml.Node
	if c := mapValue(with, "cache"); strings.Contains(uses.Value, "/setup-") && c != nil && c.Value == "false" {
		off = c
	}
	if s := mapValue(with, "skip-cache"); strings.HasPrefix(uses.Value, "golangci/golangci-lint-action@") && s != nil && s.Value == "true" {
		off = s
	}
	if off == nil {
		return nil
	}
	return []Finding{{Rule: RuleSkipCache, Path: path, Line: off.Line, Message: uses.Value + " runs with its cache off"}}
}

// branchFindings: a push or pull_request branch filter includes the default
// branch, or the workflow never runs on it. branches-ignore is not read.
func branchFindings(path string, on *yaml.Node, defaultBranch string) ([]Finding, error) {
	var out []Finding
	for _, event := range []string{"push", "pull_request"} {
		b := mapValue(mapValue(on, event), "branches")
		if b == nil {
			continue
		}
		if defaultBranch == "" {
			return nil, ErrNoDefaultBranch
		}
		if !matchesBranch(scalars(b), defaultBranch) {
			out = append(out, Finding{Rule: RuleBranchFilter, Path: path, Line: b.Line,
				Message: event + " filters on branches that do not include " + defaultBranch})
		}
	}
	return out, nil
}

// matchesBranch reads the filter as GitHub does: in order, the last pattern
// that matches decides, and a ! pattern excludes.
func matchesBranch(patterns []string, branch string) bool {
	in := false
	for _, p := range patterns {
		neg := strings.HasPrefix(p, "!")
		if globRegexp(strings.TrimPrefix(p, "!")).MatchString(branch) {
			in = !neg
		}
	}
	return in
}

// globRegexp is a filter pattern as a regexp: ** crosses a slash, * and ?
// do not, and everything else is literal.
func globRegexp(p string) *regexp.Regexp {
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString("[^/]*")
		case p[i] == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	return regexp.MustCompile("^" + b.String() + "$")
}

// scalars is a scalar's value, or a sequence's scalar values.
func scalars(n *yaml.Node) []string {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.ScalarNode {
		return []string{n.Value}
	}
	var out []string
	for _, c := range seq(n) {
		if c.Kind == yaml.ScalarNode {
			out = append(out, c.Value)
		}
	}
	return out
}

func seq(n *yaml.Node) []*yaml.Node {
	if n == nil || n.Kind != yaml.SequenceNode {
		return nil
	}
	return n.Content
}
