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

// Lint returns the findings in dir's workflows that no exception covers, and
// the exceptions file's own, sorted by file, line and rule. An unreadable file
// is an error, never an empty result.
func Lint(dir, defaultBranch string) ([]Finding, error) {
	var all []Finding
	for _, wd := range workflowDirs {
		files, err := workflowFiles(filepath.Join(dir, wd))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			rel, _ := filepath.Rel(dir, f)
			found, err := lintWorkflow(filepath.ToSlash(rel), data, defaultBranch)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", filepath.ToSlash(rel), err)
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
