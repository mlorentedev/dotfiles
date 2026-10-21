// Package lessons keeps a repository's lessons in one format and renders their
// index on read (DOCS-020, #2038; CLI-131). Each lesson file is the source of
// truth for its title and date. The index is never committed: two PRs that each
// add a lesson would conflict on it, and under a merge queue a conflict ejects
// the PR. It is rendered on demand by Index, so it cannot disagree with the
// files.
package lessons

import (
	"bytes"
	"fmt"
	"io/fs"
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

// FileNumber returns the number a lesson file name declares, and whether name
// is a lesson file (lesson-NNN-<slug>.md) at all.
func FileNumber(name string) (int, bool) {
	m := fileRE.FindStringSubmatch(name)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

// Normalize returns the lesson's canonical form and what it declares:
// frontmatter with id (the file name), type, status, title and created, an H1
// equal to the title, and the date in one place. The body is left as written.
func Normalize(name string, raw []byte) (Lesson, []byte, error) {
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
	l.Title = firstNonEmpty(fm.get("title"), h1PrefixRE.ReplaceAllString(h1[1], ""))

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

// renderTable renders lessons as one markdown table sorted by number, each link
// prefixed with rel (the directory below the root, "" for the root).
func renderTable(lessons []Lesson, rel string) string {
	sort.Slice(lessons, func(i, j int) bool { return lessons[i].Number < lessons[j].Number })
	var t strings.Builder
	t.WriteString("| Lesson | Date |\n|---|---|\n")
	for _, l := range lessons {
		fmt.Fprintf(&t, "| [%03d - %s](%s%s) | %s |\n", l.Number, strings.ReplaceAll(l.Title, "|", `\|`), rel, l.File, l.Created)
	}
	return t.String()
}

// Index renders the index of the lessons under dir: a table for the lessons in
// dir itself, then one section per first-level category directory that holds
// lessons. It reads only, and a lesson that cannot be normalised is an error,
// never a missing row.
func Index(dir string) ([]byte, error) {
	dirs, err := lessonDirs(dir)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString("# Lessons\n")
	seen := map[int]string{}
	for _, d := range dirs {
		ls, _, err := readDir(d, seen)
		if err != nil {
			return nil, err
		}
		if len(ls) == 0 {
			continue
		}
		rel := ""
		if d != dir {
			name := filepath.Base(d)
			out.WriteString("\n## " + name + "\n")
			rel = name + "/"
		}
		out.WriteString("\n" + renderTable(ls, rel))
	}
	return []byte(out.String()), nil
}

// StrayIndexes names every committed _index.md under dir, at any depth. fmt no
// longer reads or writes one: a repository that still commits it holds a table
// nothing regenerates, so --check reports it rather than let it go stale in
// silence. It walks the whole tree, not lessonDirs, because a stray index is
// wrong wherever it sits (#2700); a directory it cannot read is an error, not
// an absence. WalkDir does not follow a symlinked root, so the root is resolved
// first and every path is reported under dir as given.
func StrayIndexes(dir string) ([]string, error) {
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "_index.md" {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out = append(out, filepath.Join(dir, rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Change is one file fmt would rewrite.
type Change struct {
	Path string
	Want []byte
}

// Plan normalises every lesson under dir and its first-level category
// directories. It reads only. A number used twice anywhere in the tree is an
// error, and so is a wikilink that names no lesson.
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
	dangling, err := danglingLinks(dir, seen)
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

// danglingLinks names every wikilink, in any lesson of the tree, that
// resolves to no lesson file. Two forms are in use and both resolve:
// [[lesson-212]] names a lesson by number, [[lesson-268-full-slug]] names the
// file (#1514). Each is reported with its path relative to root.
func danglingLinks(root string, lessons map[int]string) ([]string, error) {
	stems := make([]string, 0, len(lessons))
	files := make([]string, 0, len(lessons))
	for _, p := range lessons {
		stems = append(stems, strings.TrimSuffix(filepath.Base(p), ".md"))
		files = append(files, p)
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

// readDir normalises every lesson file directly in dir. It returns the lessons
// and the changes that would rewrite them, and records each number in seen,
// failing when one is used twice.
func readDir(dir string, seen map[int]string) ([]Lesson, []Change, error) {
	// os.ReadDir, not filepath.Glob: Glob drops an I/O error (an unreadable
	// category directory) and returns no matches, so its lessons would vanish
	// from the index without a word.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var changes []Change
	var all []Lesson
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "lesson-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		f := filepath.Join(dir, name)
		raw, err := os.ReadFile(f) //nolint:gosec // the repository's own docs
		if err != nil {
			return nil, nil, err
		}
		l, want, err := Normalize(filepath.Base(f), raw)
		if err != nil {
			return nil, nil, err
		}
		if prev, dup := seen[l.Number]; dup {
			return nil, nil, fmt.Errorf("lesson number %03d is used twice: %s and %s", l.Number, prev, f)
		}
		seen[l.Number] = f
		all = append(all, l)
		if !bytes.Equal(raw, want) {
			changes = append(changes, Change{Path: f, Want: want})
		}
	}
	return all, changes, nil
}

func planDir(dir string, seen map[int]string) ([]Change, error) {
	_, changes, err := readDir(dir, seen)
	return changes, err
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
