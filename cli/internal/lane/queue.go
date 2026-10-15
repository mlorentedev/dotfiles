package lane

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ConfigPath is where a repo declares its lane, relative to the repo root.
const ConfigPath = ".dotf/lane.json"

// maxSlug bounds a branch name derived from an issue title.
const maxSlug = 48

// Config is a repo's lane: the ordered queue and how a successor is set up.
// It is read, never written: the queue is not rotated, because "the next item"
// is derived from the issues' own state (see Next), so no commit sits between
// two bugs and two sessions cannot disagree.
type Config struct {
	Queue        []int  `json:"queue"`
	BranchPrefix string `json:"branch_prefix"`
	Base         string `json:"base"`
	Template     string `json:"template"`
}

// LoadConfig reads repoRoot's lane file and fills the defaults. A repo with no
// file has no lane, and a key it does not know is an error, not a silent drop.
func LoadConfig(repoRoot string) (Config, error) {
	b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(ConfigPath)))
	if err != nil {
		return Config{}, fmt.Errorf("lane: %s: %w", ConfigPath, err)
	}
	var cfg Config
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("lane: %s: %w", ConfigPath, err)
	}
	if cfg.BranchPrefix == "" {
		cfg.BranchPrefix = "fix"
	}
	if cfg.Base == "" {
		cfg.Base = "origin/main"
	}
	return cfg, nil
}

// Issue is the part of a forge issue the lane reads.
type Issue struct {
	Number    int
	Title     string
	State     string // OPEN or CLOSED
	Assignees []string
}

// IssueSource reads one issue; production asks `gh`, tests ask a map.
type IssueSource interface {
	Issue(number int) (Issue, error)
}

// Next is the first queued issue that is open and held by nobody. An assignee
// is the claim another session leaves (Standing Order #8), so a held issue is
// skipped, and an issue kept open on purpose after its fix landed stays out of
// the way once somebody owns it. ok is false when nothing is free.
func Next(cfg Config, src IssueSource) (Issue, bool, error) {
	for _, n := range cfg.Queue {
		is, err := src.Issue(n)
		if err != nil {
			return Issue{}, false, fmt.Errorf("lane: issue #%d: %w", n, err)
		}
		if is.State == "OPEN" && len(is.Assignees) == 0 {
			return is, true, nil
		}
	}
	return Issue{}, false, nil
}

// Slug turns an issue title into a branch-name tail: the leading "BUG-143: "
// ticket id is dropped, the rest lower-cased into dash-separated words, cut at
// a word boundary within maxSlug. A title with no word yields "lane".
func Slug(title string) string {
	if id, rest, ok := strings.Cut(title, ": "); ok && !strings.ContainsAny(id, " \t") {
		title = rest
	}
	var words []string
	for _, w := range strings.FieldsFunc(strings.ToLower(strings.NewReplacer("'", "", "’", "").Replace(title)), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		if len(strings.Join(append(words, w), "-")) > maxSlug {
			break
		}
		words = append(words, w)
	}
	if len(words) == 0 {
		return "lane"
	}
	return strings.Join(words, "-")
}

// PromptData is what a repo's prompt template can name.
type PromptData struct {
	Number    int
	Title     string
	Branch    string
	Remaining []int // queued issues after this one
}

// RenderPrompt fills a repo's template. A field the data lacks is an error: a
// successor started on a prompt with a hole in it is worse than no successor.
func RenderPrompt(tmpl string, d PromptData) (string, error) {
	t, err := template.New("lane").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("lane: prompt template: %w", err)
	}
	var out bytes.Buffer
	if err := t.Execute(&out, d); err != nil {
		return "", errors.Join(errors.New("lane: prompt template"), err)
	}
	return out.String(), nil
}
