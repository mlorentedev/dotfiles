package search

import (
	"bufio"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// ItemType defines knowledge classification.
type ItemType string

const (
	TypeAll     ItemType = "all"
	TypePattern ItemType = "pattern"
	TypeSkill   ItemType = "skill"
	TypeLesson  ItemType = "lesson"
	TypeSpec    ItemType = "spec"
	TypeDoc     ItemType = "doc"
)

// Document represents an indexed knowledge artifact.
type Document struct {
	Type        ItemType
	ID          string
	Title       string
	Path        string
	Description string
	Tags        []string
	Keywords    []string
	Content     string
}

// Result represents a scored search hit.
type Result struct {
	Type        ItemType `json:"type"`
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Path        string   `json:"path"`
	Description string   `json:"description,omitempty"`
	Score       float64  `json:"score"`
	Snippet     string   `json:"snippet,omitempty"`
}

// Frontmatter extracts key metadata from YAML header.
type Frontmatter struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Title       string   `yaml:"title"`
	Type        string   `yaml:"type"`
	Description string   `yaml:"description"`
	Summary     string   `yaml:"summary"`
	Tags        []string `yaml:"tags"`
	Keywords    []string `yaml:"keywords"`
	Aliases     []string `yaml:"aliases"`
}

var nonWordRe = regexp.MustCompile(`[^\p{L}\p{N}_\-]+`)

func tokenize(text string) []string {
	clean := nonWordRe.ReplaceAllString(strings.ToLower(text), " ")
	fields := strings.Fields(clean)
	var tokens []string
	for _, f := range fields {
		f = strings.Trim(f, "_-")
		if len(f) > 1 {
			tokens = append(tokens, f)
		}
	}
	return tokens
}

// ParseDocument parses a markdown file into an indexed Document.
func ParseDocument(path string, data []byte) (*Document, error) {
	doc := &Document{
		Path: path,
	}

	content := string(data)
	body := content

	// Parse YAML frontmatter if present
	if strings.HasPrefix(content, "---\n") || strings.HasPrefix(content, "---\r\n") {
		parts := strings.SplitN(content[3:], "\n---", 2)
		if len(parts) == 2 {
			fmYAML := parts[0]
			body = strings.TrimPrefix(parts[1], "\n")
			body = strings.TrimPrefix(body, "\r\n")

			var fm Frontmatter
			if err := yaml.Unmarshal([]byte(fmYAML), &fm); err == nil {
				doc.ID = fm.ID
				if fm.Name != "" {
					doc.Title = fm.Name
				} else if fm.Title != "" {
					doc.Title = fm.Title
				}
				if fm.Description != "" {
					doc.Description = fm.Description
				} else if fm.Summary != "" {
					doc.Description = fm.Summary
				}
				doc.Tags = fm.Tags
				doc.Keywords = append(fm.Keywords, fm.Aliases...)
				if fm.Type != "" {
					switch strings.ToLower(fm.Type) {
					case "pattern":
						doc.Type = TypePattern
					case "skill":
						doc.Type = TypeSkill
					case "lesson":
						doc.Type = TypeLesson
					case "spec":
						doc.Type = TypeSpec
					}
				}
			}
		}
	}

	doc.Content = body

	// Infer type and title from path/headings if not set in frontmatter
	slashPath := filepath.ToSlash(path)
	if doc.Type == "" {
		if strings.Contains(slashPath, "/00_meta/patterns/") || strings.Contains(slashPath, "/patterns/") {
			doc.Type = TypePattern
		} else if strings.Contains(slashPath, "/00_meta/skills/") || strings.Contains(slashPath, "/skills/") {
			doc.Type = TypeSkill
		} else if strings.Contains(slashPath, "/lessons/") {
			doc.Type = TypeLesson
		} else if strings.Contains(slashPath, "/specs/") {
			doc.Type = TypeSpec
		} else {
			doc.Type = TypeDoc
		}
	}

	if doc.ID == "" {
		base := filepath.Base(path)
		doc.ID = strings.TrimSuffix(base, filepath.Ext(base))
	}

	if doc.Title == "" {
		// Read first Markdown heading # Title
		scanner := bufio.NewScanner(strings.NewReader(body))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "# ") {
				doc.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
				break
			}
		}
		if doc.Title == "" {
			doc.Title = doc.ID
		}
	}

	return doc, nil
}

