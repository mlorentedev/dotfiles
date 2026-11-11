package fleetlint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ExceptionsPath is where a repository declares its exceptions, relative to
// its root.
const ExceptionsPath = ".github/lint-exceptions.yml"

// RuleException is the id of a finding on the exceptions file itself: an
// exception that does not say why, or which issue tracks it, silences nothing.
const RuleException = "fleet/exception"

type exception struct{ rule, path string }

// exceptions maps each honoured exception to its line in the file.
type exceptions map[exception]int

// rules are the ids an exception may name: one naming any other silences
// nothing, so a typo would otherwise pass as a declared exception.
var rules = map[string]bool{
	RuleTimeout: true, RuleSHAPin: true, RulePersistCredentials: true, RuleNoSecretsInherit: true,
	RuleCancelInProgress: true, RuleOSOnPullRequest: true, RuleSkipCache: true, RuleBranchFilter: true,
}

// filter drops the findings an exception covers: its rule on its path, and
// nothing else. An exception that covers no finding is itself one: the
// workflow was fixed or moved, and the exception would silence its return.
func (e exceptions) filter(in []Finding) []Finding {
	var out []Finding
	used := map[exception]bool{}
	for _, f := range in {
		k := exception{f.Rule, f.Path}
		if _, ok := e[k]; ok {
			used[k] = true
			continue
		}
		out = append(out, f)
	}
	for k, line := range e {
		if !used[k] {
			out = append(out, Finding{Rule: RuleException, Path: ExceptionsPath, Line: line,
				Message: "exception for " + k.rule + " on " + k.path + " matches no finding; remove it"})
		}
	}
	return out
}

// readExceptions reads the repository's exceptions file. A missing file
// declares none; one that does not parse is an error. An entry missing its
// rule, path, reason or issue, or naming a rule that does not exist, is a
// finding and is not honoured.
func readExceptions(dir string) (exceptions, []Finding, error) {
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(ExceptionsPath)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var doc struct {
		Exceptions []yaml.Node `yaml:"exceptions"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", ExceptionsPath, err)
	}
	exc := exceptions{}
	var bad []Finding
	for i := range doc.Exceptions {
		n := &doc.Exceptions[i]
		var e struct{ Rule, Path, Reason, Issue string }
		if err := n.Decode(&e); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", ExceptionsPath, err)
		}
		if missing := missingFields(e.Rule, e.Path, e.Reason, e.Issue); missing != "" {
			bad = append(bad, Finding{Rule: RuleException, Path: ExceptionsPath, Line: n.Line,
				Message: "exception without " + missing + " is not honoured"})
			continue
		}
		if !rules[e.Rule] {
			bad = append(bad, Finding{Rule: RuleException, Path: ExceptionsPath, Line: n.Line,
				Message: "exception names " + e.Rule + ", which is not a fleet rule"})
			continue
		}
		exc[exception{e.Rule, e.Path}] = n.Line
	}
	return exc, bad, nil
}

func missingFields(rule, path, reason, issue string) string {
	var m []string
	for _, f := range []struct{ name, v string }{{"rule", rule}, {"path", path}, {"reason", reason}, {"issue", issue}} {
		if strings.TrimSpace(f.v) == "" {
			m = append(m, f.name)
		}
	}
	return strings.Join(m, ", ")
}
