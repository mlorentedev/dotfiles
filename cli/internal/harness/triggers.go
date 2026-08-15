package harness

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed triggers.json
var defaultTriggersJSON []byte

// TriggersFile is the repo-relative path to the triggers definition.
const TriggersFile = "harness/triggers.json"

// TriggerRule defines a path-to-pattern mapping.
type TriggerRule struct {
	ID          string   `json:"id"`
	Pattern     string   `json:"pattern"`
	Globs       []string `json:"globs"`
	Description string   `json:"description,omitempty"`
}

// TriggerConfig represents the top-level structure of harness/triggers.json.
type TriggerConfig struct {
	Version  int           `json:"version"`
	Triggers []TriggerRule `json:"triggers"`
}

// LoadTriggers loads triggers from repoRoot/harness/triggers.json or embedded defaults.
func LoadTriggers(repoRoot string) (*TriggerConfig, error) {
	if repoRoot != "" {
		if data, err := os.ReadFile(filepath.Join(repoRoot, TriggersFile)); err == nil {
			return ParseTriggers(data)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read %s: %w", TriggersFile, err)
		}
	}
	return ParseTriggers(defaultTriggersJSON)
}

// ParseTriggers unmarshals raw JSON into a TriggerConfig.
func ParseTriggers(data []byte) (*TriggerConfig, error) {
	var cfg TriggerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse triggers JSON: %w", err)
	}
	return &cfg, nil
}

// MatchPaths evaluates paths against trigger rules and returns sorted unique pattern IDs.
func MatchPaths(triggers []TriggerRule, paths []string) []string {
	matched := make(map[string]struct{})
	for _, p := range paths {
		norm := filepath.ToSlash(strings.TrimSpace(p))
		norm = strings.TrimPrefix(strings.TrimPrefix(norm, "./"), "/")
		if norm == "" || norm == "/dev/null" || norm == "dev/null" {
			continue
		}
		for _, rule := range triggers {
			for _, glob := range rule.Globs {
				if MatchGlob(glob, norm) {
					matched[rule.Pattern] = struct{}{}
					break
				}
			}
		}
	}

	result := make([]string, 0, len(matched))
	for pat := range matched {
		result = append(result, pat)
	}
	sort.Strings(result)
	return result
}

// MatchDiff extracts modified file paths from a unified diff and returns matching pattern IDs.
func MatchDiff(triggers []TriggerRule, diffContent string) []string {
	return MatchPaths(triggers, ExtractPathsFromDiff(diffContent))
}

// ExtractPathsFromDiff parses touched file paths from unified diff headers.
func ExtractPathsFromDiff(diff string) []string {
	seen := make(map[string]struct{})
	var paths []string

	for _, line := range strings.Split(diff, "\n") {
		line = strings.TrimRight(line, "\r")
		var p string
		if strings.HasPrefix(line, "diff --git ") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				p = strings.TrimPrefix(fields[3], "b/")
				if p == "/dev/null" || p == "dev/null" || strings.HasPrefix(p, "b/") {
					p = strings.TrimPrefix(fields[2], "a/")
				}
			}
		} else if strings.HasPrefix(line, "+++ b/") {
			p = strings.TrimPrefix(line, "+++ b/")
		} else if strings.HasPrefix(line, "--- a/") && p == "" {
			p = strings.TrimPrefix(line, "--- a/")
		}

		p = strings.TrimPrefix(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(p)), "./"), "/")
		if p != "" && p != "/dev/null" && p != "dev/null" {
			if _, exists := seen[p]; !exists {
				seen[p] = struct{}{}
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// MatchGlob tests whether a path matches the given glob pattern.
func MatchGlob(glob, path string) bool {
	g := filepath.ToSlash(strings.TrimSpace(glob))
	p := strings.TrimPrefix(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(path)), "./"), "/")
	if g == "" || p == "" {
		return false
	}
	if g == p || g == filepath.Base(p) {
		return true
	}
	re, err := globToRegex(g)
	if err != nil {
		return false
	}
	return re.MatchString(p)
}

func globToRegex(glob string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	if !strings.Contains(glob, "/") {
		b.WriteString("(?:.*/)?")
	}

	for i := 0; i < len(glob); i++ {
		switch c := glob[i]; c {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				i++
				if i+1 < len(glob) && glob[i+1] == '/' {
					i++
					b.WriteString("(?:.+/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			if strings.ContainsRune(`.+()|[]{}^$\\`, rune(c)) {
				b.WriteByte('\\')
			}
			b.WriteByte(c)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
