package prreview

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// fileFilter is the files PR-Agent drops before the model sees them: its
// `[ignore]` patterns (pr_agent/algo/file_filter.py) and the file types
// is_valid_file refuses (pr_agent/algo/language_handler.py). The type lists are
// read at the audited commit, so nothing here is a copy that can drift.
//
// It must never drop a file PR-Agent would keep: that would skip a review with a
// real diff. So a difference may only narrow it: a pattern RE2 cannot compile is
// dropped, as upstream drops one Python cannot, and a list that cannot be read is
// empty.
type fileFilter struct {
	patterns []*regexp.Regexp
	bad      map[string]bool // extensions, lower case
	locks    map[string]bool // exact file names
	suffixes []string
}

// warnFunc reports what narrowed the filter, so that a list which stopped being
// readable is seen. The lockfile list read empty for releases and nothing said so.
type warnFunc func(format string, a ...any)

// newFileFilter reads the three sources. An empty source narrows the filter; a
// source that does not parse is an error, and the caller keeps the review.
func newFileFilter(cfg, extensions, handler []byte, warn warnFunc) (*fileFilter, error) {
	patterns, err := ignorePatterns(cfg, warn)
	if err != nil {
		return nil, fmt.Errorf(".pr_agent.toml: %w", err)
	}
	var ext struct {
		BadExtensions struct {
			Default []string `toml:"default"`
		} `toml:"bad_extensions"`
	}
	if err := toml.Unmarshal(extensions, &ext); err != nil {
		return nil, fmt.Errorf("language_extensions.toml: %w", err)
	}
	f := &fileFilter{patterns: patterns, bad: map[string]bool{}, locks: map[string]bool{}}
	for _, e := range ext.BadExtensions.Default {
		f.bad[strings.ToLower(e)] = true
	}
	for _, name := range pythonStringList(handler, "AUTO_GENERATED_FILES_EXACT") {
		f.locks[name] = true
	}
	f.suffixes = pythonStringList(handler, "auto_generated_suffixes")
	if len(f.bad) == 0 || len(f.locks) == 0 || len(f.suffixes) == 0 {
		warn("PR-Agent's file type lists read incomplete (%d extensions, %d lockfiles, %d suffixes), "+
			"so fewer PRs are recognised as empty. Check them at the audited commit.",
			len(f.bad), len(f.locks), len(f.suffixes))
	}
	return f, nil
}

// kept counts the files PR-Agent would send to the model.
func (f *fileFilter) kept(files []string) int {
	n := 0
	for _, file := range files {
		if !f.typedOut(file) && !f.ignored(file) {
			n++
		}
	}
	return n
}

func (f *fileFilter) ignored(file string) bool {
	return slices.ContainsFunc(f.patterns, func(p *regexp.Regexp) bool { return p.MatchString(file) })
}