// IndexVault scans a directory and extracts indexed Documents.
func IndexVault(root string) ([]Document, error) {
	var docs []Document

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") && name != "." {
				return filepath.SkipDir
			}
			if name == "node_modules" || name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		doc, err := ParseDocument(path, data)
		if err == nil && doc != nil {
			docs = append(docs, *doc)
		}
		return nil
	})

	return docs, err
}

// ScoreDocument calculates the relevance score of a Document for query terms.
func ScoreDocument(doc *Document, queryTerms []string, rawQuery string) (float64, string) {
	if len(queryTerms) == 0 {
		return 0, ""
	}

	rawQueryLower := strings.ToLower(strings.TrimSpace(rawQuery))
	idLower := strings.ToLower(doc.ID)
	titleLower := strings.ToLower(doc.Title)
	descLower := strings.ToLower(doc.Description)
	contentLower := strings.ToLower(doc.Content)

	var score float64
	termMatches := 0

	// Exact query matches
	if idLower == rawQueryLower {
		score += 30.0
	} else if strings.Contains(idLower, rawQueryLower) {
		score += 15.0
	}
	if strings.Contains(titleLower, rawQueryLower) {
		score += 12.0
	}
	if strings.Contains(descLower, rawQueryLower) {
		score += 6.0
	}

	for _, term := range queryTerms {
		matchedTerm := false
		if idLower == term {
			score += 10.0
			matchedTerm = true
		} else if strings.Contains(idLower, term) {
			score += 5.0
			matchedTerm = true
		}

		if strings.Contains(titleLower, term) {
			score += 6.0
			matchedTerm = true
		}

		for _, tag := range doc.Tags {
			if strings.ToLower(tag) == term {
				score += 4.0
				matchedTerm = true
				break
			}
		}

		for _, kw := range doc.Keywords {
			if strings.Contains(strings.ToLower(kw), term) {
				score += 4.0
				matchedTerm = true
				break
			}
		}

		if strings.Contains(descLower, term) {
			score += 2.5
			matchedTerm = true
		}

		count := strings.Count(contentLower, term)
		if count > 0 {
			matchedTerm = true
			score += math.Min(float64(count)*0.5, 5.0)
		}

		if matchedTerm {
			termMatches++
		}
	}

	// Reward documents matching all query terms
	if termMatches == len(queryTerms) {
		score *= 1.5
	} else if termMatches == 0 {
		return 0, ""
	}

	// Extract snippet around first match
	snippet := extractSnippet(doc.Content, queryTerms)

	return score, snippet
}

func extractSnippet(content string, terms []string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		lineLower := strings.ToLower(trimmed)
		for _, t := range terms {
			if strings.Contains(lineLower, t) {
				if len(trimmed) > 160 {
					return trimmed[:157] + "..."
				}
				return trimmed
			}
		}
	}
	if len(lines) > 0 {
		for _, l := range lines {
			t := strings.TrimSpace(l)
			if t != "" && !strings.HasPrefix(t, "#") {
				if len(t) > 160 {
					return t[:157] + "..."
				}
				return t
			}
		}
	}
	return ""
}

// Search searches documents in a root directory according to the query and filters.
func Search(root string, query string, filterType ItemType, limit int) ([]Result, error) {
	docs, err := IndexVault(root)
	if err != nil {
		return nil, fmt.Errorf("index root: %w", err)
	}

	terms := tokenize(query)
	if len(terms) == 0 {
		return nil, nil
	}

	var results []Result
	for i := range docs {
		d := &docs[i]
		if filterType != "" && filterType != TypeAll && d.Type != filterType {
			continue
		}

		score, snippet := ScoreDocument(d, terms, query)
		if score > 0 {
			relPath := d.Path
			if rel, err := filepath.Rel(root, d.Path); err == nil {
				relPath = rel
			}
			results = append(results, Result{
				Type:        d.Type,
				ID:          d.ID,
				Title:       d.Title,
				Path:        filepath.ToSlash(relPath),
				Description: d.Description,
				Score:       math.Round(score*100) / 100,
				Snippet:     snippet,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ID < results[j].ID
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}
