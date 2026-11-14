// Package fleetlint checks a repository's CI workflows against the fleet's CI
// standard: every rule has a stable id, and a repository silences one rule on
// one file by declaring an exception with a reason and an issue.
package fleetlint

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Finding is one violation: the rule's id, the file relative to the
// repository with forward slashes, and the line it was found on.
type Finding struct {
	Rule, Path string
	Line       int
	Message    string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", f.Path, f.Line, f.Rule, f.Message)
}

// workflowDirs are read on every forge the fleet runs on.
var workflowDirs = []string{".github/workflows", ".gitea/workflows"}

// actionDirs hold local composite actions, one directory each, on every forge.
var actionDirs = []string{".github/actions", ".gitea/actions"}

// Lint returns the findings in dir's workflows that no exception covers, and
// the exceptions file's own, sorted by file, line and rule. An unreadable file
// is an error, never an empty result.
func Lint(dir, defaultBranch string) ([]Finding, error) {
	return LintRules(dir, defaultBranch, nil)
}

// LintRules is Lint keeping only the findings of the rules in only, every rule
// when only is empty. The branch rule runs only when it, or the exception
// check that judges its exceptions, is selected: no other rule needs the
// default branch, so a selection without them passes where none is known.
func LintRules(dir, defaultBranch string, only []string) ([]Finding, error) {
	all, err := lint(dir, defaultBranch, len(only) == 0 || slices.Contains(only, RuleBranchFilter) || slices.Contains(only, RuleException))
	if err != nil || len(only) == 0 {
		return all, err
	}
	return slices.DeleteFunc(all, func(f Finding) bool { return !slices.Contains(only, f.Rule) }), nil
}

func lint(dir, defaultBranch string, branches bool) ([]Finding, error) {
	var all []Finding
	for _, wd := range workflowDirs {
		files, err := workflowFiles(filepath.Join(dir, wd))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			found, err := lintFile(dir, f, func(rel string, data []byte) ([]Finding, error) {
				return lintWorkflow(rel, data, defaultBranch, branches)
			})
			if err != nil {
				return nil, err
			}
			all = append(all, found...)
		}
	}
	for _, ad := range actionDirs {
		files, err := actionFiles(filepath.Join(dir, ad))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			found, err := lintFile(dir, f, func(rel string, data []byte) ([]Finding, error) { return lintAction(rel, data) })
			if err != nil {
				return nil, err
			}
			all = append(all, found...)
		}
	}
	exc, bad, err := readExceptions(dir)
	if err != nil {
		return nil, err
	}
	out := append(exc.filter(all), bad...)
	slices.SortStableFunc(out, func(a, b Finding) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), strings.Compare(a.Rule, b.Rule))
	})
	return out, nil
}

// lintFile reads f and lints it under its path relative to dir, with forward
// slashes; an error names the file.
func lintFile(dir, f string, lint func(rel string, data []byte) ([]Finding, error)) ([]Finding, error) {
	data, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	rel, _ := filepath.Rel(dir, f)
	rel = filepath.ToSlash(rel)
	found, err := lint(rel, data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return found, nil
}

// actionFiles are the manifests of the actions directly under dir: each
// subdirectory's action.yml or action.yaml, a symlinked one included. Only a
// manifest that does not exist is skipped: any other failure to stat one is
// an error, since skipping it would read as a clean check.
func actionFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		// Stat follows a symlink, as Repos does: e.IsDir() would not.
		fi, err := os.Stat(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			continue
		}
		for _, name := range []string{"action.yml", "action.yaml"} {
			f := filepath.Join(dir, e.Name(), name)
			_, err := os.Stat(f)
			switch {
			case err == nil:
				out = append(out, f)
			case !errors.Is(err, os.ErrNotExist):
				return nil, err
			}
		}
	}
	return out, nil
}

func workflowFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && (strings.HasSuffix(n, ".yml") || strings.HasSuffix(n, ".yaml")) {
			out = append(out, filepath.Join(dir, n))
		}
	}
	return out, nil
}

// decodeDocs decodes every YAML document in data: yaml.Unmarshal would read
// only the first.
func decodeDocs(data []byte) ([]*yaml.Node, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var docs []*yaml.Node
	for {
		var n yaml.Node
		err := dec.Decode(&n)
		if errors.Is(err, io.EOF) {
			return docs, nil
		}
		if err != nil {
			return nil, err
		}
		docs = append(docs, &n)
	}
}

// mapValue is the value under key in a mapping node, or nil.
func mapValue(m *yaml.Node, key string) *yaml.Node {
	_, v := mapEntry(m, key)
	return v
}

// mapEntry is the key node and the value under key in a mapping node, or nils.
func mapEntry(m *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i], m.Content[i+1]
		}
	}
	return nil, nil
}

// Repos are the names of dir's immediate subdirectories that are git
// checkouts, sorted, a symlinked one included. A .git file counts: a worktree
// is a checkout too.
func Repos(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		// Stat follows a symlink: a checkout linked in from elsewhere is
		// still a checkout, and skipping it would read as clean.
		if fi, err := os.Stat(filepath.Join(dir, e.Name())); err != nil || !fi.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), ".git")); err == nil {
			out = append(out, e.Name())
		}
	}
	return out, nil
}
