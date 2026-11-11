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

type exceptions map[exception]bool

// filter drops the findings an exception covers: its rule on its path, and
// nothing else.
func (e exceptions) filter(in []Finding) []Finding {
	var out []Finding
	for _, f := range in {
		if !e[exception{f.Rule, f.Path}] {
			out = append(out, f)
		}
	}
	return out
}

// readExceptions reads the repository's exceptions file. A missing file
// declares none; one that does not parse is an error. An entry missing its
// rule, path, reason or issue is a finding and is not honoured.
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
		exc[exception{e.Rule, e.Path}] = true
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
