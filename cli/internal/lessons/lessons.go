// Package lessons keeps a repository's lessons in one format and their index
// generated from them (DOCS-020, #2038). Each lesson file is the source of
// truth for its title and date; the index table is derived, so it can never
// disagree with the files and a conflict on it is resolved by re-running fmt.
package lessons

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Lesson is one lesson file after normalisation.
type Lesson struct {
	File    string // base name, lesson-NNN-<slug>.md
	Number  int
	Title   string
	Created string // YYYY-MM-DD
}

var (
	fileRE = regexp.MustCompile(`^lesson-(\d+)-.*\.md$`)
	h1RE   = regexp.MustCompile(`(?m)^# ([^\n]+?)[ \t]*$`) // [ \t], not \s: \s would eat the blank lines after the H1
	// Only the numbering forms measured in the repository are a prefix: "Lesson
	// NNN" with a separator, or a bare NNN followed by a spaced em or en dash.
	// A hyphen glued to digits ("3-2-1", "12-factor") is part of the title.
	h1PrefixRE   = regexp.MustCompile(`^(?:Lesson\s+\d+\s*[:—–-]|\d+\s+[—–])\s*`)
	inlineDateRE = regexp.MustCompile(`(?m)^[ \t]*(?:>[ \t]*)?\*\*Date:\*\*[ \t]*(\d{4}-\d{2}-\d{2})[ \t]*\n`)
	dateRE       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// leadingKeys is the order every lesson's frontmatter starts with: identity
// first, as the vault's frontmatter rule requires. Other keys keep their order.
var leadingKeys = []string{"id", "type", "status", "title", "created"}

// Normalize returns the lesson's canonical form and what it declares:
// frontmatter with id (the file name), type, status, title and created, an H1
// equal to the title, and the date in one place. The body is left as written.
// seedTitle is used only when the file declares no title yet (the one-time
// migration from a hand-kept index).
func Normalize(name string, raw []byte, seedTitle string) (Lesson, []byte, error) {
	m := fileRE.FindStringSubmatch(name)
	if m == nil {
		return Lesson{}, nil, fmt.Errorf("%s: not a lesson-NNN-<slug>.md file", name)
	}
	num, _ := strconv.Atoi(m[1])
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	fm, body := splitFrontmatter(text)

	l := Lesson{File: name, Number: num}
	l.Created = firstNonEmpty(fm.get("created"), fm.get("date"), inlineDate(body))
	if !dateRE.MatchString(l.Created) {
		return Lesson{}, nil, fmt.Errorf("%s: no date (frontmatter created/date, or an inline **Date:**)", name)
	}
	h1 := h1RE.FindStringSubmatch(body)
	if h1 == nil {
		return Lesson{}, nil, fmt.Errorf("%s: no H1 title", name)
	}
	l.Title = firstNonEmpty(fm.get("title"), seedTitle, h1PrefixRE.ReplaceAllString(h1[1], ""))

	body = removeInlineDate(body, l.Created)
	body = strings.Replace(body, h1[0], "# "+l.Title, 1)
	fm.set("id", strconv.Quote(strings.TrimSuffix(name, ".md")))
	fm.set("type", "lesson")
	if fm.get("status") == "" {
		fm.set("status", "active")
	}
	fm.set("title", strconv.Quote(l.Title))
	fm.set("created", strconv.Quote(l.Created))
	fm.del("date")
	return l, []byte(fm.render() + "\n" + strings.TrimLeft(body, "\n")), nil
}

func inlineDate(body string) string {
	if m := inlineDateRE.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

// removeInlineDate drops the inline **Date:** line that repeats the declared
// date; a line carrying another date is left alone.
func removeInlineDate(body, date string) string {
	return inlineDateRE.ReplaceAllStringFunc(body, func(line string) string {
		if strings.Contains(line, date) {
			return ""
		}
		return line
	})
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// frontmatter is an ordered list of top-level keys, each with its raw lines
// (a key line plus any indented or list continuation lines).
type frontmatter struct {
	keys  []string
	lines map[string][]string
}

func splitFrontmatter(text string) (frontmatter, string) {
	fm := frontmatter{lines: map[string][]string{}}
	if !strings.HasPrefix(text, "---\n") {
		return fm, text
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return fm, text
	}
	current := ""
	for _, line := range strings.Split(text[4:4+end], "\n") {
		key, _, isKey := strings.Cut(line, ":")
		if isKey && key != "" && !strings.ContainsAny(key[:1], " \t-#") {
			current = key
			if _, seen := fm.lines[key]; !seen {
				fm.keys = append(fm.keys, key)
			}
			fm.lines[key] = []string{line}
			continue
		}
		if current != "" {
			fm.lines[current] = append(fm.lines[current], line)
		}
	}
	return fm, text[4+end+5:]
}

// get returns a single-line value without quotes or a trailing comment.
func (f frontmatter) get(key string) string {
	ls, ok := f.lines[key]
	if !ok || len(ls) == 0 {
		return ""
	}
	_, v, _ := strings.Cut(ls[0], ":")
	v = strings.TrimSpace(v)
	if q, err := strconv.Unquote(v); err == nil {
		return q
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return strings.Trim(v, `"'`)
}

func (f *frontmatter) set(key, value string) {
	if _, ok := f.lines[key]; !ok {
		f.keys = append(f.keys, key)
	}
	f.lines[key] = []string{key + ": " + value}
}

func (f *frontmatter) del(key string) {
	delete(f.lines, key)
	f.keys = slicesDelete(f.keys, key)
}

func (f frontmatter) render() string {
	var b strings.Builder
	b.WriteString("---\n")
	done := map[string]bool{}
	for _, k := range append(append([]string{}, leadingKeys...), f.keys...) {
		if done[k] || f.lines[k] == nil {
			continue
		}
		done[k] = true
		for _, l := range f.lines[k] {
			b.WriteString(l + "\n")
		}
	}
	b.WriteString("---\n")
	return b.String()
}

func slicesDelete(s []string, v string) []string {
	out := s[:0]
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// Index markers delimit the generated table inside _index.md; everything
// outside them is the index's own prose and is kept.
const (
	IndexBegin = "<!-- BEGIN GENERATED: dotf lessons fmt — edit the lesson files, not this table -->"
	IndexEnd   = "<!-- END GENERATED -->"
)

var rowRE = regexp.MustCompile(`(?m)^\| \[\d+ - (.*?)\]\((lesson-[^)]+\.md)\) \|`)

// IndexTitles reads the titles a hand-kept index gives each lesson file: the
// seed for files that do not declare a title yet.
func IndexTitles(index []byte) map[string]string {
	out := map[string]string{}
	for _, m := range rowRE.FindAllSubmatch(index, -1) {
		out[string(m[2])] = strings.ReplaceAll(string(m[1]), `\|`, "|")
	}
	return out
}

// RenderIndex returns index with its generated table replaced by one built
// from lessons, sorted by number. An index without markers has its first
// table (the hand-kept one) replaced, and gains the markers.
func RenderIndex(index []byte, lessons []Lesson) []byte {
	sort.Slice(lessons, func(i, j int) bool { return lessons[i].Number < lessons[j].Number })
	var t bytes.Buffer
	t.WriteString(IndexBegin + "\n| Lesson | Date |\n|---|---|\n")
	for _, l := range lessons {
		fmt.Fprintf(&t, "| [%03d - %s](%s) | %s |\n", l.Number, strings.ReplaceAll(l.Title, "|", `\|`), l.File, l.Created)
	}
	t.WriteString(IndexEnd + "\n")
	s := strings.ReplaceAll(string(index), "\r\n", "\n")
	if b := strings.Index(s, IndexBegin); b >= 0 {
		if e := strings.Index(s[b:], IndexEnd); e >= 0 {
			rest := strings.TrimPrefix(s[b+e+len(IndexEnd):], "\n")
			return []byte(s[:b] + t.String() + rest)
		}
	}
	start, end := firstTable(s)
	if start < 0 {
		return []byte(strings.TrimRight(s, "\n") + "\n\n" + t.String())
	}
	return []byte(s[:start] + t.String() + s[end:])
}

// firstTable returns the byte span of the first markdown table, or -1.
func firstTable(s string) (int, int) {
	start, pos := -1, 0
	for _, line := range strings.SplitAfter(s, "\n") {
		isRow := strings.HasPrefix(line, "|")
		switch {
		case isRow && start < 0:
			start = pos
		case !isRow && start >= 0:
			return start, pos
		}
		pos += len(line)
	}
	if start >= 0 {
		return start, len(s)
	}
	return -1, -1
}

// Change is one file fmt would rewrite.
type Change struct {
	Path string
	Want []byte
}

// Plan normalises every lesson under dir and its first-level category
// directories, and regenerates each directory's _index.md. It reads only.
// A number used twice anywhere in the tree is an error.
func Plan(dir string) ([]Change, error) {
	dirs, err := lessonDirs(dir)
	if err != nil {
		return nil, err
	}
	var changes []Change
	seen := map[int]string{}
	for _, d := range dirs {
		cs, err := planDir(d, seen)
		if err != nil {
			return nil, err
		}
		changes = append(changes, cs...)
	}
	// The root index is the one the repository's pointers link, so it must
	// exist whenever a lesson exists anywhere, even only in a category.
	if _, err := os.Stat(filepath.Join(dir, "_index.md")); len(seen) > 0 && err != nil {
		return nil, fmt.Errorf("%s has lessons but no root _index.md", dir)
	}
	dangling, err := danglingLinks(dir, dirs, seen)
	if err != nil {
		return nil, err
	}
	if len(dangling) > 0 {
		return nil, fmt.Errorf("wikilinks that name no lesson; a renumbering moves the file and leaves its links behind, "+
			"so point each at the lesson's new name:\n  %s", strings.Join(dangling, "\n  "))
	}
	return changes, nil
}

// wikilink captures the target of an Obsidian-style link to a lesson, up to an
// alias or heading separator.
var wikilink = regexp.MustCompile(`\[\[(lesson-[^\]|#]+)`)

// danglingLinks names every wikilink, in any lesson or index of the tree, that
// resolves to no lesson file. An index counts because its prose above the
// generated table is hand-written. Two forms are in use and both resolve:
// [[lesson-212]] names a lesson by number, [[lesson-268-full-slug]] names the
// file (#1514). Each is reported with its path relative to root.
func danglingLinks(root string, dirs []string, lessons map[int]string) ([]string, error) {
	stems := make([]string, 0, len(lessons))
	files := make([]string, 0, len(lessons)+len(dirs))
	for _, p := range lessons {
		stems = append(stems, strings.TrimSuffix(filepath.Base(p), ".md"))
		files = append(files, p)
	}
	for _, d := range dirs {
		if idx := filepath.Join(d, "_index.md"); fileExists(idx) {
			files = append(files, idx)
		}
	}
	resolves := func(target string) bool {
		for _, s := range stems {
			if s == target || strings.HasPrefix(s, target+"-") {
				return true
			}
		}
		return false
	}
	var out []string
	for _, p := range files {
		raw, err := os.ReadFile(p) //nolint:gosec // the repository's own docs
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil, err
		}
		for _, m := range wikilink.FindAllSubmatch(raw, -1) {
			if target := strings.TrimSpace(string(m[1])); !resolves(target) {
				out = append(out, fmt.Sprintf("%s links [[%s]]", rel, target))
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func lessonDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	dirs := []string{dir}
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(dir, e.Name()))
		}
	}
	return dirs, nil
}

func planDir(dir string, seen map[int]string) ([]Change, error) {
	files, err := filepath.Glob(filepath.Join(dir, "lesson-*.md"))
	if err != nil {
		return nil, err
	}
	indexPath := filepath.Join(dir, "_index.md")
	index, err := os.ReadFile(indexPath) //nolint:gosec // the repository's own docs
	if len(files) == 0 {
		// No lessons here, but a generated table may still list some that
		// moved into a category: regenerate it empty so stale rows go.
		if err == nil && bytes.Contains(index, []byte(IndexBegin)) {
			if want := RenderIndex(index, nil); !bytes.Equal(index, want) {
				return []Change{{Path: indexPath, Want: want}}, nil
			}
		}
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s has lessons but no _index.md", dir)
	}
	seeds := IndexTitles(index)
	var changes []Change
	var all []Lesson
	for _, f := range files {
		raw, err := os.ReadFile(f) //nolint:gosec // the repository's own docs
		if err != nil {
			return nil, err
		}
		l, want, err := Normalize(filepath.Base(f), raw, seeds[filepath.Base(f)])
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[l.Number]; dup {
			return nil, fmt.Errorf("lesson number %03d is used twice: %s and %s", l.Number, prev, f)
		}
		seen[l.Number] = f
		all = append(all, l)
		if !bytes.Equal(raw, want) {
			changes = append(changes, Change{Path: f, Want: want})
		}
	}
	if want := RenderIndex(index, all); !bytes.Equal(index, want) {
		changes = append(changes, Change{Path: indexPath, Want: want})
	}
	return changes, nil
}

// Apply writes the planned changes.
func Apply(changes []Change) error {
	for _, c := range changes {
		if err := os.WriteFile(c.Path, c.Want, 0o644); err != nil { //nolint:gosec // docs, world-readable by design
			return err
		}
	}
	return nil
}
