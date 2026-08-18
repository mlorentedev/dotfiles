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

	yaml "go.yaml.in/yaml/v3"
)

//go:embed triggers.json
var defaultTriggersJSON []byte

// TriggersFile is the repo-relative path to the triggers definition.
const TriggersFile = "harness/triggers.json"

// TriggerRule defines a path-and-prompt-to-pattern mapping.
type TriggerRule struct {
	ID          string   `json:"id"`
	Pattern     string   `json:"pattern"`
	Globs       []string `json:"globs"`
	Skills      []string `json:"skills,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Description string   `json:"description,omitempty"`
}

// TriggerConfig represents the top-level structure of harness/triggers.json.
type TriggerConfig struct {
	Version  int           `json:"version"`
	Triggers []TriggerRule `json:"triggers"`
}

// LoadTriggers loads triggers from explicit repoRoot, local worktree/repo walk-up, or embedded defaults.
func LoadTriggers(repoRoot string) (*TriggerConfig, error) {
	if repoRoot != "" {
		if data, err := os.ReadFile(filepath.Join(repoRoot, TriggersFile)); err == nil {
			return ParseTriggers(data)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read %s: %w", TriggersFile, err)
		}
	}

	// Walk up from current working directory to discover local harness/triggers.json (e.g. in worktrees)
	if cwd, err := os.Getwd(); err == nil {
		for d := cwd; d != "" && d != "/" && d != "."; {
			p := filepath.Join(d, TriggersFile)
			if data, err := os.ReadFile(p); err == nil {
				return ParseTriggers(data)
			}
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
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

// MatchPrompt evaluates prompt text against keywords in trigger rules and returns sorted unique pattern IDs and skill names.
func MatchPrompt(triggers []TriggerRule, prompt string) ([]string, []string) {
	promptLower := strings.ToLower(prompt)
	matchedPats := make(map[string]struct{})
	matchedSkills := make(map[string]struct{})

	for _, rule := range triggers {
		for _, kw := range rule.Keywords {
			kwLower := strings.ToLower(strings.TrimSpace(kw))
			if kwLower != "" && strings.Contains(promptLower, kwLower) {
				if rule.Pattern != "" {
					matchedPats[rule.Pattern] = struct{}{}
				}
				for _, sk := range rule.Skills {
					if sk != "" {
						matchedSkills[sk] = struct{}{}
					}
				}
				break
			}
		}
	}

	pats := make([]string, 0, len(matchedPats))
	for p := range matchedPats {
		pats = append(pats, p)
	}
	sort.Strings(pats)

	skills := make([]string, 0, len(matchedSkills))
	for s := range matchedSkills {
		skills = append(skills, s)
	}
	sort.Strings(skills)

	return pats, skills
}

// Suggestion aggregates matched patterns and skills from prompts and paths.
type Suggestion struct {
	Patterns []string `json:"patterns"`
	Skills   []string `json:"skills"`
}

// DefaultSkillDependencies maps composite skills to their declared prerequisite skills.
var DefaultSkillDependencies = map[string][]string{
	"spec":                    {"adversarial-review", "verification-before-completion"},
	"adversarial-review":      {"verification-before-completion"},
	"executing-plans":         {"systematic-debugging", "test-driven-development", "verification-before-completion"},
	"writing-plans":           {"executing-plans"},
	"architecture-session":    {"read-all-adrs", "spec"},
	"project-maturation":      {"audit", "test", "verification-before-completion"},
	"vault-doctor":            {"insights"},
	"test-driven-development": {"test"},
	"handoff":                 {"verification-before-completion"},
	"pr-review-triage":        {"verification-before-completion"},
}

// ResolveDependencies computes the transitive closure of required skills,
// protecting against cycles and returning a sorted list of unique skills.
func ResolveDependencies(initialSkills []string, depsMap map[string][]string) []string {
	if len(initialSkills) == 0 {
		return nil
	}
	result := make(map[string]struct{})
	queue := make([]string, 0, len(initialSkills))

	for _, s := range initialSkills {
		if s != "" {
			if _, exists := result[s]; !exists {
				result[s] = struct{}{}
				queue = append(queue, s)
			}
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, dep := range depsMap[curr] {
			if dep != "" {
				if _, seen := result[dep]; !seen {
					result[dep] = struct{}{}
					queue = append(queue, dep)
				}
			}
		}
	}

	res := make([]string, 0, len(result))
	for s := range result {
		res = append(res, s)
	}
	sort.Strings(res)
	return res
}

// LoadSkillDependencies walks a skills directory and parses YAML frontmatter to build a dependency map.
func LoadSkillDependencies(skillsDir string) (map[string][]string, error) {
	deps := make(map[string][]string)
	if skillsDir == "" {
		return deps, nil
	}

	err := filepath.WalkDir(skillsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "SKILL.md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
			return nil
		}

		parts := strings.SplitN(content, "---", 3)
		if len(parts) < 3 {
			return nil
		}

		var meta struct {
			Name     string   `yaml:"name"`
			Requires []string `yaml:"requires"`
		}
		if err := yaml.Unmarshal([]byte(parts[1]), &meta); err != nil {
			return nil
		}

		if meta.Name == "" {
			meta.Name = filepath.Base(filepath.Dir(path))
		}
		if len(meta.Requires) > 0 {
			deps[meta.Name] = meta.Requires
		}
		return nil
	})

	return deps, err
}

// Suggest evaluates prompt text and modified paths against trigger rules,
// automatically resolving transitive skill dependencies.
func Suggest(triggers []TriggerRule, prompt string, paths []string) Suggestion {
	return SuggestWithDeps(triggers, prompt, paths, DefaultSkillDependencies)
}

// SuggestWithDeps evaluates prompt text and modified paths against trigger rules
// using a provided skill dependency map for transitive resolution.
func SuggestWithDeps(triggers []TriggerRule, prompt string, paths []string, depsMap map[string][]string) Suggestion {
	patsPrompt, skillsPrompt := MatchPrompt(triggers, prompt)
	patsPaths := MatchPaths(triggers, paths)

	patMap := make(map[string]struct{})
	for _, p := range patsPrompt {
		patMap[p] = struct{}{}
	}
	for _, p := range patsPaths {
		patMap[p] = struct{}{}
	}

	skillMap := make(map[string]struct{})
	for _, s := range skillsPrompt {
		skillMap[s] = struct{}{}
	}

	// Also link skills mapped to triggered patterns
	for _, rule := range triggers {
		if _, ok := patMap[rule.Pattern]; ok {
			for _, sk := range rule.Skills {
				if sk != "" {
					skillMap[sk] = struct{}{}
				}
			}
		}
	}

	pats := make([]string, 0, len(patMap))
	for p := range patMap {
		pats = append(pats, p)
	}
	sort.Strings(pats)

	initialSkills := make([]string, 0, len(skillMap))
	for s := range skillMap {
		initialSkills = append(initialSkills, s)
	}

	resolvedSkills := ResolveDependencies(initialSkills, depsMap)

	return Suggestion{
		Patterns: pats,
		Skills:   resolvedSkills,
	}
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
	if g == p {
		return true
	}

	// 1. Regex match for full path
	re, err := globToRegex(g)
	if err == nil && re.MatchString(p) {
		return true
	}

	// 2. If glob has no slash, test against basename and path segments
	if !strings.Contains(g, "/") {
		base := filepath.Base(p)
		if m, _ := filepath.Match(g, base); m {
			return true
		}
		for _, seg := range strings.Split(p, "/") {
			if m, _ := filepath.Match(g, seg); m {
				return true
			}
		}
	}

	return false
}

func globToRegex(glob string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")

	g := glob
	if strings.HasPrefix(g, "**/") {
		b.WriteString("(?:.*/)?")
		g = strings.TrimPrefix(g, "**/")
	}

	parts := strings.Split(g, "/")
	for i, part := range parts {
		if i > 0 {
			b.WriteString("/")
		}
		if part == "**" {
			b.WriteString(".*")
		} else {
			for j := 0; j < len(part); j++ {
				switch c := part[j]; c {
				case '*':
					b.WriteString("[^/]*")
				case '?':
					b.WriteString("[^/]")
				default:
					if strings.ContainsRune(`.+()|[]{}^$\\`, rune(c)) {
						b.WriteByte('\\')
					}
					b.WriteByte(c)
				}
			}
		}
	}
	b.WriteString("$")

	pattern := b.String()
	if strings.HasSuffix(pattern, "/.*$") {
		prefix := strings.TrimSuffix(pattern, "/.*$")
		pattern = prefix + "(?:/.*)?$"
	}

	return regexp.Compile(pattern)
}