// typedOut is is_valid_file at the audited commit, inverted. The extension is
// what follows the last dot of the whole path, as upstream splits it.
func (f *fileFilter) typedOut(file string) bool {
	slashed := strings.ReplaceAll(file, `\`, "/")
	if f.locks[slashed[strings.LastIndexByte(slashed, '/')+1:]] {
		return true
	}
	if slices.ContainsFunc(f.suffixes, func(s string) bool { return strings.HasSuffix(file, s) }) {
		return true
	}
	return f.bad[strings.ToLower(file[strings.LastIndexByte(file, '.')+1:])]
}

// ignorePatterns compiles `[ignore]`. Entries of `regex` are Python's, matched
// from the start of the path; globs expand as globSources says. A config with
// neither, or no config at all, has none.
func ignorePatterns(cfg []byte, warn warnFunc) ([]*regexp.Regexp, error) {
	var parsed struct {
		Ignore struct {
			Glob  []string `toml:"glob"`
			Regex []string `toml:"regex"`
		} `toml:"ignore"`
	}
	if err := toml.Unmarshal(cfg, &parsed); err != nil {
		return nil, err
	}
	var sources []string
	for _, r := range parsed.Ignore.Regex {
		sources = append(sources, `\A(?:`+r+`)`)
	}
	sources = append(sources, globSources(parsed.Ignore.Glob)...)
	var patterns []*regexp.Regexp
	for _, source := range sources {
		p, err := regexp.Compile(source)
		if err != nil {
			warn("skipping an ignore pattern this cannot compile, so a PR of those files runs the review: %v", err)
			continue
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}

// Upstream's bounds on the zero-directory forms. They are part of the behaviour:
// past them PR-Agent keeps a file, so dropping it here would skip a real diff.
const (
	maxGlobstars    = 6
	maxGlobVariants = 256
	maxExpandedGlob = 256
	globstarSegment = "**/"
)

// globSources is translate_globs_to_regexes: every glob, the glob without a
// leading `**/`, and then each form that drops a subset of its whole `**/`
// segments, since `a/**/b` also means `a/b`. Translations are compared, not
// globs, as upstream counts its quota.
func globSources(globs []string) []string {
	var sources []string
	seen := map[string]bool{}
	add := func(glob string) bool {
		source := translateGlob(glob)
		if glob == "" || seen[source] {
			return false
		}
		seen[source] = true
		sources = append(sources, source)
		return true
	}
	for _, glob := range globs {
		add(glob)
		if rest, found := strings.CutPrefix(glob, globstarSegment); found {
			add(rest)
		}
	}
	variants := 0
	for _, glob := range globs {
		for _, variant := range globVariants(glob) {
			if variants == maxGlobVariants {
				return sources
			}
			if add(variant) {
				variants++
			}
		}
	}
	return sources
}

// globVariants is glob without each non-empty subset of its `**/` segments, in
// upstream's order, or nothing when glob is past the bounds.
func globVariants(glob string) []string {
	runes := []rune(glob)
	offsets := globstarOffsets(runes)
	if len(offsets) == 0 || len(offsets) > maxGlobstars || len(runes) > maxExpandedGlob {
		return nil
	}
	segments := strings.Split(glob, "/")
	var variants []string
	for mask := 1; mask < 1<<len(offsets); mask++ {
		dropped := map[int]bool{}
		for bit, offset := range offsets {
			if mask>>bit&1 == 1 {
				dropped[strings.Count(string(runes[:offset]), "/")] = true
			}
		}
		var kept []string
		for i, segment := range segments {
			if !dropped[i] {
				kept = append(kept, segment)
			}
		}
		variants = append(variants, strings.Join(kept, "/"))
	}
	return variants
}

// globstarOffsets is _globstar_offsets: where whole `**/` segments start, outside
// bracket sets.
func globstarOffsets(runes []rune) []int {
	var offsets []int
	for i := 0; i < len(runes); {
		if runes[i] == '[' {
			if end := closingBracket(runes, i+1); end >= 0 {
				i = end + 1
				continue
			}
		}
		if isGlobstarSegment(runes, i) {
			offsets = append(offsets, i)
		}
		i++
	}
	return offsets
}

func isGlobstarSegment(runes []rune, i int) bool {
	return strings.HasPrefix(string(runes[i:]), globstarSegment) && (i == 0 || runes[i-1] == '/')
}

// translateGlob is Python's fnmatch.translate, for RE2. `*` crosses `/`: fnmatch
// is not path-aware, and neither is PR-Agent's filter.
func translateGlob(glob string) string {
	var b strings.Builder
	b.WriteString(`\A(?s:`)
	runes := []rune(glob)
	for i := 0; i < len(runes); i++ {
		switch c := runes[i]; c {
		case '*':
			if i == 0 || runes[i-1] != '*' { // fnmatch compresses a run of stars
				b.WriteString(".*")
			}
		case '?':
			b.WriteString(".")
		case '[':
			end := closingBracket(runes, i+1)
			if end < 0 {
				b.WriteString(`\[`)
				continue
			}
			b.WriteString(bracketClass(runes[i+1 : end]))
			i = end
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString(`)\z`)
	return b.String()
}

// closingBracket is the index of the `]` closing the set that starts at i, or -1
// when none does and the `[` is literal. A `]` first in the set is a member.
func closingBracket(runes []rune, i int) int {
	if i < len(runes) && runes[i] == '!' {
		i++
	}
	if i < len(runes) && runes[i] == ']' {
		i++
	}
	for ; i < len(runes); i++ {
		if runes[i] == ']' {
			return i
		}
	}
	return -1
}

// bracketClass is a fnmatch set as an RE2 class: `!` negates, `-` keeps meaning a
// range, and everything else is literal, RE2's `[:alpha:]` included.
func bracketClass(set []rune) string {
	var b strings.Builder
	b.WriteByte('[')
	if len(set) > 0 && set[0] == '!' {
		b.WriteByte('^')
		set = set[1:]
	}
	for _, c := range set {
		if strings.ContainsRune(`\[]^`, c) {
			b.WriteByte('\\')
		}
		b.WriteRune(c)
	}
	b.WriteByte(']')
	return b.String()
}

// pythonStringList reads the literal strings assigned to name in Python source,
// from a set, tuple or list, optionally wrapped in frozenset(). Upstream's source
// is not executed and no Python parser is pulled in for two literals: anything
// else this does not expect reads as no list, which narrows the filter.
func pythonStringList(src []byte, name string) []string {
	assignment := regexp.MustCompile(`(?mi)^[ \t]*` + name + `[ \t]*=[ \t]*(?:frozenset\()?`)
	loc := assignment.FindIndex(src)
	if loc == nil {
		return nil
	}
	return scanStrings(string(src[loc[1]:]))
}

// scanStrings reads one bracketed collection of plain string literals.
func scanStrings(s string) []string {
	closers := map[byte]byte{'{': '}', '(': ')', '[': ']'}
	if s == "" || closers[s[0]] == 0 {
		return nil
	}
	closer := closers[s[0]]
	var items []string
	for i := 1; i < len(s); {
		switch c := s[i]; {
		case c == closer:
			return items
		case strings.IndexByte(", \t\r\n", c) >= 0:
			i++
		case c == '#':
			end := strings.IndexByte(s[i:], '\n')
			if end < 0 {
				return nil
			}
			i += end
		case c == '\'' || c == '"':
			end := strings.IndexByte(s[i+1:], c)
			if end < 0 || strings.ContainsAny(s[i+1:i+1+end], "\\\n") {
				return nil
			}
			items = append(items, s[i+1:i+1+end])
			i += end + 2
		default:
			return nil
		}
	}
	return nil
}
